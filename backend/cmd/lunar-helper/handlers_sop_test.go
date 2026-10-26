package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	sopinfrastructure "lunar/backend/internal/sop/infrastructure"
)

func newSOPScanScratchRepository(t *testing.T) (string, string) {
	t.Helper()
	requireGitTool(t)

	rootPath := t.TempDir()
	repositoryPath := filepath.Join(rootPath, "everest")

	writeUTDocsFile(t, filepath.Join(repositoryPath, "README.md"), "initial\n")

	runGitTool(t, repositoryPath, "init", "-q", "-b", "development")
	runGitTool(t, repositoryPath, "config", "user.email", "test@example.com")
	runGitTool(t, repositoryPath, "config", "user.name", "SOP Test")
	runGitTool(t, repositoryPath, "add", "-A")
	runGitTool(t, repositoryPath, "commit", "-q", "-m", "initial commit")
	runGitTool(t, repositoryPath, "checkout", "-q", "-b", "feature/CRMMS-123")

	writeUTDocsFile(t, filepath.Join(repositoryPath, "migrations/2026_add_mdr.sql"), "CREATE TABLE log_program_mdr (id BIGINT);\n")
	writeUTDocsFile(t, filepath.Join(repositoryPath, "internal/repo/store.go"), "package repo\n\nfunc Migrate(db DB) { db.AutoMigrate(&Log{}) }\n")
	runGitTool(t, repositoryPath, "add", "-A")
	runGitTool(t, repositoryPath, "commit", "-q", "-m", "feature change")
	runGitTool(t, repositoryPath, "checkout", "-q", "development")

	return rootPath, repositoryPath
}

func newSOPTestHandler(t *testing.T, allowedRoots []string) *sopHandler {
	t.Helper()

	return &sopHandler{
		allowedRoots: allowedRoots,
		scanner:      sopinfrastructure.NewScanner(),
		logger:       discardLogger(),
	}
}

func assertWorktreeClean(t *testing.T, repositoryPath string) {
	t.Helper()

	if status := strings.TrimSpace(runGitTool(t, repositoryPath, "status", "--porcelain")); status != "" {
		t.Fatalf("expected a clean worktree, got %q", status)
	}
}

func newSOPScanBody(rootPath string, branch string, baseRef string) string {
	return fmt.Sprintf(`{"root":%q,"repo":"everest","branch":%q,"base_ref":%q}`, rootPath, branch, baseRef)
}

func performSOPScan(t *testing.T, mux *http.ServeMux, body string, withToken bool) *httptest.ResponseRecorder {
	t.Helper()
	return performUTDocsRequest(t, mux, http.MethodPost, sopScanPath, body, withToken)
}

func TestHelperSOPScanFindsDatabaseChanges(t *testing.T) {
	rootPath, repositoryPath := newSOPScanScratchRepository(t)
	handler := newSOPTestHandler(t, []string{rootPath})
	mux := http.NewServeMux()
	registerSOPHandler(mux, testHelperToken, handler)

	recorder := performSOPScan(t, mux, newSOPScanBody(rootPath, "feature/CRMMS-123", "development"), true)
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d with body %s", recorder.Code, recorder.Body.String())
	}

	body := recorder.Body.String()
	for _, expected := range []string{
		`"repository":"everest"`,
		`"path":"migrations/2026_add_mdr.sql"`,
		`"action":"CREATE"`,
		`"table":"log_program_mdr"`,
		`"name":"log_program_mdr"`,
		"AutoMigrate found in internal/repo/store.go",
	} {
		if !strings.Contains(body, expected) {
			t.Fatalf("expected the response to contain %s, got %s", expected, body)
		}
	}

	currentBranch := strings.TrimSpace(runGitTool(t, repositoryPath, "rev-parse", "--abbrev-ref", "HEAD"))
	if currentBranch != "development" {
		t.Fatalf("expected the current branch to be unchanged, got %q", currentBranch)
	}
	assertWorktreeClean(t, repositoryPath)
}

func TestHelperSOPScanRefusesWithoutAllowedRoots(t *testing.T) {
	handler := newSOPTestHandler(t, nil)
	mux := http.NewServeMux()
	registerSOPHandler(mux, testHelperToken, handler)

	recorder := performSOPScan(t, mux, newSOPScanBody(t.TempDir(), "feature/CRMMS-123", "development"), true)
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d with body %s", recorder.Code, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), messageWriteAllowlist) {
		t.Fatalf("expected the allowlist message, got %s", recorder.Body.String())
	}
}

func TestHelperSOPScanRejectsInvalidBodies(t *testing.T) {
	rootPath, _ := newSOPScanScratchRepository(t)
	handler := newSOPTestHandler(t, []string{rootPath})
	mux := http.NewServeMux()
	registerSOPHandler(mux, testHelperToken, handler)

	testCases := []struct {
		name    string
		body    string
		message string
	}{
		{name: "malformed json", body: "{", message: messageSOPScanInvalidBody},
		{name: "unknown field", body: `{"root":"/tmp","unknown":true}`, message: messageSOPScanInvalidBody},
		{name: "missing fields", body: `{"root":"/tmp"}`, message: messageSOPScanFieldsRequired},
		{name: "invalid branch", body: newSOPScanBody(rootPath, "-bad", "development"), message: messageBranchNameInvalid},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			recorder := performSOPScan(t, mux, testCase.body, true)
			if recorder.Code != http.StatusBadRequest {
				t.Fatalf("expected 400, got %d with body %s", recorder.Code, recorder.Body.String())
			}
			if !strings.Contains(recorder.Body.String(), testCase.message) {
				t.Fatalf("expected %q, got %s", testCase.message, recorder.Body.String())
			}
		})
	}
}

