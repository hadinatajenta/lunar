package application

import (
	"context"
	"errors"
	"testing"
)

func TestSuggestionScansSiblingKeysOnlyInRepositoriesItCannotSeeLocally(t *testing.T) {
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

	if calls := scanner.calledLists(); len(calls) != 1 || calls[0] != "NME/atlas" {
		t.Fatalf("only the repository the local scan cannot see may be listed, got %v", calls)
	}
	if !suggestions.Scope.VerifiedRemotely || suggestions.Scope.RemoteRepositoriesChecked != 1 {
		t.Fatalf("expected the remote coverage to be reported, got %+v", suggestions.Scope)
	}
	if suggestions.Suggestions[0].Repo != "mms-api" || suggestions.Suggestions[1].Repo != "atlas" {
		t.Fatalf("a branch for another key must not invent a ranking change, got %+v", suggestions.Suggestions)
	}

	assertEverySuggestionHasAReason(t, suggestions)
	assertReasonInvariants(t, suggestions)
}

func TestSuggestionFindsASiblingBranchOnlyOnBitbucket(t *testing.T) {
	catalogue := testCatalogueService(
		stubCatalogueLocal{repositories: testLocalRepositories("mms-api")},
		stubCatalogueRemote{repositories: testRemoteRepositories("NME", "mms-api", "atlas")},
	)
	branches := &stubSuggestionBranches{matches: map[string][]BranchMatch{
		"CRMMS-77912": {{Repository: "mms-api", Branch: "feature/CRMMS-77912-first"}},
	}}
	scanner := &stubSuggestionRemoteScanner{branchesByRepository: map[string][]string{
		"NME/atlas": {"feature/CRMMS-77913-second", "feature/CRMMS-77914-third"},
	}}
	service := testSuggestionService(catalogue, stubSuggestionIssues{
		issue:   testIssue("CRMMS-77911", "CRMMS-100", "Enhancement Manage User", "Halaman utama manage user"),
		backlog: testBacklog("CRMMS-77912", "CRMMS-77913", "CRMMS-77914"),
	}, branches, WithRemoteCandidateScanner(NewRemoteCandidateScanner(scanner)))

	suggestions, err := service.Suggest(context.Background(), "user-1", "CRMMS-77911")
	if err != nil {
		t.Fatalf("suggest failed: %v", err)
	}

	if len(suggestions.Suggestions) != 2 || suggestions.Suggestions[0].Repo != "atlas" {
		t.Fatalf("the repository found only on Bitbucket must lead, got %+v", suggestions.Suggestions)
	}
	leader := suggestions.Suggestions[0]
	if leader.Score != 3.35 || leader.IsCloned {
		t.Fatalf("unexpected leader: %+v", leader)
	}
	reason, found := findReason(leader, SignalEpicSiblings)
	if !found || reason.Weight != 5.0 || reason.Strength != 0.67 {
		t.Fatalf("unexpected sibling reason: %+v", reason)
	}
	if reason.Text != "2 other issues in this epic were worked in atlas, found only on Bitbucket" {
		t.Fatalf("the reason must state that the evidence came from Bitbucket, got %q", reason.Text)
	}
	if suggestions.Suggestions[1].Repo != "mms-api" || suggestions.Suggestions[1].Score != 1.65 {
		t.Fatalf("unexpected runner up: %+v", suggestions.Suggestions[1])
	}
	localReason, found := findReason(suggestions.Suggestions[1], SignalEpicSiblings)
	if !found || localReason.Text != "1 other issue in this epic was worked in mms-api" {
		t.Fatalf("a locally found sibling must not mention Bitbucket, got %q", localReason.Text)
	}
	if suggestions.Confidence != ConfidenceHigh {
		t.Fatalf("the leader is twice the runner up, so it is high confidence, got %q", suggestions.Confidence)
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
	if suggestions.Scope.RemoteRepositoriesChecked != 1 {
		t.Fatalf("the attempt is still reported, got %+v", suggestions.Scope)
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

func TestSuggestionSkipsBitbucketWithoutSiblingsToLookFor(t *testing.T) {
	catalogue := testCatalogueService(
		stubCatalogueLocal{repositories: testLocalRepositories("mms-api")},
		stubCatalogueRemote{repositories: testRemoteRepositories("NME", "mms-api", "atlas")},
	)
	scanner := &stubSuggestionRemoteScanner{branchesByRepository: map[string][]string{
		"NME/atlas": {"feature/CRMMS-77913-second"},
	}}
	service := testSuggestionService(catalogue, stubSuggestionIssues{
		issue:   testIssue("CRMMS-77911", "CRMMS-100", "Enhancement Manage User", "Halaman utama manage user"),
		backlog: testBacklog(),
	}, &stubSuggestionBranches{}, WithRemoteCandidateScanner(NewRemoteCandidateScanner(scanner)))

	suggestions, err := service.Suggest(context.Background(), "user-1", "CRMMS-77911")
	if err != nil {
		t.Fatalf("suggest failed: %v", err)
	}

	if calls := scanner.calledLists(); len(calls) != 0 {
		t.Fatalf("there is no sibling to look for, so Bitbucket must not be called, got %v", calls)
	}
	if suggestions.Scope.VerifiedRemotely || suggestions.Scope.RemoteRepositoriesChecked != 0 {
		t.Fatalf("nothing was checked, so nothing may be claimed: %+v", suggestions.Scope)
	}
}

func TestSuggestionPropagatesACancelledRemoteScan(t *testing.T) {
	catalogue := testCatalogueService(
		stubCatalogueLocal{repositories: testLocalRepositories("mms-api")},
		stubCatalogueRemote{repositories: testRemoteRepositories("NME", "mms-api", "atlas")},
	)
	branches := &stubSuggestionBranches{matches: map[string][]BranchMatch{
		"CRMMS-77912": {{Repository: "mms-api", Branch: "feature/CRMMS-77912-first"}},
	}}
	scanner := &stubSuggestionRemoteScanner{branchesByRepository: map[string][]string{
		"NME/atlas": {"feature/CRMMS-77913-second"},
	}}
	service := testSuggestionService(catalogue, stubSuggestionIssues{
		issue:   testIssue("CRMMS-77911", "CRMMS-100", "Enhancement Manage User", "Halaman utama manage user"),
		backlog: testBacklog("CRMMS-77912", "CRMMS-77913"),
	}, branches, WithRemoteCandidateScanner(NewRemoteCandidateScanner(scanner)))

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if _, err := service.Suggest(ctx, "user-1", "CRMMS-77911"); !errors.Is(err, context.Canceled) {
		t.Fatalf("a cancelled caller must be reported, not swallowed, got %v", err)
	}
}

func TestSiblingEvidenceOnlySendsCanonicalKeysRemotely(t *testing.T) {
	evidence := newSiblingEvidence("CRMMS-100", false)
	evidence.siblingKeys["crmms-77911"] = struct{}{}
	evidence.siblingKeys["CRMMS-6397"] = struct{}{}
	evidence.siblingKeys["not-a-key"] = struct{}{}

	keys := evidence.remoteLookupKeys()
	expected := []string{"CRMMS-6397", "CRMMS-77911"}
	if len(keys) != len(expected) {
		t.Fatalf("expected %v, got %v", expected, keys)
	}
	for index, key := range expected {
		if keys[index] != key {
			t.Fatalf("expected %v, got %v", expected, keys)
		}
	}
}
