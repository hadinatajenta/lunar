package application

import (
	"context"
	"lunar/backend/internal/agent/domain"
	codeindexdomain "lunar/backend/internal/codeindex/domain"
	"strings"
	"testing"
)

func TestAgentRun_StreamingPhaseOneEmitsLiveEvents(t *testing.T) {
	registry := newStubRegistry(newStubTool("search_code", nil))
	streamer := &stubStreamToolCaller{
		stubAIClient: &stubAIClient{streamChunks: []string{"Final ", "answer."}},
		handlers: []func(callbacks domain.ToolCallStreamCallbacks) (*domain.ToolCallResponse, error){
			func(callbacks domain.ToolCallStreamCallbacks) (*domain.ToolCallResponse, error) {
				if err := callbacks.OnReasoningDelta("I need to "); err != nil {
					return nil, err
				}
				if err := callbacks.OnReasoningDelta("check the code."); err != nil {
					return nil, err
				}
				if err := callbacks.OnToolCallStart("call_1", "search_code"); err != nil {
					return nil, err
				}
				if err := callbacks.OnToolCallArgsDelta("call_1", `{"qu`); err != nil {
					return nil, err
				}
				if err := callbacks.OnToolCallArgsDelta("call_1", `ery":"payment"}`); err != nil {
					return nil, err
				}
				return &domain.ToolCallResponse{
					FinishReason:     "tool_calls",
					ReasoningContent: "I need to check the code.",
					ToolCalls: []domain.ToolCall{{
						ID:       "call_1",
						Type:     "function",
						Function: domain.ToolCallFunction{Name: "search_code", Arguments: `{"query":"payment"}`},
					}},
				}, nil
			},
			func(callbacks domain.ToolCallStreamCallbacks) (*domain.ToolCallResponse, error) {
				return &domain.ToolCallResponse{FinishReason: "stop"}, nil
			},
		},
	}

	agent := NewAgent(streamer, registry)
	agent.ToolChoiceFunc = alwaysAutoToolChoice

	emit, events := collectEvents()
	if _, err := agent.Run(context.Background(), codeindexdomain.AgentQuestion{
		Question: "Trace the payment flow",
		Model:    "deepseek-chat",
	}, emit); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	wantPrefix := []EventType{
		EventAgentStart, EventRoundStart,
		EventReasoningDelta, EventReasoningDelta,
		EventToolCallStart, EventToolArgsDelta, EventToolArgsDelta,
		EventToolCallStart,
		EventToolResult, EventRoundEnd,
	}
	got := eventTypes(*events)
	if len(got) < len(wantPrefix) {
		t.Fatalf("event sequence too short: %v", got)
	}
	for index, wantType := range wantPrefix {
		if got[index] != wantType {
			t.Fatalf("event[%d] = %q, want %q (full: %v)", index, got[index], wantType, got)
		}
	}
	if got[len(got)-1] != EventDone {
		t.Errorf("last event = %q, want done", got[len(got)-1])
	}
	if count := countEventType(*events, EventReasoningDelta); count != 2 {
		t.Errorf("reasoning_delta events = %d, want exactly 2 (streamed live, not repeated)", count)
	}

	startPayload := payloadOf[ToolCallStartPayload](t, (*events)[4])
	if startPayload.Round != 1 || startPayload.Name != "search_code" || startPayload.CallID != "call_1" {
		t.Errorf("tool call start payload = %+v", startPayload)
	}
	if startPayload.Action == "" {
		t.Errorf("tool call start action must be populated")
	}

	deltaPayload := payloadOf[ToolArgsDeltaPayload](t, (*events)[5])
	if deltaPayload.Round != 1 || deltaPayload.CallID != "call_1" || deltaPayload.Delta != `{"qu` {
		t.Errorf("tool args delta payload = %+v", deltaPayload)
	}

	refreshPayload := payloadOf[ToolCallStartPayload](t, (*events)[7])
	if refreshPayload.Name != "search_code" {
		t.Errorf("refreshed tool call payload = %+v", refreshPayload)
	}
}

