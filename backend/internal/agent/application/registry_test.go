package application

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"lunar/backend/internal/agent/domain"
)

func TestMapToolRegistry_RegisterAndGet(t *testing.T) {
	registry := NewMapToolRegistry()
	tool := newStubTool("search_code", nil)

	registry.Register(nil)
	if _, exists := registry.Get("search_code"); exists {
		t.Fatal("registering nil must not add a tool")
	}

	registry.Register(tool)
	resolved, exists := registry.Get("search_code")
	if !exists {
		t.Fatal("registered tool must be retrievable")
	}
	if resolved.Definition().Function.Name != "search_code" {
		t.Errorf("resolved tool = %q, want search_code", resolved.Definition().Function.Name)
	}
	if _, exists := registry.Get("missing_tool"); exists {
		t.Error("unknown tool must not be retrievable")
	}
}

func TestMapToolRegistry_ReRegisterReplacesDefinition(t *testing.T) {
	registry := NewMapToolRegistry()
	registry.Register(newStubTool("search_code", nil))

	replacement := newStubTool("search_code", nil)
	replacement.definition.Function.Description = "updated description"
	registry.Register(replacement)

	definitions := registry.Definitions()
	if len(definitions) != 1 {
		t.Fatalf("definitions = %d, want 1 after re-registration", len(definitions))
	}
	if definitions[0].Function.Description != "updated description" {
		t.Errorf("description = %q, want the replacement definition", definitions[0].Function.Description)
	}

	resolved, _ := registry.Get("search_code")
	if resolved != domain.Tool(replacement) {
		t.Error("re-registration must replace the stored tool instance")
	}
}

func TestMapToolRegistry_DefinitionsReturnsCopy(t *testing.T) {
	registry := NewMapToolRegistry()
	registry.RegisterAll(newStubTool("search_code", nil), newStubTool("get_git_diff", nil))

	definitions := registry.Definitions()
	if len(definitions) != 2 {
		t.Fatalf("definitions = %d, want 2", len(definitions))
	}
	definitions[0].Function.Name = "mutated"

	if registry.Definitions()[0].Function.Name == "mutated" {
		t.Error("Definitions must return a copy that callers cannot mutate")
	}
}

func TestMapToolRegistry_Execute(t *testing.T) {
	registry := NewMapToolRegistry()
	registry.Register(newStubTool("search_code", func(ctx context.Context, args json.RawMessage) (domain.ToolResult, error) {
		if string(args) != `{"query":"payment"}` {
			t.Errorf("args = %s, want the raw model arguments", args)
		}
		return domain.ToolResult{Content: "found", Sources: []domain.SourceReference{{Repo: "aurora"}}}, nil
	}))

	result, err := registry.Execute(context.Background(), "search_code", json.RawMessage(`{"query":"payment"}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Content != "found" || len(result.Sources) != 1 {
		t.Errorf("result = %+v, want the tool output", result)
	}

	_, err = registry.Execute(context.Background(), "missing_tool", nil)
	if err == nil {
		t.Fatal("expected an error for an unknown tool")
	}
	if err.Error() != "tool not found: missing_tool" {
		t.Errorf("error = %q, want the English not-found message", err.Error())
	}
}

func TestMapToolRegistry_ExecuteWrapsToolError(t *testing.T) {
	registry := NewMapToolRegistry()
	toolErr := errors.New("disk unavailable")
	registry.Register(newStubTool("get_file_content", func(ctx context.Context, args json.RawMessage) (domain.ToolResult, error) {
		return domain.ToolResult{}, toolErr
	}))

	_, err := registry.Execute(context.Background(), "get_file_content", json.RawMessage(`{}`))
	if !errors.Is(err, toolErr) {
		t.Fatalf("error = %v, want it to wrap the tool error", err)
	}
	if !strings.Contains(err.Error(), "execute tool get_file_content") {
		t.Errorf("error = %v, want the tool name in the context", err)
	}
}

func TestMapToolRegistry_ScopedDefinitions(t *testing.T) {
	registry := NewMapToolRegistry()
	registry.RegisterAll(
		newStubTool("search_code", nil),
		newStubTool("get_file_content", nil),
		newStubTool("get_jira_issue_detail", nil),
		newStubTool("get_pull_request_diff", nil),
		newStubTool("get_query_review_context", nil),
		newStubTool("unmapped_tool", nil),
	)

	tests := []struct {
		name        string
		domains     []string
		wantNames   []string
		wantExclude []string
	}{
		{
			name:      "empty domain list returns every definition",
			domains:   nil,
			wantNames: []string{"search_code", "get_file_content", "get_jira_issue_detail", "get_pull_request_diff", "get_query_review_context", "unmapped_tool"},
		},
		{
			name:        "servicemap domain returns only code intelligence tools",
			domains:     []string{DomainServiceMap},
			wantNames:   []string{"search_code", "get_file_content"},
			wantExclude: []string{"get_jira_issue_detail", "get_pull_request_diff", "unmapped_tool"},
		},
		{
			name:        "jira domain returns only jira tools",
			domains:     []string{DomainJira},
			wantNames:   []string{"get_jira_issue_detail"},
			wantExclude: []string{"search_code"},
		},
		{
			name:      "domain names are trimmed and case insensitive",
			domains:   []string{"  JIRA  "},
			wantNames: []string{"get_jira_issue_detail"},
		},
		{
			name:        "multiple domains are combined",
			domains:     []string{DomainBitbucket, DomainQueryReview},
			wantNames:   []string{"get_pull_request_diff", "get_query_review_context"},
			wantExclude: []string{"search_code", "get_jira_issue_detail"},
		},
		{
			name:        "unknown domain yields no definitions",
			domains:     []string{"unknown"},
			wantExclude: []string{"search_code", "get_jira_issue_detail", "get_pull_request_diff"},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			scoped := registry.ScopedDefinitions(testCase.domains)
			names := make(map[string]bool, len(scoped))
			for _, definition := range scoped {
				names[definition.Function.Name] = true
			}

			for _, wantName := range testCase.wantNames {
				if !names[wantName] {
					t.Errorf("missing %q in scoped definitions %v", wantName, names)
				}
			}
			for _, excludedName := range testCase.wantExclude {
				if names[excludedName] {
					t.Errorf("%q must be excluded from scoped definitions %v", excludedName, names)
				}
			}
			if len(testCase.domains) > 0 && len(scoped) != len(testCase.wantNames) {
				t.Errorf("scoped definitions = %d, want %d (%v)", len(scoped), len(testCase.wantNames), names)
			}
		})
	}
}

func TestMapToolRegistry_ScopedDefinitionsDoesNotMutateRegistry(t *testing.T) {
	registry := NewMapToolRegistry()
	registry.Register(newStubTool("search_code", nil))

	scoped := registry.ScopedDefinitions([]string{DomainJira})
	if len(scoped) != 0 {
		t.Fatalf("scoped definitions = %v, want empty for an unrelated domain", scoped)
	}
	if len(registry.Definitions()) != 1 {
		t.Error("scoped filtering must not remove registered tools")
	}
}
