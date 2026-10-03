package application

import (
	"context"
	"encoding/json"
	"errors"
	"lunar/backend/internal/agent/domain"
	codeindexdomain "lunar/backend/internal/codeindex/domain"
	"strings"
	"testing"
)

func TestAgentRun_ToolCallingLoopThenSynthesis(t *testing.T) {
	searchTool := newStubTool("search_code", func(ctx context.Context, args json.RawMessage) (domain.ToolResult, error) {
		if string(args) != `{"query":"payment"}` {
			t.Errorf("tool args = %s, want the model arguments", args)
		}
		return domain.ToolResult{
			Content: "func processPayment() {}",
			Sources: []domain.SourceReference{
				{Repo: "aurora", File: "payment.go", Type: "code_block", Snippet: "func processPayment()"},
			},
		}, nil
	})
	registry := newStubRegistry(searchTool)

	aiClient := &stubAIClient{
		toolResponses: []*domain.ToolCallResponse{
			{
				FinishReason: "tool_calls",
				ToolCalls: []domain.ToolCall{{
					ID:       "call_1",
					Type:     "function",
					Function: domain.ToolCallFunction{Name: "search_code", Arguments: `{"query":"payment"}`},
				}},
			},
			{FinishReason: "stop", Content: domain.MessageContent{}},
		},
		streamChunks: []string{"The payment flow ", "starts at processPayment."},
	}
	agent := NewAgent(aiClient, registry)
	agent.ToolChoiceFunc = alwaysAutoToolChoice

	emit, events := collectEvents()
	answer, err := agent.Run(context.Background(), codeindexdomain.AgentQuestion{
		Question: "Trace the payment flow",
		Model:    "deepseek-chat",
	}, emit)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if searchTool.callCount.Load() != 1 {
		t.Errorf("tool executions = %d, want 1", searchTool.callCount.Load())
	}
	for _, wantType := range []EventType{EventToolCallStart, EventToolResult, EventRoundEnd, EventPhaseChange, EventDone} {
		if countEventType(*events, wantType) == 0 {
			t.Errorf("missing %q event in %v", wantType, eventTypes(*events))
		}
	}
	if answer.Answer != "The payment flow starts at processPayment." {
		t.Errorf("answer = %q, want the synthesized text", answer.Answer)
	}
	if len(answer.Sources) != 1 || answer.Sources[0].File != "payment.go" {
		t.Errorf("sources = %+v, want the search_code source", answer.Sources)
	}
	if answer.IsTruncated {
		t.Errorf("is_truncated = true, want false")
	}

	lastCall := aiClient.capturedMessages[len(aiClient.capturedMessages)-1]
	if len(lastCall) != 4 {
		t.Fatalf("second round-trip has %d messages, want 4 (system, question, assistant, tool)", len(lastCall))
	}
	toolMessage := lastCall[len(lastCall)-1]
	if toolMessage.Role != toolRole || toolMessage.ToolCallID != "call_1" {
		t.Errorf("tool message = %+v, want role tool with call_1", toolMessage)
	}
	if toolMessage.Content.Text() != "func processPayment() {}" {
		t.Errorf("tool message content = %q", toolMessage.Content.Text())
	}
	assistantMessage := lastCall[len(lastCall)-2]
	if assistantMessage.Role != agentRole || len(assistantMessage.ToolCalls) != 1 {
		t.Errorf("assistant message = %+v, want assistant with one tool call", assistantMessage)
	}
	if assistantMessage.ReasoningContent == "" {
		t.Errorf("assistant reasoning content must be populated for tool calling")
	}
}

func TestAgentRun_DispatchesToolBatchInParallel(t *testing.T) {
	first := newStubTool("search_code", func(ctx context.Context, args json.RawMessage) (domain.ToolResult, error) {
		return domain.ToolResult{Content: "first", Sources: []domain.SourceReference{{Repo: "a", File: "a.go", Type: "code_block", Snippet: "a"}}}, nil
	})
	second := newStubTool("get_file_content", func(ctx context.Context, args json.RawMessage) (domain.ToolResult, error) {
		return domain.ToolResult{Content: "second", Sources: []domain.SourceReference{{Repo: "b", File: "b.go", Type: "code_block", Snippet: "b"}}}, nil
	})
	registry := newStubRegistry(first, second)

	aiClient := &stubAIClient{
		toolResponses: []*domain.ToolCallResponse{
			{
				FinishReason: "tool_calls",
				ToolCalls: []domain.ToolCall{
					{ID: "call_a", Type: "function", Function: domain.ToolCallFunction{Name: "search_code", Arguments: `{"query":"a"}`}},
					{ID: "call_b", Type: "function", Function: domain.ToolCallFunction{Name: "get_file_content", Arguments: `{"file_path":"b.go"}`}},
				},
			},
			{FinishReason: "stop", Content: domain.MessageContent{}},
		},
		streamChunks: []string{"Batch synthesis."},
	}
	agent := NewAgent(aiClient, registry)
	agent.ToolChoiceFunc = alwaysAutoToolChoice

	emit, events := collectEvents()
	answer, err := agent.Run(context.Background(), codeindexdomain.AgentQuestion{
		Question: "Search and read",
		Model:    "deepseek-chat",
	}, emit)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if first.callCount.Load() != 1 || second.callCount.Load() != 1 {
		t.Errorf("tool executions = (%d, %d), want (1, 1)", first.callCount.Load(), second.callCount.Load())
	}
	if count := countEventType(*events, EventToolResult); count != 2 {
		t.Errorf("tool_result events = %d, want 2", count)
	}
	if len(answer.Sources) != 2 {
		t.Errorf("sources = %+v, want both tool sources", answer.Sources)
	}
	roundEnd := RoundEndPayload{}
	for _, event := range *events {
		if event.Type == EventRoundEnd {
			roundEnd = payloadOf[RoundEndPayload](t, event)
		}
	}
	if roundEnd.ToolsCalled != 2 {
		t.Errorf("tools called = %d, want 2", roundEnd.ToolsCalled)
	}
}

