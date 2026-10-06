package application

import (
	"context"
	"testing"
)

func TestSuggestionRanksEpicSiblingsFirstAndNamesThem(t *testing.T) {
	catalogue := testCatalogueService(
		stubCatalogueLocal{repositories: testLocalRepositories("mms-api")},
		stubCatalogueRemote{repositories: testRemoteRepositories("NME", "mms-api", "atlas")},
	)
	branches := &stubSuggestionBranches{matches: map[string][]BranchMatch{
		"CRMMS-77912": {{Repository: "mms-api", Branch: "feature/CRMMS-77912-first"}},
		"CRMMS-77913": {{Repository: "mms-api", Branch: "feature/CRMMS-77913-second"}},
		"CRMMS-77914": {{Repository: "atlas", Branch: "feature/CRMMS-77914-third"}},
	}}
	service := testSuggestionService(catalogue, stubSuggestionIssues{
		issue:   testIssue("CRMMS-77911", "CRMMS-100", "Enhancement Manage User", "Halaman utama manage user"),
		backlog: testBacklog("CRMMS-77912", "CRMMS-77913", "CRMMS-77914"),
	}, branches)

	suggestions, err := service.Suggest(context.Background(), "user-1", "crmms-77911")
	if err != nil {
		t.Fatalf("suggest failed: %v", err)
	}

	if suggestions.Key != "CRMMS-77911" {
		t.Fatalf("expected the canonical key, got %q", suggestions.Key)
	}
	if suggestions.Confidence != ConfidenceHigh {
		t.Fatalf("expected high confidence, got %q", suggestions.Confidence)
	}
	if len(suggestions.Suggestions) != 2 {
		t.Fatalf("expected two candidates, got %d", len(suggestions.Suggestions))
	}

	leader := suggestions.Suggestions[0]
	if leader.Repo != "mms-api" || !leader.IsCloned || leader.BitbucketProject != "NME" {
		t.Fatalf("unexpected leader: %+v", leader)
	}
	if leader.Score != 3.35 {
		t.Fatalf("expected a leader score of 3.35, got %v", leader.Score)
	}
	reason, found := findReason(leader, SignalEpicSiblings)
	if !found {
		t.Fatalf("the leader must carry an epic sibling reason, got %+v", leader.Reasons)
	}
	if reason.Weight != 5.0 || reason.Strength != 0.67 {
		t.Fatalf("unexpected sibling reason: %+v", reason)
	}
	if reason.Text != "2 other issues in this epic were worked in mms-api" {
		t.Fatalf("unexpected sibling text: %q", reason.Text)
	}
	if _, found := findReason(leader, SignalExplicitHint); found {
		t.Fatal("the issue does not name mms-api, so no hint reason may appear")
	}

	runnerUp := suggestions.Suggestions[1]
	if runnerUp.Repo != "atlas" || runnerUp.IsCloned {
		t.Fatalf("unexpected runner up: %+v", runnerUp)
	}
	if runnerUp.Score != 1.65 {
		t.Fatalf("expected a runner up score of 1.65, got %v", runnerUp.Score)
	}
	if _, found := findUnavailable(suggestions, SignalEpicSiblings); found {
		t.Fatal("a signal that produced evidence must not be reported as unavailable")
	}
	if _, found := findUnavailable(suggestions, SignalCodeIndex); !found {
		t.Fatal("an unconfigured code index must be reported")
	}
	if _, found := findUnavailable(suggestions, SignalConfluence); !found {
		t.Fatal("unconfigured linked pages must be reported")
	}

	if suggestions.Catalogue.LocalRepositoryCount != 1 || suggestions.Catalogue.RemoteRepositoryCount != 2 {
		t.Fatalf("unexpected catalogue counts: %+v", suggestions.Catalogue)
	}
	scope := suggestions.Scope
	if scope.LocalRepositoryCount != 1 || scope.RemoteRepositoryCount != 2 || scope.CoveragePercent != 50 {
		t.Fatalf("unexpected scope: %+v", scope)
	}
	if scope.VerifiedRemotely || scope.RemoteRepositoriesChecked != 0 {
		t.Fatalf("no Bitbucket scanner was configured: %+v", scope)
	}
	if len(branches.keys) != 3 {
		t.Fatalf("expected the sibling keys to drive one branch lookup, got %v", branches.keys)
	}

	assertEverySuggestionHasAReason(t, suggestions)
	assertReasonInvariants(t, suggestions)
}

