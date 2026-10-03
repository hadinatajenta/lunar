package application

import (
	"context"
	"fmt"
	"strings"

	sharedErrors "lunar/backend/internal/shared/errors"
)

type ProviderCredentials struct {
	Provider string
	APIKey   string
	BaseURL  string
	Model    string
}

func (s *CopilotService) ResolveProviderCredentials(ctx context.Context, userID string, model string) (ProviderCredentials, error) {
	trimmedModel := strings.TrimSpace(model)
	if trimmedModel == "" {
		return ProviderCredentials{}, fmt.Errorf("%w: a model is required", sharedErrors.ErrBadRequest)
	}

	provider, actualModel, _ := s.llmClient.ResolveProviderAndModel(trimmedModel)
	if strings.TrimSpace(provider) == "" {
		return ProviderCredentials{}, fmt.Errorf("%w: unsupported model %s", sharedErrors.ErrBadRequest, trimmedModel)
	}

	apiKey, err := s.resolveAPIKeyForProvider(ctx, userID, provider)
	if err != nil {
		return ProviderCredentials{}, fmt.Errorf("resolveProviderCredentials: %w", err)
	}

	baseURL, err := s.llmClient.ProviderBaseURL(provider)
	if err != nil {
		return ProviderCredentials{}, err
	}

	return ProviderCredentials{
		Provider: provider,
		APIKey:   apiKey,
		BaseURL:  baseURL,
		Model:    actualModel,
	}, nil
}
