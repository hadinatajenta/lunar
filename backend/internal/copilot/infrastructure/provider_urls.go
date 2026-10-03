package infrastructure

import (
	"fmt"
	"strings"

	sharedErrors "lunar/backend/internal/shared/errors"
)

const (
	openAIBaseURL       = "https://api.openai.com/v1"
	geminiOpenAIBaseURL = "https://generativelanguage.googleapis.com/v1beta/openai"
	defaultMimoBaseURL  = "https://api.xiaomimimo.com/v1"
)

func (c *LLMClient) ProviderBaseURL(provider string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case "deepseek":
		return c.deepseekURL, nil
	case "openai":
		return openAIBaseURL, nil
	case "gemini":
		return geminiOpenAIBaseURL, nil
	case "mimo":
		if strings.TrimSpace(c.mimoURL) == "" {
			return defaultMimoBaseURL, nil
		}
		return c.mimoURL, nil
	case "claude":
		return "", fmt.Errorf("%w: claude does not expose an OpenAI-compatible endpoint for tool calling", sharedErrors.ErrBadRequest)
	default:
		return "", fmt.Errorf("%w: unsupported AI provider %s", sharedErrors.ErrBadRequest, provider)
	}
}
