package application

import (
	"context"
	"testing"

	bitbucketDomain "lunar/backend/internal/bitbucket/domain"
	copilotDomain "lunar/backend/internal/copilot/domain"
	"lunar/backend/internal/dashboard/domain"
	jiraDomain "lunar/backend/internal/jira/domain"
)

func TestDashboardService_JiraSummaryComputation(t *testing.T) {
	issues := []jiraDomain.JiraIssue{
		{Key: "DEV-1", SprintName: "Sprint 45 - Fortune Squad", SubLabel: "DEV Document · UT"},
		{Key: "DEV-2", SprintName: "Sprint 45 - Fortune Squad", SubLabel: "DEV Document · Query"},
		{Key: "DEV-3", SprintName: "Sprint 45 - Fortune Squad", SubLabel: "Bug Fix"},
		{Key: "DEV-4", SprintName: "Sprint 44 - Legacy Crew", SubLabel: "MMS Core"},
		{Key: "DEV-5", SubLabel: "DEV Document"},
	}
	service := NewDashboardService(
		&fakeJiraProvider{issues: issues},
		&fakeBitbucketProvider{},
		&fakeCopilotProvider{},
		configuredCredentials(),
	)

	jiraSection := service.GetSummary(context.Background(), testUserID).Jira

	if jiraSection.Status != domain.StatusOK {
		t.Errorf("expected status %q, got %q", domain.StatusOK, jiraSection.Status)
	}
	if !jiraSection.IsConfigured {
		t.Error("expected is_configured true")
	}
	if jiraSection.AssignedTickets != 5 {
		t.Errorf("expected 5 assigned tickets, got %d", jiraSection.AssignedTickets)
	}
	if jiraSection.TechnicalDocuments != 3 {
		t.Errorf("expected 3 technical documents, got %d", jiraSection.TechnicalDocuments)
	}
	if jiraSection.ActiveSprint != "Sprint 45 - Fortune Squad" {
		t.Errorf("expected active sprint %q, got %q", "Sprint 45 - Fortune Squad", jiraSection.ActiveSprint)
	}
}

func TestDashboardService_JiraSummaryWithoutSprintNames(t *testing.T) {
	service := NewDashboardService(
		&fakeJiraProvider{issues: []jiraDomain.JiraIssue{
			{Key: "DOC-1", SubLabel: "DEV Document · SOP"},
			{Key: "BUG-1", SubLabel: "Bug Fix"},
		}},
		&fakeBitbucketProvider{},
		&fakeCopilotProvider{},
		configuredCredentials(),
	)

	jiraSection := service.GetSummary(context.Background(), testUserID).Jira

	if jiraSection.Status != domain.StatusOK {
		t.Errorf("expected status %q, got %q", domain.StatusOK, jiraSection.Status)
	}
	if jiraSection.ActiveSprint != "" {
		t.Errorf("expected empty active sprint, got %q", jiraSection.ActiveSprint)
	}
	if jiraSection.AssignedTickets != 2 {
		t.Errorf("expected 2 assigned tickets, got %d", jiraSection.AssignedTickets)
	}
	if jiraSection.TechnicalDocuments != 1 {
		t.Errorf("expected 1 technical document, got %d", jiraSection.TechnicalDocuments)
	}
}

func TestDashboardService_BitbucketSummaryCounting(t *testing.T) {
	service := NewDashboardService(
		&fakeJiraProvider{},
		&fakeBitbucketProvider{
			openPullRequests: []bitbucketDomain.PullRequest{
				{ID: "#1", Status: "open"},
				{ID: "#2", Status: "open"},
				{ID: "#3", Status: "open"},
				{ID: "#4", Status: "merged"},
			},
			assignedPullRequests: []bitbucketDomain.PullRequest{
				{ID: "#1", Status: "open", IsAssignedToMe: true},
				{ID: "#2", Status: "open", IsAssignedToMe: true},
			},
		},
		&fakeCopilotProvider{},
		configuredCredentials(),
	)

	bitbucketSection := service.GetSummary(context.Background(), testUserID).Bitbucket

	if bitbucketSection.Status != domain.StatusOK {
		t.Errorf("expected status %q, got %q", domain.StatusOK, bitbucketSection.Status)
	}
	if !bitbucketSection.IsConfigured {
		t.Error("expected is_configured true")
	}
	if bitbucketSection.OpenPullRequests != 3 {
		t.Errorf("expected 3 open pull requests, got %d", bitbucketSection.OpenPullRequests)
	}
	if bitbucketSection.ReviewRequested != 2 {
		t.Errorf("expected 2 review requested, got %d", bitbucketSection.ReviewRequested)
	}
}

func TestDashboardService_CopilotSummaryCounting(t *testing.T) {
	models := []copilotDomain.ModelInfo{
		{ID: "g1", Provider: "gemini", IsConfigured: true},
		{ID: "g2", Provider: "gemini", IsConfigured: true},
		{ID: "d1", Provider: "deepseek", IsConfigured: true},
		{ID: "c1", Provider: "claude", IsConfigured: true},
		{ID: "o1", Provider: "openai", IsConfigured: false},
		{ID: "m1", Provider: "mimo", IsConfigured: false},
	}
	service := NewDashboardService(
		&fakeJiraProvider{},
		&fakeBitbucketProvider{},
		&fakeCopilotProvider{models: models},
		configuredCredentials(),
	)

	copilotSection := service.GetSummary(context.Background(), testUserID).Copilot

	if copilotSection.Status != domain.StatusOK {
		t.Errorf("expected status %q, got %q", domain.StatusOK, copilotSection.Status)
	}
	if copilotSection.TotalTools != 5 {
		t.Errorf("expected 5 total tools, got %d", copilotSection.TotalTools)
	}
	if copilotSection.ActiveTools != 3 {
		t.Errorf("expected 3 active tools, got %d", copilotSection.ActiveTools)
	}
	expectedProviders := []string{"claude", "deepseek", "gemini"}
	if len(copilotSection.ConfiguredProviders) != len(expectedProviders) {
		t.Fatalf("expected providers %v, got %v", expectedProviders, copilotSection.ConfiguredProviders)
	}
	for index, provider := range expectedProviders {
		if copilotSection.ConfiguredProviders[index] != provider {
			t.Errorf("expected configured providers %v, got %v", expectedProviders, copilotSection.ConfiguredProviders)
			break
		}
	}
}
