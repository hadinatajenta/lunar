package infrastructure

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"lunar/backend/internal/postman/domain"
)

func writeCompletion(t *testing.T, w http.ResponseWriter, content string) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	payload := map[string]any{
		"choices": []any{
			map[string]any{"message": map[string]any{"content": content}},
		},
	}
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		t.Errorf("cannot write the provider response: %v", err)
	}
}

func TestProviderClientSendsTheMediumEffortToReasoningProviders(t *testing.T) {
	captured := make(chan map[string]any, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Errorf("cannot decode the provider request: %v", err)
		}
		captured <- payload
		writeCompletion(t, w, "collection-content")
	}))
	defer server.Close()

	client := NewProviderClient()
	content, err := client.Generate(context.Background(), domain.GenerateRequest{
		Provider: "deepseek",
		APIKey:   "key",
		BaseURL:  server.URL,
		Model:    "deepseek-chat",
		System:   "system prompt",
		User:     "user prompt",
	})
	if err != nil {
		t.Fatalf("generating the collection failed: %v", err)
	}
	if content != "collection-content" {
		t.Fatalf("expected the provider content, received %q", content)
	}

	payload := <-captured
	if payload["reasoning_effort"] != "medium" {
		t.Fatalf("expected the medium reasoning effort, received %v", payload["reasoning_effort"])
	}
	if payload["model"] != "deepseek-chat" {
		t.Fatalf("expected the upstream model, received %v", payload["model"])
	}
	messages, ok := payload["messages"].([]any)
	if !ok || len(messages) != 2 {
		t.Fatalf("expected a system and a user message, received %v", payload["messages"])
	}
}

func TestProviderClientUsesTemperatureForOtherProviders(t *testing.T) {
	captured := make(chan map[string]any, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Errorf("cannot decode the provider request: %v", err)
		}
		captured <- payload
		writeCompletion(t, w, "collection-content")
	}))
	defer server.Close()

	client := NewProviderClient()
	if _, err := client.Generate(context.Background(), domain.GenerateRequest{Provider: "gemini", APIKey: "key", BaseURL: server.URL, Model: "gemini-1.5-pro"}); err != nil {
		t.Fatalf("generating the collection failed: %v", err)
	}

	payload := <-captured
	if _, found := payload["reasoning_effort"]; found {
		t.Fatalf("a provider without reasoning support must not receive an effort field")
	}
	if payload["temperature"] == nil {
		t.Fatalf("expected a temperature for a provider without reasoning support")
	}
}

func TestProviderClientKeepsTheEffortOutOfErrorText(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		payload := map[string]any{"error": map[string]any{"message": "the model is overloaded"}}
		if err := json.NewEncoder(w).Encode(payload); err != nil {
			t.Errorf("cannot write the provider error: %v", err)
		}
	}))
	defer server.Close()

	client := NewProviderClient()
	_, err := client.Generate(context.Background(), domain.GenerateRequest{Provider: "deepseek", APIKey: "key", BaseURL: server.URL, Model: "deepseek-chat"})
	if err == nil {
		t.Fatalf("expected a provider failure")
	}
	if !strings.Contains(err.Error(), "the model is overloaded") {
		t.Fatalf("expected the provider message, received %q", err.Error())
	}
	if strings.Contains(strings.ToLower(err.Error()), "effort") {
		t.Fatalf("the error text must not mention an effort setting: %q", err.Error())
	}
}

func TestProviderClientMapsATimeoutToAProviderTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(300 * time.Millisecond)
		writeCompletion(t, w, "collection-content")
	}))
	defer server.Close()

	client := NewProviderClient()
	client.httpClient = &http.Client{Timeout: 20 * time.Millisecond}
	_, err := client.Generate(context.Background(), domain.GenerateRequest{Provider: "deepseek", APIKey: "key", BaseURL: server.URL, Model: "deepseek-chat"})
	if !errors.Is(err, domain.ErrProviderTimeout) {
		t.Fatalf("expected ErrProviderTimeout, received %v", err)
	}
}

func TestProviderClientRefusesAnEmptyCompletion(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeCompletion(t, w, "   ")
	}))
	defer server.Close()

	client := NewProviderClient()
	_, err := client.Generate(context.Background(), domain.GenerateRequest{Provider: "deepseek", APIKey: "key", BaseURL: server.URL, Model: "deepseek-chat"})
	if !errors.Is(err, domain.ErrCollectionInvalid) {
		t.Fatalf("expected ErrCollectionInvalid, received %v", err)
	}
}

func TestProviderClientSendsAnOpenCodeGoSessionHeader(t *testing.T) {
	capturedHeaders := make(chan http.Header, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedHeaders <- r.Header.Clone()
		writeCompletion(t, w, "collection-content")
	}))
	defer server.Close()

	client := NewProviderClient()
	if _, err := client.Generate(context.Background(), domain.GenerateRequest{
		Provider: openCodeGoProvider,
		APIKey:   "key",
		BaseURL:  server.URL,
		Model:    "deepseek-v4.1-flash",
	}); err != nil {
		t.Fatalf("generating the collection failed: %v", err)
	}

	header := <-capturedHeaders
	sessionID := header.Get(openCodeGoSessionHeader)
	if _, err := uuid.Parse(sessionID); err != nil {
		t.Fatalf("%s = %q, want a UUID: %v", openCodeGoSessionHeader, sessionID, err)
	}
}

func TestProviderClientOmitsTheOpenCodeGoSessionHeaderForOtherProviders(t *testing.T) {
	capturedHeaders := make(chan http.Header, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedHeaders <- r.Header.Clone()
		writeCompletion(t, w, "collection-content")
	}))
	defer server.Close()

	client := NewProviderClient()
	if _, err := client.Generate(context.Background(), domain.GenerateRequest{
		Provider: "deepseek",
		APIKey:   "key",
		BaseURL:  server.URL,
		Model:    "deepseek-chat",
	}); err != nil {
		t.Fatalf("generating the collection failed: %v", err)
	}

	header := <-capturedHeaders
	if got := header.Get(openCodeGoSessionHeader); got != "" {
		t.Fatalf("%s = %q, want empty for a non opencode-go provider", openCodeGoSessionHeader, got)
	}
}
