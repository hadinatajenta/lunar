package application

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	authApp "lunar/backend/internal/auth/application"
	authDomain "lunar/backend/internal/auth/domain"
	"lunar/backend/internal/jira/domain"
	"lunar/backend/internal/shared/crypto"
	sharedErrors "lunar/backend/internal/shared/errors"
)

const secondaryConfiguredUserID = "usr-configured-secondary"

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
	if pat == "invalid" {
		return nil, fmt.Errorf("%w: invalid or expired Jira PAT", sharedErrors.ErrUnauthorized)
	}
	if m.err != nil {
		return nil, m.err
	}
	return m.issues, nil
}

func (m *mockJiraRepo) GetBacklogSprints(ctx context.Context, pat string) ([]domain.JiraSprint, error) {
	m.sprintCalls.Add(1)
	if pat == "invalid" {
		return nil, fmt.Errorf("%w: invalid or expired Jira PAT", sharedErrors.ErrUnauthorized)
	}
	if m.err != nil {
		return nil, m.err
	}
	return m.sprints, nil
}

func (m *mockJiraRepo) GetIssueDetail(ctx context.Context, pat string, issueKey string) (*domain.JiraIssue, error) {
	if pat == "invalid" {
		return nil, fmt.Errorf("%w: invalid or expired Jira PAT", sharedErrors.ErrUnauthorized)
	}
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
	if pat == "invalid" {
		return nil, fmt.Errorf("%w: invalid or expired Jira PAT", sharedErrors.ErrUnauthorized)
	}
	if m.err != nil {
		return nil, m.err
	}
	return &domain.JiraUser{
		DisplayName:  "Lunar Developer (BRI MMS)",
		EmailAddress: "developer@lunar.dev",
		Name:         "developer",
	}, nil
}

func setupTestService(repo domain.JiraRepository) (*JiraService, string, string) {
	encKey := "0123456789abcdef0123456789abcdef"
	vaultRepo := &mockVaultRepo{
		secrets: make(map[string]*authDomain.UserSecrets),
	}

	validPatEnc, _ := crypto.EncryptSecret("valid-pat-token", encKey)
	invalidPatEnc, _ := crypto.EncryptSecret("invalid", encKey)

	configuredUserID := "usr-configured"
	invalidUserID := "usr-invalid"

	vaultRepo.secrets[configuredUserID] = &authDomain.UserSecrets{
		UserID:       configuredUserID,
		JiraPATEnc:   validPatEnc,
		JiraUsername: "developer",
		UpdatedAt:    time.Now().UTC(),
	}

	vaultRepo.secrets[invalidUserID] = &authDomain.UserSecrets{
		UserID:       invalidUserID,
		JiraPATEnc:   invalidPatEnc,
		JiraUsername: "developer",
		UpdatedAt:    time.Now().UTC(),
	}

	vaultRepo.secrets[secondaryConfiguredUserID] = &authDomain.UserSecrets{
		UserID:       secondaryConfiguredUserID,
		JiraPATEnc:   validPatEnc,
		JiraUsername: "developer",
		UpdatedAt:    time.Now().UTC(),
	}

	authService := authApp.NewAuthService(nil, vaultRepo, encKey, "jwt-secret", time.Hour)
	return NewJiraService(repo, authService), configuredUserID, invalidUserID
}

func cacheTestIssues() []domain.JiraIssue {
	return []domain.JiraIssue{
		{
			ID:   "10101",
			Key:  "CRMMS-77911",
			Kind: "story",
			Fields: domain.JiraIssueFields{
				Summary: "Merchant Onboarding Multi-Tier Verification Service",
			},
		},
	}
}

func cacheTestSprints() []domain.JiraSprint {
	return []domain.JiraSprint{
		{
			ID:     44,
			Name:   "Sprint 44 (MMS Modernization)",
			State:  "active",
			Issues: cacheTestIssues(),
		},
	}
}

