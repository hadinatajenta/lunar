package application

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	authApp "lunar/backend/internal/auth/application"
	authDomain "lunar/backend/internal/auth/domain"
	confluenceInfra "lunar/backend/internal/confluence/infrastructure"
	"lunar/backend/internal/shared/cache"
	"lunar/backend/internal/shared/crypto"
	sharedErrors "lunar/backend/internal/shared/errors"
)

const (
	configuredUserID      = "usr-configured"
	secondaryUserID       = "usr-configured-secondary"
	unconfiguredUserID    = "usr-unconfigured"
	missingVaultUserID    = "usr-not-in-vault"
	validConfluencePAT    = "valid-confluence-pat-token"
	encryptionTestKey     = "0123456789abcdef0123456789abcdef"
	confluenceListFixture = `{"results":[{"id":"101","title":"UT Coverage Report","space":{"key":"DEV","name":"DEV Team"},"version":{"when":"2026-01-02T03:04:05.000+07:00","by":{"displayName":"Jane Doe"}},"_links":{"base":"https://confluence.example.com","webui":"/display/DEV/UT-Coverage-Report"}}]}`
)

type mockVaultRepo struct {
	secrets map[string]*authDomain.UserSecrets
}

func (m *mockVaultRepo) SaveSecrets(ctx context.Context, s *authDomain.UserSecrets) error {
	m.secrets[s.UserID] = s
	return nil
}

func (m *mockVaultRepo) GetSecretsByUserID(ctx context.Context, userID string) (*authDomain.UserSecrets, error) {
	secret, ok := m.secrets[userID]
	if !ok || secret == nil {
		return nil, sharedErrors.ErrNotFound
	}
	return secret, nil
}

func setupTestService(baseURL string) (*ConfluenceService, string, string) {
	vaultRepo := &mockVaultRepo{
		secrets: make(map[string]*authDomain.UserSecrets),
	}

	validPATEnc, _ := crypto.EncryptSecret(validConfluencePAT, encryptionTestKey)

	vaultRepo.secrets[configuredUserID] = &authDomain.UserSecrets{
		UserID:           configuredUserID,
		ConfluencePATEnc: validPATEnc,
		UpdatedAt:        time.Now().UTC(),
	}
	vaultRepo.secrets[secondaryUserID] = &authDomain.UserSecrets{
		UserID:           secondaryUserID,
		ConfluencePATEnc: validPATEnc,
		UpdatedAt:        time.Now().UTC(),
	}
	vaultRepo.secrets[unconfiguredUserID] = &authDomain.UserSecrets{
		UserID:    unconfiguredUserID,
		UpdatedAt: time.Now().UTC(),
	}

	authService := authApp.NewAuthService(nil, vaultRepo, encryptionTestKey, "jwt-secret", time.Hour)
	client := confluenceInfra.NewConfluenceClient(baseURL)
	return NewConfluenceService(client, authService, nil), configuredUserID, unconfiguredUserID
}

func TestConfluenceService_MissingPATReturnsUnauthorized(t *testing.T) {
	service, _, _ := setupTestService("https://confluence.example.com")
	ctx := context.Background()

	t.Run("empty userID", func(t *testing.T) {
		_, err := service.GetDocuments(ctx, "", CONFLUENCE_SCOPE_ALL)
		if !errors.Is(err, sharedErrors.ErrUnauthorized) {
			t.Fatalf("expected ErrUnauthorized, got %v", err)
		}
		if !strings.Contains(err.Error(), "Confluence PAT is not configured") {
			t.Errorf("expected missing PAT message, got %q", err.Error())
		}
	})

	t.Run("userID without vault entry", func(t *testing.T) {
		_, err := service.GetDocuments(ctx, missingVaultUserID, CONFLUENCE_SCOPE_ALL)
		if !errors.Is(err, sharedErrors.ErrUnauthorized) {
			t.Fatalf("expected ErrUnauthorized, got %v", err)
		}
		if !strings.Contains(err.Error(), "Confluence PAT is not configured") {
			t.Errorf("expected missing PAT message, got %q", err.Error())
		}
	})

	t.Run("userID with empty Confluence PAT", func(t *testing.T) {
		_, err := service.GetDocuments(ctx, unconfiguredUserID, CONFLUENCE_SCOPE_ALL)
		if !errors.Is(err, sharedErrors.ErrUnauthorized) {
			t.Fatalf("expected ErrUnauthorized, got %v", err)
		}
		if !strings.Contains(err.Error(), "Confluence PAT is not configured") {
			t.Errorf("expected missing PAT message, got %q", err.Error())
		}
	})

	t.Run("detail with userID without vault entry", func(t *testing.T) {
		_, err := service.GetDocumentDetail(ctx, missingVaultUserID, "101")
		if !errors.Is(err, sharedErrors.ErrUnauthorized) {
			t.Fatalf("expected ErrUnauthorized, got %v", err)
		}
		if !strings.Contains(err.Error(), "Confluence PAT is not configured") {
			t.Errorf("expected missing PAT message, got %q", err.Error())
		}
	})
}