func TestHelperSOPScanRequiresTheToken(t *testing.T) {
	rootPath, _ := newSOPScanScratchRepository(t)
	handler := newSOPTestHandler(t, []string{rootPath})
	mux := http.NewServeMux()
	registerSOPHandler(mux, testHelperToken, handler)

	recorder := performSOPScan(t, mux, newSOPScanBody(rootPath, "feature/CRMMS-123", "development"), false)
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without the token, got %d", recorder.Code)
	}
}

func newSOPRemoteOnlyScratchRepository(t *testing.T) (string, string) {
	t.Helper()
	requireGitTool(t)

	rootPath := t.TempDir()
	repositoryPath := filepath.Join(rootPath, "everest")

	writeUTDocsFile(t, filepath.Join(repositoryPath, "README.md"), "initial\n")
	runGitTool(t, repositoryPath, "init", "-q", "-b", "development")
	runGitTool(t, repositoryPath, "add", "-A")
	runGitTool(t, repositoryPath, "commit", "-q", "-m", "initial commit")

	runGitTool(t, repositoryPath, "checkout", "-q", "-b", "feature/CRMMS-REMOTE")
	writeUTDocsFile(t, filepath.Join(repositoryPath, "migrations/2026_remote.sql"), "CREATE TABLE remote_table (id BIGINT);\n")
	runGitTool(t, repositoryPath, "add", "-A")
	runGitTool(t, repositoryPath, "commit", "-q", "-m", "remote change")

	remoteHead := strings.TrimSpace(runGitTool(t, repositoryPath, "rev-parse", "HEAD"))
	runGitTool(t, repositoryPath, "checkout", "-q", "development")
	runGitTool(t, repositoryPath, "update-ref", "refs/remotes/origin/feature/CRMMS-REMOTE", remoteHead)
	runGitTool(t, repositoryPath, "branch", "-D", "feature/CRMMS-REMOTE")

	return rootPath, repositoryPath
}

func TestHelperSOPScanReadsARemoteOnlyBranch(t *testing.T) {
	rootPath, repositoryPath := newSOPRemoteOnlyScratchRepository(t)
	handler := newSOPTestHandler(t, []string{rootPath})
	mux := http.NewServeMux()
	registerSOPHandler(mux, testHelperToken, handler)

	recorder := performSOPScan(t, mux, newSOPScanBody(rootPath, "feature/CRMMS-REMOTE", "development"), true)
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d with body %s", recorder.Code, recorder.Body.String())
	}

	body := recorder.Body.String()
	for _, expected := range []string{
		`"path":"migrations/2026_remote.sql"`,
		`"table":"remote_table"`,
	} {
		if !strings.Contains(body, expected) {
			t.Fatalf("expected the response to contain %s, got %s", expected, body)
		}
	}

	currentBranch := strings.TrimSpace(runGitTool(t, repositoryPath, "rev-parse", "--abbrev-ref", "HEAD"))
	if currentBranch != "development" {
		t.Fatalf("expected HEAD to stay on development, got %q", currentBranch)
	}
	if localBranches := strings.TrimSpace(runGitTool(t, repositoryPath, "branch", "--list", "feature/CRMMS-REMOTE")); localBranches != "" {
		t.Fatalf("expected no local tracking branch, got %q", localBranches)
	}
	assertWorktreeClean(t, repositoryPath)
}

func TestHelperSOPScanSkipsAMissingBranch(t *testing.T) {
	rootPath, repositoryPath := newSOPScanScratchRepository(t)
	handler := newSOPTestHandler(t, []string{rootPath})
	mux := http.NewServeMux()
	registerSOPHandler(mux, testHelperToken, handler)

	recorder := performSOPScan(t, mux, newSOPScanBody(rootPath, "feature/DOES-NOT-EXIST", "development"), true)
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d with body %s", recorder.Code, recorder.Body.String())
	}

	body := recorder.Body.String()
	if !strings.Contains(body, "the branch feature/DOES-NOT-EXIST does not exist in this repository; the repository was skipped") {
		t.Fatalf("expected the missing branch warning, got %s", body)
	}
	if !strings.Contains(body, `"database_files":[]`) || !strings.Contains(body, `"tables":[]`) {
		t.Fatalf("expected empty database files and tables, got %s", body)
	}

	currentBranch := strings.TrimSpace(runGitTool(t, repositoryPath, "rev-parse", "--abbrev-ref", "HEAD"))
	if currentBranch != "development" {
		t.Fatalf("expected the original branch to remain checked out, got %q", currentBranch)
	}
	assertWorktreeClean(t, repositoryPath)
}

func TestHelperSOPScanReportsAGitFailure(t *testing.T) {
	rootPath, repositoryPath := newSOPScanScratchRepository(t)
	handler := newSOPTestHandler(t, []string{rootPath})
	mux := http.NewServeMux()
	registerSOPHandler(mux, testHelperToken, handler)

	recorder := performSOPScan(t, mux, newSOPScanBody(rootPath, "feature/CRMMS-123", "no-such-base"), true)
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d with body %s", recorder.Code, recorder.Body.String())
	}

	body := recorder.Body.String()
	if !strings.Contains(body, "the repository everest could not be scanned for branch feature/CRMMS-123") {
		t.Fatalf("expected the git failure warning, got %s", body)
	}
	if !strings.Contains(body, `"database_files":[]`) || !strings.Contains(body, `"tables":[]`) {
		t.Fatalf("expected empty database files and tables, got %s", body)
	}

	currentBranch := strings.TrimSpace(runGitTool(t, repositoryPath, "rev-parse", "--abbrev-ref", "HEAD"))
	if currentBranch != "development" {
		t.Fatalf("expected the original branch to remain checked out, got %q", currentBranch)
	}
	assertWorktreeClean(t, repositoryPath)
}
