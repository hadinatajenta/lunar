package application

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	jiraDomain "lunar/backend/internal/jira/domain"
	sharedErrors "lunar/backend/internal/shared/errors"
)

func TestSuggestReturnsTheModelNameAndItsEvidence(t *testing.T) {
	issues := &stubBranchNameIssues{issue: branchNameIssue()}
	model := &stubBranchNameModel{answers: []string{branchNameModelAnswer}}
	service := NewBranchNameService(issues, model, nil)

	suggestion, err := service.Suggest(context.Background(), "user-a", "crmms-77911")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if suggestion.Key != "CRMMS-77911" {
		t.Fatalf("key = %q", suggestion.Key)
	}
	if suggestion.Branch != "feature/CRMMS-77911-Enhancement-Manage-User" {
		t.Fatalf("branch = %q", suggestion.Branch)
	}
	if suggestion.Category != "Enhancement" || suggestion.Core != "Manage-User" {
		t.Fatalf("category = %q, core = %q", suggestion.Category, suggestion.Core)
	}
	if suggestion.Source != jiraDomain.BranchNameSourceModel {
		t.Fatalf("source = %q", suggestion.Source)
	}
	if suggestion.Evidence != "Halaman Utama Manage User (Grouping by PN)" {
		t.Fatalf("evidence = %q", suggestion.Evidence)
	}
	if suggestion.EvidenceSource != EvidenceSourceAcceptanceCriteria {
		t.Fatalf("evidence source = %q, want the acceptance criteria", suggestion.EvidenceSource)
	}
	if model.calls != 1 {
		t.Fatalf("the model must be asked once, got %d calls", model.calls)
	}
	if len(issues.requests) != 1 || issues.requests[0] != "user-a|CRMMS-77911" {
		t.Fatalf("issue reads = %v", issues.requests)
	}
	if !strings.Contains(model.prompts[0], "Halaman Utama Manage User (Grouping by PN)") {
		t.Fatal("the prompt must carry the acceptance criteria heading")
	}
	if !model.hasDeadline || model.budget <= 0 || model.budget > branchNameModelTimeout {
		t.Fatalf("the model call must be bounded by %s, got %s", branchNameModelTimeout, model.budget)
	}
}

