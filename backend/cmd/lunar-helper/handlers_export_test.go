package main

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const helperExportCollection = `{
  "info": {"name": "atlas API", "schema": "https://schema.getpostman.com/json/collection/v2.1.0/collection.json"},
  "item": [
    {"name": "manage_user", "item": [
      {"name": "Get level user", "request": {"method": "GET", "header": [{"key": "Accept", "value": "application/json"}], "url": {"raw": "{{baseUrl}}/v1/user/level", "host": ["{{baseUrl}}"], "path": ["v1", "user", "level"]}}}
    ]}
  ],
  "variable": [{"key": "baseUrl", "value": "http://localhost:8080"}]
}`

func writeHelperFixture(t *testing.T, root string, relativePath string, content string) {
	t.Helper()
	fullPath := filepath.Join(root, filepath.FromSlash(relativePath))
	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		t.Fatalf("cannot create the fixture directory for %s: %v", relativePath, err)
	}
	if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
		t.Fatalf("cannot write the fixture file %s: %v", relativePath, err)
	}
}

func newExportProviderServer(t *testing.T) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/chat/completions") {
			t.Errorf("unexpected provider path %s", r.URL.Path)
		}
		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Errorf("cannot decode the provider request: %v", err)
		}
		if payload["reasoning_effort"] != "medium" {
			t.Errorf("expected the medium reasoning effort, received %v", payload["reasoning_effort"])
		}
		w.Header().Set("Content-Type", "application/json")
		response := map[string]any{"choices": []any{map[string]any{"message": map[string]any{"content": helperExportCollection}}}}
		if err := json.NewEncoder(w).Encode(response); err != nil {
			t.Errorf("cannot write the provider response: %v", err)
		}
	}))
	t.Cleanup(server.Close)
	return server
}

func newProviderKeyServer(t *testing.T, providerBaseURL string, statusCode int) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/copilot/provider-key" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if r.Header.Get("Authorization") != "Bearer session-token" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if statusCode != http.StatusOK {
			w.WriteHeader(statusCode)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "no AI provider configured"})
			return
		}
		credentials := map[string]string{
			"provider": "deepseek",
			"api_key":  "provider-key",
			"base_url": providerBaseURL,
			"model":    "deepseek-chat",
		}
		if err := json.NewEncoder(w).Encode(credentials); err != nil {
			t.Errorf("cannot write the credentials: %v", err)
		}
	}))
	t.Cleanup(server.Close)
	return server
}

func newExportRequest(t *testing.T, workspaceRoot string, repositoryName string, backendURL string) *http.Request {
	t.Helper()
	body := map[string]string{
		"model":         "deepseek-chat",
		"backend_url":   backendURL,
		"session_token": "session-token",
	}
	encoded, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("cannot encode the export request: %v", err)
	}
	target := "/export?root=" + url.QueryEscape(workspaceRoot) + "&name=" + url.QueryEscape(repositoryName)
	return httptest.NewRequest(http.MethodPost, target, strings.NewReader(string(encoded)))
}

func newHelperExportServer(workspaceRoot string) *helperServer {
	return &helperServer{
		allowedRoots: []string{workspaceRoot},
		logger:       slog.New(slog.NewTextHandler(os.Stderr, nil)),
	}
}

func TestHelperExportStreamsTheValidatedCollection(t *testing.T) {
	workspaceRoot := t.TempDir()
	writeHelperFixture(t, workspaceRoot, "atlas/src/manage_user/routes.go", "package manage_user\n\nimport \"github.com/gin-gonic/gin\"\n\nfunc Register(r *gin.Engine) { route := r.Group(\"/v1/user\"); route.GET(\"/level\", nil) }\n")
	writeHelperFixture(t, workspaceRoot, "atlas/src/wire_gen.go", "package main\n")

	provider := newExportProviderServer(t)
	backend := newProviderKeyServer(t, provider.URL, http.StatusOK)

	server := newHelperExportServer(workspaceRoot)
	recorder := httptest.NewRecorder()
	server.handleExport(recorder, newExportRequest(t, workspaceRoot, "atlas", backend.URL))

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, received %d with body %s", recorder.Code, recorder.Body.String())
	}
	if contentType := recorder.Header().Get("Content-Type"); contentType != "text/event-stream" {
		t.Fatalf("expected text/event-stream, received %q", contentType)
	}

	stream := recorder.Body.String()
	for _, marker := range []string{
		"event: stage",
		`"stage":"selecting"`,
		`"stage":"prompt"`,
		`"stage":"model"`,
		`"stage":"validating"`,
		"event: result",
		`"service":"atlas"`,
		`"request_count":1`,
		`"folder_count":1`,
		`"collection":{`,
	} {
		if !strings.Contains(stream, marker) {
			t.Fatalf("expected the stream to contain %s, received %s", marker, stream)
		}
	}
	if strings.Contains(strings.ToLower(stream), "effort") {
		t.Fatalf("the stream must not mention an effort setting")
	}
}

