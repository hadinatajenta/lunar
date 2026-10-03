package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestOpenIndexDatabaseCreatesRestrictedPermissions(t *testing.T) {
	baseDir := t.TempDir()

	database, err := openIndexDatabase(baseDir)
	if err != nil {
		t.Fatalf("openIndexDatabase returned error: %v", err)
	}
	defer database.Close()

	directoryInfo, err := os.Stat(filepath.Join(baseDir, "index"))
	if err != nil {
		t.Fatalf("stat index directory: %v", err)
	}
	if permissions := directoryInfo.Mode().Perm(); permissions != 0o700 {
		t.Fatalf("expected index directory permissions 0700, got %o", permissions)
	}

	databaseInfo, err := os.Stat(filepath.Join(baseDir, "index", indexDatabaseFileName))
	if err != nil {
		t.Fatalf("stat index database: %v", err)
	}
	if permissions := databaseInfo.Mode().Perm(); permissions != 0o600 {
		t.Fatalf("expected index database permissions 0600, got %o", permissions)
	}
}

func TestOpenIndexDatabaseAppliesEveryTable(t *testing.T) {
	baseDir := t.TempDir()

	database, err := openIndexDatabase(baseDir)
	if err != nil {
		t.Fatalf("openIndexDatabase returned error: %v", err)
	}
	defer database.Close()

	expectedTables := []string{
		"service_nodes",
		"service_deps",
		"code_files",
		"code_chunks_fts",
		"domain_knowledge_fts",
	}

	for _, tableName := range expectedTables {
		var found int
		query := `SELECT COUNT(*) FROM sqlite_master WHERE name = ?`
		if err := database.QueryRow(query, tableName).Scan(&found); err != nil {
			t.Fatalf("query sqlite_master for %s: %v", tableName, err)
		}
		if found != 1 {
			t.Fatalf("expected table %s to exist, found %d rows", tableName, found)
		}
	}
}

func TestIndexSchemaSupportsFullTextSearch(t *testing.T) {
	baseDir := t.TempDir()

	database, err := openIndexDatabase(baseDir)
	if err != nil {
		t.Fatalf("openIndexDatabase returned error: %v", err)
	}
	defer database.Close()

	insert := `INSERT INTO code_chunks_fts(repo_name, file_path, chunk_type, raw_content, content)
	           VALUES (?, ?, ?, ?, ?)`
	if _, err := database.Exec(
		insert,
		"settlement-service",
		"internal/fee/calculator.go",
		"code_block",
		"func CalculateLateFee() {}",
		"func calculate late fee calculator",
	); err != nil {
		t.Fatalf("insert chunk: %v", err)
	}

	var repoName string
	var filePath string
	search := `SELECT repo_name, file_path FROM code_chunks_fts
	           WHERE code_chunks_fts MATCH ? ORDER BY bm25(code_chunks_fts) LIMIT 1`
	if err := database.QueryRow(search, "late fee").Scan(&repoName, &filePath); err != nil {
		t.Fatalf("full text search failed: %v", err)
	}

	if repoName != "settlement-service" {
		t.Fatalf("expected settlement-service, got %s", repoName)
	}
	if filePath != "internal/fee/calculator.go" {
		t.Fatalf("expected internal/fee/calculator.go, got %s", filePath)
	}
}

func TestIndexSchemaIsIdempotent(t *testing.T) {
	baseDir := t.TempDir()

	first, err := openIndexDatabase(baseDir)
	if err != nil {
		t.Fatalf("first open failed: %v", err)
	}
	if err := first.Close(); err != nil {
		t.Fatalf("close first database: %v", err)
	}

	second, err := openIndexDatabase(baseDir)
	if err != nil {
		t.Fatalf("second open failed: %v", err)
	}
	defer second.Close()

	var tableCount int
	if err := second.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type = 'table'`).Scan(&tableCount); err != nil {
		t.Fatalf("count tables: %v", err)
	}
	if tableCount < 5 {
		t.Fatalf("expected at least 5 tables after reopening, got %d", tableCount)
	}
}
