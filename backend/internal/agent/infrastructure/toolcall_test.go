package infrastructure

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestChatClient_RunToolCall_ToolChoiceAndThinkingCompatibility(t *testing.T) {
	tests := []struct {
		name                 string
		modelOption          string
		toolChoice           string
		expectedThinkingType string
		expectedModel        string
		expectedToolChoice   interface{}
	}{
		{
			name:                 "required tool choice disables thinking for a reasoner model",
			modelOption:          "deepseek-flash-thinking",
			toolChoice:           "required",
			expectedThinkingType: "disabled",
			expectedModel:        "deepseek-flash",
			expectedToolChoice:   "required",
		},
		{
			name:                 "named tool choice disables thinking and sends a function object",
			modelOption:          "deepseek-flash-thinking",
			toolChoice:           "search_code",
			expectedThinkingType: "disabled",
			expectedModel:        "deepseek-flash",
			expectedToolChoice: map[string]interface{}{
				"type":     "function",
				"function": map[string]interface{}{"name": "search_code"},
			},
		},
		{
			name:                 "auto tool choice keeps thinking enabled for a reasoner model",
			modelOption:          "deepseek-flash-thinking",
			toolChoice:           "auto",
			expectedThinkingType: "enabled",
			expectedModel:        "deepseek-flash",
			expectedToolChoice:   "auto",
		},
		{
			name:                 "none tool choice disables thinking",
			modelOption:          "deepseek-flash",
			toolChoice:           "none",
			expectedThinkingType: "disabled",
			expectedModel:        "deepseek-flash",
			expectedToolChoice:   "none",
		},
		{
			name:                 "pro option resolves to the pro model with high effort",
			modelOption:          "deepseek-pro",
			toolChoice:           "auto",
			expectedThinkingType: "enabled",
			expectedModel:        "deepseek-v4-pro",
			expectedToolChoice:   "auto",
		},
		{
			name:                 "empty choice defaults to auto",
			modelOption:          "deepseek-flash",
			toolChoice:           "",
			expectedThinkingType: "disabled",
			expectedModel:        "deepseek-flash",
			expectedToolChoice:   "auto",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			var capturedBody map[string]interface{}
			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				body, err := io.ReadAll(request.Body)
				if err != nil {
					t.Errorf("read request body: %v", err)
					return
				}
				if err := json.Unmarshal(body, &capturedBody); err != nil {
					t.Errorf("unmarshal request body: %v", err)
					return
				}
				writer.Header().Set("Content-Type", "application/json")
				response := map[string]interface{}{
					"model": "deepseek-chat",
					"choices": []map[string]interface{}{
						{"finish_reason": "stop", "message": map[string]interface{}{"content": "ok"}},
					},
				}
				if err := json.NewEncoder(writer).Encode(response); err != nil {
					t.Errorf("encode response: %v", err)
				}
			}))
			defer server.Close()

			client := newTestChatClient(server.URL)
			if _, err := client.RunToolCall(context.Background(), userMessage("test"), searchCodeTool(), testCase.modelOption, testCase.toolChoice); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			thinking, ok := capturedBody[thinkingFieldName].(map[string]interface{})
			if !ok {
				t.Fatalf("thinking field = %v, want a map", capturedBody[thinkingFieldName])
			}
			if thinking["type"] != testCase.expectedThinkingType {
				t.Errorf("thinking type = %v, want %q", thinking["type"], testCase.expectedThinkingType)
			}
			if capturedBody["model"] != testCase.expectedModel {
				t.Errorf("model = %v, want %q", capturedBody["model"], testCase.expectedModel)
			}

			switch expected := testCase.expectedToolChoice.(type) {
			case string:
				if capturedBody[toolChoiceFieldName] != expected {
					t.Errorf("tool_choice = %v, want %q", capturedBody[toolChoiceFieldName], expected)
				}
			case map[string]interface{}:
				actual, ok := capturedBody[toolChoiceFieldName].(map[string]interface{})
				if !ok {
					t.Fatalf("tool_choice = %T, want a map", capturedBody[toolChoiceFieldName])
				}
				function, ok := actual["function"].(map[string]interface{})
				if !ok || function["name"] != expected["function"].(map[string]interface{})["name"] {
					t.Errorf("tool_choice = %v, want %v", actual, expected)
				}
			}
		})
	}
}

