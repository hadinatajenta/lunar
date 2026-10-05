package transport

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	sharedAuth "lunar/backend/internal/shared/auth"
	sharedDatabase "lunar/backend/internal/shared/database"
	"lunar/backend/internal/workspace/application"
	"lunar/backend/internal/workspace/infrastructure"
)

func requireGitTool(t *testing.T) {
	t.Helper()

	if _, err := exec.LookPath("git"); err != nil {
		t.Skipf("git executable is not available: %v", err)
	}
}

func runGitTool(t *testing.T, repositoryPath string, args ...string) string {
	t.Helper()

	commandArguments := append([]string{"--no-optional-locks", "-C", repositoryPath}, args...)
	command := exec.Command("git", commandArguments...)
	command.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=Lunar Test",
		"GIT_AUTHOR_EMAIL=lunar@test.local",
		"GIT_COMMITTER_NAME=Lunar Test",
		"GIT_COMMITTER_EMAIL=lunar@test.local",
		"GIT_CONFIG_NOSYSTEM=1",
	)

	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v: %s", args, err, output)
	}
	return string(output)
}

func writeGitFixtureFile(t *testing.T, path string, content string) {
	t.Helper()

	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("cannot write git fixture file %s: %v", path, err)
	}
}

func newScratchGitWorkspace(t *testing.T) (string, string) {
	t.Helper()
	requireGitTool(t)

	rootPath := t.TempDir()
	repositoryPath := filepath.Join(rootPath, "scratch")
	if err := os.MkdirAll(repositoryPath, 0o755); err != nil {
		t.Fatalf("cannot create the scratch repository directory: %v", err)
	}
	runGitTool(t, repositoryPath, "init", "-q", "-b", "main", ".")
	writeGitFixtureFile(t, filepath.Join(repositoryPath, "README.md"), "initial\n")
	runGitTool(t, repositoryPath, "add", "-A")
	runGitTool(t, repositoryPath, "commit", "-q", "-m", "initial commit")
	runGitTool(t, repositoryPath, "checkout", "-q", "-b", "feature/second")

	return rootPath, repositoryPath
}

func setupWorkspaceGitHandler(t *testing.T) (http.Handler, string, string) {
	t.Helper()

	db, err := sharedDatabase.OpenDB(filepath.Join(t.TempDir(), "handler-git.db"))
	if err != nil {
		t.Fatalf("cannot open test database: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})

	if err := sharedDatabase.AutoMigrate(db); err != nil {
		t.Fatalf("cannot migrate test database: %v", err)
	}

	seedHandlerTestUsers(t, db)

	rootPath, repositoryPath := newScratchGitWorkspace(t)
	inspector := infrastructure.NewGitInspector([]string{rootPath})
	workspaceRepo := infrastructure.NewSQLiteWorkspaceRepository(db)
	selectionRepo := infrastructure.NewSQLiteSelectionRepository(db)
	service := application.NewWorkspaceService(workspaceRepo, selectionRepo, inspector, inspector, []string{rootPath}, handlerTestEncryptKey)
	handler := NewWorkspaceHandler(service)

	authMiddleware := sharedAuth.RequireAuth(handlerTestJWTSecret)
	mux := http.NewServeMux()
	mux.Handle("PUT /api/workspace", authMiddleware(http.HandlerFunc(handler.SaveWorkspace)))
	mux.Handle("GET /api/workspace/branches", authMiddleware(http.HandlerFunc(handler.ListBranches)))
	mux.Handle("POST /api/workspace/checkout", authMiddleware(http.HandlerFunc(handler.CheckoutBranch)))
	mux.Handle("POST /api/workspace/fetch", authMiddleware(http.HandlerFunc(handler.FetchRepository)))
	mux.Handle("POST /api/workspace/pull", authMiddleware(http.HandlerFunc(handler.PullRepository)))
	mux.Handle("POST /api/workspace/stash", authMiddleware(http.HandlerFunc(handler.StashRepository)))
	mux.Handle("POST /api/workspace/discard", authMiddleware(http.HandlerFunc(handler.DiscardRepository)))

	return mux, rootPath, repositoryPath
}

func saveServerWorkspace(t *testing.T, handler http.Handler, rootPath string) {
	t.Helper()

	requestBody, err := json.Marshal(saveWorkspaceRequest{Source: "server", RootPath: rootPath})
	if err != nil {
		t.Fatalf("cannot encode the save workspace request: %v", err)
	}

	recorder := performRequest(handler, authorizedRequest(t, http.MethodPut, "/api/workspace", string(requestBody), handlerTestUserA))
	if recorder.Code != http.StatusOK {
		t.Fatalf("cannot save the server workspace: %d %s", recorder.Code, recorder.Body.String())
	}
}

func postGitRequest(t *testing.T, handler http.Handler, path string, requestBody any) *httptest.ResponseRecorder {
	t.Helper()

	encodedBody, err := json.Marshal(requestBody)
	if err != nil {
		t.Fatalf("cannot encode %s request: %v", path, err)
	}

	return performRequest(handler, authorizedRequest(t, http.MethodPost, path, string(encodedBody), handlerTestUserA))
}

func decodeBranchesResponse(t *testing.T, recorder *httptest.ResponseRecorder) branchesResponse {
	t.Helper()

	var response branchesResponse
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("cannot decode branches response: %v", err)
	}
	return response
}

func TestWorkspaceHandler_BranchesListsCurrentFirst(t *testing.T) {
	handler, rootPath, _ := setupWorkspaceGitHandler(t)
	saveServerWorkspace(t, handler, rootPath)

	recorder := performRequest(handler, authorizedRequest(t, http.MethodGet, "/api/workspace/branches?name=scratch", "", handlerTestUserA))
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", recorder.Code, recorder.Body.String())
	}

	response := decodeBranchesResponse(t, recorder)
	if len(response.Branches) != 2 {
		t.Fatalf("expected 2 branches, got %+v", response.Branches)
	}
	if response.Branches[0].Name != "feature/second" || !response.Branches[0].IsCurrent {
		t.Fatalf("expected the current branch first, got %+v", response.Branches)
	}
	if response.Branches[1].Name != "main" {
		t.Fatalf("expected main second, got %+v", response.Branches)
	}
	if response.Branches[0].UpdatedRelative == "" {
		t.Fatal("expected a relative update time for the current branch")
	}
}

