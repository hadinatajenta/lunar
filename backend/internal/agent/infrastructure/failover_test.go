package infrastructure

import (
	"context"
	"errors"
	"lunar/backend/internal/agent/domain"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestFailover_PrimarySuccessDoesNotTouchSecondary(t *testing.T) {
	primary := &stubAIClient{}
	secondary := &stubAIClient{}

	var hookCalled bool
	client := NewFailoverAIClient(primary, secondary, FailoverConfig{FallbackModel: "mimo-v2.5-pro"}, func(ctx context.Context, from, to, reason, targetModel string) {
		hookCalled = true
	})

	response, err := client.RunToolCall(context.Background(), nil, nil, "auto", "auto")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if response.Content.Text() != "ok" {
		t.Errorf("content = %q, want ok", response.Content.Text())
	}
	if primary.runToolCallCount != 1 {
		t.Errorf("primary calls = %d, want 1", primary.runToolCallCount)
	}
	if secondary.runToolCallCount != 0 {
		t.Errorf("secondary calls = %d, want 0", secondary.runToolCallCount)
	}
	if hookCalled {
		t.Error("failover hook must not fire on success")
	}
	if client.FailoverCount() != 0 {
		t.Errorf("failover count = %d, want 0", client.FailoverCount())
	}
}

func TestFailover_QuotaErrorSwitchesToSecondary(t *testing.T) {
	primary := &stubAIClient{
		runToolCallFunc: func(ctx context.Context, messages []domain.ChatMessage, tools []domain.ToolDefinition, model string, toolChoice string) (*domain.ToolCallResponse, error) {
			return nil, quotaError()
		},
	}
	secondary := &stubAIClient{
		runToolCallFunc: func(ctx context.Context, messages []domain.ChatMessage, tools []domain.ToolDefinition, model string, toolChoice string) (*domain.ToolCallResponse, error) {
			return &domain.ToolCallResponse{Model: model, FinishReason: "stop", Content: domain.TextMessage("secondary answer")}, nil
		},
	}

	var hookReason, hookModel, hookTo string
	client := NewFailoverAIClient(primary, secondary, FailoverConfig{FallbackModel: "mimo-v2.5-pro"}, func(ctx context.Context, from, to, reason, targetModel string) {
		hookReason = reason
		hookModel = targetModel
		hookTo = to
	})

	response, err := client.RunToolCall(context.Background(), nil, nil, "deepseek-chat", "auto")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if response.Content.Text() != "secondary answer" {
		t.Errorf("content = %q, want the secondary answer", response.Content.Text())
	}
	if hookReason != reasonQuotaExhausted {
		t.Errorf("reason = %q, want %q", hookReason, reasonQuotaExhausted)
	}
	if hookModel != "mimo-v2.5-pro" {
		t.Errorf("target model = %q, want the configured fallback", hookModel)
	}
	if hookTo != secondaryProviderName {
		t.Errorf("to = %q, want %q", hookTo, secondaryProviderName)
	}
	if secondary.lastModel != "mimo-v2.5-pro" {
		t.Errorf("secondary model = %q, want the fallback model", secondary.lastModel)
	}
	if !client.PrimaryInCooldown() {
		t.Error("primary must enter cooldown after a quota error")
	}
	if client.FailoverCount() != 1 {
		t.Errorf("failover count = %d, want 1", client.FailoverCount())
	}
}

func TestFailover_NonQuotaErrorIsReturnedWithoutFailover(t *testing.T) {
	badRequest := &ProviderError{Provider: "deepseek", StatusCode: http.StatusBadRequest, Message: "bad request"}
	primary := &stubAIClient{
		runToolCallFunc: func(ctx context.Context, messages []domain.ChatMessage, tools []domain.ToolDefinition, model string, toolChoice string) (*domain.ToolCallResponse, error) {
			return nil, badRequest
		},
	}
	secondary := &stubAIClient{}
	client := NewFailoverAIClient(primary, secondary, FailoverConfig{FallbackModel: "mimo-v2.5-pro"}, nil)

	_, err := client.RunToolCall(context.Background(), nil, nil, "deepseek-chat", "auto")
	if !errors.Is(err, badRequest) {
		t.Fatalf("error = %v, want the original provider error", err)
	}
	if secondary.runToolCallCount != 0 {
		t.Errorf("secondary calls = %d, want 0 for a non-quota error", secondary.runToolCallCount)
	}
	if client.PrimaryInCooldown() {
		t.Error("primary must not enter cooldown for a non-quota error")
	}
}

func TestFailover_ExplicitSecondaryModelRouting(t *testing.T) {
	primary := &stubAIClient{}
	secondary := &stubAIClient{}
	client := NewFailoverAIClient(primary, secondary, FailoverConfig{ExplicitModels: []string{"mimo-v2.5-pro"}}, nil)

	if _, err := client.RunToolCall(context.Background(), nil, nil, "MiMo-V2.5-Pro", "auto"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if primary.runToolCallCount != 0 {
		t.Errorf("primary calls = %d, want 0 for an explicit secondary model", primary.runToolCallCount)
	}
	if secondary.runToolCallCount != 1 {
		t.Errorf("secondary calls = %d, want 1", secondary.runToolCallCount)
	}
}

func TestFailover_ExplicitSecondaryModelWithoutSecondaryFails(t *testing.T) {
	client := NewFailoverAIClient(&stubAIClient{}, nil, FailoverConfig{ExplicitModels: []string{"mimo-v2.5-pro"}}, nil)

	_, err := client.RunToolCall(context.Background(), nil, nil, "mimo-v2.5-pro", "auto")
	if err == nil {
		t.Fatal("expected an error when the secondary provider is not configured")
	}
	if !strings.Contains(err.Error(), "secondary provider is not configured") {
		t.Errorf("error = %v, want the missing secondary message", err)
	}
}

func TestFailover_SecondaryNilPassesThroughPrimaryError(t *testing.T) {
	primaryErr := &ProviderError{Provider: "deepseek", StatusCode: http.StatusPaymentRequired, Code: "insufficient_balance", Message: "Balance empty"}
	primary := &stubAIClient{
		runToolCallFunc: func(ctx context.Context, messages []domain.ChatMessage, tools []domain.ToolDefinition, model string, toolChoice string) (*domain.ToolCallResponse, error) {
			return nil, primaryErr
		},
	}
	client := NewFailoverAIClient(primary, nil, FailoverConfig{}, nil)

	_, err := client.RunToolCall(context.Background(), nil, nil, "deepseek-chat", "auto")
	var providerError *ProviderError
	if !errors.As(err, &providerError) || providerError.StatusCode != http.StatusPaymentRequired {
		t.Fatalf("error = %v, want the primary payment error passed through", err)
	}
	if client.PrimaryInCooldown() {
		t.Error("cooldown must not be marked when there is no secondary provider")
	}
}

func TestFailover_CooldownFastPathSkipsPrimary(t *testing.T) {
	primary := &stubAIClient{
		runToolCallFunc: func(ctx context.Context, messages []domain.ChatMessage, tools []domain.ToolDefinition, model string, toolChoice string) (*domain.ToolCallResponse, error) {
			return nil, quotaError()
		},
	}
	secondary := &stubAIClient{}
	client := NewFailoverAIClient(primary, secondary, FailoverConfig{FallbackModel: "mimo-v2.5-pro", Cooldown: time.Minute}, nil)

	if _, err := client.RunToolCall(context.Background(), nil, nil, "deepseek-chat", "auto"); err != nil {
		t.Fatalf("first call: unexpected error: %v", err)
	}
	if _, err := client.RunToolCall(context.Background(), nil, nil, "deepseek-chat", "auto"); err != nil {
		t.Fatalf("second call: unexpected error: %v", err)
	}

	if primary.runToolCallCount != 1 {
		t.Errorf("primary calls = %d, want 1 (the second call must use the cooldown fast path)", primary.runToolCallCount)
	}
	if secondary.runToolCallCount != 2 {
		t.Errorf("secondary calls = %d, want 2", secondary.runToolCallCount)
	}
	if client.FailoverCount() != 2 {
		t.Errorf("failover count = %d, want 2", client.FailoverCount())
	}

	client.ResetCooldown()
	if client.PrimaryInCooldown() {
		t.Error("reset must clear the cooldown")
	}
}

func TestFailover_StripsFilePartsForSecondary(t *testing.T) {
	primary := &stubAIClient{
		runToolCallFunc: func(ctx context.Context, messages []domain.ChatMessage, tools []domain.ToolDefinition, model string, toolChoice string) (*domain.ToolCallResponse, error) {
			return nil, quotaError()
		},
	}
	secondary := &stubAIClient{}
	client := NewFailoverAIClient(primary, secondary, FailoverConfig{FallbackModel: "mimo-v2.5-pro"}, nil)

	messages := []domain.ChatMessage{{
		Role: "user",
		Content: domain.MessageContent{Parts: []domain.ContentPart{
			domain.TextPart{Type: domain.ContentPartTypeText, Text: "explain this file"},
			domain.FilePart{Type: domain.ContentPartTypeFile, FileID: "file_1"},
		}},
	}}

	if _, err := client.RunToolCall(context.Background(), messages, nil, "deepseek-chat", "auto"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if hasFileParts(secondary.lastMessages) {
		t.Error("file parts must be stripped for the secondary provider")
	}
	if secondary.lastMessages[0].Content.Text() != "explain this file" {
		t.Errorf("secondary content = %q, want the text part preserved", secondary.lastMessages[0].Content.Text())
	}
}

func TestFailover_ExplicitSecondaryRejectsFileParts(t *testing.T) {
	client := NewFailoverAIClient(&stubAIClient{}, &stubAIClient{}, FailoverConfig{ExplicitModels: []string{"mimo-v2.5-pro"}}, nil)

	messages := []domain.ChatMessage{{
		Role:    "user",
		Content: domain.MessageContent{Parts: []domain.ContentPart{domain.FilePart{Type: domain.ContentPartTypeFile, FileID: "file_1"}}},
	}}
	_, err := client.RunToolCall(context.Background(), messages, nil, "mimo-v2.5-pro", "auto")
	if err == nil {
		t.Fatal("expected an error for file parts on an explicit secondary model")
	}
	if !strings.Contains(err.Error(), "file content parts are not supported") {
		t.Errorf("error = %v, want the unsupported file parts message", err)
	}
}
