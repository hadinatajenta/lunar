package infrastructure

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	sharedErrors "lunar/backend/internal/shared/errors"
)

func TestConfluenceClient_GetPageAncestorsReadsTheAncestorPath(t *testing.T) {
	var receivedPath string
	var receivedExpand string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedPath = r.URL.Path
		receivedExpand = r.URL.Query().Get("expand")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"999","ancestors":[{"id":"1","title":"Root"},{"id":"2","title":"Parent"}]}`))
	}))
	defer upstream.Close()

	client := NewConfluenceClient(upstream.URL)
	ancestors, err := client.GetPageAncestors(context.Background(), testPAT, "999")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if receivedPath != "/rest/api/content/999" {
		t.Errorf("expected content path, got %q", receivedPath)
	}
	if receivedExpand != "ancestors" {
		t.Errorf("expected expand=ancestors, got %q", receivedExpand)
	}
	if len(ancestors) != 2 {
		t.Fatalf("expected 2 ancestors, got %d", len(ancestors))
	}
	if ancestors[0].ID != "1" || ancestors[0].Title != "Root" {
		t.Errorf("expected the root first, got %+v", ancestors[0])
	}
	if ancestors[1].ID != "2" || ancestors[1].Title != "Parent" {
		t.Errorf("expected the parent last, got %+v", ancestors[1])
	}
}

func TestConfluenceClient_GetChildPagesReadsTheChildPath(t *testing.T) {
	var receivedPath string
	var receivedLimit string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedPath = r.URL.Path
		receivedLimit = r.URL.Query().Get("limit")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"results":[{"id":"10","title":"CRMMS-1-MIG"},{"id":"11","title":"Other"}]}`))
	}))
	defer upstream.Close()

	client := NewConfluenceClient(upstream.URL)
	children, err := client.GetChildPages(context.Background(), testPAT, "2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if receivedPath != "/rest/api/content/2/child/page" {
		t.Errorf("expected the child page path, got %q", receivedPath)
	}
	if receivedLimit != "100" {
		t.Errorf("expected limit=100, got %q", receivedLimit)
	}
	if len(children) != 2 {
		t.Fatalf("expected 2 children, got %d", len(children))
	}
	if children[0].ID != "10" || children[0].Title != "CRMMS-1-MIG" {
		t.Errorf("expected the first child, got %+v", children[0])
	}
}

func TestConfluenceClient_GetPageStorageReturnsTheRawStorageValue(t *testing.T) {
	var receivedExpand string
	storageValue := `<table><tbody><tr><th>Repository</th></tr></tbody></table>`
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedExpand = r.URL.Query().Get("expand")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"10","body":{"storage":{"value":"` + storageValue + `"}}}`))
	}))
	defer upstream.Close()

	client := NewConfluenceClient(upstream.URL)
	storage, err := client.GetPageStorage(context.Background(), testPAT, "10")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if receivedExpand != "body.storage" {
		t.Errorf("expected expand=body.storage, got %q", receivedExpand)
	}
	if storage != storageValue {
		t.Errorf("expected the raw storage value, got %q", storage)
	}
}

func TestConfluenceClient_PageTreeUnauthorizedMapsToErrUnauthorized(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer upstream.Close()

	client := NewConfluenceClient(upstream.URL)
	ctx := context.Background()

	if _, err := client.GetPageAncestors(ctx, testPAT, "10"); !errors.Is(err, sharedErrors.ErrUnauthorized) {
		t.Errorf("expected ErrUnauthorized for ancestors, got %v", err)
	}
	if _, err := client.GetChildPages(ctx, testPAT, "10"); !errors.Is(err, sharedErrors.ErrUnauthorized) {
		t.Errorf("expected ErrUnauthorized for children, got %v", err)
	}
	if _, err := client.GetPageStorage(ctx, testPAT, "10"); !errors.Is(err, sharedErrors.ErrUnauthorized) {
		t.Errorf("expected ErrUnauthorized for storage, got %v", err)
	}
}

func TestConfluenceClient_PageTreeNotFoundMapsToErrNotFound(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer upstream.Close()

	client := NewConfluenceClient(upstream.URL)
	ctx := context.Background()

	if _, err := client.GetPageAncestors(ctx, testPAT, "10"); !errors.Is(err, sharedErrors.ErrNotFound) {
		t.Errorf("expected ErrNotFound for ancestors, got %v", err)
	}
	if _, err := client.GetChildPages(ctx, testPAT, "10"); !errors.Is(err, sharedErrors.ErrNotFound) {
		t.Errorf("expected ErrNotFound for children, got %v", err)
	}
	if _, err := client.GetPageStorage(ctx, testPAT, "10"); !errors.Is(err, sharedErrors.ErrNotFound) {
		t.Errorf("expected ErrNotFound for storage, got %v", err)
	}
}

func TestConfluenceClient_PageTreeUndecodableBodyReturnsError(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{not json`))
	}))
	defer upstream.Close()

	client := NewConfluenceClient(upstream.URL)
	ctx := context.Background()

	_, err := client.GetPageAncestors(ctx, testPAT, "10")
	if err == nil {
		t.Fatal("expected a decode error for ancestors, got nil")
	}
	if errors.Is(err, sharedErrors.ErrNotFound) || errors.Is(err, sharedErrors.ErrUnauthorized) {
		t.Errorf("expected a decode error, got %v", err)
	}
	if !strings.Contains(err.Error(), "ancestors") {
		t.Errorf("expected the ancestors decode message, got %q", err.Error())
	}

	if _, err := client.GetPageStorage(ctx, testPAT, "10"); err == nil {
		t.Fatal("expected a decode error for storage, got nil")
	}
}

func TestConfluenceClient_PageTreeInputValidation(t *testing.T) {
	client := NewConfluenceClient("https://confluence.example.com")
	ctx := context.Background()

	if _, err := client.GetPageAncestors(ctx, testPAT, "   "); !errors.Is(err, sharedErrors.ErrBadRequest) {
		t.Errorf("expected ErrBadRequest for a blank page id, got %v", err)
	}
	if _, err := client.GetChildPages(ctx, "", "10"); !errors.Is(err, sharedErrors.ErrUnauthorized) {
		t.Errorf("expected ErrUnauthorized for a blank PAT, got %v", err)
	}
}
