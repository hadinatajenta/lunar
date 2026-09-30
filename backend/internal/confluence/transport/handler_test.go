package transport

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	authApp "lunar/backend/internal/auth/application"
	authDomain "lunar/backend/internal/auth/domain"
	confluenceApp "lunar/backend/internal/confluence/application"
	confluenceInfra "lunar/backend/internal/confluence/infrastructure"
	sharedAuth "lunar/backend/internal/shared/auth"
	"lunar/backend/internal/shared/crypto"
	sharedErrors "lunar/backend/internal/shared/errors"
)

const (
	handlerEncryptionKey  = "0123456789abcdef0123456789abcdef"
	handlerConfiguredUser = "usr-configured"
	handlerUnconfigured   = "usr-unconfigured"
	handlerConfluencePAT  = "handler-confluence-pat"
	handlerListUpstream   = `{"results":[{"id":"101","title":"UT Coverage Report","space":{"key":"DEV","name":"DEV Team"},"version":{"when":"2026-01-02T03:04:05.000+07:00","by":{"displayName":"Jane Doe"}},"_links":{"base":"https://confluence.example.com","webui":"/display/DEV/UT-Coverage-Report"}}]}`
	handlerEmptyUpstream  = `{"results":[]}`
	handlerDetailUpstream = `{"id":"101","title":"UT Coverage Report","space":{"key":"DEV","name":"DEV Team"},"version":{"when":"2026-01-02T03:04:05.000+07:00","by":{"displayName":"Jane Doe"}},"metadata":{"labels":{"results":[{"name":"unit-test"}]}},"body":{"storage":{"value":"<p>Coverage   &amp; regression</p>"}},"_links":{"base":"https://confluence.example.com","webui":"/display/DEV/UT-Coverage-Report"}}`
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

func setupTestHandler(t *testing.T, upstream http.HandlerFunc) (*ConfluenceHandler, string, string) {
	t.Helper()

	upstreamServer := httptest.NewServer(upstream)
	t.Cleanup(upstreamServer.Close)

	vaultRepo := &mockVaultRepo{
		secrets: make(map[string]*authDomain.UserSecrets),
	}

	validPATEnc, _ := crypto.EncryptSecret(handlerConfluencePAT, handlerEncryptionKey)
	vaultRepo.secrets[handlerConfiguredUser] = &authDomain.UserSecrets{
		UserID:           handlerConfiguredUser,
		ConfluencePATEnc: validPATEnc,
		UpdatedAt:        time.Now().UTC(),
	}
	vaultRepo.secrets[handlerUnconfigured] = &authDomain.UserSecrets{
		UserID:    handlerUnconfigured,
		UpdatedAt: time.Now().UTC(),
	}

	authService := authApp.NewAuthService(nil, vaultRepo, handlerEncryptionKey, "jwt-secret", time.Hour)
	client := confluenceInfra.NewConfluenceClient(upstreamServer.URL)
	service := confluenceApp.NewConfluenceService(client, authService)
	return NewConfluenceHandler(service), handlerConfiguredUser, handlerUnconfigured
}

func performRequest(t *testing.T, handlerFunc http.HandlerFunc, userID string, target string) *httptest.ResponseRecorder {
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

func performDetailRequest(t *testing.T, handler *ConfluenceHandler, userID string, documentID string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/api/confluence/documents/"+documentID, nil)
	req.SetPathValue("id", documentID)
	ctx := sharedAuth.ContextWithUser(req.Context(), &sharedAuth.UserContext{
		UserID: userID,
		Email:  "developer@bri.co.id",
	})
	rec := httptest.NewRecorder()
	handler.GetDocument(rec, req.WithContext(ctx))
	return rec
}

func TestConfluenceHandler_UnauthenticatedReturns401WithoutUpstreamHit(t *testing.T) {
	var upstreamHits atomic.Int64
	handler, _, _ := setupTestHandler(t, func(w http.ResponseWriter, r *http.Request) {
		upstreamHits.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(handlerListUpstream))
	})

	rec := performRequest(t, handler.ListDocuments, "", "/api/confluence/documents")

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
	if hits := upstreamHits.Load(); hits != 0 {
		t.Errorf("expected no upstream hit for unauthenticated request, got %d", hits)
	}
}

func TestConfluenceHandler_MissingPATReturns401(t *testing.T) {
	var upstreamHits atomic.Int64
	handler, _, unconfiguredUser := setupTestHandler(t, func(w http.ResponseWriter, r *http.Request) {
		upstreamHits.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(handlerListUpstream))
	})

	rec := performRequest(t, handler.ListDocuments, unconfiguredUser, "/api/confluence/documents")

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "Confluence PAT is not configured") {
		t.Errorf("expected missing PAT message, got %q", rec.Body.String())
	}
	if hits := upstreamHits.Load(); hits != 0 {
		t.Errorf("expected no upstream hit without PAT, got %d", hits)
	}
}

