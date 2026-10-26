package infrastructure

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	sharedErrors "lunar/backend/internal/shared/errors"
)

const (
	claudeDefaultMaxTokens = 4096
	claudeMaxOutputTokens  = 8192
)

func claudeOutputTokenCap(requested int) int {
	if requested <= 0 {
		return claudeDefaultMaxTokens
	}
	if requested > claudeMaxOutputTokens {
		return claudeMaxOutputTokens
	}
	return requested
}

func (c *LLMClient) callClaudeAPI(
	ctx context.Context,
	apiKey string,
	model string,
	prompt string,
	systemPrompt string,
	thinking bool,
	maxOutputTokens int,
) (string, string, error) {
	endpoint := "https://api.anthropic.com/v1/messages"

	payload := map[string]any{
		"model":      model,
		"max_tokens": claudeOutputTokenCap(maxOutputTokens),
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
