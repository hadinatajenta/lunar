package database

import (
	"database/sql"
	"testing"
	"time"
)

func openMigratedDatabase(t *testing.T) *sql.DB {
	t.Helper()

	db, err := OpenDB(t.TempDir() + "/schema.db")
	if err != nil {
		t.Fatalf("cannot open database: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})

	if err := AutoMigrate(db); err != nil {
		t.Fatalf("cannot migrate: %v", err)
	}

	return db
}

func tableExists(t *testing.T, db *sql.DB, name string) bool {
	t.Helper()

	var count int
	err := db.QueryRow(
		"SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = ?",
		name,
	).Scan(&count)
	if err != nil {
		t.Fatalf("cannot inspect sqlite_master: %v", err)
	}

	return count == 1
}

func TestAutoMigrateCreatesWorkspaceTables(t *testing.T) {
	db := openMigratedDatabase(t)

	for _, table := range []string{"workspaces", "repo_selections"} {
		if !tableExists(t, db, table) {
			t.Fatalf("expected table %q to exist after migration", table)
		}
	}
}

func TestAutoMigrateIsIdempotent(t *testing.T) {
	db := openMigratedDatabase(t)

	for attempt := 0; attempt < 3; attempt++ {
		if err := AutoMigrate(db); err != nil {
			t.Fatalf("migration attempt %d failed: %v", attempt, err)
		}
	}

	if !tableExists(t, db, "workspaces") || !tableExists(t, db, "repo_selections") {
		t.Fatal("tables must survive repeated migrations")
	}
}

func TestWorkspaceColumnsMatchContract(t *testing.T) {
	db := openMigratedDatabase(t)

	expected := []string{"user_id", "source", "root_path", "helper_url", "helper_token_enc", "created_at", "updated_at"}
	rows, err := db.Query("SELECT name FROM pragma_table_info('workspaces')")
	if err != nil {
		t.Fatalf("cannot read workspaces columns: %v", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	present := map[string]bool{}
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatalf("cannot scan column name: %v", err)
		}
		present[name] = true
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("row iteration failed: %v", err)
	}

	for _, column := range expected {
		if !present[column] {
			t.Fatalf("workspaces is missing column %q", column)
		}
	}
}

func TestRepoSelectionColumnsMatchContract(t *testing.T) {
	db := openMigratedDatabase(t)

	expected := []string{"user_id", "repo_name", "is_selected", "updated_at"}
	rows, err := db.Query("SELECT name FROM pragma_table_info('repo_selections')")
	if err != nil {
		t.Fatalf("cannot read repo_selections columns: %v", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	present := map[string]bool{}
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatalf("cannot scan column name: %v", err)
		}
		present[name] = true
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("row iteration failed: %v", err)
	}

	for _, column := range expected {
		if !present[column] {
			t.Fatalf("repo_selections is missing column %q", column)
		}
	}
}

func TestRepoSelectionPrimaryKeyPreventsDuplicates(t *testing.T) {
	db := openMigratedDatabase(t)

	now := time.Now().UTC()
	if _, err := db.Exec(
		"INSERT INTO users (id, email, password_hash, salt, full_name, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)",
		"user-a", "user-a@example.com", "hash", "salt", "User A", now, now,
	); err != nil {
		t.Fatalf("cannot seed user: %v", err)
	}

	insert := "INSERT INTO repo_selections (user_id, repo_name, is_selected, updated_at) VALUES (?, ?, ?, ?)"
	if _, err := db.Exec(insert, "user-a", "everest", 1, now); err != nil {
		t.Fatalf("cannot insert selection: %v", err)
	}
	if _, err := db.Exec(insert, "user-a", "everest", 1, now); err == nil {
		t.Fatal("expected a duplicate composite key to be rejected")
	}
}

func TestWorkspaceForeignKeysCascade(t *testing.T) {
	db := openMigratedDatabase(t)

	now := time.Now().UTC()
	if _, err := db.Exec(
		"INSERT INTO users (id, email, password_hash, salt, full_name, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)",
		"user-a", "user-a@example.com", "hash", "salt", "User A", now, now,
	); err != nil {
		t.Fatalf("cannot seed user: %v", err)
	}
	if _, err := db.Exec(
		"INSERT INTO workspaces (user_id, source, root_path, helper_url, helper_token_enc, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)",
		"user-a", "helper", "/Users/erendt/BRI", "", "", now, now,
	); err != nil {
		t.Fatalf("cannot insert workspace: %v", err)
	}
	if _, err := db.Exec("INSERT INTO repo_selections (user_id, repo_name, is_selected, updated_at) VALUES (?, ?, ?, ?)", "user-a", "everest", 1, now); err != nil {
		t.Fatalf("cannot insert selection: %v", err)
	}

	if _, err := db.Exec("DELETE FROM users WHERE id = ?", "user-a"); err != nil {
		t.Fatalf("cannot delete user: %v", err)
	}

	var workspaceCount int
	if err := db.QueryRow("SELECT COUNT(*) FROM workspaces").Scan(&workspaceCount); err != nil {
		t.Fatalf("cannot count workspaces: %v", err)
	}
	if workspaceCount != 0 {
		t.Fatalf("expected workspace to cascade, found %d rows", workspaceCount)
	}

	var selectionCount int
	if err := db.QueryRow("SELECT COUNT(*) FROM repo_selections").Scan(&selectionCount); err != nil {
		t.Fatalf("cannot count selections: %v", err)
	}
	if selectionCount != 0 {
		t.Fatalf("expected selections to cascade, found %d rows", selectionCount)
	}
}

func TestWorkspaceForeignKeysRejectUnknownUser(t *testing.T) {
	db := openMigratedDatabase(t)

	now := time.Now().UTC()
	_, err := db.Exec(
		"INSERT INTO workspaces (user_id, source, root_path, helper_url, helper_token_enc, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)",
		"unknown-user", "helper", "/Users/erendt/BRI", "", "", now, now,
	)
	if err == nil {
		t.Fatal("expected a foreign key violation for an unknown user")
	}
}

func TestAutoMigrateAddsMissingColumnToExistingTable(t *testing.T) {
	db := openMigratedDatabase(t)

	if _, err := db.Exec("DROP TABLE workspaces"); err != nil {
		t.Fatalf("cannot drop table: %v", err)
	}

	legacyDDL := `
		CREATE TABLE workspaces (
			user_id    TEXT PRIMARY KEY,
			source     TEXT NOT NULL,
			root_path  TEXT NOT NULL DEFAULT '',
			helper_url TEXT NOT NULL DEFAULT '',
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL
		)
	`
	if _, err := db.Exec(legacyDDL); err != nil {
		t.Fatalf("cannot create legacy table: %v", err)
	}

	now := time.Now().UTC()
	if _, err := db.Exec(
		"INSERT INTO users (id, email, password_hash, salt, full_name, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)",
		"legacy-user", "legacy@example.com", "hash", "salt", "Legacy", now, now,
	); err != nil {
		t.Fatalf("cannot seed user: %v", err)
	}
	if _, err := db.Exec(
		"INSERT INTO workspaces (user_id, source, root_path, helper_url, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)",
		"legacy-user", "helper", "/Users/erendt/BRI", "", now, now,
	); err != nil {
		t.Fatalf("cannot insert legacy row: %v", err)
	}

	if err := AutoMigrate(db); err != nil {
		t.Fatalf("migration over legacy schema failed: %v", err)
	}

	present, err := hasColumn(db, "workspaces", "helper_token_enc")
	if err != nil {
		t.Fatalf("cannot inspect column: %v", err)
	}
	if !present {
		t.Fatal("expected helper_token_enc to be added to the legacy table")
	}

	var rootPath string
	var token string
	if err := db.QueryRow(
		"SELECT root_path, helper_token_enc FROM workspaces WHERE user_id = ?",
		"legacy-user",
	).Scan(&rootPath, &token); err != nil {
		t.Fatalf("cannot read migrated row: %v", err)
	}
	if rootPath != "/Users/erendt/BRI" {
		t.Fatalf("existing data must survive migration, got %q", rootPath)
	}
	if token != "" {
		t.Fatalf("expected an empty default for the new column, got %q", token)
	}
}

func TestAutoMigrateIsIdempotentOnAdditiveColumns(t *testing.T) {
	db := openMigratedDatabase(t)

	for attempt := 0; attempt < 3; attempt++ {
		if err := AutoMigrate(db); err != nil {
			t.Fatalf("attempt %d failed: %v", attempt, err)
		}
	}

	present, err := hasColumn(db, "workspaces", "helper_token_enc")
	if err != nil {
		t.Fatalf("cannot inspect column: %v", err)
	}
	if !present {
		t.Fatal("expected helper_token_enc to remain present")
	}
}
