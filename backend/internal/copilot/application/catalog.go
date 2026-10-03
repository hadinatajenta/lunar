package application

import (
	"context"
	"strings"

	"lunar/backend/internal/copilot/domain"
)

var standardModelCatalog = []struct {
	ID          string
	Name        string
	Provider    string
	Description string
	HasThinking bool
}{
	{
		ID:          "deepseek-v4-pro",
		Name:        "DeepSeek-V4 Pro (Thinking)",
		Provider:    "deepseek",
		Description: "High-reasoning chain-of-thought model for deep architectural and code analysis.",
		HasThinking: true,
	},
	{
		ID:          "deepseek-flash",
		Name:        "DeepSeek Flash",
		Provider:    "deepseek",
		Description: "Fast, low-latency code completion and concise general Q&A.",
		HasThinking: false,
	},
	{
		ID:          "gpt-6-astra",
		Name:        "GPT-6 Astra (Reasoning)",
		Provider:    "openai",
		Description: "Advanced reasoning model with deep inference token budget.",
		HasThinking: true,
	},
	{
		ID:          "gpt-6-sol",
		Name:        "GPT-6 Sol",
		Provider:    "openai",
		Description: "High-throughput general developer model for fast iteration.",
		HasThinking: false,
	},
	{
		ID:          "claude-opus-5-5",
		Name:        "Claude Opus 5.5 (Adaptive Thinking)",
		Provider:    "claude",
		Description: "Nuanced systems thinking and multi-turn refactoring capability.",
		HasThinking: true,
	},
	{
		ID:          "claude-sonnet-5-5",
		Name:        "Claude Sonnet 5.5",
		Provider:    "claude",
		Description: "Precise engineering assistant with high coding benchmark accuracy.",
		HasThinking: false,
	},
	{
		ID:          "claude-fable-5-1",
		Name:        "Claude Fable 5.1 (Deep Reasoning)",
		Provider:    "claude",
		Description: "Extended reasoning depth for complex architectural designs.",
		HasThinking: true,
	},
	{
		ID:          "gemini-3-8-flash",
		Name:        "Gemini 3.8 Flash (Extended Thinking)",
		Provider:    "gemini",
		Description: "Fast multimodal reasoning model with extended context window.",
		HasThinking: true,
	},
	{
		ID:          "gemini-3-1-pro",
		Name:        "Gemini 3.1 Pro",
		Provider:    "gemini",
		Description: "General enterprise assistant with comprehensive ecosystem knowledge.",
		HasThinking: false,
	},
	{
		ID:          "mimo-v2-5-pro",
		Name:        "Xiaomi MiMo-V2.5 Pro",
		Provider:    "mimo",
		Description: "Efficient multilingual engineering model with specialized reasoning mode.",
		HasThinking: true,
	},
}

func (s *CopilotService) ListModels(ctx context.Context, userID string) ([]domain.ModelInfo, error) {
	redacted, err := s.authService.GetRedactedSecrets(ctx, userID)
	if err != nil {
		return nil, err
	}

	configuredSet := make(map[string]bool)
	for _, provider := range redacted.ConfiguredAIProviders {
		configuredSet[strings.ToLower(provider)] = true
	}
	for provider, key := range s.systemProviders {
		if strings.TrimSpace(key) != "" {
			configuredSet[strings.ToLower(provider)] = true
		}
	}

	modelList := make([]domain.ModelInfo, 0, len(standardModelCatalog))
	for _, catalogEntry := range standardModelCatalog {
		isConfigured := configuredSet[catalogEntry.Provider]
		modelList = append(modelList, domain.ModelInfo{
			ID:           catalogEntry.ID,
			Name:         catalogEntry.Name,
			Provider:     catalogEntry.Provider,
			Description:  catalogEntry.Description,
			HasThinking:  catalogEntry.HasThinking,
			IsConfigured: isConfigured,
		})
	}

	return modelList, nil
}
