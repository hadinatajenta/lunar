package infrastructure

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestChatClient_RunToolCall_ParsesToolCallsAndReasoning(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		response := map[string]interface{}{
			"model": "deepseek-flash",
			"choices": []map[string]interface{}{
				{
					"finish_reason": "tool_calls",
					"message": map[string]interface{}{
						"role":              "assistant",
						"content":           "",
						"reasoning_content": "I should search the index.",
						"tool_calls": []map[string]interface{}{
							{
								"id":   "call_abc",
								"type": "function",
								"function": map[string]interface{}{
									"name":      "search_code",
									"arguments": `{"query":"payment"}`,
								},
							},
						},
					},
				},
			},
		}
		if err := json.NewEncoder(writer).Encode(response); err != nil {
			t.Errorf("encode response: %v", err)
		}
	}))
	defer server.Close()

	client := newTestChatClient(server.URL)
	response, err := client.RunToolCall(context.Background(), userMessage("Find the payment flow"), searchCodeTool(), "deepseek-flash", "auto")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if response.FinishReason != "tool_calls" {
		t.Errorf("finish_reason = %q, want tool_calls", response.FinishReason)
	}
	if response.ReasoningContent != "I should search the index." {
		t.Errorf("reasoning = %q, want the provider reasoning", response.ReasoningContent)
	}
	if response.Model != "deepseek-flash" {
		t.Errorf("model = %q, want deepseek-flash", response.Model)
	}
	if len(response.ToolCalls) != 1 {
		t.Fatalf("tool calls = %d, want 1", len(response.ToolCalls))
	}
	if response.ToolCalls[0].ID != "call_abc" || response.ToolCalls[0].Function.Name != "search_code" {
		t.Errorf("tool call = %+v, want the parsed identity", response.ToolCalls[0])
	}
	if response.ToolCalls[0].Function.Arguments != `{"query":"payment"}` {
		t.Errorf("arguments = %q, want the raw argument string", response.ToolCalls[0].Function.Arguments)
	}
	if !response.Content.IsEmpty() {
		t.Errorf("content = %q, want empty for a tool call round", response.Content.Text())
	}
}

func TestChatClient_RunToolCall_DSMLFallback(t *testing.T) {
	dsml := `<|DSML|calls>
<|DSML|invoke name="search_code">
<|DSML|parameter name="query" string="true">settlement route</|DSML|parameter>
</|DSML|invoke>
</|DSML|calls>`

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		response := map[string]interface{}{
			"model": "deepseek-chat",
			"choices": []map[string]interface{}{
				{"finish_reason": "stop", "message": map[string]interface{}{"content": dsml}},
			},
		}
		if err := json.NewEncoder(writer).Encode(response); err != nil {
			t.Errorf("encode response: %v", err)
		}
	}))
	defer server.Close()

	client := newTestChatClient(server.URL)
	response, err := client.RunToolCall(context.Background(), userMessage("Find the settlement route"), nil, "deepseek-chat", "auto")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(response.ToolCalls) != 1 {
		t.Fatalf("tool calls = %d, want the DSML fallback to parse 1", len(response.ToolCalls))
	}
	if response.ToolCalls[0].Function.Name != "search_code" {
		t.Errorf("name = %q, want search_code", response.ToolCalls[0].Function.Name)
	}
	if response.FinishReason != "tool_calls" {
		t.Errorf("finish_reason = %q, want it overridden to tool_calls", response.FinishReason)
	}
	if !response.Content.IsEmpty() {
		t.Errorf("content = %q, want the DSML markup stripped", response.Content.Text())
	}
}

func TestChatClient_RunToolCall_ProviderError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		writer.WriteHeader(http.StatusTooManyRequests)
		response := map[string]interface{}{
			"error": map[string]interface{}{
				"message": "Rate limit exceeded",
				"code":    "rate_limit_exceeded",
			},
		}
		if err := json.NewEncoder(writer).Encode(response); err != nil {
			t.Errorf("encode response: %v", err)
		}
	}))
	defer server.Close()

	client := newTestChatClient(server.URL)
	_, err := client.RunToolCall(context.Background(), userMessage("test"), nil, "deepseek-chat", "auto")
	if err == nil {
		t.Fatal("expected a provider error")
	}

	var providerError *ProviderError
	if !errors.As(err, &providerError) {
		t.Fatalf("error = %v, want a ProviderError", err)
	}
	if providerError.StatusCode != http.StatusTooManyRequests {
		t.Errorf("status = %d, want 429", providerError.StatusCode)
	}
	if providerError.Code != "rate_limit_exceeded" {
		t.Errorf("code = %q, want rate_limit_exceeded", providerError.Code)
	}
	if providerError.Provider != "deepseek" {
		t.Errorf("provider = %q, want deepseek", providerError.Provider)
	}
	if !isQuotaOrRateLimitError(err) {
		t.Error("a 429 provider error must be classified as quota or rate limit")
	}
}