func TestSuggestionWeighsAnExplicitHintAboveSiblings(t *testing.T) {
	catalogue := testCatalogueService(
		stubCatalogueLocal{repositories: testLocalRepositories("mms-api")},
		stubCatalogueRemote{repositories: testRemoteRepositories("NME", "mms-api", "atlas")},
	)
	branches := &stubSuggestionBranches{matches: map[string][]BranchMatch{
		"CRMMS-77912": {{Repository: "mms-api", Branch: "feature/CRMMS-77912-first"}},
		"CRMMS-77913": {{Repository: "mms-api", Branch: "feature/CRMMS-77913-second"}},
		"CRMMS-77914": {{Repository: "atlas", Branch: "feature/CRMMS-77914-third"}},
	}}
	service := testSuggestionService(catalogue, stubSuggestionIssues{
		issue:   testIssue("CRMMS-77911", "CRMMS-100", "Enhancement Manage User", "Halaman utama pada service atlas"),
		backlog: testBacklog("CRMMS-77912", "CRMMS-77913", "CRMMS-77914"),
	}, branches)

	suggestions, err := service.Suggest(context.Background(), "user-1", "CRMMS-77911")
	if err != nil {
		t.Fatalf("suggest failed: %v", err)
	}

	if suggestions.Suggestions[0].Repo != "atlas" {
		t.Fatalf("the named repository must lead, got %+v", suggestions.Suggestions)
	}
	if suggestions.Suggestions[0].Score != 5.65 {
		t.Fatalf("expected 1.65 from the sibling signal plus 4.0 from the hint, got %v", suggestions.Suggestions[0].Score)
	}
	hint, found := findReason(suggestions.Suggestions[0], SignalExplicitHint)
	if !found || hint.Weight != 4.0 || hint.Strength != 1.0 || hint.Text != "the issue names atlas" {
		t.Fatalf("unexpected hint reason: %+v", hint)
	}
	siblings, found := findReason(suggestions.Suggestions[0], SignalEpicSiblings)
	if !found || siblings.Weight != 5.0 || siblings.Strength != 0.33 {
		t.Fatalf("unexpected sibling reason: %+v", siblings)
	}
	if suggestions.Confidence != ConfidenceMedium {
		t.Fatalf("a leader below twice the runner up must not be high confidence, got %q", suggestions.Confidence)
	}

	assertEverySuggestionHasAReason(t, suggestions)
	assertReasonInvariants(t, suggestions)
}

func TestSuggestionReportsLowConfidenceWhenTheTopTwoAreWithinTwentyPercent(t *testing.T) {
	catalogue := testCatalogueService(
		stubCatalogueLocal{},
		stubCatalogueRemote{repositories: testRemoteRepositories("NME", "mms-api", "atlas")},
	)
	branches := &stubSuggestionBranches{matches: map[string][]BranchMatch{
		"CRMMS-77912": {{Repository: "mms-api", Branch: "feature/CRMMS-77912-first"}},
		"CRMMS-77913": {{Repository: "atlas", Branch: "feature/CRMMS-77913-second"}},
	}}
	service := testSuggestionService(catalogue, stubSuggestionIssues{
		issue:   testIssue("CRMMS-77911", "CRMMS-100", "Enhancement Manage User", "Halaman utama manage user"),
		backlog: testBacklog("CRMMS-77912", "CRMMS-77913"),
	}, branches)

	suggestions, err := service.Suggest(context.Background(), "user-1", "CRMMS-77911")
	if err != nil {
		t.Fatalf("suggest failed: %v", err)
	}

	if suggestions.Confidence != ConfidenceLow {
		t.Fatalf("a tie must be low confidence, got %q", suggestions.Confidence)
	}
	if len(suggestions.Suggestions) != 2 {
		t.Fatalf("a low confidence result is still a list of candidates, got %d", len(suggestions.Suggestions))
	}
	if suggestions.Suggestions[0].Score != suggestions.Suggestions[1].Score {
		t.Fatalf("expected tied candidates, got %+v", suggestions.Suggestions)
	}

	assertEverySuggestionHasAReason(t, suggestions)
	assertReasonInvariants(t, suggestions)
}

func TestSuggestionCapsTheListAtThree(t *testing.T) {
	catalogue := testCatalogueService(
		stubCatalogueLocal{repositories: testLocalRepositories("mms-api")},
		stubCatalogueRemote{repositories: testRemoteRepositories("NME", "mms-api", "atlas", "billing", "core")},
	)
	branches := &stubSuggestionBranches{matches: map[string][]BranchMatch{
		"CRMMS-77912": {{Repository: "mms-api", Branch: "feature/CRMMS-77912-a"}},
		"CRMMS-77913": {{Repository: "mms-api", Branch: "feature/CRMMS-77913-b"}},
		"CRMMS-77914": {{Repository: "mms-api", Branch: "feature/CRMMS-77914-c"}},
		"CRMMS-77915": {{Repository: "atlas", Branch: "feature/CRMMS-77915-d"}},
		"CRMMS-77916": {{Repository: "billing", Branch: "feature/CRMMS-77916-e"}},
		"CRMMS-77917": {{Repository: "core", Branch: "feature/CRMMS-77917-f"}},
	}}
	service := testSuggestionService(catalogue, stubSuggestionIssues{
		issue:   testIssue("CRMMS-77911", "CRMMS-100", "Enhancement Manage User", "Halaman utama manage user"),
		backlog: testBacklog("CRMMS-77912", "CRMMS-77913", "CRMMS-77914", "CRMMS-77915", "CRMMS-77916", "CRMMS-77917"),
	}, branches)

	suggestions, err := service.Suggest(context.Background(), "user-1", "CRMMS-77911")
	if err != nil {
		t.Fatalf("suggest failed: %v", err)
	}

	if len(suggestions.Suggestions) != maximumSuggestions {
		t.Fatalf("expected %d suggestions, got %d", maximumSuggestions, len(suggestions.Suggestions))
	}
	if suggestions.Suggestions[0].Repo != "mms-api" {
		t.Fatalf("the best supported repository must lead, got %+v", suggestions.Suggestions)
	}
	for index := 1; index < len(suggestions.Suggestions); index++ {
		if suggestions.Suggestions[index-1].Score < suggestions.Suggestions[index].Score {
			t.Fatalf("suggestions must be ordered by score, got %+v", suggestions.Suggestions)
		}
	}
	for _, suggestion := range suggestions.Suggestions {
		if suggestion.Repo == "core" {
			t.Fatal("the fourth repository must be cut by the cap")
		}
	}

	assertEverySuggestionHasAReason(t, suggestions)
	assertReasonInvariants(t, suggestions)
}
