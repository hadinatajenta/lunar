package application

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	sharedErrors "lunar/backend/internal/shared/errors"
)

func TestConfluenceService_UpdatePageStorageResolvesTheCallerPAT(t *testing.T) {
	var getAuth string
	var putAuth string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodGet {
			getAuth = r.Header.Get("Authorization")
			_, _ = w.Write([]byte(`{"id":"10","title":"Title","version":{"number":4}}`))
			return
		}
		putAuth = r.Header.Get("Authorization")
		_, _ = w.Write([]byte(`{"id":"10","version":{"number":5},"_links":{"webui":"/display/DEV/Page"}}`))
	}))
	defer upstream.Close()

	service, userID, _ := setupTestService(upstream.URL)

	version, pageURL, err := service.UpdatePageStorage(context.Background(), userID, "10", "<p>body</p>")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if getAuth != "Bearer "+validConfluencePAT || putAuth != "Bearer "+validConfluencePAT {
		t.Errorf("expected the decrypted PAT on both calls, got %q and %q", getAuth, putAuth)
	}
	if version != 5 || pageURL != "/display/DEV/Page" {
		t.Errorf("unexpected write result %d/%q", version, pageURL)
	}
}

func TestConfluenceService_UpdatePageStorageValidation(t *testing.T) {
	service, _, unconfiguredUserID := setupTestService("https://confluence.example.com")
	ctx := context.Background()

	if _, _, err := service.UpdatePageStorage(ctx, configuredUserID, "   ", "<p>x</p>"); !errors.Is(err, sharedErrors.ErrBadRequest) {
		t.Errorf("expected ErrBadRequest for a blank page id, got %v", err)
	}
	if _, _, err := service.UpdatePageStorage(ctx, configuredUserID, "10", "   "); !errors.Is(err, sharedErrors.ErrBadRequest) {
		t.Errorf("expected ErrBadRequest for a blank storage value, got %v", err)
	}
	if _, _, err := service.UpdatePageStorage(ctx, unconfiguredUserID, "10", "<p>x</p>"); !errors.Is(err, sharedErrors.ErrUnauthorized) {
		t.Errorf("expected ErrUnauthorized without a PAT, got %v", err)
	}
	if _, _, err := service.UpdatePageStorage(ctx, missingVaultUserID, "10", "<p>x</p>"); !errors.Is(err, sharedErrors.ErrUnauthorized) {
		t.Errorf("expected ErrUnauthorized for a missing vault entry, got %v", err)
	}
}

func TestConfluenceService_UpdatePageStorageMapsConflict(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodGet {
			_, _ = w.Write([]byte(`{"id":"10","title":"Title","version":{"number":4}}`))
			return
		}
		w.WriteHeader(http.StatusConflict)
	}))
	defer upstream.Close()

	service, userID, _ := setupTestService(upstream.URL)

	_, _, err := service.UpdatePageStorage(context.Background(), userID, "10", "<p>x</p>")
	if !errors.Is(err, sharedErrors.ErrConflict) {
		t.Fatalf("expected ErrConflict, got %v", err)
	}
}