func TestChatClient_RunToolCall_RetriesRetryableStatus(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		attempts++
		if attempts == 1 {
			writer.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		writer.Header().Set("Content-Type", "application/json")
		if _, err := writer.Write([]byte(`{"model":"deepseek-chat","choices":[{"finish_reason":"stop","message":{"content":"ok"}}]}`)); err != nil {
			t.Errorf("write response: %v", err)
		}
	}))
	defer server.Close()

	client := NewChatClient(ProviderConfig{
		Provider:   "deepseek",
		APIKey:     "test-api-key",
		BaseURL:    server.URL,
		Timeout:    5 * time.Second,
		MaxRetries: 1,
		RetryBase:  time.Millisecond,
	})

	response, err := client.RunToolCall(context.Background(), userMessage("test"), nil, "deepseek-chat", "auto")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if response.FinishReason != "stop" {
		t.Errorf("finish_reason = %q, want the retried success", response.FinishReason)
	}
	if attempts != 2 {
		t.Errorf("attempts = %d, want 2", attempts)
	}
}

func TestChatClient_RunToolCall_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		<-request.Context().Done()
	}))
	defer server.Close()

	client := newTestChatClient(server.URL)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := client.RunToolCall(ctx, userMessage("test"), nil, "deepseek-chat", "auto")
	if err == nil {
		t.Fatal("expected an error on cancelled context")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("error = %v, want it to wrap context.Canceled", err)
	}
}

func TestChatClient_StreamAnswerWithReasoning(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "text/event-stream")
		writeSSEPayload(t, writer, map[string]interface{}{
			"choices": []map[string]interface{}{
				{"delta": map[string]interface{}{"reasoning_content": "Thinking "}, "finish_reason": nil},
			},
		})
		writeSSEPayload(t, writer, map[string]interface{}{
			"choices": []map[string]interface{}{
				{"delta": map[string]interface{}{"reasoning_content": "hard."}, "finish_reason": nil},
			},
		})
		writeSSEPayload(t, writer, map[string]interface{}{
			"choices": []map[string]interface{}{
				{"delta": map[string]interface{}{"content": "The answer."}, "finish_reason": nil},
			},
		})
		writeSSEPayload(t, writer, map[string]interface{}{
			"choices": []map[string]interface{}{
				{"delta": map[string]interface{}{}, "finish_reason": "stop"},
			},
		})
		if _, err := fmt.Fprint(writer, "data: [DONE]\n\n"); err != nil {
			t.Errorf("write done marker: %v", err)
		}
	}))
	defer server.Close()

	client := newTestChatClient(server.URL)

	var reasoning, content string
	err := client.StreamAnswerWithReasoning(context.Background(), userMessage("Answer"), "deepseek-chat",
		func(chunk string) error {
			reasoning += chunk
			return nil
		},
		func(chunk string) error {
			content += chunk
			return nil
		},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if reasoning != "Thinking hard." {
		t.Errorf("reasoning = %q, want both reasoning deltas", reasoning)
	}
	if content != "The answer." {
		t.Errorf("content = %q, want the answer delta", content)
	}
}

func TestChatClient_StreamAnswer_PropagatesCallbackError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "text/event-stream")
		writeSSEPayload(t, writer, map[string]interface{}{
			"choices": []map[string]interface{}{
				{"delta": map[string]interface{}{"content": "first"}, "finish_reason": nil},
			},
		})
	}))
	defer server.Close()

	client := newTestChatClient(server.URL)
	callbackErr := errors.New("consumer disconnected")
	err := client.StreamAnswer(context.Background(), userMessage("Answer"), "deepseek-chat", func(chunk string) error {
		return callbackErr
	})
	if !errors.Is(err, callbackErr) {
		t.Errorf("error = %v, want the callback error", err)
	}
}
