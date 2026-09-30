package transport

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	authApp "lunar/backend/internal/auth/application"
	authDomain "lunar/backend/internal/auth/domain"
	jiraApp "lunar/backend/internal/jira/application"
	"lunar/backend/internal/jira/domain"
	sharedAuth "lunar/backend/internal/shared/auth"
	"lunar/backend/internal/shared/crypto"
	sharedErrors "lunar/backend/internal/shared/errors"
)

type mockVaultRepo struct {
	secrets map[string]*authDomain.UserSecrets
}

func (m *mockVaultRepo) SaveSecrets(ctx context.Context, s *authDomain.UserSecrets) error {
	m.secrets[s.UserID] = s
	return nil
}

func (m *mockVaultRepo) GetSecretsByUserID(ctx context.Context, userID string) (*authDomain.UserSecrets, error) {
	s, ok := m.secrets[userID]
	if !ok || s == nil {
		return nil, sharedErrors.ErrNotFound
	}
	return s, nil
}

type mockJiraRepo struct {
	issues      []domain.JiraIssue
	sprints     []domain.JiraSprint
	detail      *domain.JiraIssue
	err         error
	detailErr   error
	searchCalls atomic.Int64
	sprintCalls atomic.Int64
}

func (m *mockJiraRepo) SearchMyIssues(ctx context.Context, pat string) ([]domain.JiraIssue, error) {
	m.searchCalls.Add(1)
	if m.err != nil {
		return nil, m.err
	}
	return m.issues, nil
}

func (m *mockJiraRepo) GetBacklogSprints(ctx context.Context, pat string) ([]domain.JiraSprint, error) {
	m.sprintCalls.Add(1)
	if m.err != nil {
		return nil, m.err
	}
	return m.sprints, nil
}

func (m *mockJiraRepo) GetIssueDetail(ctx context.Context, pat string, issueKey string) (*domain.JiraIssue, error) {
	if m.detailErr != nil {
		return nil, m.detailErr
	}
	if m.err != nil {
		return nil, m.err
	}
	if m.detail != nil && m.detail.Key == issueKey {
		return m.detail, nil
	}
	return nil, fmt.Errorf("%w: issue %s not found", sharedErrors.ErrNotFound, issueKey)
}

func (m *mockJiraRepo) VerifyPAT(ctx context.Context, pat string) (*domain.JiraUser, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &domain.JiraUser{
		DisplayName:  "Lunar Developer (BRI MMS)",
		EmailAddress: "developer@lunar.dev",
		Name:         "developer",
	}, nil
}

func setupTestHandler(repo domain.JiraRepository) (*JiraHandler, string, string) {
	encKey := "0123456789abcdef0123456789abcdef"
	vaultRepo := &mockVaultRepo{
		secrets: make(map[string]*authDomain.UserSecrets),
	}

	validPatEnc, _ := crypto.EncryptSecret("valid-pat", encKey)
	configuredUser := "usr-configured"
	unconfiguredUser := "usr-unconfigured"

	vaultRepo.secrets[configuredUser] = &authDomain.UserSecrets{
		UserID:       configuredUser,
		JiraPATEnc:   validPatEnc,
		JiraUsername: "developer",
		UpdatedAt:    time.Now().UTC(),
	}

	authService := authApp.NewAuthService(nil, vaultRepo, encKey, "jwt-secret", time.Hour)
	jiraService := jiraApp.NewJiraService(repo, authService)
	return NewJiraHandler(jiraService), configuredUser, unconfiguredUser
}

func TestJiraHandler_ListMyIssues(t *testing.T) {
	repo := &mockJiraRepo{
		issues: []domain.JiraIssue{
			{
				ID:   "10101",
				Key:  "CRMMS-77911",
				Kind: "story",
				Fields: domain.JiraIssueFields{
					Summary: "Merchant Onboarding Multi-Tier Verification Service",
				},
			},
		},
	}

	handler, configuredUser, unconfiguredUser := setupTestHandler(repo)

	t.Run("Success returns 200 with issues", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/jira/issues", nil)
		ctx := sharedAuth.ContextWithUser(req.Context(), &sharedAuth.UserContext{
			UserID: configuredUser,
			Email:  "developer@bri.co.id",
		})
		rec := httptest.NewRecorder()

		handler.ListMyIssues(rec, req.WithContext(ctx))

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}

		var issues []domain.JiraIssue
		if err := json.NewDecoder(rec.Body).Decode(&issues); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if len(issues) != 1 || issues[0].Key != "CRMMS-77911" {
			t.Errorf("unexpected issues: %v", issues)
		}
	})

	t.Run("Unconfigured PAT returns 401", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/jira/issues", nil)
		ctx := sharedAuth.ContextWithUser(req.Context(), &sharedAuth.UserContext{
			UserID: unconfiguredUser,
			Email:  "developer@bri.co.id",
		})
		rec := httptest.NewRecorder()

		handler.ListMyIssues(rec, req.WithContext(ctx))

		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", rec.Code)
		}
	})

	t.Run("VPN network error returns 502", func(t *testing.T) {
		vpnRepo := &mockJiraRepo{
			err: errors.New("unable to reach Jira BRI API. Please check your VPN connection."),
		}
		vpnHandler, user, _ := setupTestHandler(vpnRepo)

		req := httptest.NewRequest(http.MethodGet, "/api/jira/issues", nil)
		ctx := sharedAuth.ContextWithUser(req.Context(), &sharedAuth.UserContext{
			UserID: user,
			Email:  "developer@bri.co.id",
		})
		rec := httptest.NewRecorder()

		vpnHandler.ListMyIssues(rec, req.WithContext(ctx))

		if rec.Code != http.StatusBadGateway {
			t.Fatalf("expected 502, got %d", rec.Code)
		}
	})
}

