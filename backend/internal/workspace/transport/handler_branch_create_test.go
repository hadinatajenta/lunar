package transport

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	sharedAuth "lunar/backend/internal/shared/auth"
	sharedDatabase "lunar/backend/internal/shared/database"
	"lunar/backend/internal/workspace/application"
	workspaceDomain "lunar/backend/internal/workspace/domain"
	"lunar/backend/internal/workspace/infrastructure"
)

const (
	branchCreatePath      = "/api/workspace/branches"
	baseBranchOptionsPath = "/api/workspace/repos/scratch/base-branch-options"
	secondScratchRepoName = "scratch-two"
	rememberedBranchName  = "feature/CRMMS-77911-Enhancement-Manage-User"
	missingRepositoryName = "not-cloned-here"
)

type recordingBranchFinder struct {
	branches []string
	err      error
	calls    []string
}

func (f *recordingBranchFinder) SearchBranches(_ context.Context, _ string, repo string, filterText string) ([]string, error) {
	f.calls = append(f.calls, repo+"|"+filterText)
	return f.branches, f.err
}

func createScratchRepository(t *testing.T, rootPath string, repositoryName string) string {
	t.Helper()

	repositoryPath := filepath.Join(rootPath, repositoryName)
	if err := os.MkdirAll(repositoryPath, 0o755); err != nil {
		t.Fatalf("cannot create %s: %v", repositoryName, err)
	}
	runGitTool(t, repositoryPath, "init", "-q", "-b", "main", ".")
	writeGitFixtureFile(t, filepath.Join(repositoryPath, "README.md"), "initial\n")
	runGitTool(t, repositoryPath, "add", "-A")
	runGitTool(t, repositoryPath, "commit", "-q", "-m", "initial commit")
	return repositoryPath
}

func setupBranchCreateHandler(t *testing.T, finder application.RemoteBranchFinder, allowedRoots []string) (http.Handler, string, string, *sql.DB) {
	t.Helper()

	db, err := sharedDatabase.OpenDB(filepath.Join(t.TempDir(), "handler-branch-create.db"))
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
	createScratchRepository(t, rootPath, secondScratchRepoName)

	if len(allowedRoots) == 0 {
		allowedRoots = []string{rootPath}
	}

	inspector := infrastructure.NewGitInspector(allowedRoots)
	workspaceRepo := infrastructure.NewSQLiteWorkspaceRepository(db)
	selectionRepo := infrastructure.NewSQLiteSelectionRepository(db)

	serviceOptions := make([]application.WorkspaceServiceOption, 0, 1)
	if finder != nil {
		serviceOptions = append(serviceOptions, application.WithRemoteBranchFinder(finder))
	}

	service := application.NewWorkspaceService(
		workspaceRepo,
		selectionRepo,
		inspector,
		inspector,
		allowedRoots,
		handlerTestEncryptKey,
		serviceOptions...,
	)
	handler := NewWorkspaceHandler(service)

	authMiddleware := sharedAuth.RequireAuth(handlerTestJWTSecret)
	mux := http.NewServeMux()
	mux.Handle("PUT /api/workspace", authMiddleware(http.HandlerFunc(handler.SaveWorkspace)))
	mux.Handle("POST "+branchCreatePath, authMiddleware(http.HandlerFunc(handler.CreateBranches)))
	mux.Handle("GET /api/workspace/repos/{repo}/base-branch-options", authMiddleware(http.HandlerFunc(handler.ListBaseBranchOptions)))

	return mux, rootPath, repositoryPath, db
}

func saveServerWorkspaceDirectly(t *testing.T, db *sql.DB, rootPath string) {
	t.Helper()

	workspaceRepo := infrastructure.NewSQLiteWorkspaceRepository(db)
	now := time.Now().UTC()
	workspace := &workspaceDomain.Workspace{
		UserID:    handlerTestUserA,
		Source:    workspaceDomain.SourceServer,
		RootPath:  rootPath,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := workspaceRepo.UpsertWorkspace(context.Background(), workspace); err != nil {
		t.Fatalf("cannot save the workspace directly: %v", err)
	}
}

func createBranchTargets(targets ...createBranchTargetRequest) createBranchesRequest {
	return createBranchesRequest{
		Key:     "CRMMS-77911",
		Branch:  rememberedBranchName,
		Targets: targets,
	}
}

func decodeCreateBranchesResponse(t *testing.T, recorder *httptest.ResponseRecorder) createBranchesResponse {
	t.Helper()

	var response createBranchesResponse
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("cannot decode the create branches response: %v", err)
	}
	return response
}

func scratchBranchNames(t *testing.T, repositoryPath string) []string {
	t.Helper()

	output := runGitTool(t, repositoryPath, "branch", "--format=%(refname:short)")
	names := make([]string, 0)
	for _, line := range strings.Split(output, "\n") {
		if trimmed := strings.TrimSpace(line); trimmed != "" {
			names = append(names, trimmed)
		}
	}
	return names
}

func countOutcomeTargets(response createBranchesResponse) int {
	return len(response.Created) + len(response.Failed) + len(response.Skipped)
}

