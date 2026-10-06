package infrastructure

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	copilotApplication "lunar/backend/internal/copilot/application"
	copilotDomain "lunar/backend/internal/copilot/domain"
	copilotInfrastructure "lunar/backend/internal/copilot/infrastructure"
	sharedErrors "lunar/backend/internal/shared/errors"
)

type capturedChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type capturedChatRequest struct {
	model          string
	authorization  string
	responseFormat string
	messages       []capturedChatMessage
	decodeErr      error
}

type stubModelProvider struct {
	models        []copilotDomain.ModelInfo
	credentials   copilotApplication.ProviderCredentials
	modelsErr     error
	credentialErr error
	resolved      []string
}

func (s *stubModelProvider) ListModels(_ context.Context, _ string) ([]copilotDomain.ModelInfo, error) {
	if s.modelsErr != nil {
		return nil, s.modelsErr
	}

	return s.models, nil
}

func (s *stubModelProvider) ResolveProviderCredentials(_ context.Context, _ string, model string) (copilotApplication.ProviderCredentials, error) {
	s.resolved = append(s.resolved, model)
	if s.credentialErr != nil {
		return copilotApplication.ProviderCredentials{}, s.credentialErr
	}

	return s.credentials, nil
}

func newChatRecorder(t *testing.T, content string, status int) (*httptest.Server, *capturedChatRequest) {
	t.Helper()

	captured := &capturedChatRequest{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Model          string                `json:"model"`
			Messages       []capturedChatMessage `json:"messages"`
			ResponseFormat *struct {
				Type string `json:"type"`
			} `json:"response_format"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			captured.decodeErr = err
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		captured.model = body.Model
		captured.messages = body.Messages
		captured.authorization = r.Header.Get("Authorization")
		if body.ResponseFormat != nil {
			captured.responseFormat = body.ResponseFormat.Type
		}
		if status != http.StatusOK {
			w.WriteHeader(status)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		response := map[string]any{"choices": []map[string]any{{"message": map[string]string{"content": content}}}}
		if err := json.NewEncoder(w).Encode(response); err != nil {
			t.Errorf("cannot write the fake answer: %v", err)
		}
	}))
	t.Cleanup(server.Close)

	return server, captured
}

func TestCompleteJSONUsesTheFastConfiguredModel(t *testing.T) {
	fencedAnswer := "```json\n{\"category\":\"Enhancement\",\"core\":\"Manage-User\",\"evidence\":\"Halaman\"}\n```"
	server, captured := newChatRecorder(t, fencedAnswer, http.StatusOK)
	provider := &stubModelProvider{
		models: []copilotDomain.ModelInfo{
			{Name: "DeepSeek-V4 Pro (Thinking)", IsConfigured: true, HasThinking: true},
			{Name: "DeepSeek Flash", IsConfigured: true, HasThinking: false},
			{Name: "Gemini 3.1 Pro", IsConfigured: true, HasThinking: false},
		},
		credentials: copilotApplication.ProviderCredentials{Provider: "deepseek", APIKey: "secret-key", Model: "deepseek-chat"},
	}
	client := NewModelJSONClient(copilotInfrastructure.NewLLMClient(server.URL, ""), provider)

	answer, err := client.CompleteJSON(context.Background(), "user-a", "Answer in JSON only.", "Name this branch.")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if answer != `{"category":"Enhancement","core":"Manage-User","evidence":"Halaman"}` {
		t.Fatalf("answer = %q", answer)
	}
	if captured.decodeErr != nil {
		t.Fatalf("the fake provider could not read the request: %v", captured.decodeErr)
	}
	if captured.model != "deepseek-chat" {
		t.Fatalf("model = %q, want the resolved fast model", captured.model)
	}
	if captured.authorization != "Bearer secret-key" {
		t.Fatalf("authorization = %q", captured.authorization)
	}
	if captured.responseFormat != "json_object" {
		t.Fatalf("response format = %q, want json_object", captured.responseFormat)
	}
	if len(captured.messages) != 2 || captured.messages[0].Role != "system" || captured.messages[0].Content != "Answer in JSON only." {
		t.Fatalf("messages = %+v", captured.messages)
	}
	if len(provider.resolved) != 1 || provider.resolved[0] != "DeepSeek Flash" {
		t.Fatalf("resolved models = %v", provider.resolved)
	}
}

func TestCompleteJSONFallsBackToTheOnlyConfiguredModel(t *testing.T) {
	server, captured := newChatRecorder(t, `{"category":"Enhancement"}`, http.StatusOK)
	provider := &stubModelProvider{
		models:      []copilotDomain.ModelInfo{{Name: "DeepSeek-V4 Pro (Thinking)", IsConfigured: true, HasThinking: true}},
		credentials: copilotApplication.ProviderCredentials{Provider: "deepseek", APIKey: "secret-key"},
	}
	client := NewModelJSONClient(copilotInfrastructure.NewLLMClient(server.URL, ""), provider)

	if _, err := client.CompleteJSON(context.Background(), "user-a", "Answer in JSON only.", "Name this branch."); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if captured.model != "deepseek-reasoner" {
		t.Fatalf("model = %q, want the resolved thinking model", captured.model)
	}
	if len(captured.messages) != 1 || !strings.Contains(captured.messages[0].Content, "Answer in JSON only.") {
		t.Fatalf("the thinking model must receive the system prompt inside the user turn: %+v", captured.messages)
	}
}

func TestCompleteJSONWithoutAConfiguredProviderFails(t *testing.T) {
	server, captured := newChatRecorder(t, `{"category":"Enhancement"}`, http.StatusOK)
	provider := &stubModelProvider{models: []copilotDomain.ModelInfo{{Name: "DeepSeek Flash", IsConfigured: false}}}
	client := NewModelJSONClient(copilotInfrastructure.NewLLMClient(server.URL, ""), provider)

	if _, err := client.CompleteJSON(context.Background(), "user-a", "system", "prompt"); !errors.Is(err, sharedErrors.ErrBadRequest) {
		t.Fatalf("error = %v, want a bad request", err)
	}
	if captured.model != "" {
		t.Fatalf("the provider must not be called, model = %q", captured.model)
	}
}

func TestCompleteJSONRejectsANonJSONAnswer(t *testing.T) {
	server, _ := newChatRecorder(t, "Manage User", http.StatusOK)
	provider := &stubModelProvider{
		models:      []copilotDomain.ModelInfo{{Name: "DeepSeek Flash", IsConfigured: true}},
		credentials: copilotApplication.ProviderCredentials{Provider: "deepseek", APIKey: "secret-key"},
	}
	client := NewModelJSONClient(copilotInfrastructure.NewLLMClient(server.URL, ""), provider)

	if _, err := client.CompleteJSON(context.Background(), "user-a", "system", "prompt"); err == nil {
		t.Fatal("expected an error for an answer that is not a JSON object")
	}
}

func TestCompleteJSONPropagatesAProviderFailure(t *testing.T) {
	server, _ := newChatRecorder(t, "", http.StatusUnauthorized)
	provider := &stubModelProvider{
		models:      []copilotDomain.ModelInfo{{Name: "DeepSeek Flash", IsConfigured: true}},
		credentials: copilotApplication.ProviderCredentials{Provider: "deepseek", APIKey: "secret-key"},
	}
	client := NewModelJSONClient(copilotInfrastructure.NewLLMClient(server.URL, ""), provider)

	if _, err := client.CompleteJSON(context.Background(), "user-a", "system", "prompt"); !errors.Is(err, sharedErrors.ErrUnauthorized) {
		t.Fatalf("error = %v, want unauthorized", err)
	}
}
