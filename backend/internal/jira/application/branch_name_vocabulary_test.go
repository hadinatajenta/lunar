package application

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	jiraDomain "lunar/backend/internal/jira/domain"
)

const observedCategoryAnswer = `{"category":"Enhancement","category_choice":"observed","core":"Manage-User","evidence":"Halaman Utama Manage User (Grouping by PN)"}`

func observedVocabulary() *stubCategoryVocabulary {
	return &stubCategoryVocabulary{counts: map[string]int{"Manage": 9, "Report": 8, "Enhancement": 3}}
}

func TestBuildCategoryVocabularyRanksByCountAndMergesSpellings(t *testing.T) {
	vocabulary := BuildCategoryVocabulary(map[string]int{"report": 1, "Report": 3, "Manage": 4, "Enhancement": 3, "enhancement": 2})
	want := []ObservedCategory{
		{Name: "Enhancement", Count: 5},
		{Name: "Manage", Count: 4},
		{Name: "Report", Count: 4},
	}
	if vocabulary.ObservedTotal != len(want) {
		t.Fatalf("observed total = %d, want %d", vocabulary.ObservedTotal, len(want))
	}
	if len(vocabulary.Listed) != len(want) {
		t.Fatalf("listed = %+v, want %+v", vocabulary.Listed, want)
	}
	for index := range want {
		if vocabulary.Listed[index] != want[index] {
			t.Fatalf("listed = %+v, want %+v", vocabulary.Listed, want)
		}
	}
}

func TestBuildCategoryVocabularyCapsTheListAndReportsTheTotal(t *testing.T) {
	counts := make(map[string]int)
	for index := 0; index < 20; index++ {
		counts[fmt.Sprintf("Category%02d", index)] = index + 1
	}

	vocabulary := BuildCategoryVocabulary(counts)
	if len(vocabulary.Listed) != maximumObservedCategories {
		t.Fatalf("listed %d categories, want the cap of %d", len(vocabulary.Listed), maximumObservedCategories)
	}
	if vocabulary.ObservedTotal != 20 {
		t.Fatalf("observed total = %d, want 20", vocabulary.ObservedTotal)
	}
	if vocabulary.Listed[0].Name != "Category19" {
		t.Fatalf("the most common category must come first, got %+v", vocabulary.Listed[0])
	}
	if _, found := vocabulary.Resolve("Category00"); found {
		t.Fatal("a category behind the cap must not be selectable")
	}
}

func TestCategoryVocabularyResolveKeepsTheObservedSpelling(t *testing.T) {
	vocabulary := BuildCategoryVocabulary(map[string]int{"CM": 2})

	resolved, found := vocabulary.Resolve("cm")
	if !found || resolved != "CM" {
		t.Fatalf("resolve = %q, %t, want CM, true", resolved, found)
	}
}

func TestCategoryVocabularyIsEmptyWithoutObservedBranches(t *testing.T) {
	if !BuildCategoryVocabulary(nil).IsEmpty() {
		t.Fatal("a workspace without keyed branches must produce an empty vocabulary")
	}
}

func TestSuggestChoosesAnObservedCategory(t *testing.T) {
	issues := &stubBranchNameIssues{issue: branchNameIssue()}
	model := &stubBranchNameModel{answers: []string{observedCategoryAnswer}}
	vocabulary := observedVocabulary()
	service := NewBranchNameService(issues, model, vocabulary)

	suggestion, err := service.Suggest(context.Background(), "user-a", "CRMMS-77911")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if suggestion.Branch != "feature/CRMMS-77911-Enhancement-Manage-User" {
		t.Fatalf("branch = %q", suggestion.Branch)
	}
	if suggestion.Category != "Enhancement" || suggestion.Source != jiraDomain.BranchNameSourceModel {
		t.Fatalf("category = %q, source = %q", suggestion.Category, suggestion.Source)
	}
	if vocabulary.calls != 1 {
		t.Fatalf("the vocabulary must be read once, got %d calls", vocabulary.calls)
	}

	prompt := model.prompts[0]
	manageIndex := strings.Index(prompt, "Manage (9)")
	reportIndex := strings.Index(prompt, "Report (8)")
	enhancementIndex := strings.Index(prompt, "Enhancement (3)")
	if manageIndex < 0 || reportIndex < 0 || enhancementIndex < 0 {
		t.Fatalf("the prompt must list every observed category with its count, got %q", prompt)
	}
	if manageIndex > reportIndex || reportIndex > enhancementIndex {
		t.Fatalf("the observed categories must be most common first, got %q", prompt)
	}
}

func TestSuggestRejectsACategoryCalledObservedThatWasNotListed(t *testing.T) {
	issues := &stubBranchNameIssues{issue: branchNameIssue()}
	model := &stubBranchNameModel{answers: []string{`{"category":"Feature","category_choice":"observed","core":"Manage-User","evidence":"Halaman Utama Manage User (Grouping by PN)"}`}}
	service := NewBranchNameService(issues, model, observedVocabulary())

	suggestion, err := service.Suggest(context.Background(), "user-a", "CRMMS-77911")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if suggestion.Source != jiraDomain.BranchNameSourceFallback {
		t.Fatalf("source = %q, want the fallback", suggestion.Source)
	}
	if suggestion.Category != "" {
		t.Fatalf("the fallback must not carry the invented category %q", suggestion.Category)
	}
}

