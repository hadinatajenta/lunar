package domain

import "testing"

func TestExtractDatabaseStatementsFindsActionsAndTables(t *testing.T) {
	content := `-- migration for the MDR log
CREATE TABLE IF NOT EXISTS log_program_mdr (
    id BIGINT PRIMARY KEY
);

/* widen the customer table */
ALTER TABLE customer ADD COLUMN mdr_enabled BOOLEAN;

INSERT INTO log_program_mdr (id) VALUES (1);
UPDATE customer SET mdr_enabled = true WHERE id = 1;
DELETE FROM customer WHERE id = 2;
DROP TABLE IF EXISTS legacy_log;`

	statements := ExtractDatabaseStatements(content)
	expected := []struct {
		action string
		table  string
	}{
		{ActionCreate, "log_program_mdr"},
		{ActionAlter, "customer"},
		{ActionInsert, "log_program_mdr"},
		{ActionUpdate, "customer"},
		{ActionDelete, "customer"},
		{ActionDrop, "legacy_log"},
	}
	if len(statements) != len(expected) {
		t.Fatalf("expected %d statements, got %d: %+v", len(expected), len(statements), statements)
	}
	for index, want := range expected {
		if statements[index].Action != want.action || statements[index].Table != want.table {
			t.Fatalf("statement %d: expected %s %s, got %s %s", index, want.action, want.table, statements[index].Action, statements[index].Table)
		}
	}
}

func TestExtractDatabaseStatementsIgnoresUnrelatedStatements(t *testing.T) {
	content := `SELECT * FROM customer;
CREATE INDEX idx_customer ON customer (id);
-- only a comment
/* block comment */`

	statements := ExtractDatabaseStatements(content)
	if len(statements) != 0 {
		t.Fatalf("expected no database statements, got %+v", statements)
	}
}

func TestExtractDatabaseStatementsStripsQuotedTableNames(t *testing.T) {
	statements := ExtractDatabaseStatements(`CREATE TABLE "my_table" (id BIGINT);`)
	if len(statements) != 1 {
		t.Fatalf("expected one statement, got %+v", statements)
	}
	if statements[0].Table != "my_table" {
		t.Fatalf("expected the unquoted table name, got %q", statements[0].Table)
	}
}

func TestExtractDatabaseStatementsBoundsTheSQL(t *testing.T) {
	statement := "CREATE TABLE big_table (" + repeatRunes('a', maxStatementLength+50) + ");"
	statements := ExtractDatabaseStatements(statement)
	if len(statements) != 1 {
		t.Fatalf("expected one statement, got %+v", statements)
	}
	if runes := []rune(statements[0].SQL); len(runes) > maxStatementLength {
		t.Fatalf("expected the SQL to be bounded to %d runes, got %d", maxStatementLength, len(runes))
	}
}

func TestMergeDatabaseTablesDeduplicates(t *testing.T) {
	files := []DatabaseFile{
		{Path: "a.sql", Statements: []DatabaseStatement{{Action: ActionCreate, Table: "alpha"}}},
		{Path: "b.sql", Statements: []DatabaseStatement{{Action: ActionCreate, Table: "alpha"}, {Action: ActionAlter, Table: "alpha"}}},
	}

	tables := MergeDatabaseTables(files)
	if len(tables) != 2 {
		t.Fatalf("expected two distinct table/action pairs, got %+v", tables)
	}
	if tables[0].Name != "alpha" || tables[0].Action != ActionCreate {
		t.Fatalf("unexpected first table %+v", tables[0])
	}
	if tables[1].Name != "alpha" || tables[1].Action != ActionAlter {
		t.Fatalf("unexpected second table %+v", tables[1])
	}
}

func TestIsDatabaseFilePath(t *testing.T) {
	testCases := []struct {
		path     string
		expected bool
	}{
		{"migrations/2026_add_mdr.sql", true},
		{"db/migrations/0001_init.sql", true},
		{"database/migrations/0001_init.up.sql", true},
		{"migration/0001_init.sql", true},
		{"internal/repo/migrations/runner.go", true},
		{"internal/repo/store.go", false},
		{"README.md", false},
		{"src/main.sql", true},
	}
	for _, testCase := range testCases {
		if got := IsDatabaseFilePath(testCase.path); got != testCase.expected {
			t.Fatalf("IsDatabaseFilePath(%q) = %v, expected %v", testCase.path, got, testCase.expected)
		}
	}
}

func repeatRunes(value rune, count int) string {
	runes := make([]rune, count)
	for index := range runes {
		runes[index] = value
	}
	return string(runes)
}
