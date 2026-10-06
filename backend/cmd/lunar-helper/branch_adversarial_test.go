package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"lunar/backend/internal/workspace/infrastructure"
)

func adversarialHelperMissWorkspace(t *testing.T) (string, string) {
	t.Helper()

	rootPath := t.TempDir()
	repositoryPath := filepath.Join(rootPath, "atlantis")
	createHelperKeyedRepository(t, repositoryPath, []string{"bugfix_CRMMS-6397-existing"})

	return rootPath, repositoryPath
}

func newIssueBranchHelperServerWithNilOperator(t *testing.T, allowedRoots []string) *httptest.Server {
	t.Helper()

	inspector := infrastructure.NewGitInspector(allowedRoots)
	handler, stop := newHelperHandler(helperDependencies{
		reader:       inspector,
		operator:     nil,
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

func TestAdversarialHelperSprintBranchCostIsIndependentOfKeyCount(t *testing.T) {
	sevenKeyRoot := newKeyedBranchHelperWorkspace(t)
	logPath := recordHelperGitInvocations(t)
	sevenKeyServer := newIssueBranchHelperServer(t, []string{sevenKeyRoot})

	sevenKeys := []string{"CRMMS-6397", "CRMMS-77745", "CRMMS-77740", "CRMMS-77911", "CRMMS-77741", "CRMMS-77742", "CRMMS-77743"}
	response := doRequest(t, http.MethodGet, sprintBranchHelperURL(sevenKeyServer, sevenKeyRoot, sevenKeys), "", bearerScheme+" "+testHelperToken, "")
	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.StatusCode, readBody(t, response))
	}

	sevenKeyInvocations := readHelperGitInvocations(t, logPath)
	if len(sevenKeyInvocations) != 3 {
		t.Fatalf("seven keys over three repositories must read refs three times, got %d: %v", len(sevenKeyInvocations), sevenKeyInvocations)
	}

	singleKeyRoot := newKeyedBranchHelperWorkspace(t)
	singleKeyServer := newIssueBranchHelperServer(t, []string{singleKeyRoot})
	baselineInvocations := len(readHelperGitInvocations(t, logPath))

	response = doRequest(t, http.MethodGet, sprintBranchHelperURL(singleKeyServer, singleKeyRoot, []string{"CRMMS-6397"}), "", bearerScheme+" "+testHelperToken, "")
	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.StatusCode, readBody(t, response))
	}

	singleKeyInvocations := readHelperGitInvocations(t, logPath)
	if len(singleKeyInvocations)-baselineInvocations != 3 {
		t.Fatalf("one key over three repositories must read refs three times too, got %d", len(singleKeyInvocations)-baselineInvocations)
	}
	for _, invocation := range singleKeyInvocations[baselineInvocations:] {
		if !strings.Contains(invocation, "for-each-ref") {
			t.Fatalf("the sprint scan must only list refs, got %q", invocation)
		}
	}
}

func TestAdversarialHelperGitRecorderCountsEveryScan(t *testing.T) {
	rootPath := newKeyedBranchHelperWorkspace(t)
	logPath := recordHelperGitInvocations(t)
	inspector := infrastructure.NewGitInspector([]string{rootPath})

	for scanIndex := 0; scanIndex < 2; scanIndex++ {
		if _, err := inspector.ScanRepositoryBranches(context.Background(), rootPath); err != nil {
			t.Fatalf("scan %d failed: %v", scanIndex, err)
		}
	}

	invocations := readHelperGitInvocations(t, logPath)
	if len(invocations) != 6 {
		t.Fatalf("two scans of three repositories must be six recorded calls, got %d: %v", len(invocations), invocations)
	}
}

func TestAdversarialHelperFindsAKeyAdjoinedByAWordCharacterAndCreatesNoDuplicate(t *testing.T) {
	rootPath, repositoryPath := adversarialHelperMissWorkspace(t)
	server := newIssueBranchHelperServer(t, []string{rootPath})

	response := doRequest(t, http.MethodGet, issueBranchHelperURL(server, "CRMMS-6397", rootPath), "", bearerScheme+" "+testHelperToken, "")
	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.StatusCode, readBody(t, response))
	}

	branches := decodeHelperBody[struct {
		Matches []struct {
			Repo   string `json:"repo"`
			Branch string `json:"branch"`
		} `json:"matches"`
	}](t, response)
	if len(branches.Matches) != 1 || branches.Matches[0].Branch != "bugfix_CRMMS-6397-existing" {
		t.Fatalf("expected the existing branch to be found, got %+v", branches.Matches)
	}

	createResponse := postHelperCreate(t, server, rootPath, createBranchesRequest{
		Key:    "CRMMS-6397",
		Branch: "bugfix_CRMMS-6397-existing",
		Targets: []createBranchTargetRequest{
			{Repo: "atlantis", BaseBranch: "main"},
		},
	})
	if createResponse.StatusCode != http.StatusMultiStatus {
		t.Fatalf("expected 207, got %d: %s", createResponse.StatusCode, readBody(t, createResponse))
	}

	created := decodeHelperBody[createBranchesResponse](t, createResponse)
	if len(created.Created) != 0 || len(created.Skipped) != 1 {
		t.Fatalf("expected the existing branch to be skipped, got %+v", created)
	}

	branchNames := helperBranchNames(t, repositoryPath)
	for _, branchName := range branchNames {
		if branchName == "feature/CRMMS-6397-Enhancement" {
			t.Fatalf("a second branch was created for an issue that already had one: %v", branchNames)
		}
	}
	if len(branchNames) != 2 {
		t.Fatalf("expected only the base and the existing branch, got %v", branchNames)
	}
}