func TestJiraHandler_GetBacklog(t *testing.T) {
	repo := &mockJiraRepo{
		sprints: []domain.JiraSprint{
			{
				ID:    44,
				Name:  "Sprint 44 (MMS Modernization)",
				State: "active",
				Issues: []domain.JiraIssue{
					{ID: "10101", Key: "CRMMS-77911"},
				},
			},
		},
	}

	handler, configuredUser, _ := setupTestHandler(repo)

	req := httptest.NewRequest(http.MethodGet, "/api/jira/backlog", nil)
	ctx := sharedAuth.ContextWithUser(req.Context(), &sharedAuth.UserContext{
		UserID: configuredUser,
		Email:  "developer@bri.co.id",
	})
	rec := httptest.NewRecorder()

	handler.GetBacklog(rec, req.WithContext(ctx))

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var backlog domain.JiraBacklogResponse
	if err := json.NewDecoder(rec.Body).Decode(&backlog); err != nil {
		t.Fatalf("failed to decode backlog: %v", err)
	}
	if len(backlog.Sprints) != 1 || backlog.TotalIssues != 1 {
		t.Errorf("unexpected backlog response: %+v", backlog)
	}
}

func TestJiraHandler_GetIssueDetail(t *testing.T) {
	sampleIssue := domain.JiraIssue{
		ID:   "10101",
		Key:  "CRMMS-77911",
		Kind: "story",
		Fields: domain.JiraIssueFields{
			Summary: "Merchant Onboarding Multi-Tier Verification Service",
		},
	}
	repo := &mockJiraRepo{
		detail: &sampleIssue,
	}

	handler, configuredUser, _ := setupTestHandler(repo)

	t.Run("Success returns 200 with issue", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/jira/issues/CRMMS-77911", nil)
		req.SetPathValue("key", "CRMMS-77911")
		ctx := sharedAuth.ContextWithUser(req.Context(), &sharedAuth.UserContext{
			UserID: configuredUser,
			Email:  "developer@bri.co.id",
		})
		rec := httptest.NewRecorder()

		handler.GetIssueDetail(rec, req.WithContext(ctx))

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}

		var issue domain.JiraIssue
		if err := json.NewDecoder(rec.Body).Decode(&issue); err != nil {
			t.Fatalf("failed to decode issue: %v", err)
		}
		if issue.Key != "CRMMS-77911" {
			t.Errorf("expected CRMMS-77911, got %s", issue.Key)
		}
	})

	t.Run("Issue not found returns 404", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/jira/issues/NONEXISTENT", nil)
		req.SetPathValue("key", "NONEXISTENT")
		ctx := sharedAuth.ContextWithUser(req.Context(), &sharedAuth.UserContext{
			UserID: configuredUser,
			Email:  "developer@bri.co.id",
		})
		rec := httptest.NewRecorder()

		handler.GetIssueDetail(rec, req.WithContext(ctx))

		if rec.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", rec.Code)
		}
	})
}

func performAuthenticatedRequest(t *testing.T, handlerFunc http.HandlerFunc, userID string, target string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, target, nil)
	if userID != "" {
		ctx := sharedAuth.ContextWithUser(req.Context(), &sharedAuth.UserContext{
			UserID: userID,
			Email:  "developer@bri.co.id",
		})
		req = req.WithContext(ctx)
	}
	rec := httptest.NewRecorder()
	handlerFunc(rec, req)
	return rec
}

func decodeIssueKeys(t *testing.T, rec *httptest.ResponseRecorder) []string {
	t.Helper()
	var issues []domain.JiraIssue
	if err := json.NewDecoder(rec.Body).Decode(&issues); err != nil {
		t.Fatalf("failed to decode issues response: %v", err)
	}
	issueKeys := make([]string, 0, len(issues))
	for _, issue := range issues {
		issueKeys = append(issueKeys, issue.Key)
	}
	return issueKeys
}

