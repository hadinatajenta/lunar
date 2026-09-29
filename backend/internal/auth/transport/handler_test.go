package transport

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"lunar/backend/internal/auth/application"
	"lunar/backend/internal/auth/domain"
	"lunar/backend/internal/auth/infrastructure"
	sharedAuth "lunar/backend/internal/shared/auth"
	sharedDatabase "lunar/backend/internal/shared/database"
)

func setupTestMux(t *testing.T) (http.Handler, string) {
	t.Helper()
	db, err := sharedDatabase.OpenDB(":memory:")
	if err != nil {
		t.Fatalf("failed to open in-memory db: %v", err)
	}

	if err := sharedDatabase.AutoMigrate(db); err != nil {
		_ = db.Close()
		t.Fatalf("failed to auto migrate: %v", err)
	}

	userRepo := infrastructure.NewSQLiteUserRepository(db)
	vaultRepo := infrastructure.NewSQLiteVaultRepository(db)

	encryptionKey := "12345678901234567890123456789012"
	jwtSecret := "integration-test-jwt-secret-key-12"
	jwtTTL := time.Hour

	authService := application.NewAuthService(userRepo, vaultRepo, encryptionKey, jwtSecret, jwtTTL)
	handler := NewAuthHandler(authService)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/auth/register", handler.Register)
	mux.HandleFunc("POST /api/auth/login", handler.Login)

	authMiddleware := sharedAuth.RequireAuth(jwtSecret)
	mux.Handle("GET /api/auth/me", authMiddleware(http.HandlerFunc(handler.Me)))
	mux.Handle("GET /api/auth/secrets", authMiddleware(http.HandlerFunc(handler.GetSecrets)))
	mux.Handle("PUT /api/auth/secrets", authMiddleware(http.HandlerFunc(handler.SaveSecrets)))

	return mux, jwtSecret
}

func TestAuthHTTPFlow(t *testing.T) {
	router, _ := setupTestMux(t)

	registerBody := `{"email":"transport@example.com","password":"Password123!","full_name":"Transport User"}`
	req := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewBufferString(registerBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected status 201 Created, got %d: %s", w.Code, w.Body.String())
	}

	var authRes application.AuthResponse
	if err := json.Unmarshal(w.Body.Bytes(), &authRes); err != nil {
		t.Fatalf("failed to parse auth response: %v", err)
	}

	if authRes.Token == "" {
		t.Fatalf("expected non-empty auth token")
	}

	unauthReq := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
	unauthW := httptest.NewRecorder()
	router.ServeHTTP(unauthW, unauthReq)

	if unauthW.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized, got %d", unauthW.Code)
	}

	meReq := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
	meReq.Header.Set("Authorization", "Bearer "+authRes.Token)
	meW := httptest.NewRecorder()
	router.ServeHTTP(meW, meReq)

	if meW.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", meW.Code, meW.Body.String())
	}

	secretsBody := `{"jira_pat":"jira-secret-key-123","jira_username":"jira.user","ai_keys":{"deepseek":"ds-key"}}`
	putReq := httptest.NewRequest(http.MethodPut, "/api/auth/secrets", bytes.NewBufferString(secretsBody))
	putReq.Header.Set("Authorization", "Bearer "+authRes.Token)
	putReq.Header.Set("Content-Type", "application/json")
	putW := httptest.NewRecorder()
	router.ServeHTTP(putW, putReq)

	if putW.Code != http.StatusOK {
		t.Fatalf("expected 200 OK on saving secrets, got %d: %s", putW.Code, putW.Body.String())
	}

	var redacted domain.RedactedSecrets
	if err := json.Unmarshal(putW.Body.Bytes(), &redacted); err != nil {
		t.Fatalf("failed to parse redacted secrets response: %v", err)
	}
	if !redacted.HasJiraPAT {
		t.Fatalf("expected HasJiraPAT to be true")
	}
	if redacted.JiraUsername != "jira.user" {
		t.Fatalf("expected jira.user, got %s", redacted.JiraUsername)
	}

	getSecretsReq := httptest.NewRequest(http.MethodGet, "/api/auth/secrets", nil)
	getSecretsReq.Header.Set("Authorization", "Bearer "+authRes.Token)
	getSecretsW := httptest.NewRecorder()
	router.ServeHTTP(getSecretsW, getSecretsReq)

	if getSecretsW.Code != http.StatusOK {
		t.Fatalf("expected 200 OK on getting secrets, got %d", getSecretsW.Code)
	}
}
