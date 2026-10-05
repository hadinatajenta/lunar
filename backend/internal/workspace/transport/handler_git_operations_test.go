package transport

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWorkspaceHandler_StashReturnsEntryCount(t *testing.T) {
	handler, rootPath, repositoryPath := setupWorkspaceGitHandler(t)
	saveServerWorkspace(t, handler, rootPath)

	writeGitFixtureFile(t, filepath.Join(repositoryPath, "README.md"), "stashed\n")

	recorder := postGitRequest(t, handler, "/api/workspace/stash", repositoryNameRequest{Name: "scratch"})
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", recorder.Code, recorder.Body.String())
	}

	var response stashResponse
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("cannot decode stash response: %v", err)
	}
	if response.StashCount != 1 {
		t.Fatalf("expected 1 stash entry, got %d", response.StashCount)
	}

	status := strings.TrimSpace(runGitTool(t, repositoryPath, "status", "--porcelain=v1"))
	if status != "" {
		t.Fatalf("expected a clean tree after stashing, got %q", status)
	}
}

func TestWorkspaceHandler_DiscardRestoresTrackedFilesAndKeepsUntracked(t *testing.T) {
	handler, rootPath, repositoryPath := setupWorkspaceGitHandler(t)
	saveServerWorkspace(t, handler, rootPath)

	writeGitFixtureFile(t, filepath.Join(repositoryPath, "README.md"), "modified\n")
	untrackedPath := filepath.Join(repositoryPath, "untracked.txt")
	writeGitFixtureFile(t, untrackedPath, "keep me\n")

	recorder := postGitRequest(t, handler, "/api/workspace/discard", repositoryNameRequest{Name: "scratch"})
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", recorder.Code, recorder.Body.String())
	}

	var response discardResponse
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("cannot decode discard response: %v", err)
	}
	if response.RestoredCount != 1 {
		t.Fatalf("expected 1 restored tracked path, got %d", response.RestoredCount)
	}

	contents, err := os.ReadFile(filepath.Join(repositoryPath, "README.md"))
	if err != nil {
		t.Fatalf("cannot read the restored file: %v", err)
	}
	if string(contents) != "initial\n" {
		t.Fatalf("expected the tracked file to be restored, got %q", contents)
	}

	if _, err := os.Stat(untrackedPath); err != nil {
		t.Fatalf("expected the untracked file to survive a discard: %v", err)
	}
}

func TestWorkspaceHandler_FetchNetworkFailureReturnsGatewayTimeout(t *testing.T) {
	handler, rootPath, repositoryPath := setupWorkspaceGitHandler(t)
	saveServerWorkspace(t, handler, rootPath)

	runGitTool(t, repositoryPath, "remote", "add", "origin", filepath.Join(rootPath, "missing-remote.git"))

	recorder := postGitRequest(t, handler, "/api/workspace/fetch", repositoryNameRequest{Name: "scratch"})
	if recorder.Code != http.StatusGatewayTimeout {
		t.Fatalf("expected 504, got %d: %s", recorder.Code, recorder.Body.String())
	}

	errorResponse := decodeErrorResponse(t, recorder)
	if !strings.Contains(errorResponse.Error, "remote git operation failed") {
		t.Fatalf("expected a typed remote failure message, got %q", errorResponse.Error)
	}
	if strings.Contains(recorder.Body.String(), rootPath) {
		t.Fatalf("expected no absolute filesystem path in the error, got %s", recorder.Body.String())
	}
}

func TestWorkspaceHandler_GitRoutesRequireAuthentication(t *testing.T) {
	handler, _, _ := setupWorkspaceGitHandler(t)

	routes := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/workspace/branches?name=scratch"},
		{http.MethodPost, "/api/workspace/checkout"},
		{http.MethodPost, "/api/workspace/fetch"},
		{http.MethodPost, "/api/workspace/pull"},
		{http.MethodPost, "/api/workspace/stash"},
		{http.MethodPost, "/api/workspace/discard"},
	}

	for _, route := range routes {
		recorder := performRequest(handler, httptest.NewRequest(route.method, route.path, strings.NewReader("{}")))
		if recorder.Code != http.StatusUnauthorized {
			t.Errorf("expected 401 for %s %s, got %d", route.method, route.path, recorder.Code)
		}
	}
}
