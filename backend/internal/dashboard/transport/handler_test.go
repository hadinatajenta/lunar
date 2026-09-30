package transport

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	authDomain "lunar/backend/internal/auth/domain"
	bitbucketDomain "lunar/backend/internal/bitbucket/domain"
	copilotDomain "lunar/backend/internal/copilot/domain"
	"lunar/backend/internal/dashboard/application"
	"lunar/backend/internal/dashboard/domain"
	jiraDomain "lunar/backend/internal/jira/domain"
	sharedAuth "lunar/backend/internal/shared/auth"
	sharedErrors "lunar/backend/internal/shared/errors"
	sharedHttp "lunar/backend/internal/shared/http"
)

const testJWTSecret = "dashboard-jwt-secret"

type fakeJiraProvider struct {
	issues []jiraDomain.JiraIssue
	err    error
}

func (f *fakeJiraProvider) GetMyIssues(ctx context.Context, userID string) ([]jiraDomain.JiraIssue, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.issues, nil
}

type fakeBitbucketProvider struct {
	pullRequests []bitbucketDomain.PullRequest
	err          error
}

func (f *fakeBitbucketProvider) ListPullRequests(ctx context.Context, userID string, filter string) ([]bitbucketDomain.PullRequest, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.pullRequests, nil
}

type fakeCopilotProvider struct {
	models []copilotDomain.ModelInfo
	err    error
}

func (f *fakeCopilotProvider) ListModels(ctx context.Context, userID string) ([]copilotDomain.ModelInfo, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.models, nil
}

type fakeCredentialProvider struct {
	secrets *authDomain.RedactedSecrets
	err     error
}

func (f *fakeCredentialProvider) GetRedactedSecrets(ctx context.Context, userID string) (*authDomain.RedactedSecrets, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.secrets, nil
}

func newTestHandler(jira application.JiraSummaryProvider, bitbucket application.BitbucketSummaryProvider, copilot application.CopilotSummaryProvider, credentials application.CredentialSummaryProvider) *DashboardHandler {
	return NewDashboardHandler(application.NewDashboardService(jira, bitbucket, copilot, credentials))
}

func serveSummary(handler *DashboardHandler, request *http.Request) *httptest.ResponseRecorder {
	recorder := httptest.NewRecorder()
	sharedAuth.RequireAuth(testJWTSecret)(http.HandlerFunc(handler.GetSummary)).ServeHTTP(recorder, request)
	return recorder
}

func newAuthorizedRequest(t *testing.T) *http.Request {
	t.Helper()
	token, err := sharedAuth.GenerateToken("usr-dashboard", "developer@lunar.dev", testJWTSecret, time.Hour)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}
	request := httptest.NewRequest(http.MethodGet, "/api/dashboard/summary", nil)
	request.Header.Set("Authorization", "Bearer "+token)
	return request
}

func decodeSummary(t *testing.T, recorder *httptest.ResponseRecorder) domain.Summary {
	t.Helper()
	var summary domain.Summary
	if err := json.NewDecoder(recorder.Body).Decode(&summary); err != nil {
		t.Fatalf("failed to decode summary response: %v", err)
	}
	return summary
}

func decodeErrorResponse(t *testing.T, recorder *httptest.ResponseRecorder) sharedHttp.ErrorResponse {
	t.Helper()
	var errorResponse sharedHttp.ErrorResponse
	if err := json.NewDecoder(recorder.Body).Decode(&errorResponse); err != nil {
		t.Fatalf("failed to decode error response: %v", err)
	}
	return errorResponse
}

func dashboardPullRequests() []bitbucketDomain.PullRequest {
	return []bitbucketDomain.PullRequest{
		{ID: "#1", Status: "open", IsAssignedToMe: true},
		{ID: "#2", Status: "open", IsAssignedToMe: true},
		{ID: "#3", Status: "open", IsAssignedToMe: false},
		{ID: "#4", Status: "merged", IsAssignedToMe: false},
	}
}

