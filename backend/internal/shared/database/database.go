package database

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	_ "modernc.org/sqlite"
)

const schemaDDL = `
CREATE TABLE IF NOT EXISTS users (
    id TEXT PRIMARY KEY,
    email TEXT UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    salt TEXT NOT NULL,
    full_name TEXT NOT NULL,
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);

CREATE TABLE IF NOT EXISTS user_secrets (
    user_id TEXT PRIMARY KEY,
    jira_pat_enc TEXT NOT NULL DEFAULT '',
    jira_username TEXT NOT NULL DEFAULT '',
    bitbucket_pat_enc TEXT NOT NULL DEFAULT '',
    bitbucket_username TEXT NOT NULL DEFAULT '',
    confluence_pat_enc TEXT NOT NULL DEFAULT '',
    ai_keys_json_enc TEXT NOT NULL DEFAULT '',
    updated_at DATETIME NOT NULL,
    FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS copilot_chats (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    title TEXT NOT NULL,
    model TEXT NOT NULL,
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL,
    FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_copilot_chats_user ON copilot_chats(user_id);

CREATE TABLE IF NOT EXISTS copilot_messages (
    id TEXT PRIMARY KEY,
    chat_id TEXT NOT NULL,
    role TEXT NOT NULL,
    content TEXT NOT NULL,
    reasoning TEXT NOT NULL DEFAULT '',
    thinking_duration_ms INTEGER NOT NULL DEFAULT 0,
    sources TEXT NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL,
    FOREIGN KEY(chat_id) REFERENCES copilot_chats(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_copilot_messages_chat ON copilot_messages(chat_id);

CREATE TABLE IF NOT EXISTS workspaces (
    user_id          TEXT PRIMARY KEY,
    source           TEXT NOT NULL,
    root_path        TEXT NOT NULL DEFAULT '',
    helper_url       TEXT NOT NULL DEFAULT '',
    helper_token_enc TEXT NOT NULL DEFAULT '',
    created_at       DATETIME NOT NULL,
    updated_at       DATETIME NOT NULL,
    FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS repo_selections (
    user_id      TEXT NOT NULL,
    repo_name    TEXT NOT NULL,
    is_selected  INTEGER NOT NULL DEFAULT 1,
    updated_at   DATETIME NOT NULL,
    PRIMARY KEY (user_id, repo_name),
    FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_repo_selections_user ON repo_selections(user_id);
`

func OpenDB(path string) (*sql.DB, error) {
	if path != ":memory:" && !strings.HasPrefix(path, "file:") {
		dir := filepath.Dir(path)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, err
		}
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	pragmas := []string{
		"PRAGMA journal_mode = WAL;",
		"PRAGMA busy_timeout = 5000;",
		"PRAGMA foreign_keys = ON;",
		"PRAGMA synchronous = NORMAL;",
	}

	for _, pragma := range pragmas {
		if _, err := db.Exec(pragma); err != nil {
			_ = db.Close()
			return nil, err
		}
	}

	return db, nil
}

type additiveColumn struct {
	Table      string
	Column     string
	Definition string
}

var additiveColumns = []additiveColumn{
	{
		Table:      "workspaces",
		Column:     "helper_token_enc",
		Definition: "TEXT NOT NULL DEFAULT ''",
	},
	{
		Table:      "copilot_messages",
		Column:     "thinking_duration_ms",
		Definition: "INTEGER NOT NULL DEFAULT 0",
	},
	{
		Table:      "copilot_messages",
		Column:     "sources",
		Definition: "TEXT NOT NULL DEFAULT ''",
	},
}

func hasColumn(db *sql.DB, table string, column string) (bool, error) {
	query := "SELECT COUNT(*) FROM pragma_table_info('" + table + "') WHERE name = ?"
	var count int
	if err := db.QueryRow(query, column).Scan(&count); err != nil {
		return false, err
	}
	return count > 0, nil
}

func addMissingColumns(db *sql.DB) error {
	for _, additive := range additiveColumns {
		present, err := hasColumn(db, additive.Table, additive.Column)
		if err != nil {
			return fmt.Errorf("inspect %s.%s: %w", additive.Table, additive.Column, err)
		}
		if present {
			continue
		}

		statement := fmt.Sprintf(
			"ALTER TABLE %s ADD COLUMN %s %s",
			additive.Table,
			additive.Column,
			additive.Definition,
		)
		if _, err := db.Exec(statement); err != nil {
			return fmt.Errorf("add %s.%s: %w", additive.Table, additive.Column, err)
		}
	}
	return nil
}

func AutoMigrate(db *sql.DB) error {
	if _, err := db.Exec(schemaDDL); err != nil {
		return err
	}
	return addMissingColumns(db)
}
