package transport

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	jiraDomain "lunar/backend/internal/jira/domain"
	sharedAuth "lunar/backend/internal/shared/auth"
	workspaceApp "lunar/backend/internal/workspace/application"
	workspaceDomain "lunar/backend/internal/workspace/domain"
)

type stubWorkspaceRoots struct {
	rootPath string
	err      error
}

func (s stubWorkspaceRoots) WorkspaceRootFor(_ context.Context, _ string) (string, error) {
	if s.err != nil {
		return "", s.err
	}

	return s.rootPath, nil
}

type stubSprintIssues struct {
	backlog  *jiraDomain.JiraBacklogResponse
	err      error
	requests int
}

func (s *stubSprintIssues) GetBacklog(_ context.Context, _ string, _ string) (*jiraDomain.JiraBacklogResponse, error) {
	s.requests++
	if s.err != nil {
		return nil, s.err
	}

	return s.backlog, nil
}

type countingBranchScanner struct {
	calls                int
	branchesByRepository map[string][]workspaceDomain.Branch
}

func (s *countingBranchScanner) ScanRepositoryBranches(_ context.Context, _ string) (map[string][]workspaceDomain.Branch, error) {
	s.calls++
	return s.branchesByRepository, nil
}

func requireGitBinary(t *testing.T) {
	t.Helper()

	if _, err := exec.LookPath("git"); err != nil {
		t.Skipf("git executable is not available: %v", err)
	}
}

func runGitCommand(t *testing.T, repositoryPath string, args ...string) {
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
}

func newKeyedTransportWorkspace(t *testing.T) string {
	t.Helper()
	requireGitBinary(t)

	rootPath := t.TempDir()
	createTransportKeyedRepository(t, filepath.Join(rootPath, "alpha"), []string{"feature/CRMMS-6397-first", "feature/CRMMS-63970-second"})
	createTransportKeyedRepository(t, filepath.Join(rootPath, "beta"), []string{"feature/crmms-6397-lower"})

	return rootPath
}

func createTransportKeyedRepository(t *testing.T, repositoryPath string, branchNames []string) {
	t.Helper()

	if err := os.MkdirAll(repositoryPath, 0o755); err != nil {
		t.Fatalf("cannot create the repository directory: %v", err)
	}
	runGitCommand(t, repositoryPath, "init", "-q", "-b", "main", ".")
	if err := os.WriteFile(filepath.Join(repositoryPath, "README.md"), []byte("initial\n"), 0o644); err != nil {
		t.Fatalf("cannot write the fixture file: %v", err)
	}
	runGitCommand(t, repositoryPath, "add", "-A")
	runGitCommand(t, repositoryPath, "commit", "-q", "-m", "initial commit")

	for _, branchName := range branchNames {
		runGitCommand(t, repositoryPath, "checkout", "-q", "-b", branchName)
	}
}

func issueBranchMux(handler *IssueBranchHandler) *http.ServeMux {
	mux := http.NewServeMux()
	mux.Handle("GET /api/jira/issues/{key}/branches", http.HandlerFunc(handler.GetIssueBranches))
	mux.Handle("GET /api/jira/sprint/branches", http.HandlerFunc(handler.GetSprintBranches))

	return mux
}

func performIssueBranchRequest(t *testing.T, mux *http.ServeMux, userID string, target string) *httptest.ResponseRecorder {
	t.Helper()

	request := httptest.NewRequest(http.MethodGet, target, nil)
	if userID != "" {
		context := sharedAuth.ContextWithUser(request.Context(), &sharedAuth.UserContext{
			UserID: userID,
			Email:  "developer@bri.co.id",
		})
		request = request.WithContext(context)
	}

	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, request)
	return recorder
}

func decodeIssueBranches(t *testing.T, recorder *httptest.ResponseRecorder) workspaceApp.IssueBranches {
	t.Helper()

	var branches workspaceApp.IssueBranches
	if err := json.NewDecoder(recorder.Body).Decode(&branches); err != nil {
		t.Fatalf("cannot decode the issue branches response: %v", err)
	}

	return branches
}

func decodeSprintBranches(t *testing.T, recorder *httptest.ResponseRecorder) workspaceApp.SprintBranches {
	t.Helper()

	var branches workspaceApp.SprintBranches
	if err := json.NewDecoder(recorder.Body).Decode(&branches); err != nil {
		t.Fatalf("cannot decode the sprint branches response: %v", err)
	}

	return branches
}

func matchedBranchNames(matches []workspaceApp.BranchMatch) []string {
	branches := make([]string, 0, len(matches))
	for _, match := range matches {
		branches = append(branches, match.Repository+"/"+match.Branch)
	}

	return branches
}
