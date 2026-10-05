package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

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

func newHelperGitServer(t *testing.T, allowedRoots []string) *httptest.Server {
	t.Helper()

	inspector := infrastructure.NewGitInspector(allowedRoots)
	handler, stop := newHelperHandler(helperDependencies{
		reader:       inspector,
		operator:     inspector,
		allowedRoots: allowedRoots,
		token:        testHelperToken,
		version:      "test-version",
		logger:       discardLogger(),
	})
	t.Cleanup(stop)

	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	return server
}

func writeHelperFile(t *testing.T, path string, content string) {
	t.Helper()

	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("cannot write helper fixture file %s: %v", path, err)
	}
}

func newHelperScratchRepository(t *testing.T) (string, string) {
	t.Helper()
	requireGitTool(t)

	rootPath := t.TempDir()
	repositoryPath := filepath.Join(rootPath, "scratch")
	if err := os.MkdirAll(repositoryPath, 0o755); err != nil {
		t.Fatalf("cannot create the scratch repository directory: %v", err)
	}
	runGitTool(t, repositoryPath, "init", "-q", "-b", "main", ".")
	writeHelperFile(t, filepath.Join(repositoryPath, "README.md"), "initial\n")
	runGitTool(t, repositoryPath, "add", "-A")
	runGitTool(t, repositoryPath, "commit", "-q", "-m", "initial commit")
	runGitTool(t, repositoryPath, "checkout", "-q", "-b", "feature/second")

	return rootPath, repositoryPath
}

func helperGitURL(server *httptest.Server, route string, rootPath string) string {
	return server.URL + route + "?root=" + url.QueryEscape(rootPath) + "&name=scratch"
}

func decodeHelperBody[T any](t *testing.T, response *http.Response) T {
	t.Helper()

	var decoded T
	if err := json.NewDecoder(response.Body).Decode(&decoded); err != nil {
		t.Fatalf("cannot decode helper response: %v", err)
	}
	return decoded
}

func decodeHelperError(t *testing.T, response *http.Response) string {
	t.Helper()

	var decoded map[string]string
	if err := json.NewDecoder(response.Body).Decode(&decoded); err != nil {
		t.Fatalf("cannot decode helper error response: %v", err)
	}
	return decoded["error"]
}

func TestHelperBranchesListsCurrentBranchFirst(t *testing.T) {
	rootPath, _ := newHelperScratchRepository(t)
	server := newHelperGitServer(t, []string{rootPath})

	response := doRequest(t, http.MethodGet, helperGitURL(server, branchesPath, rootPath), "", bearerScheme+" "+testHelperToken, "")
	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.StatusCode, readBody(t, response))
	}

	branches := decodeHelperBody[branchesResponse](t, response)
	if len(branches.Branches) != 2 {
		t.Fatalf("expected 2 branches, got %+v", branches.Branches)
	}
	if branches.Branches[0].Name != "feature/second" || !branches.Branches[0].IsCurrent {
		t.Fatalf("expected the current branch first, got %+v", branches.Branches)
	}
	if branches.Branches[1].Name != "main" {
		t.Fatalf("expected main second, got %+v", branches.Branches)
	}
	if branches.Branches[0].UpdatedRelative == "" {
		t.Fatal("expected a relative update time for the current branch")
	}
}

func TestHelperStashCreatesEntryAndCleansTree(t *testing.T) {
	rootPath, repositoryPath := newHelperScratchRepository(t)
	server := newHelperGitServer(t, []string{rootPath})

	writeHelperFile(t, filepath.Join(repositoryPath, "README.md"), "stashed\n")

	response := doRequest(t, http.MethodPost, helperGitURL(server, stashPath, rootPath), "", bearerScheme+" "+testHelperToken, "")
	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.StatusCode, readBody(t, response))
	}

	stash := decodeHelperBody[stashResponse](t, response)
	if stash.StashCount != 1 {
		t.Fatalf("expected 1 stash entry, got %d", stash.StashCount)
	}

	status := strings.TrimSpace(runGitTool(t, repositoryPath, "status", "--porcelain=v1"))
	if status != "" {
		t.Fatalf("expected a clean tree after stashing, got %q", status)
	}
}

func TestHelperDiscardRestoresTrackedFilesAndKeepsUntracked(t *testing.T) {
	rootPath, repositoryPath := newHelperScratchRepository(t)
	server := newHelperGitServer(t, []string{rootPath})

	writeHelperFile(t, filepath.Join(repositoryPath, "README.md"), "modified\n")
	untrackedPath := filepath.Join(repositoryPath, "untracked.txt")
	writeHelperFile(t, untrackedPath, "keep me\n")

	response := doRequest(t, http.MethodPost, helperGitURL(server, discardPath, rootPath), "", bearerScheme+" "+testHelperToken, "")
	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.StatusCode, readBody(t, response))
	}

	discard := decodeHelperBody[discardResponse](t, response)
	if discard.RestoredCount != 1 {
		t.Fatalf("expected 1 restored tracked path, got %d", discard.RestoredCount)
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

func TestHelperNetworkOperationsWithoutRemoteReturnConflict(t *testing.T) {
	rootPath, _ := newHelperScratchRepository(t)
	server := newHelperGitServer(t, []string{rootPath})

	fetchResult := doRequest(t, http.MethodPost, helperGitURL(server, fetchPath, rootPath), "", bearerScheme+" "+testHelperToken, "")
	if fetchResult.StatusCode != http.StatusConflict {
		t.Fatalf("expected 409 for fetch without a remote, got %d: %s", fetchResult.StatusCode, readBody(t, fetchResult))
	}

	pullResult := doRequest(t, http.MethodPost, helperGitURL(server, pullPath, rootPath), "", bearerScheme+" "+testHelperToken, "")
	if pullResult.StatusCode != http.StatusConflict {
		t.Fatalf("expected 409 for pull without an upstream, got %d: %s", pullResult.StatusCode, readBody(t, pullResult))
	}
}

func TestHelperGitEndpointsRequireToken(t *testing.T) {
	server := newHelperGitServer(t, nil)

	routes := []struct {
		method string
		route  string
	}{
		{http.MethodGet, branchesPath},
		{http.MethodPost, checkoutPath},
		{http.MethodPost, fetchPath},
		{http.MethodPost, pullPath},
		{http.MethodPost, stashPath},
		{http.MethodPost, discardPath},
	}

	for _, route := range routes {
		response := doRequest(t, route.method, server.URL+route.route, "{}", "", "")
		if response.StatusCode != http.StatusUnauthorized {
			t.Errorf("expected 401 for %s %s, got %d", route.method, route.route, response.StatusCode)
		}
	}
}