func TestJiraHandler_RefreshParamBypassesCache(t *testing.T) {
	repo := &mockJiraRepo{
		issues: []domain.JiraIssue{
			{
				ID:   "10101",
				Key:  "CRMMS-77911",
				Kind: "story",
				Fields: domain.JiraIssueFields{
					Summary: "Merchant Onboarding Multi-Tier Verification Service",
				},
			},
		},
		sprints: []domain.JiraSprint{
			{
				ID:    44,
				Name:  "Sprint 44 (MMS Modernization)",
				State: "active",
				Issues: []domain.JiraIssue{
					{ID: "10101", Key: "CRMMS-77911"},
				},
			},
		},
	}

	handler, configuredUser, _ := setupTestHandler(repo)

	t.Run("plain repeat served from cache and refresh=1 recomputes", func(t *testing.T) {
		firstRec := performAuthenticatedRequest(t, handler.ListMyIssues, configuredUser, "/api/jira/issues")
		if firstRec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", firstRec.Code)
		}
		if issueKeys := decodeIssueKeys(t, firstRec); len(issueKeys) != 1 || issueKeys[0] != "CRMMS-77911" {
			t.Fatalf("expected CRMMS-77911, got %v", issueKeys)
		}
		if calls := repo.searchCalls.Load(); calls != 1 {
			t.Fatalf("expected 1 repo hit after first request, got %d", calls)
		}

		secondRec := performAuthenticatedRequest(t, handler.ListMyIssues, configuredUser, "/api/jira/issues")
		if secondRec.Code != http.StatusOK {
			t.Fatalf("expected 200 on plain repeat, got %d", secondRec.Code)
		}
		if issueKeys := decodeIssueKeys(t, secondRec); len(issueKeys) != 1 || issueKeys[0] != "CRMMS-77911" {
			t.Fatalf("expected CRMMS-77911 from cache, got %v", issueKeys)
		}
		if calls := repo.searchCalls.Load(); calls != 1 {
			t.Errorf("expected plain repeat to be served from cache, got %d repo hits", calls)
		}

		refreshRec := performAuthenticatedRequest(t, handler.ListMyIssues, configuredUser, "/api/jira/issues?refresh=1")
		if refreshRec.Code != http.StatusOK {
			t.Fatalf("expected 200 with refresh=1, got %d", refreshRec.Code)
		}
		if issueKeys := decodeIssueKeys(t, refreshRec); len(issueKeys) != 1 || issueKeys[0] != "CRMMS-77911" {
			t.Fatalf("expected CRMMS-77911 after refresh, got %v", issueKeys)
		}
		if calls := repo.searchCalls.Load(); calls != 2 {
			t.Errorf("expected refresh=1 to reach the repo, got %d hits", calls)
		}

		thirdRec := performAuthenticatedRequest(t, handler.ListMyIssues, configuredUser, "/api/jira/issues")
		if thirdRec.Code != http.StatusOK {
			t.Fatalf("expected 200 after refresh, got %d", thirdRec.Code)
		}
		if calls := repo.searchCalls.Load(); calls != 2 {
			t.Errorf("expected refreshed result to be cached for subsequent plain requests, got %d hits", calls)
		}
	})

	t.Run("backlog refresh=1 recomputes while plain repeat served from cache", func(t *testing.T) {
		firstRec := performAuthenticatedRequest(t, handler.GetBacklog, configuredUser, "/api/jira/backlog")
		if firstRec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", firstRec.Code)
		}
		if calls := repo.sprintCalls.Load(); calls != 1 {
			t.Fatalf("expected 1 sprint repo hit after first backlog request, got %d", calls)
		}

		secondRec := performAuthenticatedRequest(t, handler.GetBacklog, configuredUser, "/api/jira/backlog")
		if secondRec.Code != http.StatusOK {
			t.Fatalf("expected 200 on plain repeat, got %d", secondRec.Code)
		}
		if calls := repo.sprintCalls.Load(); calls != 1 {
			t.Errorf("expected plain backlog repeat to be served from cache, got %d repo hits", calls)
		}

		refreshRec := performAuthenticatedRequest(t, handler.GetBacklog, configuredUser, "/api/jira/backlog?refresh=1")
		if refreshRec.Code != http.StatusOK {
			t.Fatalf("expected 200 with refresh=1, got %d", refreshRec.Code)
		}
		var backlog domain.JiraBacklogResponse
		if err := json.NewDecoder(refreshRec.Body).Decode(&backlog); err != nil {
			t.Fatalf("failed to decode backlog: %v", err)
		}
		if backlog.TotalIssues != 1 {
			t.Errorf("expected TotalIssues=1 after refresh, got %d", backlog.TotalIssues)
		}
		if calls := repo.sprintCalls.Load(); calls != 2 {
			t.Errorf("expected backlog refresh=1 to reach the repo, got %d hits", calls)
		}
	})

	t.Run("missing auth context returns 401 without repo hit", func(t *testing.T) {
		searchCallsBefore := repo.searchCalls.Load()

		rec := performAuthenticatedRequest(t, handler.ListMyIssues, "", "/api/jira/issues")
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", rec.Code)
		}
		if calls := repo.searchCalls.Load(); calls != searchCallsBefore {
			t.Errorf("expected no repo hit for unauthenticated request, got %d total hits", calls)
		}
	})
}