func TestSuggestAcceptsANewCategoryWhenItSaysSo(t *testing.T) {
	issues := &stubBranchNameIssues{issue: branchNameIssue()}
	model := &stubBranchNameModel{answers: []string{`{"category":"Feature","category_choice":"new","core":"Manage-User","evidence":"Halaman Utama Manage User (Grouping by PN)"}`}}
	service := NewBranchNameService(issues, model, observedVocabulary())

	suggestion, err := service.Suggest(context.Background(), "user-a", "CRMMS-77911")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if suggestion.Source != jiraDomain.BranchNameSourceModel || suggestion.Category != "Feature" {
		t.Fatalf("category = %q, source = %q", suggestion.Category, suggestion.Source)
	}
	if suggestion.Branch != "feature/CRMMS-77911-Feature-Manage-User" {
		t.Fatalf("branch = %q", suggestion.Branch)
	}
}

func TestSuggestFallsBackWhenTheModelDoesNotStateTheCategoryChoice(t *testing.T) {
	issues := &stubBranchNameIssues{issue: branchNameIssue()}
	model := &stubBranchNameModel{answers: []string{`{"category":"Enhancement","core":"Manage-User","evidence":"Halaman Utama Manage User (Grouping by PN)"}`}}
	service := NewBranchNameService(issues, model, observedVocabulary())

	suggestion, err := service.Suggest(context.Background(), "user-a", "CRMMS-77911")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if suggestion.Source != jiraDomain.BranchNameSourceFallback {
		t.Fatalf("an unstated choice is malformed, source = %q", suggestion.Source)
	}
}

func TestSuggestAsksForANewCategoryWithoutAVocabulary(t *testing.T) {
	issues := &stubBranchNameIssues{issue: branchNameIssue()}
	model := &stubBranchNameModel{answers: []string{branchNameModelAnswer}}
	service := NewBranchNameService(issues, model, nil)

	suggestion, err := service.Suggest(context.Background(), "user-a", "CRMMS-77911")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if suggestion.Source != jiraDomain.BranchNameSourceModel {
		t.Fatalf("source = %q, want the model", suggestion.Source)
	}
	if !strings.Contains(model.prompts[0], "none were found") {
		t.Fatalf("a worker without a vocabulary must be told to propose a category, got %q", model.prompts[0])
	}
}

func TestSuggestKeepsNamingWhenTheVocabularyCannotBeRead(t *testing.T) {
	issues := &stubBranchNameIssues{issue: branchNameIssue()}
	model := &stubBranchNameModel{answers: []string{branchNameModelAnswer}}
	vocabulary := &stubCategoryVocabulary{err: errors.New("the workspace is not configured")}
	service := NewBranchNameService(issues, model, vocabulary)

	suggestion, err := service.Suggest(context.Background(), "user-a", "CRMMS-77911")
	if err != nil {
		t.Fatalf("an unreadable vocabulary must not fail the request: %v", err)
	}
	if suggestion.Source != jiraDomain.BranchNameSourceModel {
		t.Fatalf("source = %q, want the model", suggestion.Source)
	}
	if !strings.Contains(model.prompts[0], "none were found") {
		t.Fatalf("an unreadable vocabulary must degrade to a free choice, got %q", model.prompts[0])
	}
}

func TestSuggestDropsACategoryThatRepeatsTheCore(t *testing.T) {
	issues := &stubBranchNameIssues{issue: branchNameIssue()}
	model := &stubBranchNameModel{answers: []string{`{"category":"Manage","category_choice":"observed","core":"Manage-User","evidence":"Halaman Utama Manage User (Grouping by PN)"}`}}
	service := NewBranchNameService(issues, model, observedVocabulary())

	suggestion, err := service.Suggest(context.Background(), "user-a", "CRMMS-77911")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if suggestion.Branch != "feature/CRMMS-77911-Manage-User" {
		t.Fatalf("branch = %q, want the core without a repeated category", suggestion.Branch)
	}
	if suggestion.Category != "" {
		t.Fatalf("a category repeating the core must be dropped, got %q", suggestion.Category)
	}
	if suggestion.Source != jiraDomain.BranchNameSourceModel || suggestion.Core != "Manage-User" {
		t.Fatalf("source = %q, core = %q", suggestion.Source, suggestion.Core)
	}
}

func TestSuggestReturnsTheEvidenceExactlyAsTheModelWroteIt(t *testing.T) {
	issues := &stubBranchNameIssues{issue: branchNameIssue()}
	model := &stubBranchNameModel{answers: []string{`{"category":"Enhancement","category_choice":"observed","core":"Manage-User","evidence":"1. Halaman Utama Manage User (Grouping by PN)"}`}}
	service := NewBranchNameService(issues, model, observedVocabulary())

	suggestion, err := service.Suggest(context.Background(), "user-a", "CRMMS-77911")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if suggestion.Evidence != "1. Halaman Utama Manage User (Grouping by PN)" {
		t.Fatalf("evidence = %q, want the model answer unchanged", suggestion.Evidence)
	}
}
