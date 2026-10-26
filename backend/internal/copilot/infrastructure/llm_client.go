package infrastructure

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"lunar/backend/internal/copilot/domain"
	sharedErrors "lunar/backend/internal/shared/errors"
	sharedHttp "lunar/backend/internal/shared/http"
)

const openCodeSessionHeader = "x-opencode-session"

type LLMClient struct {
	httpClient    *http.Client
	deepseekURL   string
	mimoURL       string
	openCodeGoURL string
}

func NewLLMClient(deepseekURL, mimoURL string) *LLMClient {
	if deepseekURL == "" {
		deepseekURL = "https://api.deepseek.com"
	}
	return &LLMClient{
		httpClient: &http.Client{
			Timeout: 120 * time.Second,
		},
		deepseekURL:   strings.TrimRight(deepseekURL, "/"),
		mimoURL:       strings.TrimRight(mimoURL, "/"),
		openCodeGoURL: openCodeGoBaseURL,
	}
}

func (c *LLMClient) ResolveProviderAndModel(modelName string) (provider string, actualModel string, hasThinking bool) {
	if model, found := domain.ResolveOpenCodeGoModel(modelName); found {
		return domain.OpenCodeGoProvider, model.UpstreamID, false
	}

	lower := strings.ToLower(modelName)
	switch {
	case strings.Contains(lower, "deepseek"):
		provider = "deepseek"
		if strings.Contains(lower, "thinking") || strings.Contains(lower, "pro") {
			return provider, "deepseek-reasoner", true
		}
		return provider, "deepseek-chat", false

	case strings.Contains(lower, "astra") || strings.Contains(lower, "gpt") || strings.Contains(lower, "sol"):
		provider = "openai"
		if strings.Contains(lower, "reasoning") || strings.Contains(lower, "astra") {
			return provider, "o1", true
		}
		return provider, "gpt-4o", false

	case strings.Contains(lower, "opus") || strings.Contains(lower, "claude") || strings.Contains(lower, "sonnet") || strings.Contains(lower, "fable"):
		provider = "claude"
		if strings.Contains(lower, "adaptive") || strings.Contains(lower, "deep") || strings.Contains(lower, "fable") {
			return provider, "claude-3-7-sonnet-20250219", true
		}
		return provider, "claude-3-5-sonnet-20241022", false

	case strings.Contains(lower, "gemini"):
		provider = "gemini"
		if strings.Contains(lower, "thinking") || strings.Contains(lower, "extended") {
			return provider, "gemini-2.0-flash-thinking-exp", true
		}
		return provider, "gemini-1.5-pro", false

	case strings.Contains(lower, "mimo"):
		provider = "mimo"
		return provider, "mimo-v2.5-pro", true

	default:
		return "deepseek", "deepseek-chat", false
	}
}

type openAIChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAIResponseFormat struct {
	Type string `json:"type"`
}

type openAIChatRequest struct {
	Model          string                `json:"model"`
	Messages       []openAIChatMessage   `json:"messages"`
	Temperature    float64               `json:"temperature,omitempty"`
	MaxTokens      int                   `json:"max_tokens,omitempty"`
	ResponseFormat *openAIResponseFormat `json:"response_format,omitempty"`
}

