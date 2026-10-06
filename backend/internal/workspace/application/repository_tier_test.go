package application

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	sharedErrors "lunar/backend/internal/shared/errors"
)

func TestSuggestionUsesTheModelOnlyAsAFallback(t *testing.T) {
	catalogue := testCatalogueService(
		stubCatalogueLocal{repositories: testLocalRepositories("mms-api")},
		stubCatalogueRemote{repositories: testRemoteRepositories("NME", "mms-api", "atlas")},
	)
	ranker := &stubSuggestionRanker{ranked: []RankedRepository{
		{Repo: "atlas", Rank: 1, Reason: "the page lists users, which matches atlas"},
	}}
	service := testSuggestionService(catalogue, stubSuggestionIssues{
		issue:   testIssue("CRMMS-77911", "CRMMS-100", "Enhancement Manage User", "Halaman utama manage user"),
		backlog: testBacklog(),
	}, &stubSuggestionBranches{}, WithRepositoryRanker(ranker))

	suggestions, err := service.Suggest(context.Background(), "user-1", "CRMMS-77911")
	if err != nil {
		t.Fatalf("suggest failed: %v", err)
	}

	if len(ranker.requests) != 1 {
		t.Fatalf("the model must run exactly once when nothing else fired, got %d", len(ranker.requests))
	}
	if len(ranker.requests[0].Candidates) != 2 {
		t.Fatalf("the model must see the real catalogue, got %v", ranker.requests[0].Candidates)
	}
	if len(suggestions.Suggestions) != 1 || suggestions.Suggestions[0].Repo != "atlas" {
		t.Fatalf("expected the model's first choice, got %+v", suggestions.Suggestions)
	}
	reason, found := findReason(suggestions.Suggestions[0], SignalModel)
	if !found || reason.Weight != 1.0 || reason.Strength != 1.0 {
		t.Fatalf("unexpected model reason: %+v", reason)
	}
	if reason.Text != "the page lists users, which matches atlas" {
		t.Fatalf("the model reasoning must be returned verbatim, got %q", reason.Text)
	}
	if suggestions.Confidence != ConfidenceLow {
		t.Fatalf("the model alone is never a confident recommendation, got %q", suggestions.Confidence)
	}

	assertEverySuggestionHasAReason(t, suggestions)
	assertReasonInvariants(t, suggestions)
}

func TestSuggestionSkipsTheModelWhenASignalAlreadyFired(t *testing.T) {
	catalogue := testCatalogueService(
		stubCatalogueLocal{repositories: testLocalRepositories("mms-api")},
		stubCatalogueRemote{repositories: testRemoteRepositories("NME", "mms-api")},
	)
	branches := &stubSuggestionBranches{matches: map[string][]BranchMatch{
		"CRMMS-77912": {{Repository: "mms-api", Branch: "feature/CRMMS-77912-first"}},
	}}
	ranker := &stubSuggestionRanker{}
	service := testSuggestionService(catalogue, stubSuggestionIssues{
		issue:   testIssue("CRMMS-77911", "CRMMS-100", "Enhancement Manage User", "Halaman utama manage user"),
		backlog: testBacklog("CRMMS-77912"),
	}, branches, WithRepositoryRanker(ranker))

	if _, err := service.Suggest(context.Background(), "user-1", "CRMMS-77911"); err != nil {
		t.Fatalf("suggest failed: %v", err)
	}

	if len(ranker.requests) != 0 {
		t.Fatalf("the model must not run when a deterministic signal fired, got %d calls", len(ranker.requests))
	}
}

func TestSuggestionRefusesAnInvalidKey(t *testing.T) {
	service := testSuggestionService(testCatalogueService(stubCatalogueLocal{}, nil), stubSuggestionIssues{}, &stubSuggestionBranches{})

	if _, err := service.Suggest(context.Background(), "user-1", "not-a-key"); !errors.Is(err, sharedErrors.ErrBadRequest) {
		t.Fatalf("expected a bad request for an invalid key, got %v", err)
	}
}

func TestSuggestionReportsAJiraFailure(t *testing.T) {
	service := testSuggestionService(testCatalogueService(stubCatalogueLocal{}, nil), stubSuggestionIssues{
		issueErr: fmt.Errorf("%w: unreachable", sharedErrors.ErrBadGateway),
	}, &stubSuggestionBranches{})

	_, err := service.Suggest(context.Background(), "user-1", "CRMMS-77911")
	if !errors.Is(err, sharedErrors.ErrBadGateway) {
		t.Fatalf("expected the Jira failure to be reported, got %v", err)
	}
	if err == nil || !strings.Contains(err.Error(), "CRMMS-77911") {
		t.Fatalf("the failure must name the issue, got %v", err)
	}
}

