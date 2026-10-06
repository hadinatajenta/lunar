package main

import (
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

func newKeyedBranchHelperWorkspace(t *testing.T) string {
	t.Helper()
	requireGitTool(t)

	rootPath := t.TempDir()
	createHelperKeyedRepository(t, filepath.Join(rootPath, "alpha"), []string{"feature/CRMMS-6397-first", "feature/CRMMS-63970-second"})
	createHelperKeyedRepository(t, filepath.Join(rootPath, "beta"), []string{"feature/crmms-6397-lower"})
	createHelperKeyedRepository(t, filepath.Join(rootPath, "gamma"), []string{"feature/CRMMS-77745-other"})

	return rootPath
}

func createHelperKeyedRepository(t *testing.T, repositoryPath string, branchNames []string) {
	t.Helper()

	if err := os.MkdirAll(repositoryPath, 0o755); err != nil {
		t.Fatalf("cannot create the repository directory: %v", err)
	}
	runGitTool(t, repositoryPath, "init", "-q", "-b", "main", ".")
	writeHelperFile(t, filepath.Join(repositoryPath, "README.md"), "initial\n")
	runGitTool(t, repositoryPath, "add", "-A")
	runGitTool(t, repositoryPath, "commit", "-q", "-m", "initial commit")

	for _, branchName := range branchNames {
		runGitTool(t, repositoryPath, "checkout", "-q", "-b", branchName)
	}
}

func newIssueBranchHelperServer(t *testing.T, allowedRoots []string) *httptest.Server {
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

func issueBranchHelperURL(server *httptest.Server, issueKey string, rootPath string) string {
	return server.URL + "/issues/" + issueKey + "/branches?root=" + url.QueryEscape(rootPath)
}

func sprintBranchHelperURL(server *httptest.Server, rootPath string, issueKeys []string) string {
	return server.URL + sprintBranchesPath + "?root=" + url.QueryEscape(rootPath) + "&" + keysQueryParameter + "=" + url.QueryEscape(strings.Join(issueKeys, ","))
}

func recordHelperGitInvocations(t *testing.T) string {
	t.Helper()

	realGitPath, err := exec.LookPath("git")
	if err != nil {
		t.Skipf("git executable is not available: %v", err)
	}

	shimDirectory := t.TempDir()
	logPath := filepath.Join(shimDirectory, "git-invocations.log")
	script := "#!/bin/sh\nprintf '%s\n' \"$*\" >> \"" + logPath + "\"\nexec \"" + realGitPath + "\" \"$@\"\n"
	if err := os.WriteFile(filepath.Join(shimDirectory, "git"), []byte(script), 0o755); err != nil {
		t.Fatalf("cannot write the git recorder: %v", err)
	}
	t.Setenv("PATH", shimDirectory+string(os.PathListSeparator)+os.Getenv("PATH"))

	return logPath
}

func readHelperGitInvocations(t *testing.T, logPath string) []string {
	t.Helper()

	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("cannot read the git recorder log: %v", err)
	}

	invocations := make([]string, 0)
	for _, line := range strings.Split(string(content), "\n") {
		if strings.TrimSpace(line) != "" {
			invocations = append(invocations, line)
		}
	}

	return invocations
}

func TestHelperIssueBranchesFindsTheKeyedBranch(t *testing.T) {
	rootPath := newKeyedBranchHelperWorkspace(t)
	server := newIssueBranchHelperServer(t, []string{rootPath})

	response := doRequest(t, http.MethodGet, issueBranchHelperURL(server, "CRMMS-6397", rootPath), "", bearerScheme+" "+testHelperToken, "")
	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.StatusCode, readBody(t, response))
	}

	branches := decodeHelperBody[struct {
		Key     string `json:"key"`
		Matches []struct {
			Repo      string `json:"repo"`
			Branch    string `json:"branch"`
			Source    string `json:"source"`
			IsCloned  bool   `json:"is_cloned"`
			IsCurrent bool   `json:"is_current"`
		} `json:"matches"`
		Scope struct {
			LocalRepositoryCount      int  `json:"local_repository_count"`
			RemoteRepositoryCount     int  `json:"remote_repository_count"`
			VerifiedRemotely          bool `json:"verified_remotely"`
			RemoteRepositoriesChecked int  `json:"remote_repositories_checked"`
		} `json:"scope"`
	}](t, response)

	if branches.Key != "CRMMS-6397" {
		t.Fatalf("expected the requested key, got %q", branches.Key)
	}
	if len(branches.Matches) != 2 {
		t.Fatalf("expected the keyed branch in alpha and beta, got %+v", branches.Matches)
	}
	if branches.Matches[0].Repo != "alpha" || branches.Matches[0].Branch != "feature/CRMMS-6397-first" {
		t.Fatalf("expected the alpha branch first, got %+v", branches.Matches[0])
	}
	if branches.Matches[0].Source != "local" || !branches.Matches[0].IsCloned {
		t.Fatalf("expected a local cloned match, got %+v", branches.Matches[0])
	}
	if branches.Scope.LocalRepositoryCount != 3 {
		t.Fatalf("expected 3 local repositories, got %+v", branches.Scope)
	}
	if branches.Scope.VerifiedRemotely || branches.Scope.RemoteRepositoriesChecked != 0 {
		t.Fatalf("this phase must not claim remote verification, got %+v", branches.Scope)
	}
}

