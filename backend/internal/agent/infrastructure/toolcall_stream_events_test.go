package infrastructure

import (
	"context"
	"errors"
	"fmt"
	"lunar/backend/internal/agent/domain"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestChatClient_RunToolCallStream_DSMLFallbackParity(t *testing.T) {
	dsml := "<｜｜DSML｜｜ calls>\n<｜｜DSML｜｜ invoke name=\"search_code\">\n<｜｜DSML｜｜ parameter name=\"query\" string=\"true\">route</｜｜DSML｜｜ parameter>\n</｜｜DSML｜｜ invoke>\n</｜｜DSML｜｜ calls>"

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "text/event-stream")
		writeSSEPayload(t, writer, map[string]interface{}{
			"model": "deepseek-chat",
			"choices": []map[string]interface{}{
				{"delta": map[string]interface{}{"content": dsml}, "finish_reason": nil},
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
	response, err := client.RunToolCallStream(context.Background(), userMessage("Find the route"), nil, "deepseek-chat", "auto", domain.ToolCallStreamCallbacks{})
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
		t.Errorf("finish_reason = %q, want tool_calls", response.FinishReason)
	}
	if !response.Content.IsEmpty() {
		t.Errorf("content = %q, want the DSML markup stripped", response.Content.Text())
	}
}

func TestChatClient_RunToolCallStream_InfersFinishReason(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "text/event-stream")
		writeSSEPayload(t, writer, map[string]interface{}{
			"choices": []map[string]interface{}{
				{"delta": map[string]interface{}{
					"tool_calls": []map[string]interface{}{
						{"index": 0, "id": "call_1", "type": "function", "function": map[string]interface{}{"name": "list_services", "arguments": "{}"}},
					},
				}, "finish_reason": nil},
			},
		})
		if _, err := fmt.Fprint(writer, "data: [DONE]\n\n"); err != nil {
			t.Errorf("write done marker: %v", err)
		}
	}))
	defer server.Close()

	client := newTestChatClient(server.URL)
	response, err := client.RunToolCallStream(context.Background(), userMessage("List"), nil, "deepseek-chat", "auto", domain.ToolCallStreamCallbacks{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if response.FinishReason != "tool_calls" {
		t.Errorf("finish_reason = %q, want it inferred from the tool calls", response.FinishReason)
	}
}

func TestChatClient_RunToolCallStream_PropagatesCallbackError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "text/event-stream")
		writeSSEPayload(t, writer, map[string]interface{}{
			"choices": []map[string]interface{}{
				{"delta": map[string]interface{}{"reasoning_content": "thinking"}, "finish_reason": nil},
			},
		})
	}))
	defer server.Close()

	client := newTestChatClient(server.URL)
	callbackErr := errors.New("consumer disconnected")
	_, err := client.RunToolCallStream(context.Background(), userMessage("Go"), nil, "deepseek-chat", "auto", domain.ToolCallStreamCallbacks{
		OnReasoningDelta: func(delta string) error {
			return callbackErr
		},
	})
	if !errors.Is(err, callbackErr) {
		t.Errorf("error = %v, want the callback error", err)
	}
}

func TestChatClient_RunToolCallStream_MissingAPIKey(t *testing.T) {
	client := NewChatClient(ProviderConfig{BaseURL: "http://127.0.0.1:1"})
	_, err := client.RunToolCallStream(context.Background(), userMessage("Go"), nil, "deepseek-chat", "auto", domain.ToolCallStreamCallbacks{})
	if err == nil {
		t.Fatal("expected an error when the API key is missing")
	}
	if err.Error() != missingAPIKeyMessage {
		t.Errorf("error = %q, want %q", err.Error(), missingAPIKeyMessage)
	}
}