func TestJiraService_ErrorIsNotCached(t *testing.T) {
	upstreamErr := errors.New("upstream unavailable")
	repo := &mockJiraRepo{
		issues: cacheTestIssues(),
		err:    upstreamErr,
	}
	service, configuredUserID, _ := setupTestService(repo)
	ctx := context.Background()

	if _, err := service.GetMyIssues(ctx, configuredUserID); !errors.Is(err, upstreamErr) {
		t.Fatalf("expected upstream error on first call, got %v", err)
	}
	if _, err := service.GetMyIssues(ctx, configuredUserID); !errors.Is(err, upstreamErr) {
		t.Fatalf("expected upstream error on second call, got %v", err)
	}
	if calls := repo.searchCalls.Load(); calls != 2 {
		t.Errorf("expected failing calls to always reach the repo, got %d hits", calls)
	}
}

func TestJiraService_ErrorIsFollowedByCachedSuccess(t *testing.T) {
	repo := &mockJiraRepo{
		issues: cacheTestIssues(),
		err:    errors.New("upstream unavailable"),
	}
	service, configuredUserID, _ := setupTestService(repo)
	ctx := context.Background()

	if _, err := service.GetMyIssues(ctx, configuredUserID); err == nil {
		t.Fatal("expected error on first call, got nil")
	}

	repo.err = nil

	issues, err := service.GetMyIssues(ctx, configuredUserID)
	if err != nil {
		t.Fatalf("expected success after recovery, got %v", err)
	}
	if len(issues) != 1 || issues[0].Key != "CRMMS-77911" {
		t.Fatalf("expected CRMMS-77911 after recovery, got %+v", issues)
	}

	cachedIssues, err := service.GetMyIssues(ctx, configuredUserID)
	if err != nil {
		t.Fatalf("expected cached success on third call, got %v", err)
	}
	if len(cachedIssues) != 1 || cachedIssues[0].Key != issues[0].Key {
		t.Errorf("expected cached result to match fresh result, got %+v", cachedIssues)
	}
	if calls := repo.searchCalls.Load(); calls != 2 {
		t.Errorf("expected 2 repo hits (error then cached success), got %d", calls)
	}
}

