package application

import (
	"context"
	"math"
	"testing"
	"time"
)

func TestQASuggestionScoreDepartsFromTheReasonSumOnceFeedbackExists(t *testing.T) {
	service, _ := feedbackFixture(t)

	before, err := service.Suggest(context.Background(), "user-1", "CRMMS-77911")
	if err != nil {
		t.Fatalf("suggest failed: %v", err)
	}
	if len(before.Suggestions) == 0 {
		t.Fatal("the fixture must produce at least one suggestion")
	}

	leader := before.Suggestions[0]
	sumBefore := reasonContribution(leader)
	if math.Abs(leader.Score-sumBefore) > 0.011 {
		t.Fatalf("without feedback the score must equal the reason contribution, score=%v sum=%v", leader.Score, sumBefore)
	}

	if err := service.RecordFeedback(context.Background(), "user-1", "CRMMS-77911", leader.Repo, FeedbackAdded); err != nil {
		t.Fatalf("record failed: %v", err)
	}

	after, err := service.Suggest(context.Background(), "user-1", "CRMMS-77911")
	if err != nil {
		t.Fatalf("suggest failed: %v", err)
	}
	promoted := findSuggestion(after.Suggestions, leader.Repo)
	sumAfter := reasonContribution(promoted)
	if math.Abs(promoted.Score-sumAfter) < 1.0 {
		t.Fatalf("the score no longer contains an unexplained feedback adjustment, score=%v sum=%v; update this characterisation if that is intended", promoted.Score, sumAfter)
	}

	t.Logf("QA: repo=%s reasonSum=%.2f score=%.2f hiddenAdjustment=%.2f reasons=%d", promoted.Repo, sumAfter, promoted.Score, promoted.Score-sumAfter, len(promoted.Reasons))
}

func TestQARemoteSiblingScanNeverRunsAgainstALocalOnlyCatalogue(t *testing.T) {
	catalogue := testCatalogueService(stubCatalogueLocal{repositories: testLocalRepositories("mms-api")}, nil)
	branches := &stubSuggestionBranches{matches: map[string][]BranchMatch{
		"CRMMS-77912": {{Repository: "mms-api", Branch: "feature/CRMMS-77912-first"}},
	}}
	issues := stubSuggestionIssues{
		issue:   testIssue("CRMMS-77911", "CRMMS-100", "Enhancement Manage User", "user management page"),
		backlog: testBacklog("CRMMS-77912"),
	}
	scanner := &stubSuggestionRemoteScanner{branchesByRepository: map[string][]string{}}
	service := testSuggestionService(catalogue, issues, branches, WithRemoteCandidateScanner(NewRemoteCandidateScanner(scanner)))

	result, err := service.Suggest(context.Background(), "user-1", "CRMMS-77911")
	if err != nil {
		t.Fatalf("suggest failed: %v", err)
	}
	if calls := scanner.calledLists(); len(calls) != 0 {
		t.Fatalf("a catalogue without a remote source must offer no candidate, got %v", calls)
	}
	if result.Scope.VerifiedRemotely {
		t.Fatal("a scan that never ran must never report a verified remote tier")
	}
	if result.Scope.RemoteRepositoriesChecked != 0 {
		t.Fatalf("a scan that never ran must report zero repositories checked, got %d", result.Scope.RemoteRepositoriesChecked)
	}
	if result.Catalogue.RemoteRepositoryCount != 0 {
		t.Fatalf("a catalogue without a remote source must report zero remote repositories, got %d", result.Catalogue.RemoteRepositoryCount)
	}
	if result.Scope.CoveragePercent != 100 {
		t.Fatalf("a local only catalogue must not advertise a coverage gap, got %d", result.Scope.CoveragePercent)
	}
}

func TestQAScanIssueKeysTimeoutAccountsForEveryCandidateExactlyOnce(t *testing.T) {
	repositories := []string{"NME/one", "NME/two", "NME/three", "NME/four", "NME/five", "NME/six", "NME/seven", "NME/eight"}
	scanner := &stubRemoteBranchScanner{searchDelay: 200 * time.Millisecond, listDelay: 200 * time.Millisecond}
	bounded := newRemoteCandidateScanner(scanner, 25*time.Millisecond)

	result, err := bounded.ScanIssueKeys(context.Background(), "user-1", []string{"CRMMS-77911"}, repositories)
	if err != nil {
		t.Fatalf("a timed out scan must not fail the request, got %v", err)
	}
	if !result.TimedOut {
		t.Fatal("a scan whose aggregate deadline passed must be marked timed out")
	}

	seen := map[string]int{}
	for repository := range result.Matches {
		seen[repository]++
	}
	for _, failure := range result.Failures {
		seen[failure.Repository]++
	}
	for _, repository := range result.RepositoriesNotChecked {
		seen[repository]++
	}
	for _, repository := range repositories {
		if seen[repository] != 1 {
			t.Fatalf("repository %s is accounted for %d times, timedOut=%v checked=%d failures=%d notChecked=%d", repository, seen[repository], result.TimedOut, result.RepositoriesChecked, len(result.Failures), len(result.RepositoriesNotChecked))
		}
	}
	for _, repository := range result.RepositoriesNotChecked {
		if _, present := result.Matches[repository]; present {
			t.Fatalf("an unchecked repository %s must never appear in Matches", repository)
		}
	}

	t.Logf("QA: timedOut=%v checked=%d failures=%d notChecked=%v", result.TimedOut, result.RepositoriesChecked, len(result.Failures), result.RepositoriesNotChecked)
}

func TestQASiblingEvidenceDropsAnUnusableKeyFromTheRemoteLookup(t *testing.T) {
	evidence := newSiblingEvidence("CRMMS-100", false)
	evidence.siblingKeys["CRMMS-77912"] = struct{}{}
	evidence.siblingKeys["not a key"] = struct{}{}
	evidence.siblingKeys["CRMMS-77913"] = struct{}{}

	keys := evidence.remoteLookupKeys()
	if len(keys) != 2 {
		t.Fatalf("expected two usable sibling keys, got %v", keys)
	}

	t.Logf("QA: the remote lookup silently dropped %d sibling key(s), kept=%v", 3-len(keys), keys)
}

func reasonContribution(suggestion RepositorySuggestion) float64 {
	total := 0.0
	for _, reason := range suggestion.Reasons {
		total += reason.Weight * reason.Strength
	}

	return total
}
