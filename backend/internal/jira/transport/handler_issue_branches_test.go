package transport

import (
	"errors"
	"net/http"
	"strings"
	"testing"

	jiraDomain "lunar/backend/internal/jira/domain"
	workspaceApp "lunar/backend/internal/workspace/application"
	workspaceDomain "lunar/backend/internal/workspace/domain"
	"lunar/backend/internal/workspace/infrastructure"
)

func TestIssueBranchHandlerFindsTheKeyedBranch(t *testing.T) {
	rootPath := newKeyedTransportWorkspace(t)
	handler := NewIssueBranchHandler(
		workspaceApp.NewIssueBranchService(infrastructure.NewGitInspector([]string{rootPath})),
		stubWorkspaceRoots{rootPath: rootPath},
		&stubSprintIssues{},
	)

	recorder := performIssueBranchRequest(t, issueBranchMux(handler), "user-a", "/api/jira/issues/CRMMS-6397/branches")
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", recorder.Code, recorder.Body.String())
	}

	branches := decodeIssueBranches(t, recorder)
	if branches.Key != "CRMMS-6397" {
		t.Fatalf("expected the requested key, got %q", branches.Key)
	}

	expected := []string{"alpha/feature/CRMMS-6397-first", "beta/feature/crmms-6397-lower"}
	matches := matchedBranchNames(branches.Matches)
	if len(matches) != len(expected) {
		t.Fatalf("expected %v, got %v", expected, matches)
	}
	for index, expectedMatch := range expected {
		if matches[index] != expectedMatch {
			t.Fatalf("expected %v, got %v", expected, matches)
		}
	}

	if branches.Scope.LocalRepositoryCount != 2 || branches.Scope.VerifiedRemotely || branches.Scope.RemoteRepositoriesChecked != 0 {
		t.Fatalf("unexpected scope: %+v", branches.Scope)
	}
}

func TestIssueBranchHandlerKeepsALongerKeySeparateFromItsPrefix(t *testing.T) {
	rootPath := newKeyedTransportWorkspace(t)
	handler := NewIssueBranchHandler(
		workspaceApp.NewIssueBranchService(infrastructure.NewGitInspector([]string{rootPath})),
		stubWorkspaceRoots{rootPath: rootPath},
		&stubSprintIssues{},
	)

	recorder := performIssueBranchRequest(t, issueBranchMux(handler), "user-a", "/api/jira/issues/CRMMS-63970/branches")
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", recorder.Code, recorder.Body.String())
	}

	matches := matchedBranchNames(decodeIssueBranches(t, recorder).Matches)
	if len(matches) != 1 || matches[0] != "alpha/feature/CRMMS-63970-second" {
		t.Fatalf("expected only the longer key branch, got %v", matches)
	}
}

func TestIssueBranchHandlerReturnsAnEmptyArrayWhenNothingMatches(t *testing.T) {
	rootPath := newKeyedTransportWorkspace(t)
	handler := NewIssueBranchHandler(
		workspaceApp.NewIssueBranchService(infrastructure.NewGitInspector([]string{rootPath})),
		stubWorkspaceRoots{rootPath: rootPath},
		&stubSprintIssues{},
	)

	recorder := performIssueBranchRequest(t, issueBranchMux(handler), "user-a", "/api/jira/issues/CRMMS-11111/branches")
	if recorder.Code != http.StatusOK {
		t.Fatalf("absence is not an error, expected 200, got %d: %s", recorder.Code, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), `"matches":[]`) {
		t.Fatalf("expected an empty matches array, got %s", recorder.Body.String())
	}
}

func TestIssueBranchHandlerRefusesAnInvalidKey(t *testing.T) {
	rootPath := newKeyedTransportWorkspace(t)
	handler := NewIssueBranchHandler(
		workspaceApp.NewIssueBranchService(infrastructure.NewGitInspector([]string{rootPath})),
		stubWorkspaceRoots{rootPath: rootPath},
		&stubSprintIssues{},
	)

	recorder := performIssueBranchRequest(t, issueBranchMux(handler), "user-a", "/api/jira/issues/not-a-key/branches")
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", recorder.Code, recorder.Body.String())
	}
}

func TestIssueBranchHandlerRequiresASession(t *testing.T) {
	handler := NewIssueBranchHandler(
		workspaceApp.NewIssueBranchService(&countingBranchScanner{}),
		stubWorkspaceRoots{rootPath: "/workspace"},
		&stubSprintIssues{},
	)

	recorder := performIssueBranchRequest(t, issueBranchMux(handler), "", "/api/jira/issues/CRMMS-6397/branches")
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", recorder.Code, recorder.Body.String())
	}
}

func TestIssueBranchHandlerReportsAMissingWorkspace(t *testing.T) {
	handler := NewIssueBranchHandler(
		workspaceApp.NewIssueBranchService(&countingBranchScanner{}),
		stubWorkspaceRoots{err: workspaceApp.ErrWorkspaceNotUsable},
		&stubSprintIssues{},
	)

	recorder := performIssueBranchRequest(t, issueBranchMux(handler), "user-a", "/api/jira/issues/CRMMS-6397/branches")
	if recorder.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d: %s", recorder.Code, recorder.Body.String())
	}
}

func TestIssueBranchHandlerReportsAGitTimeoutAsGatewayTimeout(t *testing.T) {
	handler := NewIssueBranchHandler(
		workspaceApp.NewIssueBranchService(&countingBranchScanner{}),
		stubWorkspaceRoots{err: workspaceDomain.ErrGitTimeout},
		&stubSprintIssues{},
	)

	recorder := performIssueBranchRequest(t, issueBranchMux(handler), "user-a", "/api/jira/issues/CRMMS-6397/branches")
	if recorder.Code != http.StatusGatewayTimeout {
		t.Fatalf("expected 504, got %d: %s", recorder.Code, recorder.Body.String())
	}
}

