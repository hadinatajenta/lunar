package application

import (
	"context"
	"errors"
	"fmt"
	"testing"

	authDomain "lunar/backend/internal/auth/domain"
	bitbucketDomain "lunar/backend/internal/bitbucket/domain"
	"lunar/backend/internal/dashboard/domain"
	sharedErrors "lunar/backend/internal/shared/errors"
)

func TestDashboardService_JiraUnauthorizedReturnsUnconfiguredStatus(t *testing.T) {
	service := NewDashboardService(
		&fakeJiraProvider{err: fmt.Errorf("%w: Jira PAT is not configured", sharedErrors.ErrUnauthorized)},
		&fakeBitbucketProvider{},
		&fakeCopilotProvider{},
		configuredCredentials(),
	)

	jiraSection := service.GetSummary(context.Background(), testUserID).Jira

	if jiraSection.Status != domain.StatusUnconfigured {
		t.Errorf("expected status %q, got %q", domain.StatusUnconfigured, jiraSection.Status)
	}
	if jiraSection.IsConfigured {
		t.Error("expected is_configured false")
	}
	if jiraSection.AssignedTickets != 0 || jiraSection.TechnicalDocuments != 0 || jiraSection.ActiveSprint != "" {
		t.Errorf("expected zeroed counters, got %+v", jiraSection)
	}
}

func TestDashboardService_JiraGenericErrorReturnsErrorStatus(t *testing.T) {
	service := NewDashboardService(
		&fakeJiraProvider{err: errors.New("unable to reach Jira BRI API")},
		&fakeBitbucketProvider{},
		&fakeCopilotProvider{},
		configuredCredentials(),
	)

	jiraSection := service.GetSummary(context.Background(), testUserID).Jira

	if jiraSection.Status != domain.StatusError {
		t.Errorf("expected status %q, got %q", domain.StatusError, jiraSection.Status)
	}
	if !jiraSection.IsConfigured {
		t.Error("expected is_configured true after PAT resolution")
	}
	if jiraSection.AssignedTickets != 0 || jiraSection.TechnicalDocuments != 0 || jiraSection.ActiveSprint != "" {
		t.Errorf("expected zeroed counters, got %+v", jiraSection)
	}
}

func TestDashboardService_BitbucketOpenPullRequestFailureReturnsErrorStatus(t *testing.T) {
	service := NewDashboardService(
		&fakeJiraProvider{},
		&fakeBitbucketProvider{err: errors.New("Bitbucket VPN unreachable")},
		&fakeCopilotProvider{},
		configuredCredentials(),
	)

	bitbucketSection := service.GetSummary(context.Background(), testUserID).Bitbucket

	if bitbucketSection.Status != domain.StatusError {
		t.Errorf("expected status %q, got %q", domain.StatusError, bitbucketSection.Status)
	}
	if !bitbucketSection.IsConfigured {
		t.Error("expected is_configured true")
	}
	if bitbucketSection.OpenPullRequests != 0 || bitbucketSection.ReviewRequested != 0 {
		t.Errorf("expected zeroed counters, got %+v", bitbucketSection)
	}
}

func TestDashboardService_BitbucketAssignedPullRequestFailureReturnsErrorStatus(t *testing.T) {
	service := NewDashboardService(
		&fakeJiraProvider{},
		&fakeBitbucketProvider{
			openPullRequests:     []bitbucketDomain.PullRequest{{ID: "#1", Status: "open"}},
			assignedPullRequests: []bitbucketDomain.PullRequest{{ID: "#1", Status: "open", IsAssignedToMe: true}},
			assignedErr:          errors.New("reviewer query failed"),
		},
		&fakeCopilotProvider{},
		configuredCredentials(),
	)

	bitbucketSection := service.GetSummary(context.Background(), testUserID).Bitbucket

	if bitbucketSection.Status != domain.StatusError {
		t.Errorf("expected status %q, got %q", domain.StatusError, bitbucketSection.Status)
	}
	if bitbucketSection.OpenPullRequests != 0 || bitbucketSection.ReviewRequested != 0 {
		t.Errorf("expected zeroed counters, got %+v", bitbucketSection)
	}
}

func TestDashboardService_BitbucketCredentialFailureReturnsUnconfiguredStatus(t *testing.T) {
	service := NewDashboardService(
		&fakeJiraProvider{},
		&fakeBitbucketProvider{
			openPullRequests: []bitbucketDomain.PullRequest{
				{ID: "#1", Status: "open"},
				{ID: "#2", Status: "open"},
			},
			assignedPullRequests: []bitbucketDomain.PullRequest{
				{ID: "#1", Status: "open", IsAssignedToMe: true},
			},
		},
		&fakeCopilotProvider{},
		&fakeCredentialProvider{err: errors.New("vault unavailable")},
	)

	bitbucketSection := service.GetSummary(context.Background(), testUserID).Bitbucket

	if bitbucketSection.Status != domain.StatusUnconfigured {
		t.Errorf("expected status %q, got %q", domain.StatusUnconfigured, bitbucketSection.Status)
	}
	if bitbucketSection.IsConfigured {
		t.Error("expected is_configured false")
	}
	if bitbucketSection.OpenPullRequests != 2 || bitbucketSection.ReviewRequested != 1 {
		t.Errorf("expected counters 2 and 1, got %+v", bitbucketSection)
	}
}

func TestDashboardService_BitbucketWithoutPATReturnsUnconfiguredStatus(t *testing.T) {
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
		&fakeCredentialProvider{secrets: &authDomain.RedactedSecrets{UserID: testUserID}},
	)

	bitbucketSection := service.GetSummary(context.Background(), testUserID).Bitbucket

	if bitbucketSection.Status != domain.StatusUnconfigured {
		t.Errorf("expected status %q, got %q", domain.StatusUnconfigured, bitbucketSection.Status)
	}
	if bitbucketSection.IsConfigured {
		t.Error("expected is_configured false")
	}
	if bitbucketSection.OpenPullRequests != 3 {
		t.Errorf("expected 3 open pull requests, got %d", bitbucketSection.OpenPullRequests)
	}
	if bitbucketSection.ReviewRequested != 2 {
		t.Errorf("expected 2 review requested, got %d", bitbucketSection.ReviewRequested)
	}
}

func TestDashboardService_CopilotProviderFailureReturnsErrorStatus(t *testing.T) {
	service := NewDashboardService(
		&fakeJiraProvider{},
		&fakeBitbucketProvider{},
		&fakeCopilotProvider{err: errors.New("model catalog unavailable")},
		configuredCredentials(),
	)

	copilotSection := service.GetSummary(context.Background(), testUserID).Copilot

	if copilotSection.Status != domain.StatusError {
		t.Errorf("expected status %q, got %q", domain.StatusError, copilotSection.Status)
	}
	if copilotSection.ActiveTools != 0 || copilotSection.TotalTools != 0 {
		t.Errorf("expected zeroed counters, got %+v", copilotSection)
	}
	if copilotSection.ConfiguredProviders == nil {
		t.Error("expected non-nil configured providers slice")
	}
	if len(copilotSection.ConfiguredProviders) != 0 {
		t.Errorf("expected empty configured providers, got %v", copilotSection.ConfiguredProviders)
	}
}
