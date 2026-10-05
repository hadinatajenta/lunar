package infrastructure

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"lunar/backend/internal/postman/domain"
	sharedHttp "lunar/backend/internal/shared/http"
)

const (
	chatCompletionsPath     = "/chat/completions"
	reasoningEffort         = "medium"
	reasoningProvider       = "deepseek"
	defaultProviderTimeout  = 120 * time.Second
	maxProviderResponseSize = 8 << 20
	maxProviderMessageSize  = 300
	providerTemperature     = 0.2
)

type ProviderClient struct {
	httpClient *http.Client
}

func NewProviderClient() *ProviderClient {
	return &ProviderClient{httpClient: &http.Client{Timeout: defaultProviderTimeout}}
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatCompletionRequest struct {
	Model           string        `json:"model"`
	Messages        []chatMessage `json:"messages"`
	Temperature     float64       `json:"temperature,omitempty"`
	ReasoningEffort string        `json:"reasoning_effort,omitempty"`
}

type chatCompletionResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

func (c *ProviderClient) Generate(ctx context.Context, request domain.GenerateRequest) (string, error) {
	baseURL := strings.TrimRight(strings.TrimSpace(request.BaseURL), "/")
	if baseURL == "" {
		return "", errors.New("generate collection: the provider base URL is not configured")
	}
	if strings.TrimSpace(request.APIKey) == "" {
		return "", errors.New("generate collection: the provider API key is not configured")
	}

	payload := chatCompletionRequest{
		Model: request.Model,
		Messages: []chatMessage{
			{Role: "system", Content: request.System},
			{Role: "user", Content: request.User},
		},
	}
	if supportsReasoningEffort(request.Provider) {
		payload.ReasoningEffort = reasoningEffort
	} else {
		payload.Temperature = providerTemperature
	}

	encoded, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("generate collection: encode request: %w", err)
	}

	httpRequest, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+chatCompletionsPath, bytes.NewReader(encoded))
	if err != nil {
		return "", fmt.Errorf("generate collection: %w", err)
	}
	httpRequest.Header.Set("Content-Type", "application/json")
	httpRequest.Header.Set("Accept", "application/json")
	httpRequest.Header.Set("Authorization", "Bearer "+request.APIKey)
	httpRequest.Header.Set("User-Agent", sharedHttp.ClientUserAgent)

	response, err := c.httpClient.Do(httpRequest)
	if err != nil {
		if isProviderTimeout(err) {
			return "", fmt.Errorf("%w: the provider did not respond in time", domain.ErrProviderTimeout)
		}
		return "", fmt.Errorf("generate collection: the provider is unreachable: %w", err)
	}
	defer response.Body.Close()

	body, err := io.ReadAll(io.LimitReader(response.Body, maxProviderResponseSize+1))
	if err != nil {
		return "", fmt.Errorf("generate collection: read provider response: %w", err)
	}
	if len(body) > maxProviderResponseSize {
		return "", fmt.Errorf("generate collection: the provider response exceeds %d bytes", maxProviderResponseSize)
	}

	if response.StatusCode != http.StatusOK {
		if response.StatusCode == http.StatusUnauthorized {
			return "", errors.New("generate collection: the provider rejected the API key")
		}
		return "", fmt.Errorf("generate collection: %s", describeProviderError(body, response.StatusCode))
	}

	var decoded chatCompletionResponse
	if err := json.Unmarshal(body, &decoded); err != nil {
		return "", fmt.Errorf("generate collection: decode provider response: %w", err)
	}
	if len(decoded.Choices) == 0 {
		return "", fmt.Errorf("%w: the provider returned no completion", domain.ErrCollectionInvalid)
	}

	content := strings.TrimSpace(decoded.Choices[0].Message.Content)
	if content == "" {
		return "", fmt.Errorf("%w: the provider returned an empty completion", domain.ErrCollectionInvalid)
	}
	return content, nil
}

func supportsReasoningEffort(provider string) bool {
	return strings.EqualFold(strings.TrimSpace(provider), reasoningProvider)
}

func isProviderTimeout(err error) bool {
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	var networkError net.Error
	if errors.As(err, &networkError) {
		return networkError.Timeout()
	}
	return false
}

func describeProviderError(body []byte, statusCode int) string {
	var decoded chatCompletionResponse
	if err := json.Unmarshal(body, &decoded); err == nil && decoded.Error != nil {
		message := strings.TrimSpace(decoded.Error.Message)
		if message != "" {
			return fmt.Sprintf("the provider returned HTTP %d: %s", statusCode, truncate(message, maxProviderMessageSize))
		}
	}
	return fmt.Sprintf("the provider returned HTTP %d", statusCode)
}

func truncate(value string, limit int) string {
	if len(value) <= limit {
		return value
	}
	return value[:limit]
}
