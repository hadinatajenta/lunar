package transport

import (
	"errors"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
)

func TestWorkspaceHandler_CreateBranchesSkipsAnExistingBranch(t *testing.T) {
	handler, rootPath, _, _ := setupBranchCreateHandler(t, nil, nil)
	saveServerWorkspace(t, handler, rootPath)

	recorder := postGitRequest(t, handler, branchCreatePath, createBranchTargets(
		createBranchTargetRequest{Repo: "scratch", BaseBranch: "main"},
	))
	if recorder.Code != http.StatusMultiStatus {
		t.Fatalf("expected 207, got %d: %s", recorder.Code, recorder.Body.String())
	}

	response := decodeCreateBranchesResponse(t, recorder)
	if len(response.Created) != 1 {
		t.Fatalf("expected the first target to be created, got %+v", response)
	}

	recorder = postGitRequest(t, handler, branchCreatePath, createBranchTargets(
		createBranchTargetRequest{Repo: "scratch", BaseBranch: "main"},
	))
	if recorder.Code != http.StatusMultiStatus {
		t.Fatalf("expected 207, got %d: %s", recorder.Code, recorder.Body.String())
	}

	response = decodeCreateBranchesResponse(t, recorder)
	if len(response.Failed) != 0 {
		t.Fatalf("expected an existing branch to be skipped, not failed: %+v", response)
	}
	if len(response.Skipped) != 1 {
		t.Fatalf("expected one skipped target, got %+v", response)
	}
	if !strings.Contains(response.Skipped[0].Reason, "already exists") {
		t.Fatalf("expected the skip reason to say the branch already exists, got %q", response.Skipped[0].Reason)
	}
}

func TestWorkspaceHandler_CreateBranchesReportsADirtyTreeWithTheCheckoutGuardMessage(t *testing.T) {
	handler, rootPath, repositoryPath, _ := setupBranchCreateHandler(t, nil, nil)
	saveServerWorkspace(t, handler, rootPath)

	writeGitFixtureFile(t, filepath.Join(repositoryPath, "README.md"), "modified\n")

	recorder := postGitRequest(t, handler, branchCreatePath, createBranchTargets(
		createBranchTargetRequest{Repo: "scratch", BaseBranch: "main"},
	))
	if recorder.Code != http.StatusMultiStatus {
		t.Fatalf("expected 207, got %d: %s", recorder.Code, recorder.Body.String())
	}

	response := decodeCreateBranchesResponse(t, recorder)
	if len(response.Failed) != 1 || len(response.Created) != 0 {
		t.Fatalf("expected a single failure, got %+v", response)
	}
	if response.Failed[0].Reason != workingTreeDirtyMessage {
		t.Fatalf("expected the checkout guard message %q, got %q", workingTreeDirtyMessage, response.Failed[0].Reason)
	}
}

func TestWorkspaceHandler_CreateBranchesReportsAMissingBaseBranch(t *testing.T) {
	handler, rootPath, repositoryPath, _ := setupBranchCreateHandler(t, nil, nil)
	saveServerWorkspace(t, handler, rootPath)

	recorder := postGitRequest(t, handler, branchCreatePath, createBranchTargets(
		createBranchTargetRequest{Repo: "scratch", BaseBranch: "release/nonexistent"},
	))
	if recorder.Code != http.StatusMultiStatus {
		t.Fatalf("expected 207, got %d: %s", recorder.Code, recorder.Body.String())
	}

	response := decodeCreateBranchesResponse(t, recorder)
	if len(response.Failed) != 1 || len(response.Created) != 0 {
		t.Fatalf("expected a single failure, got %+v", response)
	}
	if !strings.Contains(response.Failed[0].Reason, "base branch") {
		t.Fatalf("expected the reason to name the base branch, got %q", response.Failed[0].Reason)
	}

	for _, branchName := range scratchBranchNames(t, repositoryPath) {
		if branchName == rememberedBranchName {
			t.Fatal("expected no branch to be created from a missing base branch")
		}
	}
}