func TestJiraService_ForceRefreshBypassesCache(t *testing.T) {
	repo := &mockJiraRepo{
		issues:  cacheTestIssues(),
		sprints: cacheTestSprints(),
	}
	service, configuredUserID, _ := setupTestService(repo)
	ctx := context.Background()

	t.Run("GetMyIssues", func(t *testing.T) {
		freshIssues, err := service.GetMyIssues(ctx, configuredUserID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if _, err := service.GetMyIssues(ctx, configuredUserID); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if calls := repo.searchCalls.Load(); calls != 1 {
			t.Fatalf("expected repeat call to be served from cache with 1 hit, got %d", calls)
		}

		refreshedIssues, err := service.GetMyIssues(WithForceRefresh(ctx), configuredUserID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if calls := repo.searchCalls.Load(); calls != 2 {
			t.Errorf("expected force refresh to reach the repo, got %d hits", calls)
		}
		if len(refreshedIssues) != 1 || refreshedIssues[0].Key != freshIssues[0].Key {
			t.Errorf("expected refreshed result %v, got %+v", freshIssues, refreshedIssues)
		}
	})

	t.Run("GetBacklog", func(t *testing.T) {
		if _, err := service.GetBacklog(ctx, configuredUserID, "all"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if _, err := service.GetBacklog(ctx, configuredUserID, "all"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if calls := repo.sprintCalls.Load(); calls != 1 {
			t.Fatalf("expected repeat backlog call to be served from cache with 1 hit, got %d", calls)
		}

		refreshedBacklog, err := service.GetBacklog(WithForceRefresh(ctx), configuredUserID, "all")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if calls := repo.sprintCalls.Load(); calls != 2 {
			t.Errorf("expected force refresh to reach the sprint repo, got %d hits", calls)
		}
		if refreshedBacklog.TotalIssues != 1 {
			t.Errorf("expected TotalIssues=1 on refreshed backlog, got %d", refreshedBacklog.TotalIssues)
		}
	})
}

func TestJiraService_PerUserCacheIsolation(t *testing.T) {
	repo := &mockJiraRepo{
		issues: cacheTestIssues(),
	}
	service, userA, _ := setupTestService(repo)
	ctx := context.Background()

	if _, err := service.GetMyIssues(ctx, userA); err != nil {
		t.Fatalf("unexpected error for user A: %v", err)
	}
	if calls := repo.searchCalls.Load(); calls != 1 {
		t.Fatalf("expected 1 hit for user A first call, got %d", calls)
	}

	issuesForB, err := service.GetMyIssues(ctx, secondaryConfiguredUserID)
	if err != nil {
		t.Fatalf("unexpected error for user B: %v", err)
	}
	if calls := repo.searchCalls.Load(); calls != 2 {
		t.Errorf("expected user B to reach the repo instead of reusing user A's cache, got %d hits", calls)
	}
	if len(issuesForB) != 1 || issuesForB[0].Key != "CRMMS-77911" {
		t.Errorf("expected CRMMS-77911 for user B, got %+v", issuesForB)
	}

	if _, err := service.GetMyIssues(ctx, userA); err != nil {
		t.Fatalf("unexpected error for user A second call: %v", err)
	}
	if _, err := service.GetMyIssues(ctx, secondaryConfiguredUserID); err != nil {
		t.Fatalf("unexpected error for user B second call: %v", err)
	}
	if calls := repo.searchCalls.Load(); calls != 2 {
		t.Errorf("expected each user to be served from their own cache after warmup, got %d hits", calls)
	}
}

func TestJiraService_CacheExpiresAfterTTL(t *testing.T) {
	repo := &mockJiraRepo{
		issues: cacheTestIssues(),
	}
	service, configuredUserID, _ := setupTestService(repo)
	service.cacheTTL = 20 * time.Millisecond
	ctx := context.Background()

	if _, err := service.GetMyIssues(ctx, configuredUserID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := service.GetMyIssues(ctx, configuredUserID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if calls := repo.searchCalls.Load(); calls != 1 {
		t.Fatalf("expected cached hit before expiry with 1 hit, got %d", calls)
	}

	time.Sleep(60 * time.Millisecond)

	if _, err := service.GetMyIssues(ctx, configuredUserID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if calls := repo.searchCalls.Load(); calls != 2 {
		t.Errorf("expected expired entry to be recomputed, got %d hits", calls)
	}
}

func TestJiraService_CachesSuccessPerOperation(t *testing.T) {
	repo := &mockJiraRepo{
		issues:  cacheTestIssues(),
		sprints: cacheTestSprints(),
	}
	service, configuredUserID, _ := setupTestService(repo)
	ctx := context.Background()

	firstIssues, err := service.GetMyIssues(ctx, configuredUserID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	secondIssues, err := service.GetMyIssues(ctx, configuredUserID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(firstIssues) != 1 || len(secondIssues) != 1 || secondIssues[0].Key != firstIssues[0].Key {
		t.Errorf("expected identical cached issues, got %+v", secondIssues)
	}
	if calls := repo.searchCalls.Load(); calls != 1 {
		t.Errorf("expected exactly 1 repo hit for two GetMyIssues calls within TTL, got %d", calls)
	}

	firstBacklog, err := service.GetBacklog(ctx, configuredUserID, "all")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	secondBacklog, err := service.GetBacklog(ctx, configuredUserID, "all")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if secondBacklog.TotalIssues != firstBacklog.TotalIssues {
		t.Errorf("expected identical cached backlog, got %+v", secondBacklog)
	}
	if calls := repo.sprintCalls.Load(); calls != 1 {
		t.Errorf("expected exactly 1 sprint repo hit for two GetBacklog calls within TTL, got %d", calls)
	}
	if calls := repo.searchCalls.Load(); calls != 2 {
		t.Errorf("expected issues and backlog cached under separate keys (issues op 1 hit + backlog detection 1 hit), got %d", calls)
	}
}

func TestJiraService_ConcurrentCacheAccess(t *testing.T) {
	repo := &mockJiraRepo{
		issues: cacheTestIssues(),
	}
	service, configuredUserID, _ := setupTestService(repo)
	ctx := context.Background()

	if _, err := service.GetMyIssues(ctx, configuredUserID); err != nil {
		t.Fatalf("warmup call failed: %v", err)
	}

	const workerCount = 8
	workerErrors := make(chan error, workerCount)
	var waitGroup sync.WaitGroup
	for range workerCount {
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			issues, err := service.GetMyIssues(ctx, configuredUserID)
			if err != nil {
				workerErrors <- fmt.Errorf("concurrent GetMyIssues failed: %w", err)
				return
			}
			if len(issues) != 1 || issues[0].Key != "CRMMS-77911" {
				workerErrors <- fmt.Errorf("unexpected concurrent result: %+v", issues)
			}
		}()
	}
	waitGroup.Wait()
	close(workerErrors)

	for workerErr := range workerErrors {
		t.Error(workerErr)
	}
	if calls := repo.searchCalls.Load(); calls != 1 {
		t.Errorf("expected concurrent reads to be served from cache after warmup, got %d hits", calls)
	}
}

func TestJiraService_Success(t *testing.T) {
	sampleIssues := []domain.JiraIssue{
		{
			ID:         "10101",
			Key:        "CRMMS-77911",
			Kind:       "story",
			Points:     8,
			SubLabel:   "MMS Core",
			SprintName: "Sprint 44 (MMS Modernization)",
			Fields: domain.JiraIssueFields{
				Summary: "Merchant Onboarding Multi-Tier Verification Service",
			},
		},
		{
			ID:         "10201",
			Key:        "BUG-201",
			Kind:       "bug",
			Points:     3,
			SubLabel:   "Bug Fix",
			SprintName: "Sprint 44 (MMS Modernization)",
			Fields: domain.JiraIssueFields{
				Summary: "Race condition during concurrent QRIS settlement callback processing",
			},
		},
	}

	sampleSprints := []domain.JiraSprint{
		{
			ID:     44,
			Name:   "Sprint 44 (MMS Modernization)",
			State:  "active",
			Issues: sampleIssues,
		},
		{
			ID:     45,
			Name:   "Sprint 45 (MMS Next Gen)",
			State:  "future",
			Issues: []domain.JiraIssue{},
		},
	}

	repo := &mockJiraRepo{
		issues:  sampleIssues,
		sprints: sampleSprints,
		detail:  &sampleIssues[0],
	}

	service, configuredUserID, _ := setupTestService(repo)
	ctx := context.Background()

	t.Run("GetMyIssues success", func(t *testing.T) {
		issues, err := service.GetMyIssues(ctx, configuredUserID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(issues) != 2 {
			t.Fatalf("expected 2 issues, got %d", len(issues))
		}
		if issues[0].Key != "CRMMS-77911" {
			t.Errorf("expected CRMMS-77911, got %s", issues[0].Key)
		}
	})

	t.Run("GetBacklog success", func(t *testing.T) {
		backlog, err := service.GetBacklog(ctx, configuredUserID, "all")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(backlog.Sprints) != 2 {
			t.Fatalf("expected 2 sprints, got %d", len(backlog.Sprints))
		}
		if backlog.TotalIssues != 2 {
			t.Errorf("expected TotalIssues=2, got %d", backlog.TotalIssues)
		}
		if backlog.ActiveSprintName != "Sprint 44 (MMS Modernization)" {
			t.Errorf("expected active sprint name 'Sprint 44 (MMS Modernization)', got %s", backlog.ActiveSprintName)
		}
	})

	t.Run("GetIssueDetail success", func(t *testing.T) {
		issue, err := service.GetIssueDetail(ctx, configuredUserID, "CRMMS-77911")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if issue.Key != "CRMMS-77911" {
			t.Errorf("expected CRMMS-77911, got %s", issue.Key)
		}
	})

	t.Run("GetIssueDetail empty key bad request", func(t *testing.T) {
		_, err := service.GetIssueDetail(ctx, configuredUserID, "   ")
		if !errors.Is(err, sharedErrors.ErrBadRequest) {
			t.Errorf("expected ErrBadRequest for whitespace key, got %v", err)
		}
	})

	t.Run("GetIssueDetail not found", func(t *testing.T) {
		_, err := service.GetIssueDetail(ctx, configuredUserID, "UNKNOWN-999")
		if !errors.Is(err, sharedErrors.ErrNotFound) {
			t.Errorf("expected ErrNotFound for missing key, got %v", err)
		}
	})
}

func TestJiraService_UnconfiguredPATError(t *testing.T) {
	repo := &mockJiraRepo{}
	service, _, _ := setupTestService(repo)
	ctx := context.Background()

	t.Run("GetMyIssues with empty userID", func(t *testing.T) {
		_, err := service.GetMyIssues(ctx, "")
		if !errors.Is(err, sharedErrors.ErrUnauthorized) {
			t.Errorf("expected ErrUnauthorized, got %v", err)
		}
	})

	t.Run("GetMyIssues with unconfigured userID", func(t *testing.T) {
		_, err := service.GetMyIssues(ctx, "usr-not-in-vault")
		if !errors.Is(err, sharedErrors.ErrUnauthorized) {
			t.Errorf("expected ErrUnauthorized, got %v", err)
		}
	})

	t.Run("GetBacklog with unconfigured userID", func(t *testing.T) {
		_, err := service.GetBacklog(ctx, "usr-not-in-vault", "")
		if !errors.Is(err, sharedErrors.ErrUnauthorized) {
			t.Errorf("expected ErrUnauthorized, got %v", err)
		}
	})

	t.Run("GetIssueDetail with unconfigured userID", func(t *testing.T) {
		_, err := service.GetIssueDetail(ctx, "usr-not-in-vault", "CRMMS-77911")
		if !errors.Is(err, sharedErrors.ErrUnauthorized) {
			t.Errorf("expected ErrUnauthorized, got %v", err)
		}
	})
}

func TestJiraService_InvalidPATError(t *testing.T) {
	repo := &mockJiraRepo{}
	service, _, invalidUserID := setupTestService(repo)
	ctx := context.Background()

	t.Run("GetMyIssues invalid PAT", func(t *testing.T) {
		_, err := service.GetMyIssues(ctx, invalidUserID)
		if !errors.Is(err, sharedErrors.ErrUnauthorized) {
			t.Errorf("expected ErrUnauthorized for invalid PAT, got %v", err)
		}
	})

	t.Run("GetBacklog invalid PAT", func(t *testing.T) {
		_, err := service.GetBacklog(ctx, invalidUserID, "")
		if !errors.Is(err, sharedErrors.ErrUnauthorized) {
			t.Errorf("expected ErrUnauthorized for invalid PAT, got %v", err)
		}
	})

	t.Run("GetIssueDetail invalid PAT", func(t *testing.T) {
		_, err := service.GetIssueDetail(ctx, invalidUserID, "CRMMS-77911")
		if !errors.Is(err, sharedErrors.ErrUnauthorized) {
			t.Errorf("expected ErrUnauthorized for invalid PAT, got %v", err)
		}
	})
}

func TestJiraService_VPNGatewayError(t *testing.T) {
	vpnErr := errors.New("unable to reach Jira BRI API. Please check your VPN connection.")
	repo := &mockJiraRepo{
		err: vpnErr,
	}
	service, configuredUserID, _ := setupTestService(repo)
	ctx := context.Background()

	t.Run("GetMyIssues VPN gateway error", func(t *testing.T) {
		_, err := service.GetMyIssues(ctx, configuredUserID)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !errors.Is(err, vpnErr) && err.Error() != vpnErr.Error() {
			t.Errorf("expected VPN error, got %v", err)
		}
	})

	t.Run("GetBacklog VPN gateway error", func(t *testing.T) {
		_, err := service.GetBacklog(ctx, configuredUserID, "")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !errors.Is(err, vpnErr) && err.Error() != vpnErr.Error() {
			t.Errorf("expected VPN error, got %v", err)
		}
	})

	t.Run("GetIssueDetail VPN gateway error", func(t *testing.T) {
		_, err := service.GetIssueDetail(ctx, configuredUserID, "CRMMS-77911")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !errors.Is(err, vpnErr) && err.Error() != vpnErr.Error() {
			t.Errorf("expected VPN error, got %v", err)
		}
	})
}

func TestJiraService_DynamicSquadDetection(t *testing.T) {
	fortuneIssues := []domain.JiraIssue{
		{
			ID:         "10101",
			Key:        "CRMMS-77911",
			SprintName: "Sprint 44 - Fortune Squad",
			Fields: domain.JiraIssueFields{
				Status: domain.JiraStatus{
					Name: "In Progress",
					StatusCategory: domain.JiraStatusCategory{
						Key: "indeterminate",
					},
				},
			},
		},
	}

	allSprints := []domain.JiraSprint{
		{
			ID:     44,
			Name:   "Sprint 44 - Fortune Squad",
			State:  "active",
			Issues: fortuneIssues,
		},
		{
			ID:    45,
			Name:  "Sprint 45 - Fortune Squad",
			State: "future",
			Issues: []domain.JiraIssue{
				{ID: "10104", Key: "CRMMS-77925"},
			},
		},
		{
			ID:    55,
			Name:  "Sprint Azzuri #5",
			State: "active",
			Issues: []domain.JiraIssue{
				{ID: "10701", Key: "AZZ-101"},
				{ID: "10702", Key: "AZZ-102"},
				{ID: "10703", Key: "AZZ-103"},
			},
		},
	}

	repo := &mockJiraRepo{
		issues:  fortuneIssues,
		sprints: allSprints,
	}

	service, configuredUserID, _ := setupTestService(repo)
	ctx := context.Background()

	t.Run("Auto-detects Fortune Squad and filters out Azzuri", func(t *testing.T) {
		backlog, err := service.GetBacklog(ctx, configuredUserID, "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if backlog.DetectedSquad != "Fortune Squad" {
			t.Errorf("expected detected squad 'Fortune Squad', got '%s'", backlog.DetectedSquad)
		}
		if len(backlog.Sprints) != 2 {
			t.Fatalf("expected 2 Fortune sprints, got %d", len(backlog.Sprints))
		}
		if backlog.TotalIssues != 2 {
			t.Errorf("expected TotalIssues=2 for Fortune Squad, got %d", backlog.TotalIssues)
		}
		if len(backlog.AvailableSquads) != 2 {
			t.Errorf("expected 2 available squads, got %d", len(backlog.AvailableSquads))
		}
	})

	t.Run("Explicitly filters by Azzuri", func(t *testing.T) {
		backlog, err := service.GetBacklog(ctx, configuredUserID, "Azzuri")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(backlog.Sprints) != 1 {
			t.Fatalf("expected 1 Azzuri sprint, got %d", len(backlog.Sprints))
		}
		if backlog.TotalIssues != 3 {
			t.Errorf("expected TotalIssues=3 for Azzuri, got %d", backlog.TotalIssues)
		}
	})

	t.Run("Requests all squads", func(t *testing.T) {
		backlog, err := service.GetBacklog(ctx, configuredUserID, "all")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(backlog.Sprints) != 3 {
			t.Fatalf("expected all 3 sprints, got %d", len(backlog.Sprints))
		}
		if backlog.TotalIssues != 5 {
			t.Errorf("expected TotalIssues=5 across all sprints, got %d", backlog.TotalIssues)
		}
	})
}