func TestAdversarialHelperBulkKeepsALongerDigitRunOutOfAShorterKey(t *testing.T) {
	rootPath := newKeyedBranchHelperWorkspace(t)
	server := newIssueBranchHelperServer(t, []string{rootPath})

	response := doRequest(t, http.MethodGet, sprintBranchHelperURL(server, rootPath, []string{"CRMMS-6397", "CRMMS-63970"}), "", bearerScheme+" "+testHelperToken, "")
	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.StatusCode, readBody(t, response))
	}

	sprintBranches := decodeHelperBody[struct {
		Matches map[string][]struct {
			Repo   string `json:"repo"`
			Branch string `json:"branch"`
		} `json:"matches"`
	}](t, response)

	if len(sprintBranches.Matches["CRMMS-6397"]) != 2 {
		t.Fatalf("CRMMS-6397 must match only its own two branches, got %+v", sprintBranches.Matches["CRMMS-6397"])
	}
	if len(sprintBranches.Matches["CRMMS-63970"]) != 1 {
		t.Fatalf("CRMMS-63970 must match its own branch only, got %+v", sprintBranches.Matches["CRMMS-63970"])
	}
	for _, match := range sprintBranches.Matches["CRMMS-6397"] {
		if strings.Contains(match.Branch, "63970") {
			t.Fatalf("the longer key must not appear under the shorter key, got %+v", match)
		}
	}
}

func TestAdversarialHelperCreateBranchesRefusesAnEmptyTargetsArray(t *testing.T) {
	rootPath, repositoryPath := newHelperBranchWorkspace(t)
	server := newHelperGitServer(t, []string{rootPath})

	response := postHelperCreate(t, server, rootPath, createBranchesRequest{
		Key:     "CRMMS-77911",
		Branch:  helperBranchName,
		Targets: []createBranchTargetRequest{},
	})
	if response.StatusCode != http.StatusBadRequest {
		t.Fatalf("an empty targets array must be 400, got %d: %s", response.StatusCode, readBody(t, response))
	}

	for _, branchName := range helperBranchNames(t, repositoryPath) {
		if branchName == helperBranchName {
			t.Fatal("a refused request must not create a branch")
		}
	}
}

func TestAdversarialHelperCreateBranchesAccountsForADuplicatedRepositoryTarget(t *testing.T) {
	rootPath, _ := newHelperBranchWorkspace(t)
	server := newHelperGitServer(t, []string{rootPath})

	response := postHelperCreate(t, server, rootPath, helperCreateRequest(
		createBranchTargetRequest{Repo: "scratch", BaseBranch: "main"},
		createBranchTargetRequest{Repo: "scratch", BaseBranch: "main"},
	))
	if response.StatusCode != http.StatusMultiStatus {
		t.Fatalf("expected 207, got %d: %s", response.StatusCode, readBody(t, response))
	}

	outcome := decodeHelperBody[createBranchesResponse](t, response)
	total := len(outcome.Created) + len(outcome.Failed) + len(outcome.Skipped)
	if total != 2 {
		t.Fatalf("a duplicated repository must be accounted for twice, got %+v", outcome)
	}
	if len(outcome.Created) != 1 || len(outcome.Skipped) != 1 {
		t.Fatalf("expected the second attempt to be skipped, got %+v", outcome)
	}
}

func TestAdversarialHelperCreateBranchesWithoutGitOperations(t *testing.T) {
	rootPath, repositoryPath := newHelperBranchWorkspace(t)
	server := newIssueBranchHelperServerWithNilOperator(t, []string{rootPath})

	response := postHelperCreate(t, server, rootPath, helperCreateRequest(
		createBranchTargetRequest{Repo: "scratch", BaseBranch: "main"},
	))
	if response.StatusCode != http.StatusInternalServerError {
		t.Fatalf("expected 500 without git operations, got %d: %s", response.StatusCode, readBody(t, response))
	}

	for _, branchName := range helperBranchNames(t, repositoryPath) {
		if branchName == helperBranchName {
			t.Fatal("a missing git operator must not create a branch")
		}
	}
}
func TestAdversarialHelperBaseBranchOptionsRefusesARootOutsideTheAllowList(t *testing.T) {
	rootPath, repositoryPath := newHelperBranchWorkspace(t)
	server := newHelperGitServer(t, []string{filepath.Join(t.TempDir(), "allowed-elsewhere")})

	response := doRequest(t, http.MethodGet, helperRootURL(server, "/repos/scratch/base-branch-options", rootPath), "", bearerScheme+" "+testHelperToken, "")
	if response.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403 for the base options of an outside root, got %d: %s", response.StatusCode, readBody(t, response))
	}

	if _, err := os.Stat(repositoryPath); err != nil {
		t.Fatalf("the scratch repository must be untouched: %v", err)
	}
}
