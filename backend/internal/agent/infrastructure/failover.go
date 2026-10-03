package infrastructure

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync/atomic"
	"time"

	"lunar/backend/internal/agent/domain"
)

const (
	defaultCooldownDuration = 60 * time.Second
	reasonCooldownActive    = "cooldown_active"
	reasonQuotaExhausted    = "quota_exhausted"
	primaryProviderName     = "primary"
	secondaryProviderName   = "secondary"
)

type FailoverConfig struct {
	FallbackModel  string
	ExplicitModels []string
	Cooldown       time.Duration
}

type FailoverAIClient struct {
	primary            domain.AIClient
	secondary          domain.AIClient
	fallbackModel      string
	explicitModels     map[string]bool
	cooldownDuration   time.Duration
	primaryCooldownEnd atomic.Int64
	failoverCount      atomic.Int64
	onFailover         domain.FailoverHook
}

func NewFailoverAIClient(
	primary domain.AIClient,
	secondary domain.AIClient,
	cfg FailoverConfig,
	onFailover domain.FailoverHook,
) *FailoverAIClient {
	cooldown := cfg.Cooldown
	if cooldown <= 0 {
		cooldown = defaultCooldownDuration
	}
	explicitModels := make(map[string]bool, len(cfg.ExplicitModels))
	for _, model := range cfg.ExplicitModels {
		explicitModels[strings.ToLower(strings.TrimSpace(model))] = true
	}
	return &FailoverAIClient{
		primary:          primary,
		secondary:        secondary,
		fallbackModel:    strings.TrimSpace(cfg.FallbackModel),
		explicitModels:   explicitModels,
		cooldownDuration: cooldown,
		onFailover:       onFailover,
	}
}

func (f *FailoverAIClient) PrimaryInCooldown() bool {
	until := f.primaryCooldownEnd.Load()
	return until != 0 && time.Now().UnixNano() < until
}

func (f *FailoverAIClient) MarkPrimaryCooldown() {
	f.primaryCooldownEnd.Store(time.Now().Add(f.cooldownDuration).UnixNano())
}

func (f *FailoverAIClient) ResetCooldown() {
	f.primaryCooldownEnd.Store(0)
}

func (f *FailoverAIClient) FailoverCount() int64 {
	return f.failoverCount.Load()
}

func (f *FailoverAIClient) targetsSecondary(modelOption string) bool {
	return f.explicitModels[strings.ToLower(strings.TrimSpace(modelOption))]
}

func (f *FailoverAIClient) resolveFallbackModel(modelOption string) string {
	if f.fallbackModel != "" {
		return f.fallbackModel
	}
	return modelOption
}

func (f *FailoverAIClient) triggerFailover(ctx context.Context, reason string, targetModel string) {
	f.failoverCount.Add(1)
	slog.Warn("ai_provider_failover",
		"from", primaryProviderName,
		"to", secondaryProviderName,
		"reason", reason,
		"model", targetModel,
	)
	if hook := domain.FailoverHookFromContext(ctx); hook != nil {
		hook(ctx, primaryProviderName, secondaryProviderName, reason, targetModel)
	}
	if f.onFailover != nil {
		f.onFailover(ctx, primaryProviderName, secondaryProviderName, reason, targetModel)
	}
}

func (f *FailoverAIClient) secondaryMessages(messages []domain.ChatMessage) []domain.ChatMessage {
	if !hasFileParts(messages) {
		return messages
	}
	return stripFileParts(messages)
}

func runToolCallOn(
	client domain.AIClient,
	ctx context.Context,
	messages []domain.ChatMessage,
	tools []domain.ToolDefinition,
	modelOption string,
	toolChoice string,
	callbacks domain.ToolCallStreamCallbacks,
) (*domain.ToolCallResponse, error) {
	if streamer, ok := client.(domain.StreamToolCaller); ok {
		return streamer.RunToolCallStream(ctx, messages, tools, modelOption, toolChoice, callbacks)
	}
	return client.RunToolCall(ctx, messages, tools, modelOption, toolChoice)
}

func (f *FailoverAIClient) RunToolCall(
	ctx context.Context,
	messages []domain.ChatMessage,
	tools []domain.ToolDefinition,
	modelOption string,
	toolChoice string,
) (*domain.ToolCallResponse, error) {
	if f.targetsSecondary(modelOption) {
		if f.secondary == nil {
			return nil, fmt.Errorf("failover: secondary provider is not configured for model %q", modelOption)
		}
		if hasFileParts(messages) {
			return nil, fmt.Errorf("failover: file content parts are not supported by the secondary provider")
		}
		return f.secondary.RunToolCall(ctx, messages, tools, modelOption, toolChoice)
	}

	if f.secondary == nil {
		return f.primary.RunToolCall(ctx, messages, tools, modelOption, toolChoice)
	}

	fallbackModel := f.resolveFallbackModel(modelOption)

	if f.PrimaryInCooldown() {
		f.triggerFailover(ctx, reasonCooldownActive, fallbackModel)
		return f.secondary.RunToolCall(ctx, f.secondaryMessages(messages), tools, fallbackModel, toolChoice)
	}

	response, err := f.primary.RunToolCall(ctx, messages, tools, modelOption, toolChoice)
	if err == nil {
		return response, nil
	}
	if !isQuotaOrRateLimitError(err) {
		return nil, fmt.Errorf("failover: primary provider request: %w", err)
	}

	f.MarkPrimaryCooldown()
	f.triggerFailover(ctx, reasonQuotaExhausted, fallbackModel)
	return f.secondary.RunToolCall(ctx, f.secondaryMessages(messages), tools, fallbackModel, toolChoice)
}

