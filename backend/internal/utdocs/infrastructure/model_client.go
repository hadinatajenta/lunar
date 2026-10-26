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
	"lunar/backend/internal/utdocs/domain"
)

const (
	documentOutputTokens = 8000
	repairContentLimit   = 12000
	codeFence            = "```"
)

type ModelProvider interface {
	ResolveProviderCredentials(ctx context.Context, userID string, model string) (copilotApplication.ProviderCredentials, error)
}

type completionFunc func(ctx context.Context, userID string, model string, systemPrompt string, userPrompt string) (string, error)

type LLMClient struct {
	completion completionFunc
}

func NewLLMClient(client *copilotInfrastructure.LLMClient, provider ModelProvider) *LLMClient {
	return &LLMClient{
		completion: func(ctx context.Context, userID string, model string, systemPrompt string, userPrompt string) (string, error) {
			if client == nil || provider == nil {
				return "", fmt.Errorf("the model client is not configured")
			}

			credentials, err := provider.ResolveProviderCredentials(ctx, userID, model)
			if err != nil {
				return "", fmt.Errorf("resolve the provider credentials: %w", err)
			}

			content, _, err := client.ChatCompletion(
				ctx,
				credentials.Provider,
				credentials.APIKey,
				model,
				userPrompt,
				systemPrompt,
				false,
				copilotDomain.ReasoningEffortLow,
				copilotInfrastructure.ChatOptions{
					MaxOutputTokens:  documentOutputTokens,
					JSONOutput:       true,
					FailOnTruncation: true,
				},
			)
			if err != nil {
				return "", fmt.Errorf("call the %s model: %w", credentials.Provider, err)
			}
			return content, nil
		},
	}
}

func (c *LLMClient) Generate(
	ctx context.Context,
	userID string,
	model string,
	documentTitle string,
	request domain.DocumentRequest,
) (*domain.GeneratedContent, error) {
	if c.completion == nil {
		return nil, fmt.Errorf("the model client is not configured")
	}

	userPrompt, err := BuildUTDocsUserPrompt(documentTitle, request)
	if err != nil {
		return nil, err
	}

	content, err := c.completion(ctx, userID, model, utDocsSystemPrompt, userPrompt)
	if err != nil {
		return nil, err
	}

	generated, parseErr := ParseGeneratedContent(content)
	if parseErr == nil {
		return generated, nil
	}

	repaired, repairErr := c.completion(ctx, userID, model, utDocsSystemPrompt, BuildUTDocsRepairPrompt(parseErr, content))
	if repairErr != nil {
		return nil, repairErr
	}

	generated, parseErr = ParseGeneratedContent(repaired)
	if parseErr != nil {
		return nil, fmt.Errorf("%w: the unit test document could not be parsed after one repair attempt: %v", sharedErrors.ErrBadGateway, parseErr)
	}
	return generated, nil
}

func ParseGeneratedContent(content string) (*domain.GeneratedContent, error) {
	candidate := stripCodeFence(strings.TrimSpace(content))
	if !strings.HasPrefix(candidate, "{") {
		return nil, fmt.Errorf("the model did not answer with a JSON object")
	}

	var generated domain.GeneratedContent
	if err := json.Unmarshal([]byte(candidate), &generated); err != nil {
		return nil, fmt.Errorf("decode the generated document: %w", err)
	}
	if strings.TrimSpace(generated.Title) == "" {
		return nil, fmt.Errorf("the generated document has no title")
	}
	if strings.TrimSpace(generated.Summary) == "" {
		return nil, fmt.Errorf("the generated document has no summary")
	}
	for index, row := range generated.Rows {
		if strings.TrimSpace(row.TestFunctionName) == "" {
			return nil, fmt.Errorf("generated row %d has no test function name", index+1)
		}
	}
	return &generated, nil
}

func stripCodeFence(value string) string {
	if !strings.HasPrefix(value, codeFence) {
		return value
	}

	lineBreak := strings.Index(value, "\n")
	if lineBreak < 0 {
		return value
	}

	inner := strings.TrimSpace(value[lineBreak+1:])
	return strings.TrimSpace(strings.TrimSuffix(inner, codeFence))
}
