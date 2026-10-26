package application

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	sharedErrors "lunar/backend/internal/shared/errors"
)

func TestConfluenceService_GetPageAncestorsResolvesTheCallerPAT(t *testing.T) {
	var receivedAuth string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"999","ancestors":[{"id":"1","title":"Root"},{"id":"2","title":"Parent"}]}`))
	}))
	defer upstream.Close()

	service, userID, _ := setupTestService(upstream.URL)

	ancestors, err := service.GetPageAncestors(context.Background(), userID, "999")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if receivedAuth != "Bearer "+validConfluencePAT {
		t.Errorf("expected the decrypted PAT, got %q", receivedAuth)
	}
	if len(ancestors) != 2 || ancestors[1].Title != "Parent" {
		t.Errorf("expected the parent ancestor, got %+v", ancestors)
	}
}

func TestConfluenceService_GetChildPagesAndStorage(t *testing.T) {
	var childPath string
	var storageExpand string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/rest/api/content/2/child/page" {
			childPath = r.URL.Path
			_, _ = w.Write([]byte(`{"results":[{"id":"10","title":"CRMMS-1-MIG"}]}`))
			return
		}
		storageExpand = r.URL.Query().Get("expand")
		_, _ = w.Write([]byte(`{"id":"10","body":{"storage":{"value":"<p>table</p>"}}}`))
	}))
	defer upstream.Close()

	service, userID, _ := setupTestService(upstream.URL)
	ctx := context.Background()

	children, err := service.GetChildPages(ctx, userID, "2")
	if err != nil {
		t.Fatalf("unexpected children error: %v", err)
	}
	if childPath != "/rest/api/content/2/child/page" || len(children) != 1 || children[0].ID != "10" {
		t.Errorf("unexpected children result %+v from %q", children, childPath)
	}

	storage, err := service.GetPageStorage(ctx, userID, "10")
	if err != nil {
		t.Fatalf("unexpected storage error: %v", err)
	}
	if storageExpand != "body.storage" || storage != "<p>table</p>" {
		t.Errorf("unexpected storage result %q with expand %q", storage, storageExpand)
	}
}

func TestConfluenceService_PageTreeValidation(t *testing.T) {
	service, _, unconfiguredUserID := setupTestService("https://confluence.example.com")
	ctx := context.Background()

	if _, err := service.GetPageAncestors(ctx, configuredUserID, "   "); !errors.Is(err, sharedErrors.ErrBadRequest) {
		t.Errorf("expected ErrBadRequest for a blank page id, got %v", err)
	}
	if _, err := service.GetChildPages(ctx, unconfiguredUserID, "10"); !errors.Is(err, sharedErrors.ErrUnauthorized) {
		t.Errorf("expected ErrUnauthorized without a PAT, got %v", err)
	}
	if _, err := service.GetPageStorage(ctx, missingVaultUserID, "10"); !errors.Is(err, sharedErrors.ErrUnauthorized) {
		t.Errorf("expected ErrUnauthorized for a missing vault entry, got %v", err)
	}
}

func TestConfluenceService_GetPageStorageMapsNotFound(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer upstream.Close()

	service, userID, _ := setupTestService(upstream.URL)

	if _, err := service.GetPageStorage(context.Background(), userID, "10"); !errors.Is(err, sharedErrors.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}
