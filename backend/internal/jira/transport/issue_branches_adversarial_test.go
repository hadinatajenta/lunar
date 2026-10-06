package transport

import (
	"net/http"
	"path/filepath"
	"testing"

	jiraDomain "lunar/backend/internal/jira/domain"
	workspaceApp "lunar/backend/internal/workspace/application"
	"lunar/backend/internal/workspace/infrastructure"
)

func newAdversarialDigitRunWorkspace(t *testing.T) string {
	t.Helper()
	requireGitBinary(t)

	rootPath := t.TempDir()
	createTransportKeyedRepository(t, filepath.Join(rootPath, "alpha"), []string{
		"feature/CRMMS-6397-exact",
		"feature/CRMMS-63970-one-extra-digit",
		"feature/CRMMS-639712-six-digits",
		"feature/CRMMS-6397123-seven-digits",
	})

	return rootPath
}

func newAdversarialIssueBranchHandler(rootPath string, sprintIssues SprintIssueSource) *IssueBranchHandler {
	return NewIssueBranchHandler(
		workspaceApp.NewIssueBranchService(infrastructure.NewGitInspector([]string{rootPath})),
		stubWorkspaceRoots{rootPath: rootPath},
		sprintIssues,
	)
}

func TestAdversarialIssueBranchSurfaceKeepsALongerDigitRunOutOfAShorterKey(t *testing.T) {
	rootPath := newAdversarialDigitRunWorkspace(t)
	handler := newAdversarialIssueBranchHandler(rootPath, &stubSprintIssues{})

	recorder := performIssueBranchRequest(t, issueBranchMux(handler), "user-a", "/api/jira/issues/CRMMS-6397/branches")
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", recorder.Code, recorder.Body.String())
	}

	matches := matchedBranchNames(decodeIssueBranches(t, recorder).Matches)
	if len(matches) != 1 || matches[0] != "alpha/feature/CRMMS-6397-exact" {
		t.Fatalf("CRMMS-6397 must match only its exact branch, got %v", matches)
	}

	recorder = performIssueBranchRequest(t, issueBranchMux(handler), "user-a", "/api/jira/issues/CRMMS-63970/branches")
	matches = matchedBranchNames(decodeIssueBranches(t, recorder).Matches)
	if len(matches) != 1 || matches[0] != "alpha/feature/CRMMS-63970-one-extra-digit" {
		t.Fatalf("CRMMS-63970 must match only its own branch, got %v", matches)
	}

	recorder = performIssueBranchRequest(t, issueBranchMux(handler), "user-a", "/api/jira/issues/CRMMS-639712/branches")
	matches = matchedBranchNames(decodeIssueBranches(t, recorder).Matches)
	if len(matches) != 1 || matches[0] != "alpha/feature/CRMMS-639712-six-digits" {
		t.Fatalf("CRMMS-639712 must match only its own branch, got %v", matches)
	}

	recorder = performIssueBranchRequest(t, issueBranchMux(handler), "user-a", "/api/jira/issues/CRMMS-6397123/branches")
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("a seven digit issue number cannot be requested through the surface, expected 400, got %d: %s", recorder.Code, recorder.Body.String())
	}
}

