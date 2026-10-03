package tools

import (
	"strings"
	"testing"
)

const symbolFixture = `package payment

func ProcessPayment(id string) string {
	return id
}

func handle(id string) string {
	return ProcessPayment(id)
}
`

func TestFindSymbolReferencesSeparatesDefinitionsFromUsages(t *testing.T) {
	store := openIndexTestStore(t)
	saveExtractedChunks(t, store, "payments-service", "internal/payment/payment.go", symbolFixture, ".go")
	tools := newTestTools(store, t.TempDir())

	result, err := executeTool(t, mustFindTool(t, tools, ToolNameFindSymbolReferences), map[string]any{"symbol": "ProcessPayment"})
	if err != nil {
		t.Fatalf("find_symbol_references failed: %v", err)
	}
	if !strings.Contains(result.Content, "Definitions (1 found)") {
		t.Errorf("expected one definition section entry:\n%s", result.Content)
	}
	if !strings.Contains(result.Content, "Calls and usages (1 found)") {
		t.Errorf("expected one usage section entry:\n%s", result.Content)
	}
	if !strings.Contains(result.Content, "func ProcessPayment(id string) string") {
		t.Errorf("expected the definition snippet:\n%s", result.Content)
	}
	if len(result.Sources) != 2 {
		t.Fatalf("expected two sources, got %d", len(result.Sources))
	}
	if result.Sources[0].StartLine != 3 {
		t.Errorf("expected the definition on line 3, got %d", result.Sources[0].StartLine)
	}
	if result.Sources[1].StartLine != 7 {
		t.Errorf("expected the usage on line 7, got %d", result.Sources[1].StartLine)
	}
}

func TestFindSymbolReferencesReportsMissingSymbol(t *testing.T) {
	store := openIndexTestStore(t)
	saveExtractedChunks(t, store, "payments-service", "internal/payment/payment.go", symbolFixture, ".go")
	tools := newTestTools(store, t.TempDir())

	result, err := executeTool(t, mustFindTool(t, tools, ToolNameFindSymbolReferences), map[string]any{"symbol": "UnrelatedSymbol"})
	if err != nil {
		t.Fatalf("find_symbol_references failed: %v", err)
	}
	if !strings.Contains(result.Content, "No reference was found") {
		t.Errorf("expected an empty-result message, got:\n%s", result.Content)
	}
	if len(result.Sources) != 0 {
		t.Errorf("expected no sources, got %+v", result.Sources)
	}
}

func TestFindSymbolReferencesRejectsBlankSymbol(t *testing.T) {
	store := openIndexTestStore(t)
	tool := mustFindTool(t, newTestTools(store, t.TempDir()), ToolNameFindSymbolReferences)

	if _, err := executeTool(t, tool, map[string]any{"symbol": "   "}); err == nil {
		t.Error("expected an error for a blank symbol")
	}
	if _, err := executeTool(t, tool, map[string]any{"symbol": "ProcessPayment", "repo_filter": "../outside"}); err == nil {
		t.Error("expected an error for a repository filter that escapes the workspace root")
	}
}