func TestWorkspaceHandler_CheckoutSwitchesAndRefusesDirtyTree(t *testing.T) {
	handler, rootPath, repositoryPath := setupWorkspaceGitHandler(t)
	saveServerWorkspace(t, handler, rootPath)

	checkoutRecorder := postGitRequest(t, handler, "/api/workspace/checkout", checkoutBranchRequest{Name: "scratch", Branch: "main"})
	if checkoutRecorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", checkoutRecorder.Code, checkoutRecorder.Body.String())
	}

	var checkout checkoutBranchResponse
	if err := json.NewDecoder(checkoutRecorder.Body).Decode(&checkout); err != nil {
		t.Fatalf("cannot decode checkout response: %v", err)
	}
	if checkout.Branch != "main" {
		t.Fatalf("expected the response branch main, got %q", checkout.Branch)
	}

	currentBranch := strings.TrimSpace(runGitTool(t, repositoryPath, "rev-parse", "--abbrev-ref", "HEAD"))
	if currentBranch != "main" {
		t.Fatalf("expected the repository to switch to main, got %q", currentBranch)
	}

	writeGitFixtureFile(t, filepath.Join(repositoryPath, "README.md"), "dirty\n")

	dirtyRecorder := postGitRequest(t, handler, "/api/workspace/checkout", checkoutBranchRequest{Name: "scratch", Branch: "feature/second"})
	if dirtyRecorder.Code != http.StatusConflict {
		t.Fatalf("expected 409 for a dirty tree, got %d: %s", dirtyRecorder.Code, dirtyRecorder.Body.String())
	}

	errorResponse := decodeErrorResponse(t, dirtyRecorder)
	if !strings.Contains(errorResponse.Error, "uncommitted changes") {
		t.Fatalf("expected a typed dirty tree message, got %q", errorResponse.Error)
	}
	if strings.Contains(dirtyRecorder.Body.String(), rootPath) {
		t.Fatalf("expected no absolute filesystem path in the error, got %s", dirtyRecorder.Body.String())
	}

	currentBranch = strings.TrimSpace(runGitTool(t, repositoryPath, "rev-parse", "--abbrev-ref", "HEAD"))
	if currentBranch != "main" {
		t.Fatalf("expected the refused checkout to stay on main, got %q", currentBranch)
	}
}

func TestWorkspaceHandler_CheckoutRejectsUnsafeBranchName(t *testing.T) {
	handler, rootPath, _ := setupWorkspaceGitHandler(t)
	saveServerWorkspace(t, handler, rootPath)

	recorder := postGitRequest(t, handler, "/api/workspace/checkout", checkoutBranchRequest{Name: "scratch", Branch: "-x"})
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", recorder.Code, recorder.Body.String())
	}

	errorResponse := decodeErrorResponse(t, recorder)
	if !strings.Contains(errorResponse.Error, "branch name is invalid") {
		t.Fatalf("expected a typed branch name message, got %q", errorResponse.Error)
	}
}

func TestWorkspaceHandler_CheckoutAllowsUntrackedOnlyChanges(t *testing.T) {
	handler, rootPath, repositoryPath := setupWorkspaceGitHandler(t)
	saveServerWorkspace(t, handler, rootPath)

	untrackedPath := filepath.Join(repositoryPath, "untracked.txt")
	writeGitFixtureFile(t, untrackedPath, "keep me\n")

	recorder := postGitRequest(t, handler, "/api/workspace/checkout", checkoutBranchRequest{Name: "scratch", Branch: "main"})
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200 for an untracked-only tree, got %d: %s", recorder.Code, recorder.Body.String())
	}

	var checkout checkoutBranchResponse
	if err := json.NewDecoder(recorder.Body).Decode(&checkout); err != nil {
		t.Fatalf("cannot decode checkout response: %v", err)
	}
	if checkout.Branch != "main" {
		t.Fatalf("expected the response branch main, got %q", checkout.Branch)
	}

	status := strings.TrimSpace(runGitTool(t, repositoryPath, "status", "--porcelain=v1"))
	if status != "?? untracked.txt" {
		t.Fatalf("expected only the untracked file to remain changed, got %q", status)
	}

	if _, err := os.Stat(untrackedPath); err != nil {
		t.Fatalf("expected the untracked file to survive the checkout: %v", err)
	}
}
