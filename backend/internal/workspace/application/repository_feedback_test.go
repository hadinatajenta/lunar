package application

import (
	"context"
	"errors"
	"testing"

	sharedErrors "lunar/backend/internal/shared/errors"
)

func TestFeedbackLevelsRankAnAdditionAboveAConfirmation(t *testing.T) {
	if feedbackLevel(FeedbackAdded) <= feedbackLevel(FeedbackKept) {
		t.Fatalf("an addition must outrank a confirmation: added %v, kept %v", feedbackLevel(FeedbackAdded), feedbackLevel(FeedbackKept))
	}
	if feedbackLevel(FeedbackKept) <= 0 {
		t.Fatalf("keeping a suggestion is a weak positive, got %v", feedbackLevel(FeedbackKept))
	}
	if feedbackLevel(FeedbackUnticked) >= 0 {
		t.Fatalf("unticking is a negative, got %v", feedbackLevel(FeedbackUnticked))
	}
	if feedbackLevel(FeedbackRemoved) >= feedbackLevel(FeedbackUnticked) {
		t.Fatalf("a removal is a stronger negative than an untick: removed %v, unticked %v", feedbackLevel(FeedbackRemoved), feedbackLevel(FeedbackUnticked))
	}
	if feedbackLevel(FeedbackAdded) > WeightExplicitHint {
		t.Fatalf("feedback must not outweigh a deterministic signal, got %v", feedbackLevel(FeedbackAdded))
	}
}

func TestFeedbackStoreScopesAnUntickToItsIssue(t *testing.T) {
	store := NewRepositoryFeedbackStore()
	if err := store.Record("user-1", "CRMMS-100", "CRMMS-77911", "atlas", FeedbackUnticked); err != nil {
		t.Fatalf("record failed: %v", err)
	}

	if adjustment := store.Adjustment("user-1", "CRMMS-100", "CRMMS-77911", "atlas"); adjustment != -0.5 {
		t.Fatalf("expected the untick to apply to its issue, got %v", adjustment)
	}
	if adjustment := store.Adjustment("user-1", "CRMMS-100", "CRMMS-77912", "atlas"); adjustment != 0 {
		t.Fatalf("an untick must not leak to another issue, got %v", adjustment)
	}
}

func TestFeedbackStoreAppliesAnAdditionToTheWholeEpic(t *testing.T) {
	store := NewRepositoryFeedbackStore()
	if err := store.Record("user-1", "CRMMS-100", "CRMMS-77911", "atlas", FeedbackAdded); err != nil {
		t.Fatalf("record failed: %v", err)
	}

	if adjustment := store.Adjustment("user-1", "CRMMS-100", "CRMMS-77911", "atlas"); adjustment != 2.0 {
		t.Fatalf("expected an addition to boost its issue, got %v", adjustment)
	}
	if adjustment := store.Adjustment("user-1", "CRMMS-100", "CRMMS-77912", "atlas"); adjustment != 2.0 {
		t.Fatalf("an addition must carry to the next issue in the epic, got %v", adjustment)
	}
	if adjustment := store.Adjustment("user-1", "CRMMS-200", "CRMMS-77912", "atlas"); adjustment != 0 {
		t.Fatalf("an addition must not leak to another epic, got %v", adjustment)
	}
	if adjustment := store.Adjustment("user-2", "CRMMS-100", "CRMMS-77911", "atlas"); adjustment != 0 {
		t.Fatalf("feedback must not leak to another user, got %v", adjustment)
	}
}

func TestFeedbackStoreClampsRepeatedActions(t *testing.T) {
	store := NewRepositoryFeedbackStore()
	for attempt := 0; attempt < 3; attempt++ {
		if err := store.Record("user-1", "CRMMS-100", "CRMMS-77911", "atlas", FeedbackAdded); err != nil {
			t.Fatalf("record failed: %v", err)
		}
	}

	if adjustment := store.Adjustment("user-1", "CRMMS-100", "CRMMS-77911", "atlas"); adjustment != feedbackAdjustmentLimit {
		t.Fatalf("feedback must stay bounded, got %v", adjustment)
	}
}

func TestFeedbackStoreRefusesInvalidInput(t *testing.T) {
	store := NewRepositoryFeedbackStore()

	if err := store.Record("user-1", "CRMMS-100", "CRMMS-77911", "atlas", FeedbackAction("guessed")); !errors.Is(err, sharedErrors.ErrBadRequest) {
		t.Fatalf("expected an unknown action to be refused, got %v", err)
	}
	if err := store.Record("", "CRMMS-100", "CRMMS-77911", "atlas", FeedbackAdded); !errors.Is(err, sharedErrors.ErrBadRequest) {
		t.Fatalf("expected a missing user to be refused, got %v", err)
	}
	if err := store.Record("user-1", "CRMMS-100", "CRMMS-77911", "  ", FeedbackAdded); !errors.Is(err, sharedErrors.ErrBadRequest) {
		t.Fatalf("expected a missing repository to be refused, got %v", err)
	}
}