func TestConfluenceService_EmptyDocumentIDReturnsBadRequest(t *testing.T) {
	service, _, _ := setupTestService("https://confluence.example.com")

	_, err := service.GetDocumentDetail(context.Background(), configuredUserID, "   ")
	if !errors.Is(err, sharedErrors.ErrBadRequest) {
		t.Fatalf("expected ErrBadRequest, got %v", err)
	}
	if !strings.Contains(err.Error(), "document id is required") {
		t.Errorf("expected missing id message, got %q", err.Error())
	}
}

func TestConfluenceService_UpstreamUnauthorizedMapsToErrUnauthorized(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer upstream.Close()

	service, _, _ := setupTestService(upstream.URL)

	_, err := service.GetDocuments(context.Background(), configuredUserID, CONFLUENCE_SCOPE_ALL)
	if !errors.Is(err, sharedErrors.ErrUnauthorized) {
		t.Fatalf("expected ErrUnauthorized for upstream 401, got %v", err)
	}
	if !strings.Contains(err.Error(), "invalid or expired Confluence PAT") {
		t.Errorf("expected invalid PAT message, got %q", err.Error())
	}
	if strings.Contains(err.Error(), validConfluencePAT) {
		t.Error("expected PAT value to never appear in the error")
	}
}

func TestConfluenceService_UpstreamUnreachableReturnsErrorForEveryCall(t *testing.T) {
	closedUpstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	baseURL := closedUpstream.URL
	closedUpstream.Close()

	service, _, _ := setupTestService(baseURL)
	ctx := context.Background()

	firstErr := error(nil)
	if _, err := service.GetDocuments(ctx, configuredUserID, CONFLUENCE_SCOPE_ALL); err == nil {
		t.Fatal("expected error on first call, got nil")
	} else {
		firstErr = err
	}
	if _, err := service.GetDocuments(ctx, configuredUserID, CONFLUENCE_SCOPE_ALL); err == nil {
		t.Fatal("expected error on second call, got nil")
	}
	if errors.Is(firstErr, sharedErrors.ErrUnauthorized) {
		t.Error("expected unreachable upstream to not map to ErrUnauthorized")
	}
	if errors.Is(firstErr, sharedErrors.ErrNotFound) {
		t.Error("expected unreachable upstream to not map to ErrNotFound")
	}
}

func TestConfluenceService_UpstreamServerErrorIsNotCached(t *testing.T) {
	var upstreamHits atomic.Int64
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upstreamHits.Add(1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer upstream.Close()

	service, _, _ := setupTestService(upstream.URL)
	ctx := context.Background()

	for call := 1; call <= 2; call++ {
		_, err := service.GetDocuments(ctx, configuredUserID, CONFLUENCE_SCOPE_ALL)
		if err == nil {
			t.Fatalf("expected error on call %d, got nil", call)
		}
		if errors.Is(err, sharedErrors.ErrUnauthorized) || errors.Is(err, sharedErrors.ErrNotFound) {
			t.Fatalf("expected generic upstream error on call %d, got %v", call, err)
		}
	}
	if hits := upstreamHits.Load(); hits != 2 {
		t.Errorf("expected error responses to never be cached (2 upstream hits), got %d", hits)
	}
}

func TestConfluenceService_DocumentNotFoundIsNotCached(t *testing.T) {
	var upstreamHits atomic.Int64
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upstreamHits.Add(1)
		w.WriteHeader(http.StatusNotFound)
	}))
	defer upstream.Close()

	service, _, _ := setupTestService(upstream.URL)
	ctx := context.Background()

	for call := 1; call <= 2; call++ {
		_, err := service.GetDocumentDetail(ctx, configuredUserID, "999")
		if !errors.Is(err, sharedErrors.ErrNotFound) {
			t.Fatalf("expected ErrNotFound on call %d, got %v", call, err)
		}
	}
	if hits := upstreamHits.Load(); hits != 2 {
		t.Errorf("expected 404 responses to never be cached (2 upstream hits), got %d", hits)
	}
}