func dashboardModels() []copilotDomain.ModelInfo {
	return []copilotDomain.ModelInfo{
		{ID: "m1", Provider: "deepseek", IsConfigured: true},
		{ID: "m2", Provider: "gemini", IsConfigured: true},
		{ID: "m3", Provider: "openai", IsConfigured: false},
	}
}

func healthyBitbucketProvider() *fakeBitbucketProvider {
	return &fakeBitbucketProvider{pullRequests: dashboardPullRequests()}
}

func healthyCopilotProvider() *fakeCopilotProvider {
	return &fakeCopilotProvider{models: dashboardModels()}
}

func configuredCredentials() *fakeCredentialProvider {
	return &fakeCredentialProvider{secrets: &authDomain.RedactedSecrets{HasBitbucketPAT: true}}
}

func TestDashboardHandler_MissingAuthorizationHeaderReturns401(t *testing.T) {
	handler := newTestHandler(&fakeJiraProvider{}, healthyBitbucketProvider(), healthyCopilotProvider(), configuredCredentials())
	request := httptest.NewRequest(http.MethodGet, "/api/dashboard/summary", nil)

	recorder := serveSummary(handler, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", recorder.Code)
	}
	errorResponse := decodeErrorResponse(t, recorder)
	if errorResponse.Error != "missing authorization header" {
		t.Errorf("expected error %q, got %q", "missing authorization header", errorResponse.Error)
	}
}

func TestDashboardHandler_InvalidBearerTokenReturns401(t *testing.T) {
	handler := newTestHandler(&fakeJiraProvider{}, healthyBitbucketProvider(), healthyCopilotProvider(), configuredCredentials())

	t.Run("garbage bearer token", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodGet, "/api/dashboard/summary", nil)
		request.Header.Set("Authorization", "Bearer garbage-token")

		recorder := serveSummary(handler, request)

		if recorder.Code != http.StatusUnauthorized {
			t.Fatalf("expected status 401, got %d", recorder.Code)
		}
		errorResponse := decodeErrorResponse(t, recorder)
		if errorResponse.Error != "invalid or expired token" {
			t.Errorf("expected error %q, got %q", "invalid or expired token", errorResponse.Error)
		}
	})

	t.Run("malformed authorization header", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodGet, "/api/dashboard/summary", nil)
		request.Header.Set("Authorization", "Token abc.def.ghi")

		recorder := serveSummary(handler, request)

		if recorder.Code != http.StatusUnauthorized {
			t.Fatalf("expected status 401, got %d", recorder.Code)
		}
		errorResponse := decodeErrorResponse(t, recorder)
		if errorResponse.Error != "invalid authorization header format" {
			t.Errorf("expected error %q, got %q", "invalid authorization header format", errorResponse.Error)
		}
	})
}

func TestDashboardHandler_JiraUnauthorizedReturnsUnconfiguredSection(t *testing.T) {
	handler := newTestHandler(
		&fakeJiraProvider{err: fmt.Errorf("%w: Jira PAT is not configured", sharedErrors.ErrUnauthorized)},
		healthyBitbucketProvider(),
		healthyCopilotProvider(),
		configuredCredentials(),
	)

	recorder := serveSummary(handler, newAuthorizedRequest(t))

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}
	summary := decodeSummary(t, recorder)
	if summary.Jira.Status != domain.StatusUnconfigured {
		t.Errorf("expected jira status %q, got %q", domain.StatusUnconfigured, summary.Jira.Status)
	}
	if summary.Jira.IsConfigured {
		t.Error("expected jira is_configured false")
	}
	if summary.Jira.AssignedTickets != 0 || summary.Jira.TechnicalDocuments != 0 || summary.Jira.ActiveSprint != "" {
		t.Errorf("expected zeroed jira counters, got %+v", summary.Jira)
	}
	if summary.Bitbucket.Status != domain.StatusOK || summary.Bitbucket.OpenPullRequests != 3 || summary.Bitbucket.ReviewRequested != 2 {
		t.Errorf("unexpected bitbucket section: %+v", summary.Bitbucket)
	}
	if summary.Copilot.Status != domain.StatusOK || summary.Copilot.TotalTools != 3 || summary.Copilot.ActiveTools != 2 {
		t.Errorf("unexpected copilot section: %+v", summary.Copilot)
	}
}

