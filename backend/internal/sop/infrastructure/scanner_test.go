package infrastructure

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	sopdomain "lunar/backend/internal/sop/domain"
)

func requireSOPGit(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not available")
	}
}

func runSOPGitTest(t *testing.T, repositoryPath string, args ...string) string {
	t.Helper()
	command := exec.Command("git", append([]string{"-C", repositoryPath}, args...)...)
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s failed: %v: %s", strings.Join(args, " "), err, string(output))
	}
	return string(output)
}

func writeSOPFixture(t *testing.T, fullPath string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		t.Fatalf("cannot create the fixture directory: %v", err)
	}
	if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
		t.Fatalf("cannot write the fixture file: %v", err)
	}
}

func newSOPScratchRepository(t *testing.T) string {
	t.Helper()
	requireSOPGit(t)

	repositoryPath := filepath.Join(t.TempDir(), "everest")
	writeSOPFixture(t, filepath.Join(repositoryPath, "README.md"), "initial\n")

	runSOPGitTest(t, repositoryPath, "init", "-q", "-b", "development")
	runSOPGitTest(t, repositoryPath, "config", "user.email", "test@example.com")
	runSOPGitTest(t, repositoryPath, "config", "user.name", "SOP Test")
	runSOPGitTest(t, repositoryPath, "add", "-A")
	runSOPGitTest(t, repositoryPath, "commit", "-q", "-m", "initial")
	runSOPGitTest(t, repositoryPath, "checkout", "-q", "-b", "feature/CRMMS-1")

	writeSOPFixture(t, filepath.Join(repositoryPath, "migrations/2026_add_mdr.sql"), "CREATE TABLE log_program_mdr (id BIGINT);\nALTER TABLE customer ADD COLUMN mdr BOOLEAN;\n")
	writeSOPFixture(t, filepath.Join(repositoryPath, "internal/repo/store.go"), "package repo\n\nfunc Migrate(db DB) { db.AutoMigrate(&Log{}) }\n")
	runSOPGitTest(t, repositoryPath, "add", "-A")
	runSOPGitTest(t, repositoryPath, "commit", "-q", "-m", "feature change")

	return repositoryPath
}

