package infrastructure

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	copilotApplication "lunar/backend/internal/copilot/application"
	copilotDomain "lunar/backend/internal/copilot/domain"
	copilotInfrastructure "lunar/backend/internal/copilot/infrastructure"
	sharedErrors "lunar/backend/internal/shared/errors"
)

const (
	modelJSONOutputTokens = 600
	modelFence            = "```"
)

type ModelProvider interface {
	ListModels(ctx context.Context, userID string) ([]copilotDomain.ModelInfo, error)
	ResolveProviderCredentials(ctx context.Context, userID string, model string) (copilotApplication.ProviderCredentials, error)
}

type ModelJSONClient struct {
	client   *copilotInfrastructure.LLMClient
	provider ModelProvider
}

func NewModelJSONClient(client *copilotInfrastructure.LLMClient, provider ModelProvider) *ModelJSONClient {
	return &ModelJSONClient{client: client, provider: provider}
}

func (c *ModelJSONClient) CompleteJSON(ctx context.Context, userID string, systemPrompt string, prompt string) (string, error) {
	if c.client == nil || c.provider == nil {
		return "", fmt.Errorf("the model client is not configured")
	}

	modelName, err := c.preferredModel(ctx, userID)
	if err != nil {
		return "", fmt.Errorf("select the model: %w", err)
	}

	credentials, err := c.provider.ResolveProviderCredentials(ctx, userID, modelName)
	if err != nil {
		return "", fmt.Errorf("resolve the provider credentials: %w", err)
	}

	content, _, err := c.client.ChatCompletion(
		ctx,
		credentials.Provider,
		credentials.APIKey,
		modelName,
		prompt,
		systemPrompt,
		false,
		copilotDomain.ReasoningEffortLow,
		copilotInfrastructure.ChatOptions{
			MaxOutputTokens:  modelJSONOutputTokens,
			JSONOutput:       true,
			FailOnTruncation: true,
		},
	)
	if err != nil {
		return "", fmt.Errorf("call the %s model: %w", credentials.Provider, err)
	}

	return modelJSONObject(content)
}

func (c *ModelJSONClient) preferredModel(ctx context.Context, userID string) (string, error) {
	models, err := c.provider.ListModels(ctx, userID)
	if err != nil {
		return "", err
	}

	fastModel := ""
	thinkingModel := ""
	for _, model := range models {
		if !model.IsConfigured {
			continue
		}
		if model.HasThinking {
			if thinkingModel == "" {
				thinkingModel = model.Name
			}
			continue
		}
		if fastModel == "" {
			fastModel = model.Name
		}
	}

	if fastModel != "" {
		return fastModel, nil
	}
	if thinkingModel != "" {
		return thinkingModel, nil
	}

	return "", fmt.Errorf("%w: no AI provider is configured to name the branch", sharedErrors.ErrBadRequest)
}

func modelJSONObject(content string) (string, error) {
	candidate := strings.TrimSpace(content)
	if strings.HasPrefix(candidate, modelFence) {
		lineBreak := strings.Index(candidate, "\n")
		if lineBreak < 0 {
			return "", fmt.Errorf("the model answered with an unterminated code fence")
		}

		candidate = candidate[lineBreak+1:]
		candidate = strings.TrimSuffix(strings.TrimSpace(candidate), modelFence)
		candidate = strings.TrimSpace(candidate)
	}

	if !strings.HasPrefix(candidate, "{") {
		return "", fmt.Errorf("the model did not answer with a JSON object")
	}
	if !json.Valid([]byte(candidate)) {
		return "", fmt.Errorf("the model answer is not valid JSON")
	}

	return candidate, nil
}