func TestDashboardHandler_ProviderFailureKeepsOtherSectionsHealthy(t *testing.T) {
	handler := newTestHandler(
		&fakeJiraProvider{err: errors.New("unable to reach Jira BRI API")},
		healthyBitbucketProvider(),
		healthyCopilotProvider(),
		configuredCredentials(),
	)

	recorder := serveSummary(handler, newAuthorizedRequest(t))

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}
	summary := decodeSummary(t, recorder)
	if summary.Jira.Status != domain.StatusError {
		t.Errorf("expected jira status %q, got %q", domain.StatusError, summary.Jira.Status)
	}
	if summary.Jira.AssignedTickets != 0 || summary.Jira.TechnicalDocuments != 0 {
		t.Errorf("expected zeroed jira counters, got %+v", summary.Jira)
	}
	if summary.Bitbucket.Status != domain.StatusOK || !summary.Bitbucket.IsConfigured {
		t.Errorf("expected healthy bitbucket section, got %+v", summary.Bitbucket)
	}
	if summary.Bitbucket.OpenPullRequests != 3 || summary.Bitbucket.ReviewRequested != 2 {
		t.Errorf("unexpected bitbucket counters: %+v", summary.Bitbucket)
	}
	if summary.Copilot.Status != domain.StatusOK || summary.Copilot.TotalTools != 3 || summary.Copilot.ActiveTools != 2 {
		t.Errorf("unexpected copilot section: %+v", summary.Copilot)
	}
	expectedProviders := []string{"deepseek", "gemini"}
	for index, provider := range expectedProviders {
		if summary.Copilot.ConfiguredProviders[index] != provider {
			t.Fatalf("expected configured providers %v, got %v", expectedProviders, summary.Copilot.ConfiguredProviders)
		}
	}
}

func TestDashboardHandler_AllProvidersSucceedReturnsFullSummary(t *testing.T) {
	issues := []jiraDomain.JiraIssue{
		{Key: "DEV-1", SprintName: "Sprint 45 - Fortune Squad", SubLabel: "DEV Document · UT"},
		{Key: "DEV-2", SprintName: "Sprint 45 - Fortune Squad", SubLabel: "Bug Fix"},
		{Key: "DEV-3", SubLabel: "DEV Document"},
	}
	handler := newTestHandler(
		&fakeJiraProvider{issues: issues},
		healthyBitbucketProvider(),
		healthyCopilotProvider(),
		configuredCredentials(),
	)

	recorder := serveSummary(handler, newAuthorizedRequest(t))

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}
	summary := decodeSummary(t, recorder)
	if _, err := time.Parse(time.RFC3339, summary.SyncedAt); err != nil {
		t.Errorf("expected synced_at in RFC3339 format, got %q: %v", summary.SyncedAt, err)
	}
	if summary.Jira.Status != domain.StatusOK || !summary.Jira.IsConfigured {
		t.Errorf("unexpected jira section: %+v", summary.Jira)
	}
	if summary.Jira.AssignedTickets != 3 || summary.Jira.TechnicalDocuments != 2 {
		t.Errorf("unexpected jira counters: %+v", summary.Jira)
	}
	if summary.Jira.ActiveSprint != "Sprint 45 - Fortune Squad" {
		t.Errorf("expected active sprint %q, got %q", "Sprint 45 - Fortune Squad", summary.Jira.ActiveSprint)
	}
	if summary.Bitbucket.OpenPullRequests != 3 || summary.Bitbucket.ReviewRequested != 2 {
		t.Errorf("unexpected bitbucket counters: %+v", summary.Bitbucket)
	}
	if summary.Copilot.TotalTools != 3 || summary.Copilot.ActiveTools != 2 {
		t.Errorf("unexpected copilot counters: %+v", summary.Copilot)
	}
}
