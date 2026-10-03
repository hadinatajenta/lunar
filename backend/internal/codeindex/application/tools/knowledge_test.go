package tools

import (
	"context"
	"strings"
	"testing"

	codeindexdomain "lunar/backend/internal/codeindex/domain"
)

func seedKnowledge(t *testing.T, store codeindexdomain.IndexStore) {
	t.Helper()

	items := []codeindexdomain.DomainKnowledgeItem{
		{Term: "APP_JENIS", Explanation: "Application type code used by the onboarding flow."},
		{Term: "STATUS_APPROVED", Explanation: "The application passed every approval step."},
	}
	if err := store.SaveKnowledge(context.Background(), items); err != nil {
		t.Fatalf("cannot seed domain knowledge: %v", err)
	}
}

func TestGetDomainKnowledgeReturnsMatchingNotes(t *testing.T) {
	store := openIndexTestStore(t)
	seedKnowledge(t, store)
	tools := newTestTools(store, t.TempDir())

	result, err := executeTool(t, mustFindTool(t, tools, ToolNameGetDomainKnowledge), map[string]any{"query": "onboarding"})
	if err != nil {
		t.Fatalf("get_domain_knowledge failed: %v", err)
	}
	if !strings.Contains(result.Content, "APP_JENIS") {
		t.Errorf("expected the matching note term:\n%s", result.Content)
	}
	if !strings.Contains(result.Content, "Application type code") {
		t.Errorf("expected the matching note explanation:\n%s", result.Content)
	}
	if strings.Contains(result.Content, "STATUS_APPROVED") {
		t.Errorf("an unrelated note must not be returned:\n%s", result.Content)
	}
	if len(result.Sources) != 0 {
		t.Errorf("expected no sources, got %+v", result.Sources)
	}
}

func TestGetDomainKnowledgeClampsRequestedLimit(t *testing.T) {
	store := openIndexTestStore(t)
	seedKnowledge(t, store)
	tools := newTestTools(store, t.TempDir())

	result, err := executeTool(t, mustFindTool(t, tools, ToolNameGetDomainKnowledge), map[string]any{
		"query": "application",
		"limit": 1,
	})
	if err != nil {
		t.Fatalf("get_domain_knowledge failed: %v", err)
	}
	if sections := strings.Count(result.Content, "### "); sections != 1 {
		t.Errorf("expected one note section, got %d:\n%s", sections, result.Content)
	}
}

func TestGetDomainKnowledgeRejectsBlankQuery(t *testing.T) {
	store := openIndexTestStore(t)
	seedKnowledge(t, store)
	tool := mustFindTool(t, newTestTools(store, t.TempDir()), ToolNameGetDomainKnowledge)

	if _, err := executeTool(t, tool, map[string]any{"query": "   "}); err == nil {
		t.Error("expected an error for a blank query")
	}
}
