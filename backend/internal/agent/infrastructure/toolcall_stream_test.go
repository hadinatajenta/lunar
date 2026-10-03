package infrastructure

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"lunar/backend/internal/agent/domain"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestChatClient_RunToolCallStream_LiveCallbacksAndAssembly(t *testing.T) {
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
			"model": "deepseek-chat",
			"choices": []map[string]interface{}{
				{"delta": map[string]interface{}{"reasoning_content": "I need to "}, "finish_reason": nil},
			},
		})
		writeSSEPayload(t, writer, map[string]interface{}{
			"choices": []map[string]interface{}{
				{"delta": map[string]interface{}{"reasoning_content": "check the code."}, "finish_reason": nil},
			},
		})
		writeSSEPayload(t, writer, map[string]interface{}{
			"choices": []map[string]interface{}{
				{"delta": map[string]interface{}{"content": "Alright, I "}, "finish_reason": nil},
			},
		})
		writeSSEPayload(t, writer, map[string]interface{}{
			"choices": []map[string]interface{}{
				{"delta": map[string]interface{}{
					"tool_calls": []map[string]interface{}{
						{"index": 0, "id": "call_abc", "type": "function", "function": map[string]interface{}{"name": "search_code", "arguments": ""}},
					},
				}, "finish_reason": nil},
			},
		})
		writeSSEPayload(t, writer, map[string]interface{}{
			"choices": []map[string]interface{}{
				{"delta": map[string]interface{}{
					"tool_calls": []map[string]interface{}{
						{"index": 0, "function": map[string]interface{}{"arguments": `{"qu`}},
					},
				}, "finish_reason": nil},
			},
		})
		writeSSEPayload(t, writer, map[string]interface{}{
			"choices": []map[string]interface{}{
				{"delta": map[string]interface{}{
					"tool_calls": []map[string]interface{}{
						{"index": 0, "function": map[string]interface{}{"arguments": `ery":"payment"}`}},
					},
				}, "finish_reason": nil},
			},
		})
		writeSSEPayload(t, writer, map[string]interface{}{
			"choices": []map[string]interface{}{
				{"delta": map[string]interface{}{}, "finish_reason": "tool_calls"},
			},
		})
		if _, err := fmt.Fprint(writer, "data: [DONE]\n\n"); err != nil {
			t.Errorf("write done marker: %v", err)
		}
	}))
	defer server.Close()

	client := newTestChatClient(server.URL)

	type callbackEvent struct {
		kind     string
		firstArg string
		second   string
	}
	var events []callbackEvent

	callbacks := domain.ToolCallStreamCallbacks{
		OnReasoningDelta: func(delta string) error {
			events = append(events, callbackEvent{kind: "reasoning", firstArg: delta})
			return nil
		},
		OnContentDelta: func(delta string) error {
			events = append(events, callbackEvent{kind: "content", firstArg: delta})
			return nil
		},
		OnToolCallStart: func(callID, name string) error {
			events = append(events, callbackEvent{kind: "start", firstArg: callID, second: name})
			return nil
		},
		OnToolCallArgsDelta: func(callID, delta string) error {
			events = append(events, callbackEvent{kind: "args", firstArg: callID, second: delta})
			return nil
		},
	}

	response, err := client.RunToolCallStream(
		context.Background(),
		userMessage("Find the payment route"),
		searchCodeTool(),
		"auto",
		"required",
		callbacks,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if capturedBody["stream"] != true {
		t.Errorf("stream = %v, want true", capturedBody["stream"])
	}
	if capturedBody[toolChoiceFieldName] != "required" {
		t.Errorf("tool_choice = %v, want required", capturedBody[toolChoiceFieldName])
	}
	thinking, ok := capturedBody[thinkingFieldName].(map[string]interface{})
	if !ok {
		t.Fatalf("thinking = %v, want a map", capturedBody[thinkingFieldName])
	}
	if thinking["type"] != "disabled" {
		t.Errorf("thinking type = %v, want disabled for a required choice", thinking["type"])
	}

	wantEvents := []callbackEvent{
		{kind: "reasoning", firstArg: "I need to "},
		{kind: "reasoning", firstArg: "check the code."},
		{kind: "content", firstArg: "Alright, I "},
		{kind: "start", firstArg: "call_abc", second: "search_code"},
		{kind: "args", firstArg: "call_abc", second: `{"qu`},
		{kind: "args", firstArg: "call_abc", second: `ery":"payment"}`},
	}
	if len(events) != len(wantEvents) {
		t.Fatalf("callback events = %d, want %d: %+v", len(events), len(wantEvents), events)
	}
	for index, wantEvent := range wantEvents {
		if events[index] != wantEvent {
			t.Errorf("event[%d] = %+v, want %+v", index, events[index], wantEvent)
		}
	}

	if response.FinishReason != "tool_calls" {
		t.Errorf("finish_reason = %q, want tool_calls", response.FinishReason)
	}
	if response.ReasoningContent != "I need to check the code." {
		t.Errorf("reasoning = %q, want the accumulated reasoning", response.ReasoningContent)
	}
	if response.Content.Text() != "Alright, I " {
		t.Errorf("content = %q, want the content accumulated despite live streaming", response.Content.Text())
	}
	if len(response.ToolCalls) != 1 {
		t.Fatalf("tool calls = %d, want 1", len(response.ToolCalls))
	}
	toolCall := response.ToolCalls[0]
	if toolCall.ID != "call_abc" || toolCall.Function.Name != "search_code" {
		t.Errorf("tool call = %+v, want the streamed identity", toolCall)
	}
	if toolCall.Function.Arguments != `{"query":"payment"}` {
		t.Errorf("arguments = %q, want the accumulated fragments", toolCall.Function.Arguments)
	}
	if response.Model != "deepseek-chat" {
		t.Errorf("model = %q, want deepseek-chat", response.Model)
	}
}

func TestChatClient_RunToolCallStream_ParallelToolCallIndexes(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "text/event-stream")
		writeSSEPayload(t, writer, map[string]interface{}{
			"choices": []map[string]interface{}{
				{"delta": map[string]interface{}{
					"tool_calls": []map[string]interface{}{
						{"index": 1, "id": "call_b", "type": "function", "function": map[string]interface{}{"name": "get_file_content", "arguments": `{"file_path":"b.go"}`}},
						{"index": 0, "id": "call_a", "type": "function", "function": map[string]interface{}{"name": "search_code", "arguments": `{"query":"a"}`}},
					},
				}, "finish_reason": nil},
			},
		})
		writeSSEPayload(t, writer, map[string]interface{}{
			"choices": []map[string]interface{}{
				{"delta": map[string]interface{}{}, "finish_reason": "tool_calls"},
			},
		})
		if _, err := fmt.Fprint(writer, "data: [DONE]\n\n"); err != nil {
			t.Errorf("write done marker: %v", err)
		}
	}))
	defer server.Close()

	client := newTestChatClient(server.URL)
	response, err := client.RunToolCallStream(context.Background(), userMessage("Batch"), nil, "deepseek-chat", "auto", domain.ToolCallStreamCallbacks{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(response.ToolCalls) != 2 {
		t.Fatalf("tool calls = %d, want 2", len(response.ToolCalls))
	}
	if response.ToolCalls[0].Function.Name != "get_file_content" || response.ToolCalls[1].Function.Name != "search_code" {
		t.Errorf("tool call order = [%s, %s], want first-seen index order", response.ToolCalls[0].Function.Name, response.ToolCalls[1].Function.Name)
	}
}
