package infrastructure_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"lunar/backend/internal/codeindex/application"
	"lunar/backend/internal/codeindex/domain"
	"lunar/backend/internal/codeindex/infrastructure"
)

var (
	_ application.KnowledgeLoader   = (*infrastructure.KnowledgeLoader)(nil)
	_ application.GlossaryGenerator = (*infrastructure.GlossaryGenerator)(nil)
)

const knowledgeDirectoryEnv = "LUNAR_KNOWLEDGE_DIR"

const glossaryFixture = `enums:
  app_type:
    description: "Merchant application type"
    values:
      "1": "EDC"
      "3": "MOCASH"
      "2": "QRIS"
  terminal_type:
    description: ""
    values:
      "B": "Traditional"
      "A": "Android"
notes:
  - term: "ProcessPaymentRequest"
    explanation: "Payment processing flow for EDC terminals."
`

type recordingKnowledgeStore struct {
	domain.IndexStore
	savedItems []domain.DomainKnowledgeItem
	isCleared  bool
}

func (s *recordingKnowledgeStore) ClearKnowledge(context.Context) error {
	s.isCleared = true
	s.savedItems = nil
	return nil
}

func (s *recordingKnowledgeStore) SaveKnowledge(_ context.Context, items []domain.DomainKnowledgeItem) error {
	s.savedItems = append(s.savedItems, items...)
	return nil
}

func (s *recordingKnowledgeStore) SearchKnowledge(_ context.Context, query string, limit int) ([]domain.DomainKnowledgeItem, error) {
	matches := make([]domain.DomainKnowledgeItem, 0, limit)
	for _, item := range s.savedItems {
		if len(matches) >= limit {
			break
		}
		if matchesKnowledgeQuery(item, query) {
			matches = append(matches, item)
		}
	}
	return matches, nil
}

func matchesKnowledgeQuery(item domain.DomainKnowledgeItem, query string) bool {
	haystack := strings.ToLower(item.Term + " " + item.Explanation)
	for _, token := range strings.Fields(strings.ToLower(query)) {
		if !strings.Contains(haystack, token) {
			return false
		}
	}
	return true
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatalf("cannot create directory for %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("cannot write %s: %v", path, err)
	}
}

func TestResolveKnowledgeDirPrefersEnvironmentOverride(t *testing.T) {
	t.Setenv(knowledgeDirectoryEnv, "/tmp/lunar-knowledge-fixture")

	resolved, err := infrastructure.ResolveKnowledgeDir()
	if err != nil {
		t.Fatalf("ResolveKnowledgeDir with override: %v", err)
	}
	if resolved != "/tmp/lunar-knowledge-fixture" {
		t.Fatalf("expected environment override path, got %q", resolved)
	}

	if err := os.Unsetenv(knowledgeDirectoryEnv); err != nil {
		t.Fatalf("cannot unset %s: %v", knowledgeDirectoryEnv, err)
	}
	homeDirectory, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("cannot resolve home directory: %v", err)
	}
	resolved, err = infrastructure.ResolveKnowledgeDir()
	if err != nil {
		t.Fatalf("ResolveKnowledgeDir default: %v", err)
	}
	expected := filepath.Join(homeDirectory, ".lunar", "knowledge")
	if resolved != expected {
		t.Fatalf("expected default directory %q, got %q", expected, resolved)
	}
}

func TestLoadGlossaryBuildsSortedEnumPromptSection(t *testing.T) {
	glossaryPath := filepath.Join(t.TempDir(), "glossary.yaml")
	writeFile(t, glossaryPath, glossaryFixture)

	glossary, err := infrastructure.LoadGlossary(glossaryPath)
	if err != nil {
		t.Fatalf("LoadGlossary: %v", err)
	}
	if len(glossary.Enums) != 2 {
		t.Fatalf("expected 2 enums, got %d", len(glossary.Enums))
	}

	appType, isFound := glossary.Enums["app_type"]
	if !isFound {
		t.Fatalf("expected app_type enum, got %v", glossary.Enums)
	}
	if appType.Description != "Merchant application type" {
		t.Fatalf("unexpected description %q", appType.Description)
	}
	if appType.Values["1"] != "EDC" || appType.Values["2"] != "QRIS" || appType.Values["3"] != "MOCASH" {
		t.Fatalf("unexpected values %v", appType.Values)
	}
	if len(glossary.Notes) != 1 || glossary.Notes[0].Term != "ProcessPaymentRequest" {
		t.Fatalf("unexpected notes %v", glossary.Notes)
	}

	expected := "- app_type (Merchant application type): 1=EDC, 2=QRIS, 3=MOCASH\n" +
		"- terminal_type: A=Android, B=Traditional"
	if actual := application.BuildEnumPromptSection(glossary); actual != expected {
		t.Fatalf("unexpected enum prompt section:\nexpected: %q\nactual:   %q", expected, actual)
	}
}

func TestBuildEnumPromptSectionHandlesEmptyGlossary(t *testing.T) {
	if actual := application.BuildEnumPromptSection(nil); actual != "" {
		t.Fatalf("expected empty section for nil glossary, got %q", actual)
	}
	if actual := application.BuildEnumPromptSection(&domain.Glossary{}); actual != "" {
		t.Fatalf("expected empty section for empty glossary, got %q", actual)
	}
}