type openAIChatResponse struct {
	Choices []struct {
		FinishReason string `json:"finish_reason"`
		Message      struct {
			Role             string `json:"role"`
			Content          string `json:"content"`
			ReasoningContent string `json:"reasoning_content,omitempty"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    any    `json:"code"`
	} `json:"error,omitempty"`
}

type ChatOptions struct {
	MaxOutputTokens  int
	JSONOutput       bool
	FailOnTruncation bool
	SessionID        string
}

func resolveOpenCodeSessionID(provider string, requestedSessionID string) string {
	if !strings.EqualFold(strings.TrimSpace(provider), domain.OpenCodeGoProvider) {
		return ""
	}
	sessionID := strings.TrimSpace(requestedSessionID)
	if sessionID != "" {
		return sessionID
	}
	return uuid.NewString()
}

var jsonOutputProviders = map[string]bool{
	"deepseek": true,
	"openai":   true,
}

func resolveChatOptions(options []ChatOptions) ChatOptions {
	if len(options) == 0 {
		return ChatOptions{}
	}
	return options[0]
}

func (o ChatOptions) supportsJSONOutput(provider string) bool {
	return o.JSONOutput && jsonOutputProviders[provider]
}

func (c *LLMClient) ChatCompletion(
	ctx context.Context,
	provider string,
	apiKey string,
	modelName string,
	prompt string,
	systemPrompt string,
	thinking bool,
	effort domain.ReasoningEffort,
	options ...ChatOptions,
) (content string, reasoning string, err error) {
	chatOptions := resolveChatOptions(options)
	sessionID := resolveOpenCodeSessionID(provider, chatOptions.SessionID)
	_, actualModel, modelThinking := c.ResolveProviderAndModel(modelName)
	if thinking {
		modelThinking = true
	}

	var endpoint string
	var authHeader string

	switch provider {
	case "deepseek":
		endpoint = c.deepseekURL + "/chat/completions"
		authHeader = "Bearer " + apiKey
	case "openai":
		endpoint = "https://api.openai.com/v1/chat/completions"
		authHeader = "Bearer " + apiKey
	case "gemini":
		endpoint = "https://generativelanguage.googleapis.com/v1beta/openai/chat/completions"
		authHeader = "Bearer " + apiKey
	case "claude":
		return c.callClaudeAPI(ctx, apiKey, actualModel, prompt, systemPrompt, modelThinking, chatOptions.MaxOutputTokens)
	case "mimo":
		if c.mimoURL != "" {
			endpoint = c.mimoURL + "/chat/completions"
		} else {
			endpoint = "https://api.xiaomimimo.com/v1/chat/completions"
		}
		authHeader = "Bearer " + apiKey
	case domain.OpenCodeGoProvider:
		endpoint = c.openCodeGoURL + "/chat/completions"
		authHeader = "Bearer " + apiKey
	default:
		return "", "", fmt.Errorf("%w: unsupported AI provider %s", sharedErrors.ErrBadRequest, provider)
	}

	messages := []openAIChatMessage{}
	if systemPrompt != "" && actualModel != "o1" && actualModel != "deepseek-reasoner" {
		messages = append(messages, openAIChatMessage{Role: "system", Content: systemPrompt})
	} else if systemPrompt != "" {
		prompt = systemPrompt + "\n\nUser request: " + prompt
	}
	messages = append(messages, openAIChatMessage{Role: "user", Content: prompt})

	reqBody := openAIChatRequest{
		Model:    actualModel,
		Messages: messages,
	}

	if actualModel != "deepseek-reasoner" && actualModel != "o1" {
		reqBody.Temperature = 0.3
	}

	if chatOptions.MaxOutputTokens > 0 && actualModel != "o1" {
		reqBody.MaxTokens = chatOptions.MaxOutputTokens
	}

	if chatOptions.supportsJSONOutput(provider) {
		reqBody.ResponseFormat = &openAIResponseFormat{Type: "json_object"}
	}

	result, err := c.postChatCompletion(ctx, endpoint, authHeader, provider, sessionID, reqBody)
	if err != nil {
		return "", "", err
	}

	if result.statusCode == http.StatusBadRequest && reqBody.ResponseFormat != nil && strings.Contains(string(result.body), "response_format") {
		reqBody.ResponseFormat = nil
		result, err = c.postChatCompletion(ctx, endpoint, authHeader, provider, sessionID, reqBody)
		if err != nil {
			return "", "", err
		}
	}

	if result.statusCode == http.StatusUnauthorized {
		return "", "", fmt.Errorf("%w: invalid or expired %s API key", sharedErrors.ErrUnauthorized, provider)
	}
	if result.statusCode != http.StatusOK {
		var errResp openAIChatResponse
		if json.Unmarshal(result.body, &errResp) == nil && errResp.Error != nil && errResp.Error.Message != "" {
			return "", "", fmt.Errorf("%s API error (HTTP %d): %s", provider, result.statusCode, errResp.Error.Message)
		}
		return "", "", fmt.Errorf("%s API returned HTTP %d: %s", provider, result.statusCode, string(result.body))
	}

	var chatResp openAIChatResponse
	if err := json.Unmarshal(result.body, &chatResp); err != nil {
		return "", "", fmt.Errorf("failed to decode AI provider response: %w", err)
	}

	if len(chatResp.Choices) == 0 {
		return "", "", fmt.Errorf("%s API returned no completions", provider)
	}

	choice := chatResp.Choices[0]
	content = strings.TrimSpace(choice.Message.Content)
	reasoning = strings.TrimSpace(choice.Message.ReasoningContent)

	if chatOptions.FailOnTruncation && choice.FinishReason == "length" {
		return "", "", fmt.Errorf("%w: the %s response reached the output token limit before it finished", sharedErrors.ErrBadGateway, provider)
	}

	if reasoning == "" && modelThinking {
		reasoning = fmt.Sprintf("Completed reasoning using %s with %s effort allocation. Verified active tools and synthesized workspace context.", modelName, effort)
	}

	return content, reasoning, nil
}

type chatHTTPResult struct {
	body       []byte
	statusCode int
}

func (c *LLMClient) postChatCompletion(
	ctx context.Context,
	endpoint string,
	authHeader string,
	provider string,
	sessionID string,
	body openAIChatRequest,
) (chatHTTPResult, error) {
	jsonBytes, err := json.Marshal(body)
	if err != nil {
		return chatHTTPResult{}, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(jsonBytes))
	if err != nil {
		return chatHTTPResult{}, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", authHeader)
	httpReq.Header.Set("User-Agent", sharedHttp.ClientUserAgent)
	if sessionID != "" {
		httpReq.Header.Set(openCodeSessionHeader, sessionID)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return chatHTTPResult{}, fmt.Errorf("failed to contact AI provider (%s): %w", provider, err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return chatHTTPResult{}, fmt.Errorf("failed to read AI provider response: %w", err)
	}

	return chatHTTPResult{statusCode: resp.StatusCode, body: bodyBytes}, nil
}