func TestHelperIssueBranchesKeepsALongerKeySeparateFromItsPrefix(t *testing.T) {
	rootPath := newKeyedBranchHelperWorkspace(t)
	server := newIssueBranchHelperServer(t, []string{rootPath})

	response := doRequest(t, http.MethodGet, issueBranchHelperURL(server, "CRMMS-6397", rootPath), "", bearerScheme+" "+testHelperToken, "")
	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.StatusCode, readBody(t, response))
	}

	body := readBody(t, response)
	if strings.Contains(body, "CRMMS-63970-second") {
		t.Fatalf("CRMMS-6397 must not match the longer CRMMS-63970 branch, got %s", body)
	}
}

func TestHelperIssueBranchesReturnsAnEmptyArrayWhenNothingMatches(t *testing.T) {
	rootPath := newKeyedBranchHelperWorkspace(t)
	server := newIssueBranchHelperServer(t, []string{rootPath})

	response := doRequest(t, http.MethodGet, issueBranchHelperURL(server, "CRMMS-11111", rootPath), "", bearerScheme+" "+testHelperToken, "")
	if response.StatusCode != http.StatusOK {
		t.Fatalf("absence is not an error, expected 200, got %d: %s", response.StatusCode, readBody(t, response))
	}

	body := readBody(t, response)
	if !strings.Contains(body, `"matches":[]`) {
		t.Fatalf("expected an empty matches array, got %s", body)
	}
}

func TestHelperIssueBranchesRefusesAnInvalidKey(t *testing.T) {
	rootPath := newKeyedBranchHelperWorkspace(t)
	server := newIssueBranchHelperServer(t, []string{rootPath})

	response := doRequest(t, http.MethodGet, issueBranchHelperURL(server, "not-a-key", rootPath), "", bearerScheme+" "+testHelperToken, "")
	if response.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", response.StatusCode, readBody(t, response))
	}
	if decodeHelperError(t, response) != messageInvalidIssueKey {
		t.Fatalf("unexpected error message")
	}
}

func TestHelperIssueBranchesRequiresAToken(t *testing.T) {
	rootPath := newKeyedBranchHelperWorkspace(t)
	server := newIssueBranchHelperServer(t, []string{rootPath})

	response := doRequest(t, http.MethodGet, issueBranchHelperURL(server, "CRMMS-6397", rootPath), "", "", "")
	if response.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", response.StatusCode, readBody(t, response))
	}
}

func TestHelperIssueBranchesRefusesARootOutsideTheAllowedRoots(t *testing.T) {
	rootPath := newKeyedBranchHelperWorkspace(t)
	server := newIssueBranchHelperServer(t, []string{filepath.Join(rootPath, "elsewhere")})

	response := doRequest(t, http.MethodGet, issueBranchHelperURL(server, "CRMMS-6397", rootPath), "", bearerScheme+" "+testHelperToken, "")
	if response.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", response.StatusCode, readBody(t, response))
	}
}

func TestHelperSprintBranchesScansOnceForEveryKey(t *testing.T) {
	rootPath := newKeyedBranchHelperWorkspace(t)
	logPath := recordHelperGitInvocations(t)
	server := newIssueBranchHelperServer(t, []string{rootPath})
	issueKeys := []string{"CRMMS-6397", "CRMMS-77745", "CRMMS-77740", "CRMMS-77911", "CRMMS-77741", "CRMMS-77742", "CRMMS-77743"}

	response := doRequest(t, http.MethodGet, sprintBranchHelperURL(server, rootPath, issueKeys), "", bearerScheme+" "+testHelperToken, "")
	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.StatusCode, readBody(t, response))
	}

	sprintBranches := decodeHelperBody[struct {
		Matches map[string][]struct {
			Repo   string `json:"repo"`
			Branch string `json:"branch"`
		} `json:"matches"`
	}](t, response)

	if len(sprintBranches.Matches) != len(issueKeys) {
		t.Fatalf("expected %d keys, got %d", len(issueKeys), len(sprintBranches.Matches))
	}
	for _, issueKey := range issueKeys {
		if _, found := sprintBranches.Matches[issueKey]; !found {
			t.Fatalf("every requested key must be present, %s is missing", issueKey)
		}
	}
	if len(sprintBranches.Matches["CRMMS-6397"]) != 2 {
		t.Fatalf("expected two branches for CRMMS-6397, got %+v", sprintBranches.Matches["CRMMS-6397"])
	}

	invocations := readHelperGitInvocations(t, logPath)
	if len(invocations) != 3 {
		t.Fatalf("expected one scan per repository for the whole sprint, got %d: %v", len(invocations), invocations)
	}
	for _, invocation := range invocations {
		if !strings.Contains(invocation, "for-each-ref") {
			t.Fatalf("the sprint scan must only list refs, got %q", invocation)
		}
	}
}

func TestHelperSprintBranchesRefusesAnInvalidKey(t *testing.T) {
	rootPath := newKeyedBranchHelperWorkspace(t)
	server := newIssueBranchHelperServer(t, []string{rootPath})

	response := doRequest(t, http.MethodGet, sprintBranchHelperURL(server, rootPath, []string{"CRMMS-6397", "not-a-key"}), "", bearerScheme+" "+testHelperToken, "")
	if response.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", response.StatusCode, readBody(t, response))
	}
}

func TestParseIssueKeysTrimsAndDropsEmptyEntries(t *testing.T) {
	issueKeys := parseIssueKeys(" CRMMS-1 , ,CRMMS-2 ,")
	if len(issueKeys) != 2 || issueKeys[0] != "CRMMS-1" || issueKeys[1] != "CRMMS-2" {
		t.Fatalf("unexpected parsed keys: %v", issueKeys)
	}
}
