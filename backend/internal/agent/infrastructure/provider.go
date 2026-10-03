package infrastructure

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand/v2"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"lunar/backend/internal/agent/domain"
)

const (
	defaultRequestTimeout    = 120 * time.Second
	defaultChatModel         = "deepseek-chat"
	defaultBreakerThreshold  = 5
	defaultBreakerCooldown   = 30 * time.Second
	defaultRetryBase         = 300 * time.Millisecond
	maxBackoff               = 5 * time.Second
	reasoningProviderName    = "deepseek"
	chatCompletionsPath      = "/chat/completions"
	missingAPIKeyMessage     = "chat completions: API key is not configured"
	emptyChoicesMessage      = "chat completions: provider returned no choices"
	openBreakerMessage       = "chat completions: provider is temporarily unavailable (circuit breaker open)"
	thinkingModeDisabled     = "disabled"
	thinkingModeEnabled      = "enabled"
	thinkingFieldName        = "thinking"
	reasoningEffortFieldName = "reasoning_effort"
	outputConfigFieldName    = "output_config"
	toolChoiceFieldName      = "tool_choice"
)

type ProviderConfig struct {
	Provider   string
	APIKey     string
	BaseURL    string
	Model      string
	Timeout    time.Duration
	MaxRetries int
	RetryBase  time.Duration
}

type ChatClient struct {
	cfg        ProviderConfig
	httpClient *http.Client
	breaker    *circuitBreaker
}

func NewChatClient(cfg ProviderConfig) *ChatClient {
	cfg.BaseURL = strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/")
	cfg.Provider = strings.ToLower(strings.TrimSpace(cfg.Provider))
	if cfg.Timeout <= 0 {
		cfg.Timeout = defaultRequestTimeout
	}
	if cfg.MaxRetries < 0 {
		cfg.MaxRetries = 0
	}
	return &ChatClient{
		cfg:        cfg,
		httpClient: &http.Client{Timeout: cfg.Timeout},
		breaker:    newCircuitBreaker(defaultBreakerThreshold, defaultBreakerCooldown),
	}
}

func (c *ChatClient) providerName() string {
	if c.cfg.Provider == "" {
		return reasoningProviderName
	}
	return c.cfg.Provider
}

func (c *ChatClient) supportsReasoningEffort() bool {
	return c.providerName() == reasoningProviderName
}

func (c *ChatClient) setHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	if c.cfg.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.cfg.APIKey)
	}
}

func hasFileParts(messages []domain.ChatMessage) bool {
	for _, message := range messages {
		for _, part := range message.Content.Parts {
			if filePart, ok := part.(domain.FilePart); ok && filePart.Type == domain.ContentPartTypeFile {
				return true
			}
		}
	}
	return false
}

func stripFileParts(messages []domain.ChatMessage) []domain.ChatMessage {
	stripped := make([]domain.ChatMessage, len(messages))
	for index, message := range messages {
		cloned := message
		if message.Content.Parts != nil {
			var textParts []domain.ContentPart
			for _, part := range message.Content.Parts {
				if textPart, ok := part.(domain.TextPart); ok && textPart.Type == domain.ContentPartTypeText {
					textParts = append(textParts, textPart)
				}
			}
			switch {
			case len(textParts) > 0:
				cloned.Content = domain.MessageContent{Parts: textParts}
			case message.Content.String != nil:
				cloned.Content = domain.MessageContent{String: message.Content.String}
			default:
				emptyText := ""
				cloned.Content = domain.MessageContent{String: &emptyText}
			}
		}
		stripped[index] = cloned
	}
	return stripped
}

func (c *ChatClient) maxAttempts() int {
	return c.cfg.MaxRetries + 1
}

func (c *ChatClient) backoffDelay(attempt int) time.Duration {
	base := c.cfg.RetryBase
	if base <= 0 {
		base = defaultRetryBase
	}
	delay := base
	for step := 1; step < attempt; step++ {
		delay *= 2
		if delay >= maxBackoff {
			delay = maxBackoff
			break
		}
	}
	return delay/2 + time.Duration(rand.Int64N(int64(delay/2)+1))
}

func isRetryableStatus(statusCode int) bool {
	return statusCode == http.StatusTooManyRequests || statusCode >= 500
}

func retryAfterDelay(resp *http.Response) time.Duration {
	value := resp.Header.Get("Retry-After")
	if value == "" {
		return 0
	}
	seconds, err := strconv.Atoi(value)
	if err != nil || seconds <= 0 {
		return 0
	}
	delay := time.Duration(seconds) * time.Second
	if delay > maxBackoff {
		delay = maxBackoff
	}
	return delay
}

type circuitBreaker struct {
	mu        sync.Mutex
	failures  int
	threshold int
	cooldown  time.Duration
	openUntil time.Time
}

func newCircuitBreaker(threshold int, cooldown time.Duration) *circuitBreaker {
	if threshold <= 0 {
		threshold = defaultBreakerThreshold
	}
	return &circuitBreaker{threshold: threshold, cooldown: cooldown}
}

func (b *circuitBreaker) allow() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.openUntil.IsZero() {
		return true
	}
	if time.Now().After(b.openUntil) {
		b.openUntil = time.Time{}
		return true
	}
	return false
}

func (b *circuitBreaker) success() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.failures = 0
	b.openUntil = time.Time{}
}

func (b *circuitBreaker) failure() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.failures++
	if b.failures >= b.threshold {
		b.openUntil = time.Now().Add(b.cooldown)
		b.failures = 0
	}
}

func (c *ChatClient) sendJSON(ctx context.Context, endpoint string, body []byte) (*http.Response, error) {
	if c.breaker == nil || !c.breaker.allow() {
		return nil, errors.New(openBreakerMessage)
	}

	attempts := c.maxAttempts()
	var lastErr error

	for attempt := 1; attempt <= attempts; attempt++ {
		if attempt > 1 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(c.backoffDelay(attempt - 1)):
			}
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
		if err != nil {
			return nil, fmt.Errorf("build chat completions request: %w", err)
		}
		c.setHeaders(req)

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = err
			c.breaker.failure()
			continue
		}

		if !isRetryableStatus(resp.StatusCode) {
			c.breaker.success()
			return resp, nil
		}

		if attempt == attempts {
			c.breaker.failure()
			return resp, nil
		}

		delay := retryAfterDelay(resp)
		if err := resp.Body.Close(); err != nil {
			return nil, fmt.Errorf("close retryable response body: %w", err)
		}
		lastErr = fmt.Errorf("provider returned HTTP %d", resp.StatusCode)
		c.breaker.failure()
		if delay > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(delay):
			}
		}
	}

	return nil, fmt.Errorf("chat completions: request failed after %d attempts: %w", attempts, lastErr)
}

func marshalRequestBody(requestBody map[string]interface{}) ([]byte, error) {
	encoded, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("marshal chat completions request: %w", err)
	}
	return encoded, nil
}
