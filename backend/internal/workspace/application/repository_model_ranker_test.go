package application

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	sharedErrors "lunar/backend/internal/shared/errors"
)

const modelRankingReason = "the acceptance criteria name the list API the page calls"

type stubRankerModel struct {
	answers []string
	err     error
	calls   int
	users   []string
	prompts []string
}

func (s *stubRankerModel) CompleteJSON(_ context.Context, userID string, _ string, prompt string) (string, error) {
	s.calls++
	s.users = append(s.users, userID)
	s.prompts = append(s.prompts, prompt)
	if s.err != nil {
		return "", s.err
	}
	if len(s.answers) == 0 {
		return "", fmt.Errorf("the stub model holds no answer")
	}
	if s.calls <= len(s.answers) {
		return s.answers[s.calls-1], nil
	}

	return s.answers[len(s.answers)-1], nil
}

func rankerRequest(candidates ...string) RepositoryRankRequest {
	return RepositoryRankRequest{
		UserID:      "user-a",
		IssueKey:    "CRMMS-77911",
		Summary:     "Enable the bulk MDR program",
		Description: "h3. Deskripsi Acceptance Criteria (AC)",
		Candidates:  candidates,
	}
}

func TestRepositoryModelRankerRanksTheCandidatesInOrder(t *testing.T) {
	client := &stubRankerModel{answers: []string{`{"repositories":[{"repo":"mms-api","reason":"the acceptance criteria name the list API the page calls"},{"repo":"fe-mms","reason":"the criteria name the page that calls it"}]}`}}
	ranker := NewRepositoryModelRanker(client)

	ranked, err := ranker.RankRepositories(context.Background(), rankerRequest("fe-mms", "mms-api"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ranked) != 2 {
		t.Fatalf("ranked = %+v", ranked)
	}
	if ranked[0].Repo != "mms-api" || ranked[0].Rank != 1 || ranked[0].Reason != modelRankingReason {
		t.Fatalf("first ranking = %+v", ranked[0])
	}
	if ranked[1].Repo != "fe-mms" || ranked[1].Rank != 2 {
		t.Fatalf("second ranking = %+v", ranked[1])
	}
	if client.calls != 1 || client.users[0] != "user-a" {
		t.Fatalf("model calls = %d, users = %v", client.calls, client.users)
	}
	for _, candidate := range []string{"fe-mms", "mms-api"} {
		if !strings.Contains(client.prompts[0], candidate) {
			t.Fatalf("the prompt must carry %q", candidate)
		}
	}
}

func TestRepositoryModelRankerMatchesNamesCaseInsensitively(t *testing.T) {
	client := &stubRankerModel{answers: []string{`{"repositories":[{"repo":"MMS-API","reason":"named"}]}`}}
	ranker := NewRepositoryModelRanker(client)

	ranked, err := ranker.RankRepositories(context.Background(), rankerRequest("mms-api"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ranked) != 1 || ranked[0].Repo != "mms-api" {
		t.Fatalf("ranked = %+v", ranked)
	}
}

func TestRepositoryModelRankerRejectsARepositoryThatWasNotOffered(t *testing.T) {
	client := &stubRankerModel{answers: []string{`{"repositories":[{"repo":"atlas","reason":"invented"}]}`}}
	ranker := NewRepositoryModelRanker(client)

	if _, err := ranker.RankRepositories(context.Background(), rankerRequest("mms-api")); err == nil {
		t.Fatal("a repository that was not offered must be an error, never a suggestion")
	}
}

func TestRepositoryModelRankerRejectsADuplicateRepository(t *testing.T) {
	client := &stubRankerModel{answers: []string{`{"repositories":[{"repo":"mms-api","reason":"first"},{"repo":"MMS-API","reason":"second"}]}`}}
	ranker := NewRepositoryModelRanker(client)

	if _, err := ranker.RankRepositories(context.Background(), rankerRequest("mms-api")); err == nil {
		t.Fatal("a duplicated repository must be an error")
	}
}

func TestRepositoryModelRankerKeepsAtMostThreeRankings(t *testing.T) {
	client := &stubRankerModel{answers: []string{`{"repositories":[{"repo":"a","reason":"1"},{"repo":"b","reason":"2"},{"repo":"c","reason":"3"},{"repo":"d","reason":"4"}]}`}}
	ranker := NewRepositoryModelRanker(client)

	ranked, err := ranker.RankRepositories(context.Background(), rankerRequest("a", "b", "c", "d"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ranked) != maximumModelRankings {
		t.Fatalf("ranked = %+v", ranked)
	}
	if ranked[maximumModelRankings-1].Rank != maximumModelRankings {
		t.Fatalf("last rank = %d", ranked[maximumModelRankings-1].Rank)
	}
}

func TestRepositoryModelRankerAcceptsAnEmptyRanking(t *testing.T) {
	client := &stubRankerModel{answers: []string{`{"repositories":[]}`}}
	ranker := NewRepositoryModelRanker(client)

	ranked, err := ranker.RankRepositories(context.Background(), rankerRequest("mms-api"))
	if err != nil {
		t.Fatalf("a refusal is a valid answer: %v", err)
	}
	if len(ranked) != 0 {
		t.Fatalf("ranked = %+v", ranked)
	}
}

func TestRepositoryModelRankerRejectsAMalformedAnswer(t *testing.T) {
	client := &stubRankerModel{answers: []string{"mms-api is the one"}}
	ranker := NewRepositoryModelRanker(client)

	if _, err := ranker.RankRepositories(context.Background(), rankerRequest("mms-api")); err == nil {
		t.Fatal("a malformed answer must be an error")
	}
}

func TestRepositoryModelRankerRejectsAnEmptyCatalogue(t *testing.T) {
	ranker := NewRepositoryModelRanker(&stubRankerModel{})

	if _, err := ranker.RankRepositories(context.Background(), rankerRequest()); !errors.Is(err, sharedErrors.ErrBadRequest) {
		t.Fatalf("error = %v, want a bad request", err)
	}
}

func TestRepositoryModelRankerWithoutAClientFails(t *testing.T) {
	ranker := NewRepositoryModelRanker(nil)

	if _, err := ranker.RankRepositories(context.Background(), rankerRequest("mms-api")); err == nil {
		t.Fatal("a ranker without a client must fail rather than guess")
	}
}

func TestRepositoryModelRankerDeduplicatesTheCandidates(t *testing.T) {
	client := &stubRankerModel{answers: []string{`{"repositories":[]}`}}
	ranker := NewRepositoryModelRanker(client)

	if _, err := ranker.RankRepositories(context.Background(), rankerRequest("mms-api", "MMS-API", "  ")); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Count(client.prompts[0], "mms-api") != 1 {
		t.Fatalf("the prompt must list each candidate once: %q", client.prompts[0])
	}
}

func TestModelSignalIsAFallbackAndStaysLowConfidence(t *testing.T) {
	client := &stubRankerModel{answers: []string{`{"repositories":[{"repo":"mms-api","reason":"the acceptance criteria name the list API the page calls"}]}`}}
	catalogue := testCatalogueService(stubCatalogueLocal{repositories: testLocalRepositories("mms-api", "fe-mms")}, stubCatalogueRemote{})
	service := testSuggestionService(
		catalogue,
		stubSuggestionIssues{issue: testIssue("CRMMS-77911", "", "Enable the bulk MDR program", "h3. Deskripsi Acceptance Criteria (AC)")},
		&stubSuggestionBranches{},
		WithRepositoryRanker(NewRepositoryModelRanker(client)),
	)

	suggestions, err := service.Suggest(context.Background(), "user-a", "CRMMS-77911")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if client.calls != 1 {
		t.Fatalf("the model must run exactly once when nothing else fired, got %d calls", client.calls)
	}
	if len(suggestions.Suggestions) != 1 {
		t.Fatalf("suggestions = %+v", suggestions.Suggestions)
	}

	top := suggestions.Suggestions[0]
	if top.Repo != "mms-api" || top.Score != WeightModel {
		t.Fatalf("top suggestion = %+v", top)
	}
	if len(top.Reasons) != 1 {
		t.Fatalf("reasons = %+v", top.Reasons)
	}
	if top.Reasons[0].Signal != SignalModel || top.Reasons[0].Weight != WeightModel || top.Reasons[0].Text != modelRankingReason {
		t.Fatalf("reason = %+v", top.Reasons[0])
	}
	if suggestions.Confidence != ConfidenceLow {
		t.Fatalf("a model only result must stay low confidence, got %q", suggestions.Confidence)
	}
	for _, signal := range suggestions.SignalsUnavailable {
		if signal.Signal == SignalModel {
			t.Fatalf("the model signal produced a candidate and must not be reported unavailable: %+v", signal)
		}
	}
}

func TestModelSignalDoesNotRunWhenAnotherSignalFired(t *testing.T) {
	client := &stubRankerModel{answers: []string{`{"repositories":[{"repo":"mms-api","reason":"named"}]}`}}
	catalogue := testCatalogueService(stubCatalogueLocal{repositories: testLocalRepositories("mms-api")}, stubCatalogueRemote{})
	service := testSuggestionService(
		catalogue,
		stubSuggestionIssues{issue: testIssue("CRMMS-77911", "", "Change the mms-api list endpoint", "description")},
		&stubSuggestionBranches{},
		WithRepositoryRanker(NewRepositoryModelRanker(client)),
	)

	suggestions, err := service.Suggest(context.Background(), "user-a", "CRMMS-77911")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if client.calls != 0 {
		t.Fatalf("the model must not run when a deterministic signal fired, got %d calls", client.calls)
	}
	if len(suggestions.Suggestions) != 1 || len(suggestions.Suggestions[0].Reasons) == 0 {
		t.Fatalf("suggestions = %+v", suggestions.Suggestions)
	}
	if suggestions.Suggestions[0].Reasons[0].Signal != SignalExplicitHint {
		t.Fatalf("reason = %+v", suggestions.Suggestions[0].Reasons[0])
	}
}

func TestModelSignalFailureIsReportedAndSuggestsNothing(t *testing.T) {
	client := &stubRankerModel{answers: []string{"I cannot rank these repositories"}}
	catalogue := testCatalogueService(stubCatalogueLocal{repositories: testLocalRepositories("mms-api", "fe-mms")}, stubCatalogueRemote{})
	service := testSuggestionService(
		catalogue,
		stubSuggestionIssues{issue: testIssue("CRMMS-77911", "", "Enable the bulk MDR program", "h3. Deskripsi Acceptance Criteria (AC)")},
		&stubSuggestionBranches{},
		WithRepositoryRanker(NewRepositoryModelRanker(client)),
	)

	suggestions, err := service.Suggest(context.Background(), "user-a", "CRMMS-77911")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(suggestions.Suggestions) != 0 {
		t.Fatalf("a malformed answer must suggest nothing, got %+v", suggestions.Suggestions)
	}
	if suggestions.Confidence != ConfidenceLow {
		t.Fatalf("confidence = %q", suggestions.Confidence)
	}

	reported := false
	for _, signal := range suggestions.SignalsUnavailable {
		if signal.Signal != SignalModel {
			continue
		}
		reported = true
		if signal.Reason != modelFailedReason {
			t.Fatalf("reason = %q, want %q", signal.Reason, modelFailedReason)
		}
	}
	if !reported {
		t.Fatalf("the failed model signal must be reported: %+v", suggestions.SignalsUnavailable)
	}
}