func TestIssueBranchHandlerSprintScansOnceForEveryIssue(t *testing.T) {
	scanner := &countingBranchScanner{
		branchesByRepository: map[string][]workspaceDomain.Branch{
			"alpha": {
				{Name: "feature/CRMMS-6397-first", IsCurrent: true},
				{Name: "feature/CRMMS-77745-second"},
			},
		},
	}
	sprintIssues := &stubSprintIssues{backlog: &jiraDomain.JiraBacklogResponse{
		Sprints: []jiraDomain.JiraSprint{
			{
				ID:            7,
				Name:          "Sprint 7",
				State:         "active",
				OriginBoardID: 42,
				Issues: []jiraDomain.JiraIssue{
					{Key: "CRMMS-6397"},
					{Key: "CRMMS-77745"},
					{Key: "CRMMS-77740"},
					{Key: "CRMMS-77911"},
					{Key: "CRMMS-77741"},
					{Key: "CRMMS-77742"},
					{Key: "CRMMS-77743"},
				},
			},
		},
	}}
	handler := NewIssueBranchHandler(
		workspaceApp.NewIssueBranchService(scanner),
		stubWorkspaceRoots{rootPath: "/workspace"},
		sprintIssues,
	)

	recorder := performIssueBranchRequest(t, issueBranchMux(handler), "user-a", "/api/jira/sprint/branches?squad=NME&board=42")
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", recorder.Code, recorder.Body.String())
	}

	sprintBranches := decodeSprintBranches(t, recorder)
	if len(sprintBranches.Matches) != 7 {
		t.Fatalf("expected seven keys, got %d", len(sprintBranches.Matches))
	}
	if scanner.calls != 1 {
		t.Fatalf("expected one workspace scan for the whole sprint, got %d", scanner.calls)
	}
	if len(sprintBranches.Matches["CRMMS-6397"]) != 1 {
		t.Fatalf("expected the CRMMS-6397 branch, got %+v", sprintBranches.Matches["CRMMS-6397"])
	}
	if sprintBranches.Matches["CRMMS-77740"] == nil {
		t.Fatal("a key without a match must carry an empty array, not null")
	}
	if sprintIssues.requests != 1 {
		t.Fatalf("expected one Jira backlog read, got %d", sprintIssues.requests)
	}
}

func TestIssueBranchHandlerSprintRefusesAnInvalidBoard(t *testing.T) {
	handler := NewIssueBranchHandler(
		workspaceApp.NewIssueBranchService(&countingBranchScanner{}),
		stubWorkspaceRoots{rootPath: "/workspace"},
		&stubSprintIssues{},
	)

	recorder := performIssueBranchRequest(t, issueBranchMux(handler), "user-a", "/api/jira/sprint/branches?board=not-a-number")
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", recorder.Code, recorder.Body.String())
	}
}

func TestIssueBranchHandlerSprintReportsAnUnreachableJira(t *testing.T) {
	handler := NewIssueBranchHandler(
		workspaceApp.NewIssueBranchService(&countingBranchScanner{}),
		stubWorkspaceRoots{rootPath: "/workspace"},
		&stubSprintIssues{err: errors.New("jira is unreachable")},
	)

	recorder := performIssueBranchRequest(t, issueBranchMux(handler), "user-a", "/api/jira/sprint/branches?squad=NME")
	if recorder.Code != http.StatusBadGateway {
		t.Fatalf("expected 502, got %d: %s", recorder.Code, recorder.Body.String())
	}
}

func TestSelectSprintIssueKeysPrefersTheActiveSprintOfTheBoard(t *testing.T) {
	sprints := []jiraDomain.JiraSprint{
		{ID: 1, Name: "Sprint 1", State: "closed", OriginBoardID: 9, Issues: []jiraDomain.JiraIssue{{Key: "CRMMS-1"}}},
		{ID: 2, Name: "Sprint 2", State: "active", OriginBoardID: 9, Issues: []jiraDomain.JiraIssue{{Key: "CRMMS-2"}}},
		{ID: 3, Name: "Sprint 3", State: "active", OriginBoardID: 42, Issues: []jiraDomain.JiraIssue{{Key: "CRMMS-3"}}},
	}

	issueKeys := selectSprintIssueKeys(sprints, 42)
	if len(issueKeys) != 1 || issueKeys[0] != "CRMMS-3" {
		t.Fatalf("expected the active sprint of board 42, got %v", issueKeys)
	}

	issueKeys = selectSprintIssueKeys(sprints, 0)
	if len(issueKeys) != 1 || issueKeys[0] != "CRMMS-2" {
		t.Fatalf("expected the first active sprint, got %v", issueKeys)
	}

	if len(selectSprintIssueKeys(nil, 0)) != 0 {
		t.Fatalf("expected no keys when there is no sprint, got %v", selectSprintIssueKeys(nil, 0))
	}
}

func TestSelectSprintIssueKeysSortsAndDeduplicates(t *testing.T) {
	sprints := []jiraDomain.JiraSprint{
		{
			ID:            2,
			State:         "active",
			OriginBoardID: 42,
			Issues: []jiraDomain.JiraIssue{
				{Key: "CRMMS-77745"},
				{Key: "crmms-6397"},
				{Key: "CRMMS-77745"},
				{Key: ""},
			},
		},
	}

	issueKeys := selectSprintIssueKeys(sprints, 42)
	if len(issueKeys) != 2 || issueKeys[0] != "CRMMS-6397" || issueKeys[1] != "CRMMS-77745" {
		t.Fatalf("expected two sorted unique keys, got %v", issueKeys)
	}
}
