package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"lunar/backend/internal/workspace/domain"
)

const (
	helperCreateBranchPath = "/branches"
	helperBaseBranchPath   = "/repos/{repo}/base-branch-options"
	helperBranchName       = "feature/CRMMS-77911-Enhancement-Manage-User"
	helperSecondRepoName   = "scratch-two"
)

func helperRootURL(server *httptest.Server, route string, rootPath string) string {
	return server.URL + route + "?root=" + url.QueryEscape(rootPath)
}

func newHelperBranchWorkspace(t *testing.T) (string, string) {
	t.Helper()

	rootPath, repositoryPath := newHelperScratchRepository(t)

	secondPath := filepath.Join(rootPath, helperSecondRepoName)
	if err := os.MkdirAll(secondPath, 0o755); err != nil {
		t.Fatalf("cannot create the second repository: %v", err)
	}
	runGitTool(t, secondPath, "init", "-q", "-b", "main", ".")
	writeHelperFile(t, filepath.Join(secondPath, "README.md"), "initial\n")
	runGitTool(t, secondPath, "add", "-A")
	runGitTool(t, secondPath, "commit", "-q", "-m", "initial commit")

	return rootPath, repositoryPath
}

func helperCreateRequest(targets ...createBranchTargetRequest) createBranchesRequest {
	return createBranchesRequest{
		Key:     "CRMMS-77911",
		Branch:  helperBranchName,
		Targets: targets,
	}
}

func postHelperCreate(t *testing.T, server *httptest.Server, rootPath string, request createBranchesRequest) *http.Response {
	t.Helper()

	body, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("cannot encode the create branches request: %v", err)
	}

	return doRequest(t, http.MethodPost, helperRootURL(server, helperCreateBranchPath, rootPath), string(body), bearerScheme+" "+testHelperToken, "")
}

func helperBranchNames(t *testing.T, repositoryPath string) []string {
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

func TestHelperCreateBranchesCreatesEachTargetAndSkipsTheNext(t *testing.T) {
	rootPath, repositoryPath := newHelperBranchWorkspace(t)
	server := newHelperGitServer(t, []string{rootPath})

	response := postHelperCreate(t, server, rootPath, helperCreateRequest(
		createBranchTargetRequest{Repo: "scratch", BaseBranch: "main"},
		createBranchTargetRequest{Repo: helperSecondRepoName, BaseBranch: "main"},
	))
	if response.StatusCode != http.StatusMultiStatus {
		t.Fatalf("expected 207, got %d: %s", response.StatusCode, readBody(t, response))
	}

	created := decodeHelperBody[createBranchesResponse](t, response)
	if len(created.Created) != 2 || len(created.Failed) != 0 || len(created.Skipped) != 0 {
		t.Fatalf("expected two created targets, got %+v", created)
	}
	for _, branch := range created.Created {
		if branch.BaseBranch != "main" || len(branch.HeadCommit) != 40 {
			t.Fatalf("unexpected created target %+v", branch)
		}
	}

	repeated := postHelperCreate(t, server, rootPath, helperCreateRequest(
		createBranchTargetRequest{Repo: "scratch", BaseBranch: "main"},
	))
	if repeated.StatusCode != http.StatusMultiStatus {
		t.Fatalf("expected 207, got %d: %s", repeated.StatusCode, readBody(t, repeated))
	}

	skipped := decodeHelperBody[createBranchesResponse](t, repeated)
	if len(skipped.Skipped) != 1 || len(skipped.Failed) != 0 {
		t.Fatalf("expected the existing branch to be skipped, got %+v", skipped)
	}
	if !strings.Contains(skipped.Skipped[0].Reason, "already exists") {
		t.Fatalf("expected an already exists reason, got %q", skipped.Skipped[0].Reason)
	}

	if current := strings.TrimSpace(runGitTool(t, repositoryPath, "rev-parse", "--abbrev-ref", "HEAD")); current != helperBranchName {
		t.Fatalf("expected the created branch to be checked out, got %q", current)
	}
}

func TestHelperCreateBranchesRefusesATargetWithoutABaseBranch(t *testing.T) {
	rootPath, repositoryPath := newHelperBranchWorkspace(t)
	server := newHelperGitServer(t, []string{rootPath})

	response := postHelperCreate(t, server, rootPath, helperCreateRequest(
		createBranchTargetRequest{Repo: "scratch", BaseBranch: "main"},
		createBranchTargetRequest{Repo: helperSecondRepoName},
	))
	if response.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", response.StatusCode, readBody(t, response))
	}

	for _, branchName := range helperBranchNames(t, repositoryPath) {
		if branchName == helperBranchName {
			t.Fatal("expected no branch to be created when a target omits its base branch")
		}
	}
}