func TestWorkspaceHandler_CreateBranchesAccountsForEveryTarget(t *testing.T) {
	handler, rootPath, repositoryPath, _ := setupBranchCreateHandler(t, nil, nil)
	saveServerWorkspace(t, handler, rootPath)

	secondRepositoryPath := filepath.Join(rootPath, secondScratchRepoName)
	runGitTool(t, secondRepositoryPath, "branch", rememberedBranchName, "main")

	recorder := postGitRequest(t, handler, branchCreatePath, createBranchTargets(
		createBranchTargetRequest{Repo: "scratch", BaseBranch: "main"},
		createBranchTargetRequest{Repo: secondScratchRepoName, BaseBranch: "main"},
		createBranchTargetRequest{Repo: missingRepositoryName, BaseBranch: "main"},
	))
	if recorder.Code != http.StatusMultiStatus {
		t.Fatalf("expected 207, got %d: %s", recorder.Code, recorder.Body.String())
	}

	response := decodeCreateBranchesResponse(t, recorder)
	if countOutcomeTargets(response) != 3 {
		t.Fatalf("expected every target to be accounted for, got %+v", response)
	}
	if len(response.Created) != 1 || len(response.Skipped) != 1 || len(response.Failed) != 1 {
		t.Fatalf("expected one of each outcome, got %+v", response)
	}
	if response.Failed[0].Repo != missingRepositoryName {
		t.Fatalf("expected the missing repository to fail, got %+v", response.Failed)
	}
	if !strings.Contains(response.Failed[0].Reason, "not cloned") {
		t.Fatalf("expected the failure to say the repository is not cloned, got %q", response.Failed[0].Reason)
	}

	branches := scratchBranchNames(t, repositoryPath)
	found := false
	for _, branchName := range branches {
		if branchName == rememberedBranchName {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected the created branch to survive a sibling failure, got %+v", branches)
	}
}

func TestWorkspaceHandler_CreateBranchesSkipsWhenBitbucketAlreadyCarriesTheKey(t *testing.T) {
	finder := &recordingBranchFinder{branches: []string{"feature/CRMMS-77911-Existing-Tail"}}
	handler, rootPath, repositoryPath, _ := setupBranchCreateHandler(t, finder, nil)
	saveServerWorkspace(t, handler, rootPath)

	runGitTool(t, repositoryPath, "remote", "add", "origin", "ssh://git@bitbucket.bri.co.id:7999/nme/scratch.git")

	recorder := postGitRequest(t, handler, branchCreatePath, createBranchTargets(
		createBranchTargetRequest{Repo: "scratch", BaseBranch: "main"},
	))
	if recorder.Code != http.StatusMultiStatus {
		t.Fatalf("expected 207, got %d: %s", recorder.Code, recorder.Body.String())
	}

	response := decodeCreateBranchesResponse(t, recorder)
	if len(response.Created) != 0 || len(response.Failed) != 0 || len(response.Skipped) != 1 {
		t.Fatalf("expected the target to be skipped, got %+v", response)
	}
	if !strings.Contains(response.Skipped[0].Reason, "feature/CRMMS-77911-Existing-Tail") {
		t.Fatalf("expected the existing branch to be named, got %q", response.Skipped[0].Reason)
	}
	if len(finder.calls) != 1 || finder.calls[0] != "NME/scratch|CRMMS-77911" {
		t.Fatalf("expected one Bitbucket call for the named repository, got %+v", finder.calls)
	}

	for _, branchName := range scratchBranchNames(t, repositoryPath) {
		if branchName == rememberedBranchName {
			t.Fatal("expected no branch to be created when Bitbucket already carries the key")
		}
	}
}

func TestWorkspaceHandler_CreateBranchesFailsTheTargetWhenBitbucketCannotBeChecked(t *testing.T) {
	finder := &recordingBranchFinder{err: errors.New("bitbucket is unreachable")}
	handler, rootPath, repositoryPath, _ := setupBranchCreateHandler(t, finder, nil)
	saveServerWorkspace(t, handler, rootPath)

	runGitTool(t, repositoryPath, "remote", "add", "origin", "ssh://git@bitbucket.bri.co.id:7999/nme/scratch.git")

	recorder := postGitRequest(t, handler, branchCreatePath, createBranchTargets(
		createBranchTargetRequest{Repo: "scratch", BaseBranch: "main"},
	))
	if recorder.Code != http.StatusMultiStatus {
		t.Fatalf("expected 207, got %d: %s", recorder.Code, recorder.Body.String())
	}

	response := decodeCreateBranchesResponse(t, recorder)
	if len(response.Failed) != 1 || len(response.Created) != 0 {
		t.Fatalf("expected the target to fail without creating, got %+v", response)
	}
	if !strings.Contains(response.Failed[0].Reason, "Bitbucket") {
		t.Fatalf("expected the reason to name Bitbucket, got %q", response.Failed[0].Reason)
	}

	for _, branchName := range scratchBranchNames(t, repositoryPath) {
		if branchName == rememberedBranchName {
			t.Fatal("expected no branch to be created when the duplicate check could not run")
		}
	}
}

func TestWorkspaceHandler_CreateBranchesReturnsForbiddenOutsideTheAllowedRoots(t *testing.T) {
	handler, rootPath, repositoryPath, db := setupBranchCreateHandler(t, nil, []string{filepath.Join(t.TempDir(), "allowed-elsewhere")})
	saveServerWorkspaceDirectly(t, db, rootPath)

	recorder := postGitRequest(t, handler, branchCreatePath, createBranchTargets(
		createBranchTargetRequest{Repo: "scratch", BaseBranch: "main"},
	))
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", recorder.Code, recorder.Body.String())
	}

	optionsRecorder := performRequest(handler, authorizedRequest(t, http.MethodGet, baseBranchOptionsPath, "", handlerTestUserA))
	if optionsRecorder.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for the base options of an outside root, got %d: %s", optionsRecorder.Code, optionsRecorder.Body.String())
	}

	for _, branchName := range scratchBranchNames(t, repositoryPath) {
		if branchName == rememberedBranchName {
			t.Fatal("expected no branch to be created outside the allow list")
		}
	}
}
