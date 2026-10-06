package transport

import (
	"encoding/json"
	"net/http"
	"testing"
)

func TestWorkspaceHandler_BaseBranchOptionsReturnsEveryBranchWithItsCommit(t *testing.T) {
	handler, rootPath, repositoryPath, _ := setupBranchCreateHandler(t, nil, nil)
	saveServerWorkspace(t, handler, rootPath)

	runGitTool(t, repositoryPath, "checkout", "-q", "-b", "feature/CRMMS-77911-Other-Team")

	recorder := performRequest(handler, authorizedRequest(t, http.MethodGet, baseBranchOptionsPath, "", handlerTestUserA))
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", recorder.Code, recorder.Body.String())
	}

	var response baseBranchOptionsResponse
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("cannot decode the base branch options response: %v", err)
	}

	if response.Repo != "scratch" || !response.IsCloned {
		t.Fatalf("unexpected repository envelope %+v", response)
	}
	if response.Remembered != nil || response.BitbucketDefault != nil {
		t.Fatalf("expected no remembered or remote default branch, got %+v", response)
	}
	if response.OtherBranchCount != 0 {
		t.Fatalf("expected nothing to be hidden behind a show all control, got %d", response.OtherBranchCount)
	}

	foundNames := make(map[string]bool, len(response.Options))
	for _, option := range response.Options {
		foundNames[option.Name] = true
		if option.LastCommitRelative == nil {
			t.Fatalf("expected %q to carry its last commit", option.Name)
		}
		if option.Source == "" {
			t.Fatalf("expected %q to carry a source", option.Name)
		}
	}

	for _, expected := range []string{"main", "feature/second", "feature/CRMMS-77911-Other-Team"} {
		if !foundNames[expected] {
			t.Fatalf("expected %q to be selectable, got %+v", expected, response.Options)
		}
	}
}

func TestWorkspaceHandler_BaseBranchOptionsRemembersTheChosenBase(t *testing.T) {
	handler, rootPath, _, _ := setupBranchCreateHandler(t, nil, nil)
	saveServerWorkspace(t, handler, rootPath)

	recorder := postGitRequest(t, handler, branchCreatePath, createBranchTargets(
		createBranchTargetRequest{Repo: "scratch", BaseBranch: "main", RememberBase: true},
	))
	if recorder.Code != http.StatusMultiStatus {
		t.Fatalf("expected 207, got %d: %s", recorder.Code, recorder.Body.String())
	}

	optionsRecorder := performRequest(handler, authorizedRequest(t, http.MethodGet, baseBranchOptionsPath, "", handlerTestUserA))
	if optionsRecorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", optionsRecorder.Code, optionsRecorder.Body.String())
	}

	var response baseBranchOptionsResponse
	if err := json.NewDecoder(optionsRecorder.Body).Decode(&response); err != nil {
		t.Fatalf("cannot decode the base branch options response: %v", err)
	}

	if response.Remembered == nil || *response.Remembered != "main" {
		t.Fatalf("expected the chosen base to be remembered, got %+v", response.Remembered)
	}
	if len(response.Options) == 0 {
		t.Fatal("expected options for a cloned repository")
	}
	if response.Options[0].Name != "main" || response.Options[0].Source != "remembered" {
		t.Fatalf("expected the remembered branch first, got %+v", response.Options[0])
	}
}
