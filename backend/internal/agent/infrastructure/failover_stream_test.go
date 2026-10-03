package infrastructure

import (
	"context"
	"errors"
	"lunar/backend/internal/agent/domain"
	"net/http"
	"testing"
)

func TestFailover_StreamAnswerDoesNotFailoverMidStream(t *testing.T) {
	primary := &stubReasoningAIClient{stubAIClient: &stubAIClient{
		streamAnswerReasoningFunc: func(ctx context.Context, messages []domain.ChatMessage, model string, onReasoning func(chunk string) error, onChunk func(chunk string) error) error {
			if err := onChunk("partial answer"); err != nil {
				return err
			}
			return quotaError()
		},
	}}
	secondary := &stubAIClient{}

	var hookCalled bool
	client := NewFailoverAIClient(primary, secondary, FailoverConfig{FallbackModel: "mimo-v2.5-pro"}, func(ctx context.Context, from, to, reason, targetModel string) {
		hookCalled = true
	})

	var received string
	err := client.StreamAnswer(context.Background(), nil, "deepseek-chat", func(chunk string) error {
		received += chunk
		return nil
	})
	if err == nil {
		t.Fatal("expected the mid-stream error to be returned")
	}
	if received != "partial answer" {
		t.Errorf("received = %q, want the flushed chunk", received)
	}
	if hookCalled {
		t.Error("failover hook must not fire after output was flushed")
	}
	if secondary.streamAnswerCount != 0 {
		t.Errorf("secondary calls = %d, want 0 mid-stream", secondary.streamAnswerCount)
	}
}

