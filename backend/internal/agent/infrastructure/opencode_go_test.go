package infrastructure

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	sharedHttp "lunar/backend/internal/shared/http"
)

const openCodeGoChatResponse = `{"model":"deepseek-v4.1-flash","choices":[{"finish_reason":"stop","message":{"content":"ok"}}]}`

type capturedChatRequest struct {
	body    map[string]interface{}
	headers http.Header
}

func captureChatCompletionRequest(t *testing.T, responseBody string) (*httptest.Server, *capturedChatRequest) {
	t.Helper()

	captured := &capturedChatRequest{}
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		captured.headers = request.Header.Clone()
		body, err := io.ReadAll(request.Body)
		if err != nil {
			t.Errorf("read request body: %v", err)
			return
		}
		if err := json.Unmarshal(body, &captured.body); err != nil {
			t.Errorf("unmarshal request body: %v", err)
			return
		}
		writer.Header().Set("Content-Type", "application/json")
		if _, err := writer.Write([]byte(responseBody)); err != nil {
			t.Errorf("write response: %v", err)
		}
	}))
	return server, captured
}

func newOpenCodeGoTestClient(serverURL string, sessionID string) *ChatClient {
	return NewChatClient(ProviderConfig{
		Provider:  openCodeGoProviderName,
		APIKey:    "test-api-key",
		BaseURL:   serverURL,
		Model:     "deepseek-v4.1-flash",
		SessionID: sessionID,
		Timeout:   5 * time.Second,
	})
}

func TestChatClientOpenCodeGoOmitsDeepSeekReasoningFields(t *testing.T) {
	tests := []struct {
		name        string
		modelOption string
		wantModel   string
	}{
		{name: "upstream model passthrough", modelOption: "deepseek-v4.1-flash", wantModel: "deepseek-v4.1-flash"},
		{name: "thinking alias request", modelOption: "deepseek-flash-max", wantModel: "deepseek-flash"},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			server, captured := captureChatCompletionRequest(t, openCodeGoChatResponse)
			defer server.Close()

			client := newOpenCodeGoTestClient(server.URL, "conversation-1")
			if _, err := client.RunToolCall(context.Background(), userMessage("test"), searchCodeTool(), testCase.modelOption, "auto"); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			for _, field := range []string{thinkingFieldName, reasoningEffortFieldName, outputConfigFieldName} {
				if _, exists := captured.body[field]; exists {
					t.Errorf("opencode-go request must not contain %q", field)
				}
			}
			if captured.body["model"] != testCase.wantModel {
				t.Errorf("model = %v, want %q", captured.body["model"], testCase.wantModel)
			}
			if captured.body["temperature"] != 0.1 {
				t.Errorf("temperature = %v, want 0.1", captured.body["temperature"])
			}
		})
	}
}

func TestChatClientOpenCodeGoSendsUserAgentAndSessionHeader(t *testing.T) {
	server, captured := captureChatCompletionRequest(t, openCodeGoChatResponse)
	defer server.Close()

	client := newOpenCodeGoTestClient(server.URL, "conversation-42")
	if _, err := client.RunToolCall(context.Background(), userMessage("test"), searchCodeTool(), "deepseek-v4.1-flash", "auto"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := captured.headers.Get("User-Agent"); got != sharedHttp.ClientUserAgent {
		t.Errorf("User-Agent = %q, want %q", got, sharedHttp.ClientUserAgent)
	}
	if got := captured.headers.Get("x-opencode-session"); got != "conversation-42" {
		t.Errorf("x-opencode-session = %q, want %q", got, "conversation-42")
	}
	if got := captured.headers.Get("Authorization"); got != "Bearer test-api-key" {
		t.Errorf("Authorization = %q, want %q", got, "Bearer test-api-key")
	}
}

func TestChatClientOpenCodeGoOmitsSessionHeaderWithoutSessionID(t *testing.T) {
	server, captured := captureChatCompletionRequest(t, openCodeGoChatResponse)
	defer server.Close()

	client := newOpenCodeGoTestClient(server.URL, "")
	if _, err := client.RunToolCall(context.Background(), userMessage("test"), searchCodeTool(), "deepseek-v4.1-flash", "auto"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := captured.headers.Get("x-opencode-session"); got != "" {
		t.Errorf("x-opencode-session = %q, want empty", got)
	}
	if got := captured.headers.Get("User-Agent"); got != sharedHttp.ClientUserAgent {
		t.Errorf("User-Agent = %q, want %q", got, sharedHttp.ClientUserAgent)
	}
}

func TestChatClientOtherProvidersDoNotSendSessionHeader(t *testing.T) {
	server, captured := captureChatCompletionRequest(t, openCodeGoChatResponse)
	defer server.Close()

	client := NewChatClient(ProviderConfig{
		Provider:  "deepseek",
		APIKey:    "test-api-key",
		BaseURL:   server.URL,
		Model:     "deepseek-flash",
		SessionID: "conversation-42",
		Timeout:   5 * time.Second,
	})
	if _, err := client.RunToolCall(context.Background(), userMessage("test"), searchCodeTool(), "deepseek-flash", "auto"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := captured.headers.Get("x-opencode-session"); got != "" {
		t.Errorf("x-opencode-session = %q, want empty for a non opencode-go provider", got)
	}
	if got := captured.headers.Get("User-Agent"); got != sharedHttp.ClientUserAgent {
		t.Errorf("User-Agent = %q, want %q", got, sharedHttp.ClientUserAgent)
	}
}