func TestSuggestCachesTheModelNamePerIssueAndUser(t *testing.T) {
	issues := &stubBranchNameIssues{issue: branchNameIssue()}
	model := &stubBranchNameModel{answers: []string{branchNameModelAnswer}}
	service := NewBranchNameService(issues, model, nil)

	first, err := service.Suggest(context.Background(), "user-a", "CRMMS-77911")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	second, err := service.Suggest(context.Background(), "user-a", "CRMMS-77911")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if *first != *second {
		t.Fatalf("the cached answer changed: %+v then %+v", first, second)
	}
	if model.calls != 1 || len(issues.requests) != 1 {
		t.Fatalf("a cached name must not ask again: %d model calls, %d issue reads", model.calls, len(issues.requests))
	}

	if _, err := service.Suggest(context.Background(), "user-b", "CRMMS-77911"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if model.calls != 2 {
		t.Fatalf("the cache must be per user, got %d model calls", model.calls)
	}
}

func TestSuggestReturnsTheFallbackWhenTheModelFails(t *testing.T) {
	issues := &stubBranchNameIssues{issue: &jiraDomain.JiraIssue{
		Key: "CRMMS-77745",
		Fields: jiraDomain.JiraIssueFields{
			Summary: "Sebagai Maker & Approval di New MMS, I want to memiliki menu dan form terpisah untuk mengajukan program bulk MDR QRIS Statis",
		},
	}}
	model := &stubBranchNameModel{err: errors.New("no AI provider is configured")}
	service := NewBranchNameService(issues, model, nil)

	suggestion, err := service.Suggest(context.Background(), "user-a", "CRMMS-77745")
	if err != nil {
		t.Fatalf("a model failure must not fail the request: %v", err)
	}
	if suggestion.Source != jiraDomain.BranchNameSourceFallback {
		t.Fatalf("source = %q, want %q", suggestion.Source, jiraDomain.BranchNameSourceFallback)
	}
	if suggestion.Branch != "feature/CRMMS-77745-Menu-Form-Terpisah-Program-Bulk" {
		t.Fatalf("branch = %q", suggestion.Branch)
	}
	if suggestion.Core != "Menu-Form-Terpisah-Program-Bulk" {
		t.Fatalf("core = %q", suggestion.Core)
	}
	if suggestion.Category != "" {
		t.Fatalf("the fallback must not invent a category, got %q", suggestion.Category)
	}
	if suggestion.Evidence != "memiliki menu dan form terpisah untuk mengajukan program bulk MDR QRIS Statis" {
		t.Fatalf("evidence = %q", suggestion.Evidence)
	}
	if suggestion.EvidenceSource != EvidenceSourceSummary {
		t.Fatalf("the fallback names from the summary, evidence source = %q", suggestion.EvidenceSource)
	}
}

func TestSuggestAsksTheModelAgainAfterAFallback(t *testing.T) {
	issues := &stubBranchNameIssues{issue: branchNameIssue()}
	model := &stubBranchNameModel{err: errors.New("the provider is down")}
	service := NewBranchNameService(issues, model, nil)

	first, err := service.Suggest(context.Background(), "user-a", "CRMMS-77911")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if first.Source != jiraDomain.BranchNameSourceFallback {
		t.Fatalf("source = %q", first.Source)
	}

	model.err = nil
	model.answers = []string{branchNameModelAnswer}
	second, err := service.Suggest(context.Background(), "user-a", "CRMMS-77911")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if second.Source != jiraDomain.BranchNameSourceModel {
		t.Fatalf("a fallback must not be cached, source = %q", second.Source)
	}
	if model.calls != 2 {
		t.Fatalf("model calls = %d, want 2", model.calls)
	}
}

func TestSuggestNormalisesTheModelSegments(t *testing.T) {
	issues := &stubBranchNameIssues{issue: branchNameIssue()}
	model := &stubBranchNameModel{answers: []string{`{"category":"enhancement","category_choice":"new","core":"manage user","evidence":"Halaman Utama Manage User (Grouping by PN)"}`}}
	service := NewBranchNameService(issues, model, nil)

	suggestion, err := service.Suggest(context.Background(), "user-a", "CRMMS-77911")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if suggestion.Branch != "feature/CRMMS-77911-Enhancement-Manage-User" {
		t.Fatalf("branch = %q", suggestion.Branch)
	}
}

func TestSuggestFallsBackForAMalformedModelAnswer(t *testing.T) {
	tests := []struct {
		name   string
		answer string
	}{
		{name: "not json", answer: "Manage-User"},
		{name: "missing evidence", answer: `{"category":"Enhancement","category_choice":"new","core":"Manage-User","evidence":""}`},
		{name: "too many core words", answer: `{"category":"Enhancement","category_choice":"new","core":"Manage User Grouping Detail PN List","evidence":"Halaman Utama Manage User (Grouping by PN)"}`},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			issues := &stubBranchNameIssues{issue: branchNameIssue()}
			model := &stubBranchNameModel{answers: []string{testCase.answer}}
			service := NewBranchNameService(issues, model, nil)

			suggestion, err := service.Suggest(context.Background(), "user-a", "CRMMS-77911")
			if err != nil {
				t.Fatalf("a malformed answer must not fail the request: %v", err)
			}
			if suggestion.Source != jiraDomain.BranchNameSourceFallback {
				t.Fatalf("source = %q, want the fallback", suggestion.Source)
			}
			if suggestion.Core != branchNameFallbackCore {
				t.Fatalf("core = %q, want %q", suggestion.Core, branchNameFallbackCore)
			}
			if suggestion.Evidence == "" {
				t.Fatal("the fallback must still name the phrase it used")
			}
		})
	}
}

func TestSuggestRefusesModelEvidenceThatIsNotInTheIssue(t *testing.T) {
	issues := &stubBranchNameIssues{issue: branchNameIssue()}
	model := &stubBranchNameModel{answers: []string{`{"category":"Enhancement","category_choice":"new","core":"Manage-User","evidence":"Totally Invented Heading"}`}}
	service := NewBranchNameService(issues, model, nil)

	suggestion, err := service.Suggest(context.Background(), "user-a", "CRMMS-77911")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if suggestion.Source != jiraDomain.BranchNameSourceFallback {
		t.Fatalf("a name whose evidence is not in the issue is a defect, source = %q", suggestion.Source)
	}
}

