package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	agentdomain "lunar/backend/internal/agent/domain"
)

const testSessionToken = "session-token-abc123"

func newTestTools(t *testing.T, handler http.Handler) []agentdomain.Tool {
	t.Helper()

	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	return NewTools(NewBackendClient(server.URL, testSessionToken))
}

func mustFindTool(t *testing.T, tools []agentdomain.Tool, name string) agentdomain.Tool {
	t.Helper()

	for _, tool := range tools {
		if tool.Definition().Function.Name == name {
			return tool
		}
	}
	t.Fatalf("tool %q is not registered", name)
	return nil
}

func executeTool(t *testing.T, tool agentdomain.Tool, arguments map[string]any) (agentdomain.ToolResult, error) {
	t.Helper()

	encodedArguments, err := json.Marshal(arguments)
	if err != nil {
		t.Fatalf("cannot encode arguments for %q: %v", tool.Definition().Function.Name, err)
	}
	return tool.Execute(context.Background(), encodedArguments)
}

func TestListMyJiraIssuesCallsTheIssuesEndpoint(t *testing.T) {
	var capturedPath string
	var capturedAuthorization string

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		capturedAuthorization = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[
			{
				"key": "CRMMS-1",
				"fields": {
					"summary": "Fix login redirect",
					"status": {"name": "In Progress"},
					"issuetype": {"name": "Bug"},
					"priority": {"name": "High"},
					"assignee": {"displayName": "Jane Doe"}
				},
				"sprint_name": "Sprint 44 - Fortune Squad"
			}
		]`))
	})

	result, err := executeTool(t, mustFindTool(t, newTestTools(t, handler), ToolNameListMyIssues), nil)
	if err != nil {
		t.Fatalf("list_my_jira_issues failed: %v", err)
	}
	if capturedPath != "/api/jira/issues" {
		t.Fatalf("expected the issues endpoint, got %q", capturedPath)
	}
	if capturedAuthorization != "Bearer "+testSessionToken {
		t.Fatalf("expected the bearer session token, got %q", capturedAuthorization)
	}

	expectedFragments := []string{
		"CRMMS-1",
		"Fix login redirect",
		"status=In Progress",
		"type=Bug",
		"priority=High",
		"assignee=Jane Doe",
		"sprint=Sprint 44 - Fortune Squad",
	}
	for _, fragment := range expectedFragments {
		if !strings.Contains(result.Content, fragment) {
			t.Errorf("expected %q in the summary:\n%s", fragment, result.Content)
		}
	}

	if len(result.Sources) != 1 {
		t.Fatalf("expected one source, got %d", len(result.Sources))
	}
	if result.Sources[0].Type != sourceTypeJira || result.Sources[0].File != "CRMMS-1" {
		t.Fatalf("unexpected source: %+v", result.Sources[0])
	}
}

func TestListMyJiraIssuesReportsNoIssues(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`null`))
	})

	result, err := executeTool(t, mustFindTool(t, newTestTools(t, handler), ToolNameListMyIssues), nil)
	if err != nil {
		t.Fatalf("list_my_jira_issues failed: %v", err)
	}
	if !strings.Contains(result.Content, "No Jira issues") {
		t.Fatalf("expected an empty result message, got %q", result.Content)
	}
	if len(result.Sources) != 0 {
		t.Fatalf("expected no sources for an empty result, got %d", len(result.Sources))
	}
}

func TestListMyJiraIssuesRejectsAnOversizedResponse(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(bytes.Repeat([]byte("a"), maxResponseBytes+1))
	})

	_, err := executeTool(t, mustFindTool(t, newTestTools(t, handler), ToolNameListMyIssues), nil)
	if err == nil {
		t.Fatal("expected an error for an oversized response body")
	}
	if !strings.Contains(err.Error(), "exceeds") {
		t.Fatalf("expected the bound to be reported, got %v", err)
	}
}

func TestGetJiraIssueDetailCallsTheIssueEndpoint(t *testing.T) {
	var capturedPath string
	var capturedAuthorization string

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		capturedAuthorization = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"key": "CRMMS-2",
			"fields": {
				"summary": "Add export",
				"description": "Acceptance criteria here.",
				"updated": "2026-10-01T10:00:00.000+0700",
				"project": {"key": "CRMMS", "name": "CRM"},
				"status": {"name": "To Do"},
				"issuetype": {"name": "Story"},
				"priority": {"name": "Medium"},
				"assignee": {"displayName": "Budi"},
				"duedate": "2026-10-20",
				"parent": {"key": "CRMMS-100", "fields": {"summary": "Epic export"}}
			}
		}`))
	})

	result, err := executeTool(
		t,
		mustFindTool(t, newTestTools(t, handler), ToolNameGetIssueDetail),
		map[string]any{"issue_key": "crmms-2"},
	)
	if err != nil {
		t.Fatalf("get_jira_issue_detail failed: %v", err)
	}
	if capturedPath != "/api/jira/issues/CRMMS-2" {
		t.Fatalf("expected the canonical issue endpoint, got %q", capturedPath)
	}
	if capturedAuthorization != "Bearer "+testSessionToken {
		t.Fatalf("expected the bearer session token, got %q", capturedAuthorization)
	}

	expectedFragments := []string{
		"CRMMS-2: Add export",
		"status=To Do",
		"type=Story",
		"assignee=Budi",
		"project=CRMMS",
		"parent=CRMMS-100 Epic export",
		"due=2026-10-20",
		"Acceptance criteria here.",
	}
	for _, fragment := range expectedFragments {
		if !strings.Contains(result.Content, fragment) {
			t.Errorf("expected %q in the detail:\n%s", fragment, result.Content)
		}
	}

	if len(result.Sources) != 1 {
		t.Fatalf("expected one source, got %d", len(result.Sources))
	}
	if result.Sources[0].Type != sourceTypeJira || result.Sources[0].File != "CRMMS-2" {
		t.Fatalf("unexpected source: %+v", result.Sources[0])
	}
}

