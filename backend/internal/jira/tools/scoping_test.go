package tools

import (
	"slices"
	"testing"

	agentapplication "lunar/backend/internal/agent/application"
)

func jiraToolNames() []string {
	return []string{ToolNameListMyIssues, ToolNameGetIssueDetail, ToolNameGetActiveSprintIssues}
}

func TestScopedDefinitionsReturnsTheJiraToolsForTheJiraDomain(t *testing.T) {
	registry := agentapplication.NewMapToolRegistry()
	registry.RegisterAll(NewTools(NewBackendClient("http://127.0.0.1:1", "token"))...)

	jiraDefinitions := registry.ScopedDefinitions([]string{"jira"})
	if len(jiraDefinitions) != 3 {
		t.Fatalf("expected the three Jira tools for the jira domain, got %d", len(jiraDefinitions))
	}

	names := make([]string, 0, len(jiraDefinitions))
	for _, definition := range jiraDefinitions {
		names = append(names, definition.Function.Name)
	}
	for _, expected := range jiraToolNames() {
		if !slices.Contains(names, expected) {
			t.Errorf("expected %q among the jira domain tools, got %v", expected, names)
		}
	}
}

func TestScopedDefinitionsExcludesTheJiraToolsForTheServiceMapDomain(t *testing.T) {
	registry := agentapplication.NewMapToolRegistry()
	registry.RegisterAll(NewTools(NewBackendClient("http://127.0.0.1:1", "token"))...)

	serviceMapDefinitions := registry.ScopedDefinitions([]string{"servicemap"})
	for _, definition := range serviceMapDefinitions {
		if slices.Contains(jiraToolNames(), definition.Function.Name) {
			t.Errorf("the servicemap domain must not offer the Jira tool %q", definition.Function.Name)
		}
	}
	if len(serviceMapDefinitions) != 0 {
		t.Fatalf("expected no servicemap tools from a Jira-only registry, got %d", len(serviceMapDefinitions))
	}
}
