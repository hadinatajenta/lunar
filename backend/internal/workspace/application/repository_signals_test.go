package application

import (
	"context"
	"errors"
	"sort"
	"testing"

	jiraDomain "lunar/backend/internal/jira/domain"
)

func TestSuggestionScoresCodeIndexTermsByRarityAndRequiresTwoTerms(t *testing.T) {
	catalogue := testCatalogueService(
		stubCatalogueLocal{repositories: testLocalRepositories("alpha", "beta", "gamma")},
		nil,
	)
	code := stubSuggestionCode{matchesByTerm: map[string][]CodeMatch{
		"perubahan": {{Repo: "alpha", FilePath: "src/a.go"}, {Repo: "beta", FilePath: "src/b.go"}, {Repo: "gamma", FilePath: "src/c.go"}},
		"penutupan": {{Repo: "alpha", FilePath: "src/a.go"}},
		"riwayat":   {{Repo: "alpha", FilePath: "src/d.go"}},
		"store":     {{Repo: "alpha", FilePath: "src/e.go"}},
		"code":      {{Repo: "alpha", FilePath: "src/f.go"}},
	}}
	service := testSuggestionService(catalogue, stubSuggestionIssues{
		issue:   testIssue("CRMMS-77911", "CRMMS-100", "Penutupan", "penutupan store code riwayat perubahan"),
		backlog: testBacklog(),
	}, &stubSuggestionBranches{}, WithCodeSearch(code))

	suggestions, err := service.Suggest(context.Background(), "user-1", "CRMMS-77911")
	if err != nil {
		t.Fatalf("suggest failed: %v", err)
	}

	if len(suggestions.Suggestions) != 1 || suggestions.Suggestions[0].Repo != "alpha" {
		t.Fatalf("expected only alpha to clear the two term minimum, got %+v", suggestions.Suggestions)
	}
	reason, found := findReason(suggestions.Suggestions[0], SignalCodeIndex)
	if !found {
		t.Fatalf("expected a code index reason, got %+v", suggestions.Suggestions[0].Reasons)
	}
	if reason.Weight != 2.0 || reason.Strength != 1.0 {
		t.Fatalf("unexpected code index reason: %+v", reason)
	}
	if suggestions.Suggestions[0].Score != 2.0 {
		t.Fatalf("expected the code index alone to score 2.0, got %v", suggestions.Suggestions[0].Score)
	}
	if suggestions.Confidence != ConfidenceMedium {
		t.Fatalf("a code index leader is never high confidence, got %q", suggestions.Confidence)
	}

	assertReasonInvariants(t, suggestions)
}

func TestSuggestionKeepsTheCodeIndexScopedToClonedRepositories(t *testing.T) {
	catalogue := testCatalogueService(
		stubCatalogueLocal{repositories: testLocalRepositories("alpha")},
		stubCatalogueRemote{repositories: testRemoteRepositories("NME", "alpha", "uncloned")},
	)
	code := stubSuggestionCode{matchesByTerm: map[string][]CodeMatch{
		"penutupan": {{Repo: "alpha", FilePath: "src/a.go"}, {Repo: "uncloned", FilePath: "src/u.go"}},
		"riwayat":   {{Repo: "alpha", FilePath: "src/b.go"}, {Repo: "uncloned", FilePath: "src/u.go"}},
	}}
	service := testSuggestionService(catalogue, stubSuggestionIssues{
		issue:   testIssue("CRMMS-77911", "CRMMS-100", "Penutupan", "penutupan riwayat"),
		backlog: testBacklog(),
	}, &stubSuggestionBranches{}, WithCodeSearch(code))

	suggestions, err := service.Suggest(context.Background(), "user-1", "CRMMS-77911")
	if err != nil {
		t.Fatalf("suggest failed: %v", err)
	}

	if len(suggestions.Suggestions) != 1 || suggestions.Suggestions[0].Repo != "alpha" {
		t.Fatalf("a repository that is not cloned cannot be scored by the code index, got %+v", suggestions.Suggestions)
	}
}