func TestHelperCreateBranchesRefusesATraversalBranchName(t *testing.T) {
	rootPath, repositoryPath := newHelperBranchWorkspace(t)
	server := newHelperGitServer(t, []string{rootPath})

	request := helperCreateRequest(createBranchTargetRequest{Repo: "scratch", BaseBranch: "main"})
	request.Branch = "main; rm -rf /"

	response := postHelperCreate(t, server, rootPath, request)
	if response.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", response.StatusCode, readBody(t, response))
	}

	for _, branchName := range helperBranchNames(t, repositoryPath) {
		if strings.Contains(branchName, "rm -rf") {
			t.Fatalf("expected no branch to be created for a traversal name, got %q", branchName)
		}
	}
}

func TestHelperCreateBranchesRefusesARootOutsideTheAllowList(t *testing.T) {
	rootPath, repositoryPath := newHelperBranchWorkspace(t)
	server := newHelperGitServer(t, []string{filepath.Join(t.TempDir(), "allowed-elsewhere")})

	response := postHelperCreate(t, server, rootPath, helperCreateRequest(
		createBranchTargetRequest{Repo: "scratch", BaseBranch: "main"},
	))
	if response.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", response.StatusCode, readBody(t, response))
	}

	for _, branchName := range helperBranchNames(t, repositoryPath) {
		if branchName == helperBranchName {
			t.Fatal("expected no branch to be created outside the allow list")
		}
	}
}

func decodeHelperErrorFromRecorder(t *testing.T, recorder *httptest.ResponseRecorder) string {
	t.Helper()

	var decoded map[string]string
	if err := json.NewDecoder(recorder.Body).Decode(&decoded); err != nil {
		t.Fatalf("cannot decode the helper error: %v", err)
	}
	return decoded["error"]
}

func TestHelperGitOperationErrorMapsADuplicateBranchToConflict(t *testing.T) {
	recorder := httptest.NewRecorder()

	writeGitOperationError(recorder, domain.ErrBranchAlreadyExists)
	if recorder.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d", recorder.Code)
	}

	errorMessage := decodeHelperErrorFromRecorder(t, recorder)
	if !strings.Contains(errorMessage, "already exists") {
		t.Fatalf("expected the message to name the conflict, got %q", errorMessage)
	}
}

func TestHelperBaseBranchOptionsListsEveryBranchWithItsCommit(t *testing.T) {
	rootPath, _ := newHelperBranchWorkspace(t)
	server := newHelperGitServer(t, []string{rootPath})

	response := doRequest(t, http.MethodGet, helperRootURL(server, "/repos/scratch/base-branch-options", rootPath), "", bearerScheme+" "+testHelperToken, "")
	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.StatusCode, readBody(t, response))
	}

	options := decodeHelperBody[baseBranchOptionsResponse](t, response)
	if options.Repo != "scratch" || !options.IsCloned {
		t.Fatalf("unexpected envelope %+v", options)
	}
	if options.Remembered != nil || options.BitbucketDefault != nil {
		t.Fatalf("expected no remembered or remote default branch, got %+v", options)
	}

	foundNames := make(map[string]bool, len(options.Options))
	for _, option := range options.Options {
		foundNames[option.Name] = true
		if option.LastCommitRelative == nil {
			t.Fatalf("expected %q to carry its last commit", option.Name)
		}
	}

	for _, expected := range []string{"main", "feature/second"} {
		if !foundNames[expected] {
			t.Fatalf("expected %q to be selectable, got %+v", expected, options.Options)
		}
	}
}

func TestHelperBaseBranchOptionsRemembersTheChosenBase(t *testing.T) {
	rootPath, _ := newHelperBranchWorkspace(t)
	server := newHelperGitServer(t, []string{rootPath})

	response := postHelperCreate(t, server, rootPath, helperCreateRequest(
		createBranchTargetRequest{Repo: "scratch", BaseBranch: "main", RememberBase: true},
	))
	if response.StatusCode != http.StatusMultiStatus {
		t.Fatalf("expected 207, got %d: %s", response.StatusCode, readBody(t, response))
	}

	optionsResponse := doRequest(t, http.MethodGet, helperRootURL(server, "/repos/scratch/base-branch-options", rootPath), "", bearerScheme+" "+testHelperToken, "")
	if optionsResponse.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", optionsResponse.StatusCode, readBody(t, optionsResponse))
	}

	options := decodeHelperBody[baseBranchOptionsResponse](t, optionsResponse)
	if options.Remembered == nil || *options.Remembered != "main" {
		t.Fatalf("expected the chosen base to be remembered, got %+v", options.Remembered)
	}
	if len(options.Options) == 0 || options.Options[0].Name != "main" || options.Options[0].Source != "remembered" {
		t.Fatalf("expected the remembered branch first, got %+v", options.Options)
	}
}
