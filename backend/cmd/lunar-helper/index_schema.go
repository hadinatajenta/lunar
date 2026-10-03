package main

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	codeindexinfrastructure "lunar/backend/internal/codeindex/infrastructure"

	_ "modernc.org/sqlite"
)

const indexDatabaseFileName = "index.db"

const indexSchemaVersion = 2

const indexSchemaSQL = `
CREATE TABLE IF NOT EXISTS service_nodes (
    repo_name    TEXT PRIMARY KEY,
    language     TEXT NOT NULL DEFAULT '',
    chunk_count  INTEGER NOT NULL DEFAULT 0,
    indexed_at   TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);

CREATE TABLE IF NOT EXISTS service_deps (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    from_service TEXT NOT NULL,
    to_service   TEXT NOT NULL,
    call_type    TEXT NOT NULL DEFAULT '',
    endpoint     TEXT NOT NULL DEFAULT '',
    topic        TEXT NOT NULL DEFAULT '',
    source_file  TEXT NOT NULL DEFAULT '',
    UNIQUE(from_service, to_service, call_type, endpoint, topic)
);

CREATE VIRTUAL TABLE IF NOT EXISTS code_chunks_fts USING fts5(
    repo_name    UNINDEXED,
    file_path    UNINDEXED,
    chunk_type   UNINDEXED,
    raw_content  UNINDEXED,
    start_line   UNINDEXED,
    end_line     UNINDEXED,
    content,
    tokenize = 'porter unicode61'
);

CREATE VIRTUAL TABLE IF NOT EXISTS domain_knowledge_fts USING fts5(
    term,
    explanation,
    tokenize = 'porter unicode61'
);

CREATE TABLE IF NOT EXISTS code_files (
    repo_name      TEXT NOT NULL,
    file_path      TEXT NOT NULL,
    content_hash   TEXT NOT NULL,
    size_bytes     INTEGER NOT NULL,
    modified_unix  INTEGER NOT NULL,
    indexed_at     TEXT NOT NULL,
    PRIMARY KEY (repo_name, file_path)
);
`

func openHelperIndex() (*codeindexinfrastructure.Store, error) {
	homeDirectory, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("resolveHomeDirectory: %w", err)
	}

	database, err := openIndexDatabase(filepath.Join(homeDirectory, helperDirectoryName))
	if err != nil {
		return nil, err
	}

	return codeindexinfrastructure.NewStore(database), nil
}

func openIndexDatabase(baseDir string) (*sql.DB, error) {
	directory := filepath.Join(baseDir, "index")
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return nil, fmt.Errorf("createIndexDirectory: %w", err)
	}
	if err := os.Chmod(directory, 0o700); err != nil {
		return nil, fmt.Errorf("setIndexDirectoryPermissions: %w", err)
	}

	databasePath := filepath.Join(directory, indexDatabaseFileName)
	database, err := sql.Open("sqlite", databasePath)
	if err != nil {
		return nil, fmt.Errorf("openIndexDatabase: %w", err)
	}

	database.SetMaxOpenConns(4)
	database.SetMaxIdleConns(4)
	database.SetConnMaxLifetime(0)

	if err := applyIndexSchema(database); err != nil {
		_ = database.Close()
		return nil, err
	}

	if err := os.Chmod(databasePath, 0o600); err != nil {
		_ = database.Close()
		return nil, fmt.Errorf("setIndexDatabasePermissions: %w", err)
	}

	return database, nil
}

func applyIndexSchema(database *sql.DB) error {
	if _, err := database.Exec("PRAGMA journal_mode=WAL"); err != nil {
		return fmt.Errorf("enableWriteAheadLogging: %w", err)
	}
	if _, err := database.Exec("PRAGMA busy_timeout=5000"); err != nil {
		return fmt.Errorf("setBusyTimeout: %w", err)
	}
	if err := resetIndexSchemaOnVersionChange(database); err != nil {
		return err
	}
	if _, err := database.Exec(indexSchemaSQL); err != nil {
		return fmt.Errorf("applyIndexSchema: %w", err)
	}
	return recordIndexSchemaVersion(database)
}

func resetIndexSchemaOnVersionChange(database *sql.DB) error {
	var version int
	if err := database.QueryRow("PRAGMA user_version").Scan(&version); err != nil {
		return fmt.Errorf("readIndexSchemaVersion: %w", err)
	}
	if version == indexSchemaVersion {
		return nil
	}

	resetStatements := []string{
		"DROP TABLE IF EXISTS code_chunks_fts",
		"DROP TABLE IF EXISTS domain_knowledge_fts",
		"DROP TABLE IF EXISTS service_nodes",
		"DROP TABLE IF EXISTS service_deps",
		"DROP TABLE IF EXISTS code_files",
	}
	for _, statement := range resetStatements {
		if _, err := database.Exec(statement); err != nil {
			return fmt.Errorf("resetIndexSchema: %w", err)
		}
	}
	return nil
}

func recordIndexSchemaVersion(database *sql.DB) error {
	statement := fmt.Sprintf("PRAGMA user_version = %d", indexSchemaVersion)
	if _, err := database.Exec(statement); err != nil {
		return fmt.Errorf("writeIndexSchemaVersion: %w", err)
	}
	return nil
}