func TestSuggestionReportsAnUnusableWorkspace(t *testing.T) {
	catalogue := NewRepositoryCatalogueService(stubSuggestionRoots{err: ErrWorkspaceNotUsable}, stubCatalogueLocal{}, nil)
	service := NewRepositorySuggestionService(RepositorySuggestionDependencies{
		Catalogue:      catalogue,
		WorkspaceRoots: stubSuggestionRoots{err: ErrWorkspaceNotUsable},
		Issues:         stubSuggestionIssues{issue: testIssue("CRMMS-77911", "CRMMS-100", "Summary", "Description")},
		Branches:       &stubSuggestionBranches{},
	})

	if _, err := service.Suggest(context.Background(), "user-1", "CRMMS-77911"); !errors.Is(err, ErrWorkspaceNotUsable) {
		t.Fatalf("expected the unusable workspace to be reported, got %v", err)
	}
}

func TestSuggestionSerialisesEmptyListsAsArrays(t *testing.T) {
	catalogue := testCatalogueService(stubCatalogueLocal{}, nil)
	service := testSuggestionService(catalogue, stubSuggestionIssues{
		issue:   testIssue("CRMMS-77911", "CRMMS-100", "Summary", "Description"),
		backlog: testBacklog(),
	}, &stubSuggestionBranches{})

	suggestions, err := service.Suggest(context.Background(), "user-1", "CRMMS-77911")
	if err != nil {
		t.Fatalf("suggest failed: %v", err)
	}
	if suggestions.Confidence != ConfidenceLow {
		t.Fatalf("no repository is known, so the result must be low confidence, got %q", suggestions.Confidence)
	}
	if len(suggestions.Suggestions) != 0 {
		t.Fatalf("expected no suggestions, got %+v", suggestions.Suggestions)
	}
	if _, found := findUnavailable(suggestions, SignalExplicitHint); !found {
		t.Fatal("an empty catalogue must be stated as the reason the hint signal is silent")
	}

	encoded, err := json.Marshal(suggestions)
	if err != nil {
		t.Fatalf("cannot encode the response: %v", err)
	}
	if !strings.Contains(string(encoded), "\"suggestions\":[]") {
		t.Fatalf("an empty suggestion list must encode as an array, got %s", encoded)
	}
	if !strings.Contains(string(encoded), "\"catalogue\":{\"local_repository_count\":0,\"remote_repository_count\":0}") {
		t.Fatalf("the catalogue counts must always be reported, got %s", encoded)
	}
}

func TestSuggestionReportsUnavailableSignalsWithReasons(t *testing.T) {
	catalogue := testCatalogueService(stubCatalogueLocal{repositories: testLocalRepositories("mms-api")}, nil)
	service := testSuggestionService(catalogue, stubSuggestionIssues{
		issue:      testIssue("CRMMS-77911", "CRMMS-100", "Summary", "Description"),
		backlogErr: errors.New("jira is unreachable"),
	}, &stubSuggestionBranches{err: errors.New("git failed")})

	suggestions, err := service.Suggest(context.Background(), "user-1", "CRMMS-77911")
	if err != nil {
		t.Fatalf("a failed signal must not fail the request: %v", err)
	}
	if len(suggestions.Suggestions) != 0 {
		t.Fatalf("expected no suggestions, got %+v", suggestions.Suggestions)
	}

	sibling, found := findUnavailable(suggestions, SignalEpicSiblings)
	if !found || sibling.Reason != siblingBacklogReason {
		t.Fatalf("expected the sibling signal to state its reason, got %+v", suggestions.SignalsUnavailable)
	}
	codeIndex, found := findUnavailable(suggestions, SignalCodeIndex)
	if !found || codeIndex.Reason != codeIndexMissingReason {
		t.Fatalf("expected the code index to state its reason, got %+v", suggestions.SignalsUnavailable)
	}
	pages, found := findUnavailable(suggestions, SignalConfluence)
	if !found || pages.Reason != pageMissingReason {
		t.Fatalf("expected the pages signal to state its reason, got %+v", suggestions.SignalsUnavailable)
	}
	model, found := findUnavailable(suggestions, SignalModel)
	if !found || model.Reason != modelUnavailableReason {
		t.Fatalf("expected the model fallback to state its reason, got %+v", suggestions.SignalsUnavailable)
	}
}