func TestSuggestionFindsARepositoryNamedInALinkedPage(t *testing.T) {
	catalogue := testCatalogueService(stubCatalogueLocal{repositories: testLocalRepositories("alpha")}, nil)
	pages := stubSuggestionPages{pages: []LinkedPage{
		{Title: "MDR maintenance", Body: "the change lands in alpha and touches the menu"},
	}}
	service := testSuggestionService(catalogue, stubSuggestionIssues{
		issue:   testIssue("CRMMS-77911", "CRMMS-100", "Maintenance", "the menu changes"),
		backlog: testBacklog(),
	}, &stubSuggestionBranches{}, WithLinkedPages(pages))

	suggestions, err := service.Suggest(context.Background(), "user-1", "CRMMS-77911")
	if err != nil {
		t.Fatalf("suggest failed: %v", err)
	}

	if len(suggestions.Suggestions) != 1 {
		t.Fatalf("expected alpha from the linked page, got %+v", suggestions.Suggestions)
	}
	reason, found := findReason(suggestions.Suggestions[0], SignalConfluence)
	if !found || reason.Weight != 1.5 || reason.Strength != 1.0 {
		t.Fatalf("unexpected confluence reason: %+v", reason)
	}
	if reason.Text != "the linked page MDR maintenance names alpha" {
		t.Fatalf("the reason must name the page, got %q", reason.Text)
	}
	if _, found := findUnavailable(suggestions, SignalConfluence); found {
		t.Fatal("a signal that produced evidence must not be reported as unavailable")
	}
}

func TestSuggestionFallsBackToTheSprintWhenTheIssueHasNoEpic(t *testing.T) {
	catalogue := testCatalogueService(stubCatalogueLocal{repositories: testLocalRepositories("alpha")}, nil)
	issue := testIssue("CRMMS-77911", "", "Maintenance", "the menu changes")
	issue.Fields.Parent = nil
	backlog := &jiraDomain.JiraBacklogResponse{Sprints: []jiraDomain.JiraSprint{{Name: "Sprint 21", State: "active", Issues: []jiraDomain.JiraIssue{
		{Key: "CRMMS-77912", SprintName: "Sprint 21"},
		{Key: "CRMMS-77913", SprintName: "Sprint 22"},
	}}}}
	branches := &stubSuggestionBranches{matches: map[string][]BranchMatch{
		"CRMMS-77912": {{Repository: "alpha", Branch: "feature/CRMMS-77912-first"}},
		"CRMMS-77913": {{Repository: "alpha", Branch: "feature/CRMMS-77913-second"}},
	}}
	service := testSuggestionService(catalogue, stubSuggestionIssues{issue: issue, backlog: backlog}, branches)

	suggestions, err := service.Suggest(context.Background(), "user-1", "CRMMS-77911")
	if err != nil {
		t.Fatalf("suggest failed: %v", err)
	}

	if len(branches.keys) != 1 || branches.keys[0] != "CRMMS-77912" {
		t.Fatalf("the sprint fallback must stay in the issue's sprint, got %v", branches.keys)
	}
	reason, found := findReason(suggestions.Suggestions[0], SignalEpicSiblings)
	if !found {
		t.Fatalf("expected a sibling reason, got %+v", suggestions.Suggestions)
	}
	if reason.Text != "1 other issue in this sprint was worked in alpha" {
		t.Fatalf("the reason must state the sprint fallback, got %q", reason.Text)
	}
}

func TestSuggestionReportsASignalFailureWithoutFailingTheRequest(t *testing.T) {
	catalogue := testCatalogueService(stubCatalogueLocal{repositories: testLocalRepositories("alpha")}, nil)
	service := testSuggestionService(catalogue, stubSuggestionIssues{
		issue:   testIssue("CRMMS-77911", "CRMMS-100", "Penutupan", "penutupan riwayat"),
		backlog: testBacklog(),
	}, &stubSuggestionBranches{},
		WithCodeSearch(stubSuggestionCode{err: errors.New("the index is down")}),
		WithLinkedPages(stubSuggestionPages{err: errors.New("confluence is down")}),
	)

	suggestions, err := service.Suggest(context.Background(), "user-1", "CRMMS-77911")
	if err != nil {
		t.Fatalf("a degraded signal must not fail the request: %v", err)
	}

	codeIndex, found := findUnavailable(suggestions, SignalCodeIndex)
	if !found || codeIndex.Reason != codeIndexFailedReason {
		t.Fatalf("expected the code index failure to be stated, got %+v", suggestions.SignalsUnavailable)
	}
	pages, found := findUnavailable(suggestions, SignalConfluence)
	if !found || pages.Reason != pageFailedReason {
		t.Fatalf("expected the page failure to be stated, got %+v", suggestions.SignalsUnavailable)
	}
}

