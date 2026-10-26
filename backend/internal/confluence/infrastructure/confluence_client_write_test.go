package infrastructure

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	sharedErrors "lunar/backend/internal/shared/errors"
)

type updateRequestCapture struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	Title   string `json:"title"`
	Version struct {
		Number int `json:"number"`
	} `json:"version"`
	Body struct {
		Storage struct {
			Value          string `json:"value"`
			Representation string `json:"representation"`
		} `json:"storage"`
	} `json:"body"`
}

func TestConfluenceClient_UpdatePageStorageReadsVersionThenPuts(t *testing.T) {
	var getExpand string
	var putMethod string
	var putContentType string
	var captured updateRequestCapture

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodGet {
			getExpand = r.URL.Query().Get("expand")
			_, _ = w.Write([]byte(`{"id":"10","title":"Existing Title","version":{"number":4},"_links":{"webui":"/display/DEV/Old"}}`))
			return
		}
		putMethod = r.Method
		putContentType = r.Header.Get("Content-Type")
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &captured)
		_, _ = w.Write([]byte(`{"id":"10","title":"Existing Title","version":{"number":5},"_links":{"webui":"/display/DEV/Page"}}`))
	}))
	defer upstream.Close()

	client := NewConfluenceClient(upstream.URL)
	version, pageURL, err := client.UpdatePageStorage(context.Background(), testPAT, "10", "<p>body</p>")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if getExpand != "version,space" {
		t.Errorf("expected expand=version,space, got %q", getExpand)
	}
	if putMethod != http.MethodPut || putContentType != "application/json" {
		t.Errorf("expected a JSON PUT, got %s %q", putMethod, putContentType)
	}
	if captured.ID != "10" || captured.Type != "page" || captured.Title != "Existing Title" {
		t.Errorf("unexpected update identity %+v", captured)
	}
	if captured.Version.Number != 5 {
		t.Errorf("expected version 5, got %d", captured.Version.Number)
	}
	if captured.Body.Storage.Value != "<p>body</p>" || captured.Body.Storage.Representation != "storage" {
		t.Errorf("unexpected update storage %+v", captured.Body.Storage)
	}
	if version != 5 {
		t.Errorf("expected the returned version 5, got %d", version)
	}
	if pageURL != "/display/DEV/Page" {
		t.Errorf("expected the webui link, got %q", pageURL)
	}
}

func TestConfluenceClient_UpdatePageStorageFallsBackWhenUpdateVersionIsAbsent(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodGet {
			_, _ = w.Write([]byte(`{"id":"10","title":"Title","version":{"number":7}}`))
			return
		}
		_, _ = w.Write([]byte(`{"id":"10"}`))
	}))
	defer upstream.Close()

	client := NewConfluenceClient(upstream.URL)
	version, pageURL, err := client.UpdatePageStorage(context.Background(), testPAT, "10", "<p>x</p>")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if version != 8 {
		t.Errorf("expected the computed version 8, got %d", version)
	}
	if pageURL != "" {
		t.Errorf("expected no webui link, got %q", pageURL)
	}
}

func TestConfluenceClient_UpdatePageStorageUnauthorizedMapsToErrUnauthorized(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer upstream.Close()

	client := NewConfluenceClient(upstream.URL)
	_, _, err := client.UpdatePageStorage(context.Background(), testPAT, "10", "<p>x</p>")
	if !errors.Is(err, sharedErrors.ErrUnauthorized) {
		t.Fatalf("expected ErrUnauthorized, got %v", err)
	}
}

func TestConfluenceClient_UpdatePageStorageNotFoundMapsToErrNotFound(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer upstream.Close()

	client := NewConfluenceClient(upstream.URL)
	_, _, err := client.UpdatePageStorage(context.Background(), testPAT, "10", "<p>x</p>")
	if !errors.Is(err, sharedErrors.ErrNotFound) {
		t.Fatalf("expected ErrNotFound on the version read, got %v", err)
	}
}

func TestConfluenceClient_UpdatePageStorageConflictMapsToErrConflict(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodGet {
			_, _ = w.Write([]byte(`{"id":"10","title":"Title","version":{"number":3}}`))
			return
		}
		w.WriteHeader(http.StatusConflict)
	}))
	defer upstream.Close()

	client := NewConfluenceClient(upstream.URL)
	_, _, err := client.UpdatePageStorage(context.Background(), testPAT, "10", "<p>x</p>")
	if !errors.Is(err, sharedErrors.ErrConflict) {
		t.Fatalf("expected ErrConflict, got %v", err)
	}
}

func TestConfluenceClient_UpdatePageStorageNotFoundOnPutMapsToErrNotFound(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodGet {
			_, _ = w.Write([]byte(`{"id":"10","title":"Title","version":{"number":3}}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer upstream.Close()

	client := NewConfluenceClient(upstream.URL)
	_, _, err := client.UpdatePageStorage(context.Background(), testPAT, "10", "<p>x</p>")
	if !errors.Is(err, sharedErrors.ErrNotFound) {
		t.Fatalf("expected ErrNotFound on the update, got %v", err)
	}
}

func TestConfluenceClient_UpdatePageStorageUndecodableVersionBodyReturnsError(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{not json`))
	}))
	defer upstream.Close()

	client := NewConfluenceClient(upstream.URL)
	_, _, err := client.UpdatePageStorage(context.Background(), testPAT, "10", "<p>x</p>")
	if err == nil {
		t.Fatal("expected a decode error, got nil")
	}
	if errors.Is(err, sharedErrors.ErrNotFound) || errors.Is(err, sharedErrors.ErrUnauthorized) {
		t.Errorf("expected a decode error, got %v", err)
	}
	if !strings.Contains(err.Error(), "version") {
		t.Errorf("expected the version decode message, got %q", err.Error())
	}
}

func TestConfluenceClient_UpdatePageStorageUndecodableUpdateBodyReturnsError(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodGet {
			_, _ = w.Write([]byte(`{"id":"10","title":"Title","version":{"number":2}}`))
			return
		}
		_, _ = w.Write([]byte(`{not json`))
	}))
	defer upstream.Close()

	client := NewConfluenceClient(upstream.URL)
	_, _, err := client.UpdatePageStorage(context.Background(), testPAT, "10", "<p>x</p>")
	if err == nil {
		t.Fatal("expected a decode error, got nil")
	}
	if !strings.Contains(err.Error(), "update response") {
		t.Errorf("expected the update decode message, got %q", err.Error())
	}
}

func TestConfluenceClient_UpdatePageStorageInputValidation(t *testing.T) {
	client := NewConfluenceClient("https://confluence.example.com")
	ctx := context.Background()

	if _, _, err := client.UpdatePageStorage(ctx, testPAT, "10", "   "); !errors.Is(err, sharedErrors.ErrBadRequest) {
		t.Errorf("expected ErrBadRequest for a blank storage value, got %v", err)
	}
	if _, _, err := client.UpdatePageStorage(ctx, testPAT, "   ", "<p>x</p>"); !errors.Is(err, sharedErrors.ErrBadRequest) {
		t.Errorf("expected ErrBadRequest for a blank page id, got %v", err)
	}
	if _, _, err := client.UpdatePageStorage(ctx, "  ", "10", "<p>x</p>"); !errors.Is(err, sharedErrors.ErrUnauthorized) {
		t.Errorf("expected ErrUnauthorized for a blank PAT, got %v", err)
	}
}
