package application

import (
	"context"
	"errors"
	"lunar/backend/internal/agent/domain"
	codeindexdomain "lunar/backend/internal/codeindex/domain"
	"strings"
	"testing"
)

func TestAgentRun_DirectAnswerWithoutTools(t *testing.T) {
	aiClient := &stubAIClient{
		toolResponses: []*domain.ToolCallResponse{
			{FinishReason: "stop", Content: domain.TextMessage("Direct answer.")},
		},
	}
	registry := newStubRegistry(newStubTool("search_code", nil))
	agent := NewAgent(aiClient, registry)

	emit, events := collectEvents()
	answer, err := agent.Run(context.Background(), codeindexdomain.AgentQuestion{
		Question: "hello",
		Model:    "deepseek-chat",
		Domains:  []string{DomainServiceMap},
	}, emit)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := []EventType{EventAgentStart, EventRoundStart, EventTextDelta, EventSources, EventDone}
	if got := eventTypes(*events); len(got) != len(want) {
		t.Fatalf("event sequence = %v, want %v", got, want)
	} else {
		for index, wantType := range want {
			if got[index] != wantType {
				t.Fatalf("event[%d] = %q, want %q (full: %v)", index, got[index], wantType, got)
			}
		}
	}

	if answer.Answer != "Direct answer." {
		t.Errorf("answer = %q, want the direct model answer", answer.Answer)
	}
	if answer.Rounds != 1 {
		t.Errorf("rounds = %d, want 1", answer.Rounds)
	}
	if answer.IsTruncated {
		t.Errorf("is_truncated = true, want false")
	}
	if len(answer.Sources) != 0 {
		t.Errorf("sources = %v, want none", answer.Sources)
	}
}

func TestAgentRun_RequiredToolRetry(t *testing.T) {
	searchTool := newStubTool("search_code", nil)
	registry := newStubRegistry(searchTool)

	aiClient := &stubAIClient{
		toolResponses: []*domain.ToolCallResponse{
			{FinishReason: "stop", Content: domain.TextMessage("I will answer without evidence.")},
			{
				FinishReason: "tool_calls",
				ToolCalls: []domain.ToolCall{{
					ID:       "call_retry",
					Type:     "function",
					Function: domain.ToolCallFunction{Name: "search_code", Arguments: `{"query":"route"}`},
				}},
			},
			{FinishReason: "stop", Content: domain.MessageContent{}},
		},
		streamChunks: []string{"Synthesized."},
	}
	agent := NewAgent(aiClient, registry)
	agent.ToolChoiceFunc = func(question string, round int, executedTools []string) (string, bool) {
		if round == 1 {
			return requiredToolChoice, true
		}
		return autoToolChoice, false
	}

	emit, events := collectEvents()
	if _, err := agent.Run(context.Background(), codeindexdomain.AgentQuestion{
		Question: "Check the users table",
		Model:    "deepseek-chat",
	}, emit); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if searchTool.callCount.Load() != 1 {
		t.Errorf("tool executions = %d, want 1 after the retry", searchTool.callCount.Load())
	}
	if aiClient.capturedCallCount() != 3 {
		t.Errorf("model round-trips = %d, want 3 (attempt, retry, synthesis round)", aiClient.capturedCallCount())
	}

	retryMessages := aiClient.capturedMessages[1]
	lastMessage := retryMessages[len(retryMessages)-1]
	if lastMessage.Role != userRole || lastMessage.Content.Text() != requiredEvidenceReminderText {
		t.Errorf("retry reminder = %+v, want the enforcement reminder", lastMessage)
	}
	if countEventType(*events, EventToolResult) != 1 {
		t.Errorf("tool_result events = %d, want 1", countEventType(*events, EventToolResult))
	}
}