func TestSuggestRejectsAnInvalidIssueKey(t *testing.T) {
	issues := &stubBranchNameIssues{issue: branchNameIssue()}
	model := &stubBranchNameModel{answers: []string{branchNameModelAnswer}}
	service := NewBranchNameService(issues, model, nil)

	if _, err := service.Suggest(context.Background(), "user-a", "not a key"); !errors.Is(err, sharedErrors.ErrBadRequest) {
		t.Fatalf("error = %v, want a bad request", err)
	}
	if len(issues.requests) != 0 || model.calls != 0 {
		t.Fatalf("an invalid key must not reach Jira or the model: %v, %d calls", issues.requests, model.calls)
	}
}

func TestSuggestPropagatesTheJiraFailure(t *testing.T) {
	issues := &stubBranchNameIssues{err: fmt.Errorf("%w: Jira PAT is not configured", sharedErrors.ErrUnauthorized)}
	model := &stubBranchNameModel{answers: []string{branchNameModelAnswer}}
	service := NewBranchNameService(issues, model, nil)

	if _, err := service.Suggest(context.Background(), "user-a", "CRMMS-77911"); !errors.Is(err, sharedErrors.ErrUnauthorized) {
		t.Fatalf("error = %v, want unauthorized", err)
	}
	if model.calls != 0 {
		t.Fatalf("the model must not run without an issue, got %d calls", model.calls)
	}
}

func TestSuggestRejectsAnEmptyAnswerFromJira(t *testing.T) {
	issues := &stubBranchNameIssues{}
	model := &stubBranchNameModel{answers: []string{branchNameModelAnswer}}
	service := NewBranchNameService(issues, model, nil)

	if _, err := service.Suggest(context.Background(), "user-a", "CRMMS-77911"); !errors.Is(err, sharedErrors.ErrBadGateway) {
		t.Fatalf("error = %v, want a bad gateway", err)
	}
}

func TestSuggestWithoutAModelReturnsTheFallback(t *testing.T) {
	issues := &stubBranchNameIssues{issue: branchNameIssue()}
	service := NewBranchNameService(issues, nil, nil)

	suggestion, err := service.Suggest(context.Background(), "user-a", "CRMMS-77911")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if suggestion.Source != jiraDomain.BranchNameSourceFallback {
		t.Fatalf("source = %q, want the fallback", suggestion.Source)
	}
}

func TestSuggestRefusesAnIssueWithoutASummary(t *testing.T) {
	issues := &stubBranchNameIssues{issue: &jiraDomain.JiraIssue{Key: "CRMMS-77911"}}
	model := &stubBranchNameModel{err: errors.New("no AI provider is configured")}
	service := NewBranchNameService(issues, model, nil)

	if _, err := service.Suggest(context.Background(), "user-a", "CRMMS-77911"); !errors.Is(err, sharedErrors.ErrBadGateway) {
		t.Fatalf("error = %v, want a bad gateway, because a name without evidence is a defect", err)
	}
}

func TestCategoryRepeatsCoreMatchesWholeWords(t *testing.T) {
	tests := []struct {
		name     string
		category string
		core     string
		want     bool
	}{
		{name: "leading word", category: "Manage", core: "Manage-User", want: true},
		{name: "trailing word", category: "User", core: "Manage-User", want: true},
		{name: "different case", category: "manage", core: "Manage-User", want: true},
		{name: "second word of a two word category", category: "Detail-User", core: "Manage-User", want: true},
		{name: "no shared word", category: "Enhancement", core: "Manage-User", want: false},
		{name: "only a substring", category: "Manage", core: "Management-User", want: false},
		{name: "no category", category: "", core: "Manage-User", want: false},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			if got := categoryRepeatsCore(testCase.category, testCase.core); got != testCase.want {
				t.Fatalf("categoryRepeatsCore(%q, %q) = %t, want %t", testCase.category, testCase.core, got, testCase.want)
			}
		})
	}
}