func TestChatClient_RunToolCall_OmitsToolFieldsWithoutTools(t *testing.T) {
	var capturedBody map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		body, err := io.ReadAll(request.Body)
		if err != nil {
			t.Errorf("read request body: %v", err)
			return
		}
		if err := json.Unmarshal(body, &capturedBody); err != nil {
			t.Errorf("unmarshal request body: %v", err)
		}
		writer.Header().Set("Content-Type", "application/json")
		if _, err := writer.Write([]byte(`{"model":"deepseek-chat","choices":[{"finish_reason":"stop","message":{"content":"ok"}}]}`)); err != nil {
			t.Errorf("write response: %v", err)
		}
	}))
	defer server.Close()

	client := newTestChatClient(server.URL)
	if _, err := client.RunToolCall(context.Background(), userMessage("test"), nil, "deepseek-chat", "auto"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, exists := capturedBody["tools"]; exists {
		t.Error("tools must be omitted when no definitions are supplied")
	}
	if _, exists := capturedBody[toolChoiceFieldName]; exists {
		t.Error("tool_choice must be omitted when no definitions are supplied")
	}
}

func TestChatClient_NonReasoningProviderPassesModelThrough(t *testing.T) {
	var capturedBody map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		body, err := io.ReadAll(request.Body)
		if err != nil {
			t.Errorf("read request body: %v", err)
			return
		}
		if err := json.Unmarshal(body, &capturedBody); err != nil {
			t.Errorf("unmarshal request body: %v", err)
		}
		writer.Header().Set("Content-Type", "application/json")
		if _, err := writer.Write([]byte(`{"model":"mimo-v2.5-pro","choices":[{"finish_reason":"stop","message":{"content":"ok"}}]}`)); err != nil {
			t.Errorf("write response: %v", err)
		}
	}))
	defer server.Close()

	client := NewChatClient(ProviderConfig{
		Provider: "mimo",
		APIKey:   "test-api-key",
		BaseURL:  server.URL,
		Timeout:  5 * time.Second,
	})

	if _, err := client.RunToolCall(context.Background(), userMessage("test"), searchCodeTool(), "mimo-v2.5-pro", "auto"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedBody["model"] != "mimo-v2.5-pro" {
		t.Errorf("model = %v, want the provider model passed through", capturedBody["model"])
	}
	if _, exists := capturedBody[thinkingFieldName]; exists {
		t.Error("non-reasoning providers must not receive the thinking field")
	}
	if capturedBody["temperature"] != 0.1 {
		t.Errorf("temperature = %v, want 0.1", capturedBody["temperature"])
	}
}

func TestChatClient_RunToolCall_MissingAPIKey(t *testing.T) {
	client := NewChatClient(ProviderConfig{BaseURL: "http://127.0.0.1:1"})
	_, err := client.RunToolCall(context.Background(), userMessage("test"), nil, "deepseek-chat", "auto")
	if err == nil {
		t.Fatal("expected an error when the API key is missing")
	}
	if err.Error() != missingAPIKeyMessage {
		t.Errorf("error = %q, want %q", err.Error(), missingAPIKeyMessage)
	}
}

func TestChatClient_ResolveModelAliases(t *testing.T) {
	tests := []struct {
		option       string
		configured   string
		wantModel    string
		wantThinking bool
		wantEffort   string
	}{
		{option: "deepseek-flash-low", wantModel: "deepseek-flash", wantThinking: true, wantEffort: "low"},
		{option: "flash-thinking", wantModel: "deepseek-flash", wantThinking: true, wantEffort: "high"},
		{option: "flash-max", wantModel: "deepseek-flash", wantThinking: true, wantEffort: "max"},
		{option: "deepseek-pro", wantModel: "deepseek-v4-pro", wantThinking: true, wantEffort: "high"},
		{option: "deepseek-v4-pro-direct", wantModel: "deepseek-v4-pro", wantThinking: false, wantEffort: "none"},
		{option: "deepseek-chat", wantModel: "deepseek-flash", wantThinking: false, wantEffort: "none"},
		{option: "mimo-v2.5-pro", wantModel: "mimo-v2.5-pro", wantThinking: false, wantEffort: "none"},
		{option: "gpt-4o", wantModel: "gpt-4o", wantThinking: false, wantEffort: "none"},
		{option: "", configured: "deepseek-flash", wantModel: "deepseek-flash", wantThinking: false, wantEffort: "none"},
		{option: "", configured: "", wantModel: defaultChatModel, wantThinking: false, wantEffort: "none"},
	}

	for _, testCase := range tests {
		t.Run(testCase.option+"/"+testCase.configured, func(t *testing.T) {
			client := NewChatClient(ProviderConfig{Model: testCase.configured})
			model, thinking, effort := client.resolveModel(testCase.option)
			if model != testCase.wantModel {
				t.Errorf("model = %q, want %q", model, testCase.wantModel)
			}
			if thinking != testCase.wantThinking {
				t.Errorf("thinking = %t, want %t", thinking, testCase.wantThinking)
			}
			if effort != testCase.wantEffort {
				t.Errorf("effort = %q, want %q", effort, testCase.wantEffort)
			}
		})
	}
}
