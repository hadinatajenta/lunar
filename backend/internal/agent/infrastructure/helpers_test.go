package infrastructure

import (
	"context"
	"encoding/json"
	"fmt"
	"lunar/backend/internal/agent/domain"
	"net/http"
	"testing"
	"time"
)

func newTestChatClient(serverURL string) *ChatClient {
	return NewChatClient(ProviderConfig{
		Provider: "deepseek",
		APIKey:   "test-api-key",
		BaseURL:  serverURL,
		Timeout:  5 * time.Second,
	})
}

func userMessage(text string) []domain.ChatMessage {
	return []domain.ChatMessage{{Role: "user", Content: domain.TextMessage(text)}}
}

func searchCodeTool() []domain.ToolDefinition {
	return []domain.ToolDefinition{{
		Type: "function",
		Function: domain.ToolFunction{
			Name:        "search_code",
			Description: "Search indexed code",
			Parameters:  json.RawMessage(`{"type":"object"}`),
		},
	}}
}

func writeSSEPayload(t *testing.T, writer http.ResponseWriter, payload interface{}) {
	t.Helper()
	encoded, err := json.Marshal(payload)
	if err != nil {
		t.Errorf("marshal sse payload: %v", err)
		return
	}
	if _, err := fmt.Fprintf(writer, "data: %s\n\n", encoded); err != nil {
		t.Errorf("write sse payload: %v", err)
		return
	}
	if flusher, ok := writer.(http.Flusher); ok {
		flusher.Flush()
	}
}

type stubAIClient struct {
	runToolCallFunc           func(ctx context.Context, messages []domain.ChatMessage, tools []domain.ToolDefinition, model string, toolChoice string) (*domain.ToolCallResponse, error)
	streamAnswerFunc          func(ctx context.Context, messages []domain.ChatMessage, model string, onChunk func(chunk string) error) error
	streamAnswerReasoningFunc func(ctx context.Context, messages []domain.ChatMessage, model string, onReasoning func(chunk string) error, onChunk func(chunk string) error) error
	runToolCallCount          int
	streamAnswerCount         int
	lastModel                 string
	lastMessages              []domain.ChatMessage
}

func (s *stubAIClient) RunToolCall(
	ctx context.Context,
	messages []domain.ChatMessage,
	tools []domain.ToolDefinition,
	model string,
	toolChoice string,
) (*domain.ToolCallResponse, error) {
	s.runToolCallCount++
	s.lastModel = model
	s.lastMessages = messages
	if s.runToolCallFunc != nil {
		return s.runToolCallFunc(ctx, messages, tools, model, toolChoice)
	}
	return &domain.ToolCallResponse{Model: model, FinishReason: "stop", Content: domain.TextMessage("ok")}, nil
}

func (s *stubAIClient) StreamAnswer(
	ctx context.Context,
	messages []domain.ChatMessage,
	model string,
	onChunk func(chunk string) error,
) error {
	s.streamAnswerCount++
	s.lastModel = model
	s.lastMessages = messages
	if s.streamAnswerFunc != nil {
		return s.streamAnswerFunc(ctx, messages, model, onChunk)
	}
	return onChunk("ok")
}

type stubReasoningAIClient struct {
	*stubAIClient
}

func (s *stubReasoningAIClient) StreamAnswerWithReasoning(
	ctx context.Context,
	messages []domain.ChatMessage,
	model string,
	onReasoning func(chunk string) error,
	onChunk func(chunk string) error,
) error {
	s.streamAnswerCount++
	s.lastModel = model
	s.lastMessages = messages
	if s.streamAnswerReasoningFunc != nil {
		return s.streamAnswerReasoningFunc(ctx, messages, model, onReasoning, onChunk)
	}
	return onChunk("ok")
}

type stubStreamToolCaller struct {
	*stubAIClient
	runToolCallStreamFunc func(callbacks domain.ToolCallStreamCallbacks) (*domain.ToolCallResponse, error)
	streamToolCallCount   int
}

func (s *stubStreamToolCaller) RunToolCallStream(
	ctx context.Context,
	messages []domain.ChatMessage,
	tools []domain.ToolDefinition,
	model string,
	toolChoice string,
	callbacks domain.ToolCallStreamCallbacks,
) (*domain.ToolCallResponse, error) {
	s.streamToolCallCount++
	s.lastModel = model
	s.lastMessages = messages
	if s.runToolCallStreamFunc != nil {
		return s.runToolCallStreamFunc(callbacks)
	}
	return &domain.ToolCallResponse{Model: model, FinishReason: "stop", Content: domain.TextMessage("ok")}, nil
}

func quotaError() *ProviderError {
	return &ProviderError{
		Provider:   "deepseek",
		StatusCode: http.StatusTooManyRequests,
		Code:       "rate_limit_exceeded",
		Message:    "Rate limit exceeded",
	}
}