func TestHelperExportReturnsConflictWithoutAProviderKey(t *testing.T) {
	workspaceRoot := t.TempDir()
	writeHelperFixture(t, workspaceRoot, "atlas/src/manage_user/routes.go", "package manage_user\n\nimport \"github.com/gin-gonic/gin\"\n\nfunc Register(r *gin.Engine) { route := r.Group(\"/v1/user\"); route.GET(\"/level\", nil) }\n")

	backend := newProviderKeyServer(t, "http://127.0.0.1:1", http.StatusBadRequest)

	server := newHelperExportServer(workspaceRoot)
	recorder := httptest.NewRecorder()
	server.handleExport(recorder, newExportRequest(t, workspaceRoot, "atlas", backend.URL))

	if recorder.Code != http.StatusConflict {
		t.Fatalf("expected 409, received %d with body %s", recorder.Code, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), "no provider key") {
		t.Fatalf("expected a provider key reason, received %s", recorder.Body.String())
	}
}

func TestHelperExportRefusesAFrontendRepository(t *testing.T) {
	workspaceRoot := t.TempDir()
	writeHelperFixture(t, workspaceRoot, "fe-mms/src/main.js", "import Vue from \"vue\"\n")
	writeHelperFixture(t, workspaceRoot, "fe-mms/package.json", `{"dependencies":{"vue":"^3.4.0"}}`)

	provider := newExportProviderServer(t)
	backend := newProviderKeyServer(t, provider.URL, http.StatusOK)

	server := newHelperExportServer(workspaceRoot)
	recorder := httptest.NewRecorder()
	server.handleExport(recorder, newExportRequest(t, workspaceRoot, "fe-mms", backend.URL))

	if recorder.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, received %d with body %s", recorder.Code, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), "vue") {
		t.Fatalf("expected a vue reason, received %s", recorder.Body.String())
	}
}

func TestHelperExportRefusesAnUnknownRepository(t *testing.T) {
	workspaceRoot := t.TempDir()
	server := newHelperExportServer(workspaceRoot)
	recorder := httptest.NewRecorder()

	server.handleExport(recorder, newExportRequest(t, workspaceRoot, "missing-repository", "http://127.0.0.1:1"))

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("expected 404, received %d with body %s", recorder.Code, recorder.Body.String())
	}
}
func TestHelperExportRouteIsGuardedByTheTokenAndStreams(t *testing.T) {
	workspaceRoot := t.TempDir()
	writeHelperFixture(t, workspaceRoot, "atlas/src/manage_user/routes.go", "package manage_user\n\nimport \"github.com/gin-gonic/gin\"\n\nfunc Register(r *gin.Engine) { route := r.Group(\"/v1/user\"); route.GET(\"/level\", nil) }\n")

	provider := newExportProviderServer(t)
	backend := newProviderKeyServer(t, provider.URL, http.StatusOK)

	handler, stop := newHelperHandler(helperDependencies{
		allowedRoots: []string{workspaceRoot},
		token:        "helper-token",
		logger:       slog.New(slog.NewTextHandler(os.Stderr, nil)),
		version:      "test",
	})
	defer stop()

	unauthorized := httptest.NewRecorder()
	handler.ServeHTTP(unauthorized, newExportRequest(t, workspaceRoot, "atlas", backend.URL))
	if unauthorized.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without the helper token, received %d", unauthorized.Code)
	}

	authorized := httptest.NewRecorder()
	request := newExportRequest(t, workspaceRoot, "atlas", backend.URL)
	request.Header.Set("Authorization", "Bearer helper-token")
	handler.ServeHTTP(authorized, request)

	if authorized.Code != http.StatusOK {
		t.Fatalf("expected 200 through the helper handler chain, received %d with body %s", authorized.Code, authorized.Body.String())
	}
	if !strings.Contains(authorized.Body.String(), "event: result") {
		t.Fatalf("expected a result event, received %s", authorized.Body.String())
	}
}