func TestAdversarialSprintSurfaceKeepsALongerDigitRunOutOfAShorterKey(t *testing.T) {
	rootPath := newAdversarialDigitRunWorkspace(t)
	sprintIssues := &stubSprintIssues{backlog: &jiraDomain.JiraBacklogResponse{
		Sprints: []jiraDomain.JiraSprint{
			{
				ID:            7,
				State:         "active",
				OriginBoardID: 42,
				Issues: []jiraDomain.JiraIssue{
					{Key: "CRMMS-6397"},
					{Key: "CRMMS-63970"},
					{Key: "CRMMS-639712"},
				},
			},
		},
	}}
	handler := newAdversarialIssueBranchHandler(rootPath, sprintIssues)

	recorder := performIssueBranchRequest(t, issueBranchMux(handler), "user-a", "/api/jira/sprint/branches?squad=NME&board=42")
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", recorder.Code, recorder.Body.String())
	}

	sprintBranches := decodeSprintBranches(t, recorder)
	expected := map[string]string{
		"CRMMS-6397":   "alpha/feature/CRMMS-6397-exact",
		"CRMMS-63970":  "alpha/feature/CRMMS-63970-one-extra-digit",
		"CRMMS-639712": "alpha/feature/CRMMS-639712-six-digits",
	}

	if len(sprintBranches.Matches) != len(expected) {
		t.Fatalf("expected %d keys, got %+v", len(expected), sprintBranches.Matches)
	}
	for issueKey, expectedBranch := range expected {
		matches := matchedBranchNames(sprintBranches.Matches[issueKey])
		if len(matches) != 1 || matches[0] != expectedBranch {
			t.Fatalf("%s must match only %q, got %v", issueKey, expectedBranch, matches)
		}
	}
}

func TestAdversarialIssueBranchSurfaceFindsAKeyAdjoinedByAWordCharacterOnEverySurface(t *testing.T) {
	rootPath := t.TempDir()
	requireGitBinary(t)
	createTransportKeyedRepository(t, filepath.Join(rootPath, "atlantis"), []string{"bugfix_CRMMS-6397-existing"})

	handler := newAdversarialIssueBranchHandler(rootPath, &stubSprintIssues{
		backlog: &jiraDomain.JiraBacklogResponse{
			Sprints: []jiraDomain.JiraSprint{
				{ID: 7, State: "active", OriginBoardID: 42, Issues: []jiraDomain.JiraIssue{{Key: "CRMMS-6397"}}},
			},
		},
	})

	singleRecorder := performIssueBranchRequest(t, issueBranchMux(handler), "user-a", "/api/jira/issues/CRMMS-6397/branches")
	expected := []string{"atlantis/bugfix_CRMMS-6397-existing"}

	singleMatches := matchedBranchNames(decodeIssueBranches(t, singleRecorder).Matches)
	if len(singleMatches) != 1 || singleMatches[0] != expected[0] {
		t.Fatalf("the single issue surface must find the branch, expected %v, got %v", expected, singleMatches)
	}

	bulkRecorder := performIssueBranchRequest(t, issueBranchMux(handler), "user-a", "/api/jira/sprint/branches?squad=NME&board=42")
	bulkMatches := matchedBranchNames(decodeSprintBranches(t, bulkRecorder).Matches["CRMMS-6397"])
	if len(bulkMatches) != 1 || bulkMatches[0] != expected[0] {
		t.Fatalf("the sprint surface must find the branch, expected %v, got %v", expected, bulkMatches)
	}
}
func TestAdversarialSprintSurfaceFailsTheWholeBatchWhenOneBacklogKeyIsMalformed(t *testing.T) {
	rootPath := newAdversarialDigitRunWorkspace(t)
	sprintIssues := &stubSprintIssues{backlog: &jiraDomain.JiraBacklogResponse{
		Sprints: []jiraDomain.JiraSprint{
			{
				ID:            7,
				State:         "active",
				OriginBoardID: 42,
				Issues: []jiraDomain.JiraIssue{
					{Key: "CRMMS-6397"},
					{Key: "not-a-key"},
				},
			},
		},
	}}
	handler := newAdversarialIssueBranchHandler(rootPath, sprintIssues)

	recorder := performIssueBranchRequest(t, issueBranchMux(handler), "user-a", "/api/jira/sprint/branches?squad=NME&board=42")
	t.Logf("characterisation: a malformed backlog key produced %d with body %s", recorder.Code, recorder.Body.String())

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected the malformed backlog key to refuse the whole sprint, got %d: %s", recorder.Code, recorder.Body.String())
	}
}