func TestFailover_StreamAnswerFailsOverBeforeFirstChunk(t *testing.T) {
	primary := &stubReasoningAIClient{stubAIClient: &stubAIClient{
		streamAnswerReasoningFunc: func(ctx context.Context, messages []domain.ChatMessage, model string, onReasoning func(chunk string) error, onChunk func(chunk string) error) error {
			return quotaError()
		},
	}}
	secondary := &stubReasoningAIClient{stubAIClient: &stubAIClient{
		streamAnswerReasoningFunc: func(ctx context.Context, messages []domain.ChatMessage, model string, onReasoning func(chunk string) error, onChunk func(chunk string) error) error {
			if err := onReasoning("recovering"); err != nil {
				return err
			}
			return onChunk("recovered answer")
		},
	}}

	client := NewFailoverAIClient(primary, secondary, FailoverConfig{FallbackModel: "mimo-v2.5-pro"}, nil)

	var received, reasoning string
	err := client.StreamAnswerWithReasoning(context.Background(), nil, "deepseek-chat",
		func(chunk string) error {
			reasoning += chunk
			return nil
		},
		func(chunk string) error {
			received += chunk
			return nil
		},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if received != "recovered answer" || reasoning != "recovering" {
		t.Errorf("received = %q reasoning = %q, want the secondary stream", received, reasoning)
	}
	if secondary.lastModel != "mimo-v2.5-pro" {
		t.Errorf("secondary model = %q, want the fallback model", secondary.lastModel)
	}
}

func TestFailover_StreamToolCallQuotaBeforeFirstOutputFailsOver(t *testing.T) {
	primary := &stubStreamToolCaller{
		stubAIClient: &stubAIClient{},
		runToolCallStreamFunc: func(callbacks domain.ToolCallStreamCallbacks) (*domain.ToolCallResponse, error) {
			return nil, quotaError()
		},
	}
	secondary := &stubAIClient{
		runToolCallFunc: func(ctx context.Context, messages []domain.ChatMessage, tools []domain.ToolDefinition, model string, toolChoice string) (*domain.ToolCallResponse, error) {
			return &domain.ToolCallResponse{Model: model, FinishReason: "tool_calls", ToolCalls: []domain.ToolCall{{ID: "call_secondary"}}}, nil
		},
	}

	client := NewFailoverAIClient(primary, secondary, FailoverConfig{FallbackModel: "mimo-v2.5-pro"}, nil)

	response, err := client.RunToolCallStream(context.Background(), nil, nil, "deepseek-chat", "auto", domain.ToolCallStreamCallbacks{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(response.ToolCalls) != 1 || response.ToolCalls[0].ID != "call_secondary" {
		t.Errorf("tool calls = %+v, want the secondary tool call", response.ToolCalls)
	}
	if secondary.runToolCallCount != 1 {
		t.Errorf("secondary calls = %d, want 1", secondary.runToolCallCount)
	}
	if secondary.lastModel != "mimo-v2.5-pro" {
		t.Errorf("secondary model = %q, want the fallback model", secondary.lastModel)
	}
}

func TestFailover_StreamToolCallNoFailoverAfterFirstOutput(t *testing.T) {
	primary := &stubStreamToolCaller{
		stubAIClient: &stubAIClient{},
		runToolCallStreamFunc: func(callbacks domain.ToolCallStreamCallbacks) (*domain.ToolCallResponse, error) {
			if err := callbacks.OnReasoningDelta("thinking"); err != nil {
				return nil, err
			}
			return nil, quotaError()
		},
	}
	secondary := &stubAIClient{}

	client := NewFailoverAIClient(primary, secondary, FailoverConfig{FallbackModel: "mimo-v2.5-pro"}, nil)

	_, err := client.RunToolCallStream(context.Background(), nil, nil, "deepseek-chat", "auto", domain.ToolCallStreamCallbacks{
		OnReasoningDelta: func(delta string) error { return nil },
	})
	if err == nil {
		t.Fatal("expected the primary error to be returned after output was flushed")
	}
	if secondary.runToolCallCount != 0 {
		t.Errorf("secondary calls = %d, want 0 after output was flushed", secondary.runToolCallCount)
	}
}

func TestFailover_StreamToolCallNoFailoverAfterContentOutput(t *testing.T) {
	primary := &stubStreamToolCaller{
		stubAIClient: &stubAIClient{},
		runToolCallStreamFunc: func(callbacks domain.ToolCallStreamCallbacks) (*domain.ToolCallResponse, error) {
			if err := callbacks.OnContentDelta("partial"); err != nil {
				return nil, err
			}
			return nil, quotaError()
		},
	}
	secondary := &stubAIClient{}

	client := NewFailoverAIClient(primary, secondary, FailoverConfig{FallbackModel: "mimo-v2.5-pro"}, nil)

	_, err := client.RunToolCallStream(context.Background(), nil, nil, "deepseek-chat", "auto", domain.ToolCallStreamCallbacks{
		OnContentDelta: func(delta string) error { return nil },
	})
	if err == nil {
		t.Fatal("expected the primary error to be returned after content was flushed")
	}
	if secondary.runToolCallCount != 0 {
		t.Errorf("secondary calls = %d, want 0 after content was flushed", secondary.runToolCallCount)
	}
}

func TestFailover_StreamToolCallFallsBackToNonStreamingWhenUnsupported(t *testing.T) {
	primary := &stubStreamToolCaller{
		stubAIClient: &stubAIClient{},
		runToolCallStreamFunc: func(callbacks domain.ToolCallStreamCallbacks) (*domain.ToolCallResponse, error) {
			return nil, quotaError()
		},
	}
	secondary := &stubAIClient{
		runToolCallFunc: func(ctx context.Context, messages []domain.ChatMessage, tools []domain.ToolDefinition, model string, toolChoice string) (*domain.ToolCallResponse, error) {
			return &domain.ToolCallResponse{Model: model, FinishReason: "stop", Content: domain.TextMessage("non-streaming secondary")}, nil
		},
	}

	client := NewFailoverAIClient(primary, secondary, FailoverConfig{FallbackModel: "mimo-v2.5-pro"}, nil)
	response, err := client.RunToolCallStream(context.Background(), nil, nil, "deepseek-chat", "auto", domain.ToolCallStreamCallbacks{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if response.Content.Text() != "non-streaming secondary" {
		t.Errorf("content = %q, want the secondary non-streaming answer", response.Content.Text())
	}
	if secondary.runToolCallCount != 1 {
		t.Errorf("secondary RunToolCall calls = %d, want 1", secondary.runToolCallCount)
	}
}

func TestFailover_UsesHookFromContext(t *testing.T) {
	primary := &stubAIClient{
		runToolCallFunc: func(ctx context.Context, messages []domain.ChatMessage, tools []domain.ToolDefinition, model string, toolChoice string) (*domain.ToolCallResponse, error) {
			return nil, quotaError()
		},
	}
	secondary := &stubAIClient{}
	client := NewFailoverAIClient(primary, secondary, FailoverConfig{FallbackModel: "mimo-v2.5-pro"}, nil)

	var contextHookReason string
	ctx := domain.WithFailoverHook(context.Background(), func(ctx context.Context, from, to, reason, targetModel string) {
		contextHookReason = reason
	})

	if _, err := client.RunToolCall(ctx, nil, nil, "deepseek-chat", "auto"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if contextHookReason != reasonQuotaExhausted {
		t.Errorf("context hook reason = %q, want %q", contextHookReason, reasonQuotaExhausted)
	}
}

func TestFailover_ProviderErrorClassification(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "nil error", err: nil, want: false},
		{name: "payment required", err: &ProviderError{StatusCode: http.StatusPaymentRequired}, want: true},
		{name: "too many requests", err: &ProviderError{StatusCode: http.StatusTooManyRequests}, want: true},
		{name: "quota code", err: &ProviderError{StatusCode: http.StatusForbidden, Code: "insufficient_quota"}, want: true},
		{name: "resource exhausted message", err: &ProviderError{StatusCode: http.StatusBadRequest, Message: "Resource has been exhausted"}, want: true},
		{name: "bad request", err: &ProviderError{StatusCode: http.StatusBadRequest, Message: "invalid schema"}, want: false},
		{name: "untyped quota message", err: errors.New("insufficient balance"), want: true},
		{name: "untyped transport error", err: errors.New("connection reset"), want: false},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			if got := isQuotaOrRateLimitError(testCase.err); got != testCase.want {
				t.Errorf("isQuotaOrRateLimitError = %t, want %t", got, testCase.want)
			}
		})
	}
}
