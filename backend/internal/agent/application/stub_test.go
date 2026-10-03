package application

import (
	"context"
	"encoding/json"
	"lunar/backend/internal/agent/domain"
	"sync"
	"sync/atomic"
	"testing"
)

type stubAIClient struct {
	mu                  sync.Mutex
	toolResponses       []*domain.ToolCallResponse
	toolCallIndex       int
	runToolErr          error
	streamChunks        []string
	streamErr           error
	capturedMessages    [][]domain.ChatMessage
	capturedToolChoices []string
	capturedModels      []string
	capturedTools       [][]domain.ToolDefinition
}

func (s *stubAIClient) RunToolCall(
	ctx context.Context,
	messages []domain.ChatMessage,
	tools []domain.ToolDefinition,
	model string,
	toolChoice string,
) (*domain.ToolCallResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.capturedMessages = append(s.capturedMessages, messages)
	s.capturedToolChoices = append(s.capturedToolChoices, toolChoice)
	s.capturedModels = append(s.capturedModels, model)
	s.capturedTools = append(s.capturedTools, tools)

	if s.runToolErr != nil {
		return nil, s.runToolErr
	}
	if s.toolCallIndex < len(s.toolResponses) {
		response := s.toolResponses[s.toolCallIndex]
		s.toolCallIndex++
		return response, nil
	}
	return &domain.ToolCallResponse{FinishReason: "stop", Content: domain.TextMessage("default stop answer")}, nil
}

func (s *stubAIClient) StreamAnswer(
	ctx context.Context,
	messages []domain.ChatMessage,
	model string,
	onChunk func(chunk string) error,
) error {
	if s.streamErr != nil {
		return s.streamErr
	}
	for _, chunk := range s.streamChunks {
		if err := onChunk(chunk); err != nil {
			return err
		}
	}
	return nil
}

func (s *stubAIClient) capturedCallCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.capturedMessages)
}

func (s *stubAIClient) firstCapturedMessages() []domain.ChatMessage {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.capturedMessages) == 0 {
		return nil
	}
	return s.capturedMessages[0]
}

type stubStreamToolCaller struct {
	*stubAIClient
	handlers     []func(callbacks domain.ToolCallStreamCallbacks) (*domain.ToolCallResponse, error)
	handlerIndex int
}

func (s *stubStreamToolCaller) RunToolCallStream(
	ctx context.Context,
	messages []domain.ChatMessage,
	tools []domain.ToolDefinition,
	model string,
	toolChoice string,
	callbacks domain.ToolCallStreamCallbacks,
) (*domain.ToolCallResponse, error) {
	s.mu.Lock()
	s.capturedMessages = append(s.capturedMessages, messages)
	s.capturedToolChoices = append(s.capturedToolChoices, toolChoice)
	index := s.handlerIndex
	s.handlerIndex++
	var handler func(callbacks domain.ToolCallStreamCallbacks) (*domain.ToolCallResponse, error)
	if index < len(s.handlers) {
		handler = s.handlers[index]
	}
	s.mu.Unlock()

	if handler == nil {
		return &domain.ToolCallResponse{FinishReason: "stop"}, nil
	}
	return handler(callbacks)
}

type stubReasoningAIClient struct {
	*stubAIClient
	reasoningChunks []string
}

func (s *stubReasoningAIClient) StreamAnswerWithReasoning(
	ctx context.Context,
	messages []domain.ChatMessage,
	model string,
	onReasoning func(chunk string) error,
	onChunk func(chunk string) error,
) error {
	if s.streamErr != nil {
		return s.streamErr
	}
	for _, chunk := range s.reasoningChunks {
		if err := onReasoning(chunk); err != nil {
			return err
		}
	}
	for _, chunk := range s.streamChunks {
		if err := onChunk(chunk); err != nil {
			return err
		}
	}
	return nil
}

type stubTool struct {
	definition domain.ToolDefinition
	execute    func(ctx context.Context, args json.RawMessage) (domain.ToolResult, error)
	callCount  atomic.Int32
}

func (t *stubTool) Definition() domain.ToolDefinition {
	return t.definition
}

func (t *stubTool) Execute(ctx context.Context, args json.RawMessage) (domain.ToolResult, error) {
	t.callCount.Add(1)
	if t.execute != nil {
		return t.execute(ctx, args)
	}
	return domain.ToolResult{Content: "tool output"}, nil
}

func newStubTool(name string, execute func(ctx context.Context, args json.RawMessage) (domain.ToolResult, error)) *stubTool {
	return &stubTool{
		definition: domain.ToolDefinition{
			Type: "function",
			Function: domain.ToolFunction{
				Name:        name,
				Description: name + " description",
				Parameters:  json.RawMessage(`{"type":"object"}`),
			},
		},
		execute: execute,
	}
}

func newStubRegistry(tools ...domain.Tool) *MapToolRegistry {
	registry := NewMapToolRegistry()
	registry.RegisterAll(tools...)
	return registry
}

func collectEvents() (EventFunc, *[]Event) {
	events := &[]Event{}
	return func(event Event) error {
		*events = append(*events, event)
		return nil
	}, events
}

func eventTypes(events []Event) []EventType {
	types := make([]EventType, len(events))
	for index, event := range events {
		types[index] = event.Type
	}
	return types
}

func countEventType(events []Event, eventType EventType) int {
	count := 0
	for _, event := range events {
		if event.Type == eventType {
			count++
		}
	}
	return count
}

func payloadOf[T any](t *testing.T, event Event) T {
	t.Helper()
	payload, ok := event.Payload.(T)
	if !ok {
		t.Fatalf("unexpected payload type %T", event.Payload)
	}
	return payload
}

func alwaysAutoToolChoice(question string, round int, executedTools []string) (string, bool) {
	return autoToolChoice, false
}