func TestConfluenceService_CacheServesRepeatWithinTTL(t *testing.T) {
	var upstreamHits atomic.Int64
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upstreamHits.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(confluenceListFixture))
	}))
	defer upstream.Close()

	service, _, _ := setupTestService(upstream.URL)
	ctx := context.Background()

	first, err := service.GetDocuments(ctx, configuredUserID, CONFLUENCE_SCOPE_ALL)
	if err != nil {
		t.Fatalf("unexpected error on first call: %v", err)
	}
	second, err := service.GetDocuments(ctx, configuredUserID, CONFLUENCE_SCOPE_ALL)
	if err != nil {
		t.Fatalf("unexpected error on second call: %v", err)
	}
	if hits := upstreamHits.Load(); hits != 1 {
		t.Fatalf("expected second call to be served from cache with 1 upstream hit, got %d", hits)
	}
	if first.Total != 1 || second.Total != 1 || second.Documents[0].ID != first.Documents[0].ID {
		t.Errorf("expected identical cached responses, got %+v and %+v", first, second)
	}
}

func TestConfluenceService_CacheExpiresAfterTTL(t *testing.T) {
	var upstreamHits atomic.Int64
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upstreamHits.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(confluenceListFixture))
	}))
	defer upstream.Close()

	service, _, _ := setupTestService(upstream.URL)
	service.cacheStore = cache.New(20 * time.Millisecond)
	ctx := context.Background()

	if _, err := service.GetDocuments(ctx, configuredUserID, CONFLUENCE_SCOPE_ALL); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := service.GetDocuments(ctx, configuredUserID, CONFLUENCE_SCOPE_ALL); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if hits := upstreamHits.Load(); hits != 1 {
		t.Fatalf("expected cached hit before expiry with 1 upstream hit, got %d", hits)
	}

	time.Sleep(60 * time.Millisecond)

	if _, err := service.GetDocuments(ctx, configuredUserID, CONFLUENCE_SCOPE_ALL); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if hits := upstreamHits.Load(); hits != 2 {
		t.Errorf("expected expired entry to be recomputed, got %d upstream hits", hits)
	}
}

func TestConfluenceService_PerUserCacheIsolation(t *testing.T) {
	var upstreamHits atomic.Int64
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upstreamHits.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(confluenceListFixture))
	}))
	defer upstream.Close()

	service, userA, _ := setupTestService(upstream.URL)
	ctx := context.Background()

	if _, err := service.GetDocuments(ctx, userA, CONFLUENCE_SCOPE_ALL); err != nil {
		t.Fatalf("unexpected error for user A: %v", err)
	}
	if hits := upstreamHits.Load(); hits != 1 {
		t.Fatalf("expected 1 upstream hit for user A, got %d", hits)
	}

	if _, err := service.GetDocuments(ctx, secondaryUserID, CONFLUENCE_SCOPE_ALL); err != nil {
		t.Fatalf("unexpected error for user B: %v", err)
	}
	if hits := upstreamHits.Load(); hits != 2 {
		t.Errorf("expected user B to reach upstream instead of reusing user A's cache, got %d hits", hits)
	}

	if _, err := service.GetDocuments(ctx, userA, CONFLUENCE_SCOPE_ALL); err != nil {
		t.Fatalf("unexpected error for user A second call: %v", err)
	}
	if _, err := service.GetDocuments(ctx, secondaryUserID, CONFLUENCE_SCOPE_ALL); err != nil {
		t.Fatalf("unexpected error for user B second call: %v", err)
	}
	if hits := upstreamHits.Load(); hits != 2 {
		t.Errorf("expected each user to be served from their own cache after warmup, got %d hits", hits)
	}
}
