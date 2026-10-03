package infrastructure

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"lunar/backend/internal/agent/domain"
)

func TestChatClient_StreamAnswer_ResolvesThinkingConfiguration(t *testing.T) {
	tests := []struct {
		name                 string
		modelOption          string
		expectedModel        string
		expectedThinkingType string
		expectedEffort       string
	}{
		{
			name:                 "pro option enables thinking with high effort",
			modelOption:          "deepseek-pro",
			expectedModel:        "deepseek-v4-pro",
			expectedThinkingType: "enabled",
			expectedEffort:       "high",
		},
		{
			name:                 "flash option disables thinking",
			modelOption:          "deepseek-flash",
			expectedModel:        "deepseek-flash",
			expectedThinkingType: "disabled",
		},
		{
			name:                 "flash max option enables thinking with max effort",
			modelOption:          "flash-max",
			expectedModel:        "deepseek-flash",
			expectedThinkingType: "enabled",
			expectedEffort:       "max",
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
				}
				writer.Header().Set("Content-Type", "text/event-stream")
				writeSSEPayload(t, writer, map[string]interface{}{
					"choices": []map[string]interface{}{
						{"delta": map[string]interface{}{"content": "ok"}, "finish_reason": nil},
					},
				})
				if _, err := fmt.Fprint(writer, "data: [DONE]\n\n"); err != nil {
					t.Errorf("write done marker: %v", err)
				}
			}))
			defer server.Close()

			client := newTestChatClient(server.URL)
			if err := client.StreamAnswer(context.Background(), userMessage("Answer"), testCase.modelOption, func(chunk string) error {
				return nil
			}); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if capturedBody["model"] != testCase.expectedModel {
				t.Errorf("model = %v, want %q", capturedBody["model"], testCase.expectedModel)
			}
			if capturedBody["stream"] != true {
				t.Errorf("stream = %v, want true", capturedBody["stream"])
			}
			thinking, ok := capturedBody[thinkingFieldName].(map[string]interface{})
			if !ok {
				t.Fatalf("thinking = %v, want a map", capturedBody[thinkingFieldName])
			}
			if thinking["type"] != testCase.expectedThinkingType {
				t.Errorf("thinking type = %v, want %q", thinking["type"], testCase.expectedThinkingType)
			}
			if testCase.expectedEffort == "" {
				return
			}
			if capturedBody[reasoningEffortFieldName] != testCase.expectedEffort {
				t.Errorf("reasoning_effort = %v, want %q", capturedBody[reasoningEffortFieldName], testCase.expectedEffort)
			}
			outputConfig, ok := capturedBody[outputConfigFieldName].(map[string]interface{})
			if !ok || outputConfig["effort"] != testCase.expectedEffort {
				t.Errorf("output_config = %v, want effort %q", capturedBody[outputConfigFieldName], testCase.expectedEffort)
			}
		})
	}
}

func TestChatClient_RunToolCall_SanitizesAssistantReasoning(t *testing.T) {
	tests := []struct {
		name             string
		modelOption      string
		toolChoice       string
		wantReasoning    bool
		wantReasoningSet string
	}{
		{
			name:             "thinking enabled injects the reasoning fallback",
			modelOption:      "deepseek-flash-thinking",
			toolChoice:       "auto",
			wantReasoning:    true,
			wantReasoningSet: reasoningFallbackValue,
		},
		{
			name:          "thinking disabled leaves the reasoning empty",
			modelOption:   "deepseek-flash-thinking",
			toolChoice:    "required",
			wantReasoning: false,
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
				}
				writer.Header().Set("Content-Type", "application/json")
				if _, err := writer.Write([]byte(`{"model":"deepseek-flash","choices":[{"finish_reason":"stop","message":{"content":"ok"}}]}`)); err != nil {
					t.Errorf("write response: %v", err)
				}
			}))
			defer server.Close()

			messages := []domain.ChatMessage{
				{Role: "user", Content: domain.TextMessage("Find the payment route")},
				{
					Role:    "assistant",
					Content: domain.MessageContent{},
					ToolCalls: []domain.ToolCall{{
						ID:       "call_1",
						Type:     "function",
						Function: domain.ToolCallFunction{Name: "search_code", Arguments: `{"query":"payment"}`},
					}},
				},
				{Role: "tool", Content: domain.TextMessage("result"), ToolCallID: "call_1"},
			}

			client := newTestChatClient(server.URL)
			if _, err := client.RunToolCall(context.Background(), messages, searchCodeTool(), testCase.modelOption, testCase.toolChoice); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			encodedMessages, ok := capturedBody["messages"].([]interface{})
			if !ok || len(encodedMessages) != 3 {
				t.Fatalf("messages = %v, want 3 entries", capturedBody["messages"])
			}
			assistantMessage, ok := encodedMessages[1].(map[string]interface{})
			if !ok {
				t.Fatalf("assistant message = %T, want an object", encodedMessages[1])
			}

			reasoning, hasReasoning := assistantMessage["reasoning_content"]
			if hasReasoning != testCase.wantReasoning {
				t.Fatalf("reasoning_content present = %t, want %t", hasReasoning, testCase.wantReasoning)
			}
			if testCase.wantReasoning && reasoning != testCase.wantReasoningSet {
				t.Errorf("reasoning_content = %v, want %q", reasoning, testCase.wantReasoningSet)
			}
		})
	}
}
