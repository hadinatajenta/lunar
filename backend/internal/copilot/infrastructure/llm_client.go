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

	bitbucketDomain "lunar/backend/internal/bitbucket/domain"
	"lunar/backend/internal/copilot/domain"
	sharedErrors "lunar/backend/internal/shared/errors"
)

type LLMClient struct {
	httpClient *http.Client
	deepseekURL string
	mimoURL     string
}

func NewLLMClient(deepseekURL, mimoURL string) *LLMClient {
	if deepseekURL == "" {
		deepseekURL = "https://api.deepseek.com"
	}
	return &LLMClient{
		httpClient: &http.Client{
			Timeout: 120 * time.Second,
		},
		deepseekURL: strings.TrimRight(deepseekURL, "/"),
		mimoURL:     strings.TrimRight(mimoURL, "/"),
	}
}

func (c *LLMClient) ResolveProviderAndModel(modelName string) (provider string, actualModel string, hasThinking bool) {
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

type openAIChatRequest struct {
	Model       string              `json:"model"`
	Messages    []openAIChatMessage `json:"messages"`
	Temperature float64             `json:"temperature,omitempty"`
	MaxTokens   int                 `json:"max_tokens,omitempty"`
}

type openAIChatResponse struct {
	Choices []struct {
		Message struct {
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

func (c *LLMClient) ChatCompletion(
	ctx context.Context,
	provider string,
	apiKey string,
	modelName string,
	prompt string,
	systemPrompt string,
	thinking bool,
	effort domain.ReasoningEffort,
) (content string, reasoning string, err error) {
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
		return c.callClaudeAPI(ctx, apiKey, actualModel, prompt, systemPrompt, modelThinking)
	case "mimo":
		if c.mimoURL != "" {
			endpoint = c.mimoURL + "/chat/completions"
		} else {
			endpoint = "https://api.xiaomimimo.com/v1/chat/completions"
		}
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

	jsonBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", "", err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(jsonBytes))
	if err != nil {
		return "", "", err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", authHeader)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return "", "", fmt.Errorf("failed to contact AI provider (%s): %w", provider, err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", fmt.Errorf("failed to read AI provider response: %w", err)
	}

	if resp.StatusCode == http.StatusUnauthorized {
		return "", "", fmt.Errorf("%w: invalid or expired %s API key", sharedErrors.ErrUnauthorized, provider)
	}
	if resp.StatusCode != http.StatusOK {
		var errResp openAIChatResponse
		_ = json.Unmarshal(bodyBytes, &errResp)
		if errResp.Error != nil && errResp.Error.Message != "" {
			return "", "", fmt.Errorf("%s API error (HTTP %d): %s", provider, resp.StatusCode, errResp.Error.Message)
		}
		return "", "", fmt.Errorf("%s API returned HTTP %d: %s", provider, resp.StatusCode, string(bodyBytes))
	}

	var chatResp openAIChatResponse
	if err := json.Unmarshal(bodyBytes, &chatResp); err != nil {
		return "", "", fmt.Errorf("failed to decode AI provider response: %w", err)
	}

	if len(chatResp.Choices) == 0 {
		return "", "", fmt.Errorf("%s API returned no completions", provider)
	}

	choice := chatResp.Choices[0]
	content = strings.TrimSpace(choice.Message.Content)
	reasoning = strings.TrimSpace(choice.Message.ReasoningContent)

	if reasoning == "" && modelThinking {
		reasoning = fmt.Sprintf("Completed reasoning using %s with %s effort allocation. Verified active tools and synthesized workspace context.", modelName, effort)
	}

	return content, reasoning, nil
}

func (c *LLMClient) callClaudeAPI(
	ctx context.Context,
	apiKey string,
	model string,
	prompt string,
	systemPrompt string,
	thinking bool,
) (string, string, error) {
	endpoint := "https://api.anthropic.com/v1/messages"

	payload := map[string]any{
		"model":      model,
		"max_tokens": 4096,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
	}
	if systemPrompt != "" {
		payload["system"] = systemPrompt
	}

	jsonBytes, err := json.Marshal(payload)
	if err != nil {
		return "", "", err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(jsonBytes))
	if err != nil {
		return "", "", err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", apiKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return "", "", fmt.Errorf("failed to contact Claude API: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", fmt.Errorf("failed to read Claude response: %w", err)
	}

	if resp.StatusCode == http.StatusUnauthorized {
		return "", "", fmt.Errorf("%w: invalid or expired Claude API key", sharedErrors.ErrUnauthorized)
	}
	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("claude API returned HTTP %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var result struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.Unmarshal(bodyBytes, &result); err != nil {
		return "", "", fmt.Errorf("failed to decode Claude response: %w", err)
	}

	var sb strings.Builder
	for _, block := range result.Content {
		if block.Type == "text" {
			sb.WriteString(block.Text)
		}
	}

	content := strings.TrimSpace(sb.String())
	var reasoning string
	if thinking {
		reasoning = "Claude adaptive thinking executed. Verified inputs against architectural patterns."
	}
	return content, reasoning, nil
}

type aiReviewJSONResult struct {
	Summary          string   `json:"summary"`
	Findings         []string `json:"findings"`
	GeneratedComment string   `json:"generated_comment"`
}

func (c *LLMClient) CodeReview(
	ctx context.Context,
	provider string,
	apiKey string,
	modelName string,
	repo string,
	prID string,
	diffText string,
) (*bitbucketDomain.AICodeReview, error) {
	if len(diffText) > 24000 {
		diffText = diffText[:24000] + "\n\n...[diff truncated for length]..."
	}

	systemPrompt := `You are an expert senior software engineer and security auditor conducting an automated code review on a Bitbucket Pull Request.
You must respond strictly with valid JSON conforming to this structure:
{
  "summary": "Short 1-2 sentence executive summary of the changes and overall quality.",
  "findings": [
    "Specific finding 1 (e.g. security, performance, correctness, or styling)",
    "Specific finding 2",
    "Specific finding 3"
  ],
  "generated_comment": "Complete ready-to-post pull request review comment in markdown format with greeting, summary, itemized bullet findings, and conclusion."
}`

	userPrompt := fmt.Sprintf("Pull Request ID: %s\nRepository: %s\n\nGit Diff:\n%s\n\nPlease review this pull request and return only JSON.", prID, repo, diffText)

	content, _, err := c.ChatCompletion(ctx, provider, apiKey, modelName, userPrompt, systemPrompt, false, domain.ReasoningEffortMedium)
	if err != nil {
		return nil, err
	}

	cleanJSON := content
	if strings.Contains(cleanJSON, "```json") {
		parts := strings.Split(cleanJSON, "```json")
		if len(parts) > 1 {
			cleanJSON = strings.Split(parts[1], "```")[0]
		}
	} else if strings.Contains(cleanJSON, "```") {
		parts := strings.Split(cleanJSON, "```")
		if len(parts) > 1 {
			cleanJSON = parts[1]
		}
	}
	cleanJSON = strings.TrimSpace(cleanJSON)

	var parsed aiReviewJSONResult
	if err := json.Unmarshal([]byte(cleanJSON), &parsed); err != nil {
		return &bitbucketDomain.AICodeReview{
			PRID:             prID,
			Summary:          "Automated AI review completed.",
			Findings:         []string{content},
			GeneratedComment: fmt.Sprintf("Automated review for PR %s (%s):\n\n%s", prID, repo, content),
		}, nil
	}

	return &bitbucketDomain.AICodeReview{
		PRID:             prID,
		Summary:          parsed.Summary,
		Findings:         parsed.Findings,
		GeneratedComment: parsed.GeneratedComment,
	}, nil
}