func TestAgentRun_StreamingContentIsNotDuplicated(t *testing.T) {
	registry := newStubRegistry(newStubTool("search_code", nil))
	streamer := &stubStreamToolCaller{
		stubAIClient: &stubAIClient{},
		handlers: []func(callbacks domain.ToolCallStreamCallbacks) (*domain.ToolCallResponse, error){
			func(callbacks domain.ToolCallStreamCallbacks) (*domain.ToolCallResponse, error) {
				if err := callbacks.OnContentDelta("Streamed "); err != nil {
					return nil, err
				}
				if err := callbacks.OnContentDelta("answer."); err != nil {
					return nil, err
				}
				return &domain.ToolCallResponse{FinishReason: "stop", Content: domain.TextMessage("Streamed answer.")}, nil
			},
		},
	}

	agent := NewAgent(streamer, registry)
	agent.ToolChoiceFunc = alwaysAutoToolChoice

	emit, events := collectEvents()
	answer, err := agent.Run(context.Background(), codeindexdomain.AgentQuestion{
		Question: "Answer directly",
		Model:    "deepseek-chat",
	}, emit)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := []EventType{EventAgentStart, EventRoundStart, EventTextDelta, EventTextDelta, EventSources, EventDone}
	got := eventTypes(*events)
	if len(got) != len(want) {
		t.Fatalf("event sequence = %v, want %v", got, want)
	}
	for index, wantType := range want {
		if got[index] != wantType {
			t.Errorf("event[%d] = %q, want %q", index, got[index], wantType)
		}
	}

	var streamed strings.Builder
	for _, event := range *events {
		if event.Type == EventTextDelta {
			streamed.WriteString(payloadOf[TextDeltaPayload](t, event).Text)
		}
	}
	if streamed.String() != "Streamed answer." {
		t.Errorf("streamed text = %q, want it exactly once", streamed.String())
	}
	if answer.Answer != "Streamed answer." {
		t.Errorf("answer = %q, want the accumulated response content", answer.Answer)
	}
}

func TestAgentRun_StreamsReasoningDuringSynthesis(t *testing.T) {
	registry := newStubRegistry(newStubTool("search_code", nil))
	aiClient := &stubReasoningAIClient{
		stubAIClient: &stubAIClient{
			toolResponses: []*domain.ToolCallResponse{
				{FinishReason: "stop", Content: domain.MessageContent{}},
			},
			streamChunks: []string{"Synthesized answer."},
		},
		reasoningChunks: []string{"Thinking ", "about it."},
	}
	agent := NewAgent(aiClient, registry)
	agent.ToolChoiceFunc = alwaysAutoToolChoice

	emit, events := collectEvents()
	answer, err := agent.Run(context.Background(), codeindexdomain.AgentQuestion{
		Question: "Synthesize",
		Model:    "deepseek-chat",
	}, emit)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	reasoning := ""
	for _, event := range *events {
		if event.Type == EventReasoningDelta {
			reasoning += payloadOf[ReasoningDeltaPayload](t, event).Text
		}
	}
	if reasoning != "Thinking about it." {
		t.Errorf("reasoning = %q, want the synthesis reasoning streamed", reasoning)
	}
	if answer.Answer != "Synthesized answer." {
		t.Errorf("answer = %q, want the synthesis text", answer.Answer)
	}
}

func TestAgentRun_StreamingGracefulDegradationSkipsDuplicateContent(t *testing.T) {
	registry := newStubRegistry(newStubTool("search_code", nil))
	streamer := &stubStreamToolCaller{
		stubAIClient: &stubAIClient{},
		handlers: []func(callbacks domain.ToolCallStreamCallbacks) (*domain.ToolCallResponse, error){
			func(callbacks domain.ToolCallStreamCallbacks) (*domain.ToolCallResponse, error) {
				if err := callbacks.OnContentDelta("Conclusion "); err != nil {
					return nil, err
				}
				return &domain.ToolCallResponse{FinishReason: "stop", Content: domain.TextMessage("Conclusion draft")}, nil
			},
			func(callbacks domain.ToolCallStreamCallbacks) (*domain.ToolCallResponse, error) {
				if err := callbacks.OnContentDelta("Retry "); err != nil {
					return nil, err
				}
				return &domain.ToolCallResponse{FinishReason: "stop", Content: domain.TextMessage("Retry final")}, nil
			},
		},
	}

	agent := NewAgent(streamer, registry)
	agent.ToolChoiceFunc = func(question string, round int, executedTools []string) (string, bool) {
		if round == 1 {
			return requiredToolChoice, true
		}
		return autoToolChoice, false
	}

	emit, events := collectEvents()
	if _, err := agent.Run(context.Background(), codeindexdomain.AgentQuestion{
		Question: "Trace the settlement flow",
		Model:    "deepseek-chat",
	}, emit); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := []EventType{EventAgentStart, EventRoundStart, EventTextDelta, EventTextDelta, EventSources, EventDone}
	got := eventTypes(*events)
	if len(got) != len(want) {
		t.Fatalf("event sequence = %v, want %v", got, want)
	}
	for index, wantType := range want {
		if got[index] != wantType {
			t.Errorf("event[%d] = %q, want %q", index, got[index], wantType)
		}
	}

	var streamed strings.Builder
	for _, event := range *events {
		if event.Type == EventTextDelta {
			streamed.WriteString(payloadOf[TextDeltaPayload](t, event).Text)
		}
	}
	if streamed.String() != "Conclusion Retry " {
		t.Errorf("streamed text = %q, want each attempt streamed exactly once", streamed.String())
	}
}