func TestSuggestionVerifiesTheCandidatesOnBitbucket(t *testing.T) {
	catalogue := testCatalogueService(
		stubCatalogueLocal{repositories: testLocalRepositories("mms-api")},
		stubCatalogueRemote{repositories: testRemoteRepositories("NME", "mms-api", "atlas")},
	)
	branches := &stubSuggestionBranches{matches: map[string][]BranchMatch{
		"CRMMS-77912": {{Repository: "mms-api", Branch: "feature/CRMMS-77912-first"}},
		"CRMMS-77913": {{Repository: "atlas", Branch: "feature/CRMMS-77913-second"}},
		"CRMMS-77914": {{Repository: "mms-api", Branch: "feature/CRMMS-77914-third"}},
	}}
	scanner := &stubSuggestionRemoteScanner{branchesByRepository: map[string][]string{
		"NME/atlas": {"feature/CRMMS-77911-Enhancement-Manage-User"},
	}}
	service := testSuggestionService(catalogue, stubSuggestionIssues{
		issue:   testIssue("CRMMS-77911", "CRMMS-100", "Enhancement Manage User", "Halaman utama manage user"),
		backlog: testBacklog("CRMMS-77912", "CRMMS-77913", "CRMMS-77914"),
	}, branches, WithRemoteCandidateScanner(NewRemoteCandidateScanner(scanner)))

	suggestions, err := service.Suggest(context.Background(), "user-1", "CRMMS-77911")
	if err != nil {
		t.Fatalf("suggest failed: %v", err)
	}

	calls := scanner.calledSearches()
	sort.Strings(calls)
	expected := []string{"NME/atlas|CRMMS-77911", "NME/mms-api|CRMMS-77911"}
	if len(calls) != len(expected) {
		t.Fatalf("expected the top candidates to be checked, got %v", calls)
	}
	for index, call := range expected {
		if calls[index] != call {
			t.Fatalf("expected %v, got %v", expected, calls)
		}
	}
	if !suggestions.Scope.VerifiedRemotely || suggestions.Scope.RemoteRepositoriesChecked != 2 {
		t.Fatalf("expected the remote check to be reported, got %+v", suggestions.Scope)
	}
	if suggestions.Suggestions[0].Repo != "mms-api" || suggestions.Suggestions[1].Repo != "atlas" {
		t.Fatalf("the remote check must not invent a ranking change, got %+v", suggestions.Suggestions)
	}
	for _, suggestion := range suggestions.Suggestions {
		for _, reason := range suggestion.Reasons {
			if reason.Signal == SignalEpicSiblings && reason.Text == "" {
				t.Fatalf("unexpected reason: %+v", reason)
			}
		}
	}

	assertEverySuggestionHasAReason(t, suggestions)
	assertReasonInvariants(t, suggestions)
}

func TestSuggestionDegradesToTheLocalTierWhenBitbucketFails(t *testing.T) {
	catalogue := testCatalogueService(
		stubCatalogueLocal{repositories: testLocalRepositories("mms-api")},
		stubCatalogueRemote{repositories: testRemoteRepositories("NME", "mms-api", "atlas")},
	)
	branches := &stubSuggestionBranches{matches: map[string][]BranchMatch{
		"CRMMS-77912": {{Repository: "mms-api", Branch: "feature/CRMMS-77912-first"}},
		"CRMMS-77913": {{Repository: "atlas", Branch: "feature/CRMMS-77913-second"}},
	}}
	scanner := &stubSuggestionRemoteScanner{err: errors.New("bitbucket is unreachable")}
	service := testSuggestionService(catalogue, stubSuggestionIssues{
		issue:   testIssue("CRMMS-77911", "CRMMS-100", "Enhancement Manage User", "Halaman utama manage user"),
		backlog: testBacklog("CRMMS-77912", "CRMMS-77913"),
	}, branches, WithRemoteCandidateScanner(NewRemoteCandidateScanner(scanner)))

	suggestions, err := service.Suggest(context.Background(), "user-1", "CRMMS-77911")
	if err != nil {
		t.Fatalf("an unreachable Bitbucket must degrade, not fail: %v", err)
	}

	if suggestions.Scope.VerifiedRemotely {
		t.Fatalf("a failed check must never be reported as verified: %+v", suggestions.Scope)
	}
	if suggestions.Scope.RemoteRepositoriesChecked != 2 {
		t.Fatalf("the attempts are still reported, got %+v", suggestions.Scope)
	}
	if len(suggestions.Suggestions) != 2 {
		t.Fatalf("the local tier must survive, got %+v", suggestions.Suggestions)
	}
	degraded, found := findUnavailable(suggestions, SignalEpicSiblings)
	if !found || degraded.Reason != bitbucketPartialReason {
		t.Fatalf("an incomplete check must be stated, got %+v", suggestions.SignalsUnavailable)
	}

	assertEverySuggestionHasAReason(t, suggestions)
	assertReasonInvariants(t, suggestions)
}