func TestAgentRun_RequiredToolGracefulDegradation(t *testing.T) {
	registry := newStubRegistry(newStubTool("search_code", nil))
	aiClient := &stubAIClient{
		toolResponses: []*domain.ToolCallResponse{
			{FinishReason: "stop", Content: domain.TextMessage("Hello! I am doing well.")},
			{FinishReason: "stop", Content: domain.TextMessage("Hello! I am doing well, how can I help?")},
		},
	}
	agent := NewAgent(aiClient, registry)
	agent.ToolChoiceFunc = func(question string, round int, executedTools []string) (string, bool) {
		if round == 1 {
			return requiredToolChoice, true
		}
		return autoToolChoice, false
	}

	emit, events := collectEvents()
	answer, err := agent.Run(context.Background(), codeindexdomain.AgentQuestion{
		Question: "Hello, how are you?",
		Model:    "deepseek-chat",
	}, emit)
	if err != nil {
		t.Fatalf("expected graceful degradation, got error: %v", err)
	}

	var streamed strings.Builder
	for _, event := range *events {
		if event.Type != EventTextDelta {
			continue
		}
		streamed.WriteString(payloadOf[TextDeltaPayload](t, event).Text)
	}
	if streamed.String() != "Hello! I am doing well, how can I help?" {
		t.Errorf("streamed text = %q, want the retry answer exactly once", streamed.String())
	}
	if answer.Answer != "Hello! I am doing well, how can I help?" {
		t.Errorf("answer = %q, want the retry answer", answer.Answer)
	}
	if len(answer.Sources) != 0 {
		t.Errorf("sources = %+v, want none", answer.Sources)
	}
}

func TestAgentRun_RequiredToolWithoutAnyAnswerFails(t *testing.T) {
	registry := newStubRegistry(newStubTool("search_code", nil))
	aiClient := &stubAIClient{
		toolResponses: []*domain.ToolCallResponse{
			{FinishReason: "stop", Content: domain.MessageContent{}},
			{FinishReason: "stop", Content: domain.MessageContent{}},
		},
	}
	agent := NewAgent(aiClient, registry)
	agent.ToolChoiceFunc = func(question string, round int, executedTools []string) (string, bool) {
		return requiredToolChoice, true
	}

	emit, events := collectEvents()
	_, err := agent.Run(context.Background(), codeindexdomain.AgentQuestion{
		Question: "Verify the settlement flow",
		Model:    "deepseek-chat",
	}, emit)
	if err == nil {
		t.Fatal("expected an error when required evidence is never produced")
	}
	if !strings.Contains(err.Error(), "required tool evidence was not provided") {
		t.Errorf("error = %v, want the missing evidence error", err)
	}
	if countEventType(*events, EventError) != 1 {
		t.Errorf("error events = %d, want 1", countEventType(*events, EventError))
	}
	if countEventType(*events, EventDone) != 0 {
		t.Errorf("done events = %d, want 0 on failure", countEventType(*events, EventDone))
	}
}

func TestAgentRun_ContextCancellationIsReported(t *testing.T) {
	registry := newStubRegistry(newStubTool("search_code", nil))
	aiClient := &stubAIClient{runToolErr: context.Canceled}
	agent := NewAgent(aiClient, registry)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	emit, events := collectEvents()
	_, err := agent.Run(ctx, codeindexdomain.AgentQuestion{
		Question: "Trace the settlement flow",
		Model:    "deepseek-chat",
	}, emit)
	if err == nil {
		t.Fatal("expected an error on cancelled context")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("error = %v, want it to wrap context.Canceled", err)
	}
	if countEventType(*events, EventError) != 1 {
		t.Errorf("error events = %d, want 1", countEventType(*events, EventError))
	}
}

func TestAgentRun_NilEventHandlerIsTolerated(t *testing.T) {
	registry := newStubRegistry(newStubTool("search_code", nil))
	aiClient := &stubAIClient{
		toolResponses: []*domain.ToolCallResponse{
			{FinishReason: "stop", Content: domain.TextMessage("answer")},
		},
	}
	agent := NewAgent(aiClient, registry)
	agent.ToolChoiceFunc = alwaysAutoToolChoice

	answer, err := agent.Run(context.Background(), codeindexdomain.AgentQuestion{
		Question: "Question",
		Model:    "deepseek-chat",
	}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if answer.Answer != "answer" {
		t.Errorf("answer = %q, want the model answer", answer.Answer)
	}
}

func TestAgentRun_HandlerErrorAbortsRun(t *testing.T) {
	registry := newStubRegistry(newStubTool("search_code", nil))
	aiClient := &stubAIClient{
		toolResponses: []*domain.ToolCallResponse{
			{FinishReason: "stop", Content: domain.TextMessage("answer")},
		},
	}
	agent := NewAgent(aiClient, registry)
	agent.ToolChoiceFunc = alwaysAutoToolChoice

	handlerErr := errors.New("consumer disconnected")
	_, err := agent.Run(context.Background(), codeindexdomain.AgentQuestion{
		Question: "Question",
		Model:    "deepseek-chat",
	}, func(event Event) error {
		if event.Type == EventAgentStart {
			return handlerErr
		}
		return nil
	})
	if !errors.Is(err, handlerErr) {
		t.Errorf("error = %v, want the handler error", err)
	}
}
