package application

import (
	"context"
	"fmt"
	"lunar/backend/internal/agent/domain"
	codeindexdomain "lunar/backend/internal/codeindex/domain"
	"strings"
	"testing"
)

func TestAgentRun_HistorySlidingWindow(t *testing.T) {
	registry := newStubRegistry(newStubTool("search_code", nil))
	aiClient := &stubAIClient{
		toolResponses: []*domain.ToolCallResponse{
			{FinishReason: "stop", Content: domain.TextMessage("ok")},
		},
	}
	agent := NewAgent(aiClient, registry)
	agent.ToolChoiceFunc = alwaysAutoToolChoice

	history := make([]codeindexdomain.ChatHistoryItem, 0, 15)
	for index := 1; index <= 15; index++ {
		history = append(history, codeindexdomain.ChatHistoryItem{
			Role:    userRole,
			Content: fmt.Sprintf("Turn %d", index),
		})
	}

	emit, _ := collectEvents()
	if _, err := agent.Run(context.Background(), codeindexdomain.AgentQuestion{
		Question: "Final question",
		Model:    "deepseek-chat",
		History:  history,
	}, emit); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	messages := aiClient.firstCapturedMessages()
	if len(messages) != 12 {
		t.Fatalf("messages = %d, want 12 (1 system + 10 history + 1 question)", len(messages))
	}
	if messages[0].Role != systemRole {
		t.Errorf("messages[0].role = %q, want system", messages[0].Role)
	}
	if messages[1].Content.Text() != "Turn 6" {
		t.Errorf("first history turn = %q, want Turn 6", messages[1].Content.Text())
	}
	if messages[11].Content.Text() != "Final question" {
		t.Errorf("last message = %q, want the current question", messages[11].Content.Text())
	}
}

func TestAgentRun_HistorySkipsInvalidTurnsAndTrimsAssistant(t *testing.T) {
	registry := newStubRegistry(newStubTool("search_code", nil))
	aiClient := &stubAIClient{
		toolResponses: []*domain.ToolCallResponse{
			{FinishReason: "stop", Content: domain.TextMessage("ok")},
		},
	}
	agent := NewAgent(aiClient, registry)
	agent.ToolChoiceFunc = alwaysAutoToolChoice

	longAnswer := strings.Repeat("a", maxAssistantHistoryLength+50)
	emit, _ := collectEvents()
	if _, err := agent.Run(context.Background(), codeindexdomain.AgentQuestion{
		Question: "Question",
		Model:    "deepseek-chat",
		History: []codeindexdomain.ChatHistoryItem{
			{Role: "system", Content: "must be skipped"},
			{Role: agentRole, Content: "   "},
			{Role: agentRole, Content: longAnswer},
		},
	}, emit); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	messages := aiClient.firstCapturedMessages()
	if len(messages) != 3 {
		t.Fatalf("messages = %d, want 3 (system + trimmed assistant + question)", len(messages))
	}
	trimmed := messages[1].Content.Text()
	if len(trimmed) != maxAssistantHistoryLength+len(truncationSuffix) {
		t.Errorf("trimmed assistant length = %d, want %d", len(trimmed), maxAssistantHistoryLength+len(truncationSuffix))
	}
	if !strings.HasSuffix(trimmed, truncationSuffix) {
		t.Errorf("trimmed assistant answer must end with the truncation suffix")
	}
}

func TestAgentRun_TruncatesAtMaxRounds(t *testing.T) {
	registry := newStubRegistry(newStubTool("search_code", nil))
	toolCall := &domain.ToolCallResponse{
		FinishReason: "tool_calls",
		ToolCalls: []domain.ToolCall{{
			ID:       "call_loop",
			Type:     "function",
			Function: domain.ToolCallFunction{Name: "search_code", Arguments: `{"query":"x"}`},
		}},
	}
	aiClient := &stubAIClient{
		toolResponses: []*domain.ToolCallResponse{toolCall, toolCall, toolCall, toolCall},
		streamChunks:  []string{"Truncated synthesis."},
	}
	agent := NewAgent(aiClient, registry)
	agent.MaxRounds = 2
	agent.ToolChoiceFunc = alwaysAutoToolChoice

	emit, events := collectEvents()
	answer, err := agent.Run(context.Background(), codeindexdomain.AgentQuestion{
		Question: "Keep searching",
		Model:    "deepseek-chat",
	}, emit)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if answer.Rounds != 2 {
		t.Errorf("rounds = %d, want 2", answer.Rounds)
	}
	if !answer.IsTruncated {
		t.Errorf("is_truncated = false, want true when the round budget is exhausted")
	}
	if answer.Answer != "Truncated synthesis." {
		t.Errorf("answer = %q, want the synthesis output", answer.Answer)
	}
	if count := countEventType(*events, EventRoundStart); count != 2 {
		t.Errorf("round_start events = %d, want 2", count)
	}
	donePayload := payloadOf[DonePayload](t, (*events)[len(*events)-1])
	if !donePayload.IsTruncated || donePayload.Rounds != 2 {
		t.Errorf("done payload = %+v, want truncated after 2 rounds", donePayload)
	}
}

func TestAgentRun_ReasoningStreamedForNonStreamingClient(t *testing.T) {
	registry := newStubRegistry(newStubTool("search_code", nil))
	aiClient := &stubAIClient{
		toolResponses: []*domain.ToolCallResponse{
			{
				FinishReason:     "tool_calls",
				ReasoningContent: "Collected reasoning from the model.",
				ToolCalls: []domain.ToolCall{{
					ID:       "call_reason",
					Type:     "function",
					Function: domain.ToolCallFunction{Name: "search_code", Arguments: `{"query":"x"}`},
				}},
			},
			{FinishReason: "stop", Content: domain.MessageContent{}},
		},
		streamChunks: []string{"Final."},
	}
	agent := NewAgent(aiClient, registry)
	agent.ToolChoiceFunc = alwaysAutoToolChoice

	emit, events := collectEvents()
	if _, err := agent.Run(context.Background(), codeindexdomain.AgentQuestion{
		Question: "Reason about it",
		Model:    "deepseek-chat",
	}, emit); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	reasoning := ""
	for _, event := range *events {
		if event.Type == EventReasoningDelta {
			reasoning += payloadOf[ReasoningDeltaPayload](t, event).Text
		}
	}
	if reasoning != "Collected reasoning from the model." {
		t.Errorf("reasoning = %q, want the post-round reasoning emitted once", reasoning)
	}
}