func TestConfluenceHandler_UpstreamFailureReturns502(t *testing.T) {
	handler, configuredUser, _ := setupTestHandler(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	rec := performRequest(t, handler.ListDocuments, configuredUser, "/api/confluence/documents")

	if rec.Code != http.StatusBadGateway {
		t.Fatalf("expected 502, got %d", rec.Code)
	}
}

func TestConfluenceHandler_UnknownDocumentIDReturns404(t *testing.T) {
	handler, configuredUser, _ := setupTestHandler(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})

	rec := performDetailRequest(t, handler, configuredUser, "999")

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestConfluenceHandler_ListDocumentsMatchesContractShape(t *testing.T) {
	handler, configuredUser, _ := setupTestHandler(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(handlerListUpstream))
	})

	rec := performRequest(t, handler.ListDocuments, configuredUser, "/api/confluence/documents")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var payload map[string]json.RawMessage
	if err := json.NewDecoder(rec.Body).Decode(&payload); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(payload) != 2 {
		t.Fatalf("expected exactly documents and total keys, got %v", payload)
	}

	documentsRaw, hasDocuments := payload["documents"]
	if !hasDocuments {
		t.Fatal("expected documents key in response")
	}
	totalRaw, hasTotal := payload["total"]
	if !hasTotal {
		t.Fatal("expected total key in response")
	}

	var documents []map[string]json.RawMessage
	if err := json.Unmarshal(documentsRaw, &documents); err != nil {
		t.Fatalf("failed to decode documents: %v", err)
	}
	if len(documents) != 1 {
		t.Fatalf("expected 1 document, got %d", len(documents))
	}
	if string(totalRaw) != "1" {
		t.Errorf("expected total 1, got %s", string(totalRaw))
	}

	document := documents[0]
	expectedFields := map[string]string{
		"id":          "101",
		"type":        "ut",
		"type_label":  "UT",
		"title":       "UT Coverage Report",
		"status":      "open",
		"owner":       "Jane Doe",
		"updated":     "2026-01-02T03:04:05.000+07:00",
		"space":       "DEV Team",
		"description": "",
		"url":         "https://confluence.example.com/display/DEV/UT-Coverage-Report",
	}
	if len(document) != len(expectedFields) {
		t.Fatalf("expected exactly %d document fields, got %v", len(expectedFields), document)
	}
	for fieldName, expectedValue := range expectedFields {
		rawValue, exists := document[fieldName]
		if !exists {
			t.Errorf("expected field %q in document", fieldName)
			continue
		}
		var actualValue string
		if err := json.Unmarshal(rawValue, &actualValue); err != nil {
			t.Errorf("expected string value for field %q, got %s", fieldName, string(rawValue))
			continue
		}
		if actualValue != expectedValue {
			t.Errorf("expected field %q to be %q, got %q", fieldName, expectedValue, actualValue)
		}
	}
}

func TestConfluenceHandler_ListDocumentsEmptyResultsReturnEmptyArray(t *testing.T) {
	handler, configuredUser, _ := setupTestHandler(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(handlerEmptyUpstream))
	})

	rec := performRequest(t, handler.ListDocuments, configuredUser, "/api/confluence/documents")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"documents":[]`) {
		t.Errorf("expected empty documents array, got %s", rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"total":0`) {
		t.Errorf("expected total 0, got %s", rec.Body.String())
	}
}

func TestConfluenceHandler_GetDocumentReturnsDocument(t *testing.T) {
	handler, configuredUser, _ := setupTestHandler(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(handlerDetailUpstream))
	})

	rec := performDetailRequest(t, handler, configuredUser, "101")

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var document map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&document); err != nil {
		t.Fatalf("failed to decode document: %v", err)
	}
	if document["id"] != "101" {
		t.Errorf("expected id 101, got %q", document["id"])
	}
	if document["type"] != "ut" || document["type_label"] != "UT" {
		t.Errorf("expected type ut/UT, got %q/%q", document["type"], document["type_label"])
	}
	if document["description"] != "Coverage & regression" {
		t.Errorf("expected plain text description, got %q", document["description"])
	}
	if document["owner"] != "Jane Doe" {
		t.Errorf("expected owner Jane Doe, got %q", document["owner"])
	}
	if document["url"] != "https://confluence.example.com/display/DEV/UT-Coverage-Report" {
		t.Errorf("expected absolute URL, got %q", document["url"])
	}
}