func TestScannerFindsDatabaseChangesAndORMMigration(t *testing.T) {
	repositoryPath := newSOPScratchRepository(t)
	scanner := NewScanner()

	scan, err := scanner.Scan(context.Background(), sopdomain.ScanInput{
		RepositoryPath: repositoryPath,
		Repository:     "everest",
		Branch:         "feature/CRMMS-1",
		BaseRef:        "development",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(scan.DatabaseFiles) != 1 || scan.DatabaseFiles[0].Path != "migrations/2026_add_mdr.sql" {
		t.Fatalf("expected the migration file, got %+v", scan.DatabaseFiles)
	}
	if len(scan.DatabaseFiles[0].Statements) != 2 {
		t.Fatalf("expected two statements, got %+v", scan.DatabaseFiles[0].Statements)
	}
	if scan.DatabaseFiles[0].Statements[0].Action != sopdomain.ActionCreate || scan.DatabaseFiles[0].Statements[0].Table != "log_program_mdr" {
		t.Fatalf("unexpected first statement %+v", scan.DatabaseFiles[0].Statements[0])
	}
	if scan.DatabaseFiles[0].Statements[1].Action != sopdomain.ActionAlter || scan.DatabaseFiles[0].Statements[1].Table != "customer" {
		t.Fatalf("unexpected second statement %+v", scan.DatabaseFiles[0].Statements[1])
	}

	joinedWarnings := strings.Join(scan.Warnings, "|")
	if !strings.Contains(joinedWarnings, "AutoMigrate found in internal/repo/store.go") {
		t.Fatalf("expected the ORM migration warning, got %+v", scan.Warnings)
	}
}

func TestScannerBoundsTheStatementsPerFile(t *testing.T) {
	requireSOPGit(t)

	repositoryPath := filepath.Join(t.TempDir(), "everest")
	writeSOPFixture(t, filepath.Join(repositoryPath, "README.md"), "initial\n")
	runSOPGitTest(t, repositoryPath, "init", "-q", "-b", "development")
	runSOPGitTest(t, repositoryPath, "config", "user.email", "test@example.com")
	runSOPGitTest(t, repositoryPath, "config", "user.name", "SOP Test")
	runSOPGitTest(t, repositoryPath, "add", "-A")
	runSOPGitTest(t, repositoryPath, "commit", "-q", "-m", "initial")
	runSOPGitTest(t, repositoryPath, "checkout", "-q", "-b", "feature/CRMMS-1")

	statementCount := sopMaxStatementsPerFile + 5
	builder := strings.Builder{}
	for index := 0; index < statementCount; index++ {
		builder.WriteString("INSERT INTO log_program_mdr (id) VALUES (1);\n")
	}
	writeSOPFixture(t, filepath.Join(repositoryPath, "migrations/bulk.sql"), builder.String())
	runSOPGitTest(t, repositoryPath, "add", "-A")
	runSOPGitTest(t, repositoryPath, "commit", "-q", "-m", "bulk change")

	scan, err := NewScanner().Scan(context.Background(), sopdomain.ScanInput{
		RepositoryPath: repositoryPath,
		Repository:     "everest",
		Branch:         "feature/CRMMS-1",
		BaseRef:        "development",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(scan.DatabaseFiles) != 1 {
		t.Fatalf("expected one database file, got %+v", scan.DatabaseFiles)
	}
	if len(scan.DatabaseFiles[0].Statements) != sopMaxStatementsPerFile {
		t.Fatalf("expected the statements to be bounded to %d, got %d", sopMaxStatementsPerFile, len(scan.DatabaseFiles[0].Statements))
	}
	if !strings.Contains(strings.Join(scan.Warnings, "|"), "statements; the extra statements were dropped") {
		t.Fatalf("expected a bound warning, got %+v", scan.Warnings)
	}
}

func TestScannerIgnoresUnrelatedChangedFiles(t *testing.T) {
	requireSOPGit(t)

	repositoryPath := filepath.Join(t.TempDir(), "everest")
	writeSOPFixture(t, filepath.Join(repositoryPath, "README.md"), "initial\n")
	runSOPGitTest(t, repositoryPath, "init", "-q", "-b", "development")
	runSOPGitTest(t, repositoryPath, "config", "user.email", "test@example.com")
	runSOPGitTest(t, repositoryPath, "config", "user.name", "SOP Test")
	runSOPGitTest(t, repositoryPath, "add", "-A")
	runSOPGitTest(t, repositoryPath, "commit", "-q", "-m", "initial")
	runSOPGitTest(t, repositoryPath, "checkout", "-q", "-b", "feature/CRMMS-1")

	writeSOPFixture(t, filepath.Join(repositoryPath, "internal/repo/service.go"), "package repo\n\nfunc Find() int { return 1 }\n")
	runSOPGitTest(t, repositoryPath, "add", "-A")
	runSOPGitTest(t, repositoryPath, "commit", "-q", "-m", "service change")

	scan, err := NewScanner().Scan(context.Background(), sopdomain.ScanInput{
		RepositoryPath: repositoryPath,
		Repository:     "everest",
		Branch:         "feature/CRMMS-1",
		BaseRef:        "development",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(scan.DatabaseFiles) != 0 || len(scan.Warnings) != 0 {
		t.Fatalf("expected no database changes and no warnings, got files=%+v warnings=%+v", scan.DatabaseFiles, scan.Warnings)
	}
}

func TestResolveRefsReportsLocalRemoteAndMissing(t *testing.T) {
	repositoryPath := newSOPScratchRepository(t)

	head, found, err := ResolveHeadRef(context.Background(), repositoryPath, "feature/CRMMS-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !found || head != "feature/CRMMS-1" {
		t.Fatalf("expected the local head ref, got head=%q found=%v", head, found)
	}

	_, found, err = ResolveHeadRef(context.Background(), repositoryPath, "feature/absent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if found {
		t.Fatal("expected a missing head ref")
	}

	base, err := ResolveBaseRef(context.Background(), repositoryPath, "development")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if base != "development" {
		t.Fatalf("expected the local base ref, got %q", base)
	}

	fallback, err := ResolveBaseRef(context.Background(), repositoryPath, "no-such-base")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fallback != "no-such-base" {
		t.Fatalf("expected the base ref as given, got %q", fallback)
	}
}

func TestResolveHeadRefPrefersTheRemoteTrackingRef(t *testing.T) {
	requireSOPGit(t)

	repositoryPath := filepath.Join(t.TempDir(), "everest")
	writeSOPFixture(t, filepath.Join(repositoryPath, "README.md"), "initial\n")
	runSOPGitTest(t, repositoryPath, "init", "-q", "-b", "development")
	runSOPGitTest(t, repositoryPath, "config", "user.email", "test@example.com")
	runSOPGitTest(t, repositoryPath, "config", "user.name", "SOP Test")
	runSOPGitTest(t, repositoryPath, "add", "-A")
	runSOPGitTest(t, repositoryPath, "commit", "-q", "-m", "initial")
	head := strings.TrimSpace(runSOPGitTest(t, repositoryPath, "rev-parse", "HEAD"))
	runSOPGitTest(t, repositoryPath, "update-ref", "refs/remotes/origin/feature/remote", head)

	resolved, found, err := ResolveHeadRef(context.Background(), repositoryPath, "feature/remote")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !found || resolved != "origin/feature/remote" {
		t.Fatalf("expected the remote head ref, got head=%q found=%v", resolved, found)
	}
}