func TestParseFeedbackActionAcceptsTheFourActions(t *testing.T) {
	for _, rawAction := range []string{"added", "removed", "unticked", "kept", " ADDED "} {
		if _, err := ParseFeedbackAction(rawAction); err != nil {
			t.Fatalf("expected %q to parse, got %v", rawAction, err)
		}
	}
	if _, err := ParseFeedbackAction("suggested"); !errors.Is(err, sharedErrors.ErrBadRequest) {
		t.Fatalf("expected an unknown action to be refused, got %v", err)
	}
}

func TestFeedbackAdditionPromotesARepositoryInTheNextRanking(t *testing.T) {
	service, ranks := feedbackFixture(t)

	rank, err := service.Suggest(context.Background(), "user-1", "CRMMS-77911")
	if err != nil {
		t.Fatalf("suggest failed: %v", err)
	}
	if rank.Suggestions[0].Repo != "user-service" {
		t.Fatalf("expected user-service to lead before any feedback, got %+v", rank.Suggestions)
	}

	if err := service.RecordFeedback(context.Background(), "user-1", "CRMMS-77911", "atlas", FeedbackAdded); err != nil {
		t.Fatalf("record failed: %v", err)
	}
	if err := service.RecordFeedback(context.Background(), "user-1", "CRMMS-77911", "user-service", FeedbackKept); err != nil {
		t.Fatalf("record failed: %v", err)
	}

	promoted, err := service.Suggest(context.Background(), "user-1", "CRMMS-77911")
	if err != nil {
		t.Fatalf("suggest failed: %v", err)
	}
	if promoted.Suggestions[0].Repo != "atlas" {
		t.Fatalf("an addition must outrank a confirmation and promote atlas, got %+v", promoted.Suggestions)
	}
	if calls := ranks.calledLists(); len(calls) != 0 {
		t.Fatalf("the feedback test must not touch Bitbucket, got %v", calls)
	}
}

func TestFeedbackRemovalDemotesARepositoryInTheNextRanking(t *testing.T) {
	service, _ := feedbackFixture(t)

	before, err := service.Suggest(context.Background(), "user-1", "CRMMS-77911")
	if err != nil {
		t.Fatalf("suggest failed: %v", err)
	}
	if before.Suggestions[0].Repo != "user-service" {
		t.Fatalf("expected user-service to lead before any feedback, got %+v", before.Suggestions)
	}

	if err := service.RecordFeedback(context.Background(), "user-1", "CRMMS-77911", "user-service", FeedbackRemoved); err != nil {
		t.Fatalf("record failed: %v", err)
	}

	after, err := service.Suggest(context.Background(), "user-1", "CRMMS-77911")
	if err != nil {
		t.Fatalf("suggest failed: %v", err)
	}
	if after.Suggestions[0].Repo != "atlas" {
		t.Fatalf("a removal must change the next ranking, got %+v", after.Suggestions)
	}
	removed := findSuggestion(after.Suggestions, "user-service")
	if removed.Score != 3.05 {
		t.Fatalf("expected the removal to take one point off, got %v", removed.Score)
	}
}

func TestFeedbackRefusesARepositoryOutsideTheCatalogue(t *testing.T) {
	service, _ := feedbackFixture(t)

	if err := service.RecordFeedback(context.Background(), "user-1", "CRMMS-77911", "not-a-real-repo", FeedbackAdded); !errors.Is(err, sharedErrors.ErrNotFound) {
		t.Fatalf("expected an unknown repository to be refused, got %v", err)
	}
	if err := service.RecordFeedback(context.Background(), "user-1", "CRMMS-77911", "atlas", FeedbackAction("guessed")); !errors.Is(err, sharedErrors.ErrBadRequest) {
		t.Fatalf("expected an unknown action to be refused, got %v", err)
	}
}

func feedbackFixture(t *testing.T) (*RepositorySuggestionService, *stubSuggestionRemoteScanner) {
	t.Helper()

	catalogue := testCatalogueService(
		stubCatalogueLocal{repositories: testLocalRepositories("user-service")},
		stubCatalogueRemote{repositories: testRemoteRepositories("NME", "user-service", "atlas")},
	)
	branches := &stubSuggestionBranches{matches: map[string][]BranchMatch{
		"CRMMS-77912": {{Repository: "atlas", Branch: "feature/CRMMS-77912-first"}},
		"CRMMS-77913": {{Repository: "atlas", Branch: "feature/CRMMS-77913-second"}},
		"CRMMS-77914": {{Repository: "user-service", Branch: "feature/CRMMS-77914-third"}},
	}}
	issues := stubSuggestionIssues{
		issue:   testIssue("CRMMS-77911", "CRMMS-100", "Enhancement Manage User", "user management page"),
		backlog: testBacklog("CRMMS-77912", "CRMMS-77913", "CRMMS-77914"),
	}
	scanner := &stubSuggestionRemoteScanner{branchesByRepository: map[string][]string{}}

	return testSuggestionService(catalogue, issues, branches, WithRemoteCandidateScanner(NewRemoteCandidateScanner(scanner))), scanner
}

func findSuggestion(suggestions []RepositorySuggestion, repo string) RepositorySuggestion {
	for _, suggestion := range suggestions {
		if suggestion.Repo == repo {
			return suggestion
		}
	}

	return RepositorySuggestion{}
}