func TestGetJiraIssueDetailRejectsAnInvalidIssueKey(t *testing.T) {
	var requestCount int

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{}`))
	})

	tool := mustFindTool(t, newTestTools(t, handler), ToolNameGetIssueDetail)
	invalidKeys := []string{
		"",
		"   ",
		"CRMMS",
		"CRMMS-",
		"-1",
		"CRMMS/1",
		"../etc/passwd",
		"CRMMS-1/../CRMMS-2",
		"1CRMMS-1",
	}

	for _, invalidKey := range invalidKeys {
		if _, err := executeTool(t, tool, map[string]any{"issue_key": invalidKey}); err == nil {
			t.Errorf("expected an error for issue key %q", invalidKey)
		}
	}
	if requestCount != 0 {
		t.Fatalf("an invalid issue key must not reach the backend, got %d requests", requestCount)
	}
}

func TestGetActiveSprintIssuesCallsTheBacklogEndpoint(t *testing.T) {
	var capturedPath string
	var capturedAuthorization string

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		capturedAuthorization = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"sprints": [
				{
					"id": 42,
					"name": "Sprint 44 - Fortune Squad",
					"state": "active",
					"issues": [
						{"key": "CRMMS-3", "fields": {"summary": "First task", "status": {"name": "To Do"}, "issuetype": {"name": "Story"}}},
						{"key": "CRMMS-4", "fields": {"summary": "Second task", "status": {"name": "In Progress"}, "issuetype": {"name": "Bug"}}}
					]
				}
			],
			"total_issues": 2,
			"active_sprint_name": "Sprint 44 - Fortune Squad",
			"detected_squad": "Fortune Squad"
		}`))
	})

	result, err := executeTool(t, mustFindTool(t, newTestTools(t, handler), ToolNameGetActiveSprintIssues), nil)
	if err != nil {
		t.Fatalf("get_active_sprint_issues failed: %v", err)
	}
	if capturedPath != "/api/jira/backlog" {
		t.Fatalf("expected the backlog endpoint, got %q", capturedPath)
	}
	if capturedAuthorization != "Bearer "+testSessionToken {
		t.Fatalf("expected the bearer session token, got %q", capturedAuthorization)
	}

	expectedFragments := []string{
		"Active sprint backlog: Sprint 44 - Fortune Squad",
		"Squad: Fortune Squad",
		"## Sprint 44 - Fortune Squad (active)",
		"CRMMS-3 First task",
		"CRMMS-4 Second task",
		"sprint=Sprint 44 - Fortune Squad",
	}
	for _, fragment := range expectedFragments {
		if !strings.Contains(result.Content, fragment) {
			t.Errorf("expected %q in the backlog:\n%s", fragment, result.Content)
		}
	}

	if len(result.Sources) != 2 {
		t.Fatalf("expected two sources, got %d", len(result.Sources))
	}
}

func TestGetActiveSprintIssuesSurfacesABackendError(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte(`{"error":"check your VPN connection"}`))
	})

	_, err := executeTool(t, mustFindTool(t, newTestTools(t, handler), ToolNameGetActiveSprintIssues), nil)
	if err == nil {
		t.Fatal("expected an error for a backend failure")
	}
	if !strings.Contains(err.Error(), "502") {
		t.Fatalf("expected the status to be reported, got %v", err)
	}
	if !strings.Contains(err.Error(), "check your VPN connection") {
		t.Fatalf("expected the backend message to be surfaced, got %v", err)
	}
}