func TestLoadAndIndexIndexesMarkdownAndRetrievesBySearch(t *testing.T) {
	knowledgeDirectory := t.TempDir()
	glossaryPath := filepath.Join(knowledgeDirectory, "glossary.yaml")
	writeFile(t, glossaryPath, glossaryFixture)
	writeFile(t, filepath.Join(knowledgeDirectory, "process", "outlet.md"),
		"# Outlet Submission\nVerifikator checks outlet completeness.\n\n## Approval\nDepartment head grants final approval.\n")
	writeFile(t, filepath.Join(knowledgeDirectory, "process", ".private", "secret.md"),
		"# Secret\nHidden knowledge.\n")
	writeFile(t, filepath.Join(knowledgeDirectory, "ignore.txt"), "# Ignored\nignore.txt knowledge.\n")

	store := &recordingKnowledgeStore{}
	loader := infrastructure.NewKnowledgeLoader(glossaryPath)
	glossary, err := loader.LoadAndIndex(context.Background(), store)
	if err != nil {
		t.Fatalf("LoadAndIndex: %v", err)
	}
	if !store.isCleared {
		t.Fatalf("expected existing knowledge to be cleared before indexing")
	}
	if len(glossary.Notes) != 3 {
		t.Fatalf("expected fixture note and two markdown sections, got %d", len(glossary.Notes))
	}
	if len(store.savedItems) != 3 {
		t.Fatalf("expected 3 indexed items, got %d", len(store.savedItems))
	}
	if store.savedItems[0].Term != "ProcessPaymentRequest (process payment request)" {
		t.Fatalf("expected expanded identifier term, got %q", store.savedItems[0].Term)
	}
	for _, item := range store.savedItems {
		if strings.Contains(item.Explanation, "Hidden knowledge") {
			t.Fatalf("hidden markdown must not be indexed: %v", item)
		}
		if strings.Contains(item.Explanation, "ignore.txt knowledge") {
			t.Fatalf("non markdown knowledge must not be indexed: %v", item)
		}
	}

	matches, err := store.SearchKnowledge(context.Background(), "verifikator", 5)
	if err != nil {
		t.Fatalf("SearchKnowledge: %v", err)
	}
	if len(matches) != 1 {
		t.Fatalf("expected one search match, got %d", len(matches))
	}
	if matches[0].Term != "process/outlet: Outlet Submission" {
		t.Fatalf("unexpected match term %q", matches[0].Term)
	}
	if !strings.Contains(matches[0].Explanation, "outlet completeness") {
		t.Fatalf("unexpected match explanation %q", matches[0].Explanation)
	}

	approvalMatches, err := store.SearchKnowledge(context.Background(), "department approval", 5)
	if err != nil {
		t.Fatalf("SearchKnowledge for approval: %v", err)
	}
	if len(approvalMatches) != 1 || approvalMatches[0].Term != "process/outlet: Approval" {
		t.Fatalf("expected approval section match, got %v", approvalMatches)
	}
}

func TestMergeGlossariesDeduplicatesAndKeepsManualNotes(t *testing.T) {
	manual := &domain.Glossary{
		Enums: map[string]domain.GlossaryEnum{"app_type": {Description: "Type", Values: map[string]string{"1": "EDC"}}},
		Notes: []domain.GlossaryNote{{Term: "  App Type ", Explanation: "manual explanation"}},
	}
	generated := &domain.Glossary{
		Enums: map[string]domain.GlossaryEnum{"generated_enum": {Description: "Generated", Values: map[string]string{"0": "Zero"}}},
		Notes: []domain.GlossaryNote{
			{Term: "app type", Explanation: "generated duplicate"},
			{Term: "enum status pacific StatusPending", Explanation: "generated note"},
		},
	}

	merged := infrastructure.MergeGlossaries(manual, generated)
	if len(merged.Notes) != 2 {
		t.Fatalf("expected 2 merged notes, got %v", merged.Notes)
	}
	if merged.Notes[0].Explanation != "manual explanation" {
		t.Fatalf("manual note must win the duplicate, got %v", merged.Notes)
	}
	if merged.Notes[1].Term != "enum status pacific StatusPending" {
		t.Fatalf("expected generated note appended, got %v", merged.Notes[1])
	}
	if _, isFound := merged.Enums["generated_enum"]; isFound {
		t.Fatalf("generated enums must not be merged: %v", merged.Enums)
	}
	if len(manual.Notes) != 1 || len(generated.Notes) != 2 || len(manual.Enums) != 1 {
		t.Fatalf("merge must not mutate its inputs")
	}

	empty := infrastructure.MergeGlossaries(nil, nil)
	if len(empty.Enums) != 0 || len(empty.Notes) != 0 {
		t.Fatalf("expected empty glossary for nil inputs, got %v", empty)
	}
}

func TestLoadAndIndexPropagatesCorruptGeneratedArtifact(t *testing.T) {
	knowledgeDirectory := t.TempDir()
	glossaryPath := filepath.Join(knowledgeDirectory, "glossary.yaml")
	writeFile(t, glossaryPath, glossaryFixture)
	writeFile(t, infrastructure.ResolveGeneratedGlossaryPath(glossaryPath), "{{{ not a glossary")

	store := &recordingKnowledgeStore{}
	if _, err := infrastructure.NewKnowledgeLoader(glossaryPath).LoadAndIndex(context.Background(), store); err == nil {
		t.Fatalf("expected an error for a corrupt generated glossary")
	}
	if store.isCleared {
		t.Fatalf("knowledge must not be cleared when the generated artifact is unreadable")
	}
}

func TestIndexNotesRejectsNilStore(t *testing.T) {
	if err := infrastructure.IndexNotes(context.Background(), &domain.Glossary{}, nil); err == nil {
		t.Fatalf("expected an error for a nil store")
	}
}