func (f *FailoverAIClient) RunToolCallStream(
	ctx context.Context,
	messages []domain.ChatMessage,
	tools []domain.ToolDefinition,
	modelOption string,
	toolChoice string,
	callbacks domain.ToolCallStreamCallbacks,
) (*domain.ToolCallResponse, error) {
	if f.targetsSecondary(modelOption) {
		if f.secondary == nil {
			return nil, fmt.Errorf("failover: secondary provider is not configured for model %q", modelOption)
		}
		if hasFileParts(messages) {
			return nil, fmt.Errorf("failover: file content parts are not supported by the secondary provider")
		}
		return runToolCallOn(f.secondary, ctx, messages, tools, modelOption, toolChoice, callbacks)
	}

	if f.secondary == nil {
		return runToolCallOn(f.primary, ctx, messages, tools, modelOption, toolChoice, callbacks)
	}

	fallbackModel := f.resolveFallbackModel(modelOption)

	if f.PrimaryInCooldown() {
		f.triggerFailover(ctx, reasonCooldownActive, fallbackModel)
		return runToolCallOn(f.secondary, ctx, f.secondaryMessages(messages), tools, fallbackModel, toolChoice, callbacks)
	}

	var outputFlushed atomic.Bool
	wrapped := wrapToolCallCallbacks(callbacks, &outputFlushed)

	response, err := runToolCallOn(f.primary, ctx, messages, tools, modelOption, toolChoice, wrapped)
	if err == nil {
		return response, nil
	}
	if outputFlushed.Load() || !isQuotaOrRateLimitError(err) {
		return nil, fmt.Errorf("failover: primary provider stream: %w", err)
	}

	f.MarkPrimaryCooldown()
	f.triggerFailover(ctx, reasonQuotaExhausted, fallbackModel)
	return runToolCallOn(f.secondary, ctx, f.secondaryMessages(messages), tools, fallbackModel, toolChoice, callbacks)
}

func (f *FailoverAIClient) StreamAnswer(
	ctx context.Context,
	messages []domain.ChatMessage,
	modelOption string,
	onChunk func(chunk string) error,
) error {
	return f.StreamAnswerWithReasoning(ctx, messages, modelOption, nil, onChunk)
}

func (f *FailoverAIClient) StreamAnswerWithReasoning(
	ctx context.Context,
	messages []domain.ChatMessage,
	modelOption string,
	onReasoning func(chunk string) error,
	onChunk func(chunk string) error,
) error {
	if f.targetsSecondary(modelOption) {
		if f.secondary == nil {
			return fmt.Errorf("failover: secondary provider is not configured for model %q", modelOption)
		}
		if hasFileParts(messages) {
			return fmt.Errorf("failover: file content parts are not supported by the secondary provider")
		}
		return streamAnswerOn(f.secondary, ctx, messages, modelOption, onReasoning, onChunk)
	}

	if f.secondary == nil {
		return streamAnswerOn(f.primary, ctx, messages, modelOption, onReasoning, onChunk)
	}

	fallbackModel := f.resolveFallbackModel(modelOption)

	if f.PrimaryInCooldown() {
		f.triggerFailover(ctx, reasonCooldownActive, fallbackModel)
		return streamAnswerOn(f.secondary, ctx, f.secondaryMessages(messages), fallbackModel, onReasoning, onChunk)
	}

	var outputFlushed atomic.Bool
	wrappedReasoning := func(chunk string) error {
		outputFlushed.Store(true)
		if onReasoning == nil {
			return nil
		}
		return onReasoning(chunk)
	}
	wrappedChunk := func(chunk string) error {
		outputFlushed.Store(true)
		if onChunk == nil {
			return nil
		}
		return onChunk(chunk)
	}

	err := streamAnswerOn(f.primary, ctx, messages, modelOption, wrappedReasoning, wrappedChunk)
	if err == nil {
		return nil
	}
	if outputFlushed.Load() || !isQuotaOrRateLimitError(err) {
		return fmt.Errorf("failover: primary provider stream answer: %w", err)
	}

	f.MarkPrimaryCooldown()
	f.triggerFailover(ctx, reasonQuotaExhausted, fallbackModel)
	return streamAnswerOn(f.secondary, ctx, f.secondaryMessages(messages), fallbackModel, onReasoning, onChunk)
}

func streamAnswerOn(
	client domain.AIClient,
	ctx context.Context,
	messages []domain.ChatMessage,
	modelOption string,
	onReasoning func(chunk string) error,
	onChunk func(chunk string) error,
) error {
	if streamer, ok := client.(ReasoningStreamer); ok {
		return streamer.StreamAnswerWithReasoning(ctx, messages, modelOption, onReasoning, onChunk)
	}
	return client.StreamAnswer(ctx, messages, modelOption, onChunk)
}