func TestWorkspaceHandler_CreateBranchesCreatesEveryTarget(t *testing.T) {
	handler, rootPath, repositoryPath, _ := setupBranchCreateHandler(t, nil, nil)
	saveServerWorkspace(t, handler, rootPath)

	recorder := postGitRequest(t, handler, branchCreatePath, createBranchTargets(
		createBranchTargetRequest{Repo: "scratch", BaseBranch: "main"},
		createBranchTargetRequest{Repo: secondScratchRepoName, BaseBranch: "main"},
	))
	if recorder.Code != http.StatusMultiStatus {
		t.Fatalf("expected 207, got %d: %s", recorder.Code, recorder.Body.String())
	}

	response := decodeCreateBranchesResponse(t, recorder)
	if len(response.Created) != 2 || len(response.Failed) != 0 || len(response.Skipped) != 0 {
		t.Fatalf("expected two created targets, got %+v", response)
	}
	if response.Key != "CRMMS-77911" || response.Branch != rememberedBranchName {
		t.Fatalf("unexpected envelope %+v", response)
	}

	for _, created := range response.Created {
		if created.BaseBranch != "main" {
			t.Fatalf("expected the base branch main, got %q", created.BaseBranch)
		}
		if len(created.HeadCommit) != 40 {
			t.Fatalf("expected a head commit hash, got %q", created.HeadCommit)
		}
	}

	if current := strings.TrimSpace(runGitTool(t, repositoryPath, "rev-parse", "--abbrev-ref", "HEAD")); current != rememberedBranchName {
		t.Fatalf("expected the created branch to be checked out, got %q", current)
	}

	secondRepositoryPath := filepath.Join(rootPath, secondScratchRepoName)
	if current := strings.TrimSpace(runGitTool(t, secondRepositoryPath, "rev-parse", "--abbrev-ref", "HEAD")); current != rememberedBranchName {
		t.Fatalf("expected the created branch to be checked out in the second repository, got %q", current)
	}
}

func TestWorkspaceHandler_CreateBranchesRefusesATargetWithoutABaseBranch(t *testing.T) {
	handler, rootPath, repositoryPath, _ := setupBranchCreateHandler(t, nil, nil)
	saveServerWorkspace(t, handler, rootPath)

	recorder := postGitRequest(t, handler, branchCreatePath, createBranchTargets(
		createBranchTargetRequest{Repo: "scratch", BaseBranch: "main"},
		createBranchTargetRequest{Repo: secondScratchRepoName},
	))
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", recorder.Code, recorder.Body.String())
	}

	for _, branchName := range scratchBranchNames(t, repositoryPath) {
		if branchName == rememberedBranchName {
			t.Fatal("expected no branch to be created when a target omits its base branch")
		}
	}
}

func TestWorkspaceHandler_CreateBranchesRefusesATraversalBranchName(t *testing.T) {
	handler, rootPath, repositoryPath, _ := setupBranchCreateHandler(t, nil, nil)
	saveServerWorkspace(t, handler, rootPath)

	request := createBranchTargets(createBranchTargetRequest{Repo: "scratch", BaseBranch: "main"})
	request.Branch = "main; rm -rf /"

	recorder := postGitRequest(t, handler, branchCreatePath, request)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", recorder.Code, recorder.Body.String())
	}

	for _, branchName := range scratchBranchNames(t, repositoryPath) {
		if strings.Contains(branchName, "rm -rf") {
			t.Fatalf("expected no branch to be created for a traversal name, got %q", branchName)
		}
	}
}

func TestWorkspaceGitErrorResponseMapsBranchCreationConflicts(t *testing.T) {
	status, message, found := workspaceGitErrorResponse(workspaceDomain.ErrBranchAlreadyExists)
	if !found || status != http.StatusConflict {
		t.Fatalf("expected 409 for a duplicate branch, got %d found=%v", status, found)
	}
	if !strings.Contains(message, "already exists") {
		t.Fatalf("expected the message to name the conflict, got %q", message)
	}

	status, message, found = workspaceGitErrorResponse(workspaceDomain.ErrWorkingTreeDirty)
	if !found || status != http.StatusConflict {
		t.Fatalf("expected 409 for a dirty tree, got %d found=%v", status, found)
	}
	if message != workingTreeDirtyMessage {
		t.Fatalf("expected the checkout guard message %q, got %q", workingTreeDirtyMessage, message)
	}
}

func TestWorkspaceHandler_BranchCreationRoutesRequireAuthentication(t *testing.T) {
	handler, _, _, _ := setupBranchCreateHandler(t, nil, nil)

	routes := []struct {
		method string
		path   string
	}{
		{http.MethodPost, branchCreatePath},
		{http.MethodGet, baseBranchOptionsPath},
	}

	for _, route := range routes {
		recorder := performRequest(handler, httptest.NewRequest(route.method, route.path, strings.NewReader("{}")))
		if recorder.Code != http.StatusUnauthorized {
			t.Errorf("expected 401 for %s %s, got %d", route.method, route.path, recorder.Code)
		}
	}
}