func TestAgentRun_ToolFailureIsReportedAndLoopContinues(t *testing.T) {
	failingTool := newStubTool("get_file_content", func(ctx context.Context, args json.RawMessage) (domain.ToolResult, error) {
		return domain.ToolResult{}, errors.New("disk unavailable")
	})
	registry := newStubRegistry(failingTool)

	aiClient := &stubAIClient{
		toolResponses: []*domain.ToolCallResponse{
			{
				FinishReason: "tool_calls",
				ToolCalls: []domain.ToolCall{{
					ID:       "call_fail",
					Type:     "function",
					Function: domain.ToolCallFunction{Name: "get_file_content", Arguments: `{"file_path":"main.go"}`},
				}},
			},
			{FinishReason: "stop", Content: domain.MessageContent{}},
		},
		streamChunks: []string{"Recovered answer."},
	}
	agent := NewAgent(aiClient, registry)
	agent.ToolChoiceFunc = alwaysAutoToolChoice

	emit, events := collectEvents()
	answer, err := agent.Run(context.Background(), codeindexdomain.AgentQuestion{
		Question: "Read main.go",
		Model:    "deepseek-chat",
	}, emit)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var resultPayload ToolResultPayload
	found := false
	for _, event := range *events {
		if event.Type != EventToolResult {
			continue
		}
		resultPayload = payloadOf[ToolResultPayload](t, event)
		found = true
	}
	if !found {
		t.Fatalf("no tool_result event in %v", eventTypes(*events))
	}
	if !resultPayload.Failed || !strings.Contains(resultPayload.Error, "disk unavailable") {
		t.Errorf("tool result payload = %+v, want the tool failure", resultPayload)
	}
	if resultPayload.ChunksFound != 0 {
		t.Errorf("chunks found = %d, want 0 for a failed tool", resultPayload.ChunksFound)
	}

	messages := aiClient.capturedMessages[len(aiClient.capturedMessages)-1]
	toolMessage := messages[len(messages)-1]
	if !strings.Contains(toolMessage.Content.Text(), "disk unavailable") {
		t.Errorf("tool message = %q, want the failure reason appended", toolMessage.Content.Text())
	}
	if answer.Answer != "Recovered answer." {
		t.Errorf("answer = %q, want the loop to continue after a tool failure", answer.Answer)
	}
}

func TestAgentRun_DeduplicatesRepeatedSources(t *testing.T) {
	duplicateSource := domain.SourceReference{Repo: "aurora", File: "same.go", Type: "code_block", Snippet: "same"}
	tool := newStubTool("search_code", func(ctx context.Context, args json.RawMessage) (domain.ToolResult, error) {
		return domain.ToolResult{Content: "content", Sources: []domain.SourceReference{duplicateSource, duplicateSource}}, nil
	})
	registry := newStubRegistry(tool)

	aiClient := &stubAIClient{
		toolResponses: []*domain.ToolCallResponse{
			{
				FinishReason: "tool_calls",
				ToolCalls: []domain.ToolCall{{
					ID:       "call_dup",
					Type:     "function",
					Function: domain.ToolCallFunction{Name: "search_code", Arguments: `{"query":"same"}`},
				}},
			},
			{FinishReason: "stop", Content: domain.MessageContent{}},
		},
		streamChunks: []string{"Done."},
	}
	agent := NewAgent(aiClient, registry)
	agent.ToolChoiceFunc = alwaysAutoToolChoice

	emit, _ := collectEvents()
	answer, err := agent.Run(context.Background(), codeindexdomain.AgentQuestion{
		Question: "Find the duplicate",
		Model:    "deepseek-chat",
	}, emit)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(answer.Sources) != 1 {
		t.Errorf("sources = %+v, want the duplicate collapsed to one", answer.Sources)
	}
}

func TestAgentRun_ScopedDomainsLimitToolDefinitions(t *testing.T) {
	registry := newStubRegistry(
		newStubTool("search_code", nil),
		newStubTool("get_pull_request_diff", nil),
	)
	aiClient := &stubAIClient{
		toolResponses: []*domain.ToolCallResponse{
			{FinishReason: "stop", Content: domain.TextMessage("ok")},
		},
	}
	agent := NewAgent(aiClient, registry)
	agent.ToolChoiceFunc = alwaysAutoToolChoice

	emit, _ := collectEvents()
	if _, err := agent.Run(context.Background(), codeindexdomain.AgentQuestion{
		Question: "Review the pull request",
		Model:    "deepseek-chat",
		Domains:  []string{DomainBitbucket},
	}, emit); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	tools := aiClient.capturedTools[0]
	if len(tools) != 1 || tools[0].Function.Name != "get_pull_request_diff" {
		t.Errorf("tools = %+v, want only the bitbucket tool", tools)
	}
}
