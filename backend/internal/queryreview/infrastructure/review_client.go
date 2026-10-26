package infrastructure

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	copilotApplication "lunar/backend/internal/copilot/application"
	copilotDomain "lunar/backend/internal/copilot/domain"
	copilotInfrastructure "lunar/backend/internal/copilot/infrastructure"
	"lunar/backend/internal/queryreview/application"
	"lunar/backend/internal/queryreview/domain"
	sharedErrors "lunar/backend/internal/shared/errors"
)

const (
	queryReviewOutputTokens      = 8000
	queryReviewRepairContentSize = 12000

	explainKeyword = "EXPLAIN"
	explainPrefix  = "EXPLAIN "
)

type ProviderResolver interface {
	ResolveProviderCredentials(ctx context.Context, userID string, model string) (copilotApplication.ProviderCredentials, error)
}

type ReviewClient struct {
	client   *copilotInfrastructure.LLMClient
	provider ProviderResolver
}

var (
	_ application.QueryReviewer = (*ReviewClient)(nil)
	_ ProviderResolver          = (*copilotApplication.CopilotService)(nil)
)

func NewReviewClient(client *copilotInfrastructure.LLMClient, provider ProviderResolver) *ReviewClient {
	return &ReviewClient{client: client, provider: provider}
}

func (c *ReviewClient) ReviewQueries(ctx context.Context, userID string, modelName string, files []domain.PromptFile) (domain.QueryExtraction, error) {
	if c.client == nil || c.provider == nil {
		return domain.QueryExtraction{}, fmt.Errorf("%w: the query review model client is not configured", sharedErrors.ErrBadGateway)
	}
	if strings.TrimSpace(modelName) == "" {
		return domain.QueryExtraction{}, fmt.Errorf("%w: a model is required", sharedErrors.ErrBadRequest)
	}

	credentials, err := c.provider.ResolveProviderCredentials(ctx, userID, modelName)
	if err != nil {
		return domain.QueryExtraction{}, err
	}

	options := copilotInfrastructure.ChatOptions{
		MaxOutputTokens:  queryReviewOutputTokens,
		JSONOutput:       true,
		FailOnTruncation: true,
	}

	content, _, err := c.client.ChatCompletion(
		ctx,
		credentials.Provider,
		credentials.APIKey,
		modelName,
		buildQueryReviewUserPrompt(files),
		queryReviewSystemPrompt,
		false,
		copilotDomain.ReasoningEffortMedium,
		options,
	)
	if err != nil {
		return domain.QueryExtraction{}, err
	}

	payload, err := parseQueryReviewPayload(content)
	if err != nil {
		payload, err = c.repairQueryReviewPayload(ctx, credentials.Provider, credentials.APIKey, modelName, content, err, options)
		if err != nil {
			return domain.QueryExtraction{}, err
		}
	}

	return domain.QueryExtraction{
		Queries:       normalizeQueries(payload.Queries),
		Summary:       strings.TrimSpace(payload.Summary),
		PromptVersion: QueryReviewPromptVersion,
	}, nil
}

type queryReviewItemPayload struct {
	Feature        string `json:"feature"`
	Table          string `json:"table"`
	ActionType     string `json:"action_type"`
	QueryStatement string `json:"query_statement"`
	Index          string `json:"index"`
	CreateIndex    string `json:"create_index"`
	File           string `json:"file"`
	Line           int    `json:"line"`
	Notes          string `json:"notes"`
}

type queryReviewPayload struct {
	Summary string                   `json:"summary"`
	Queries []queryReviewItemPayload `json:"queries"`
}

func parseQueryReviewPayload(content string) (*queryReviewPayload, error) {
	cleaned := extractQueryReviewJSONObject(content)
	if cleaned == "" {
		return nil, fmt.Errorf("%w: the model returned no JSON object", sharedErrors.ErrBadGateway)
	}

	var payload queryReviewPayload
	if err := json.Unmarshal([]byte(cleaned), &payload); err != nil {
		return nil, fmt.Errorf("%w: the model returned invalid query review JSON: %v", sharedErrors.ErrBadGateway, err)
	}
	if strings.TrimSpace(payload.Summary) == "" && payload.Queries == nil {
		return nil, fmt.Errorf("%w: the query review JSON is empty", sharedErrors.ErrBadGateway)
	}

	return &payload, nil
}

func extractQueryReviewJSONObject(content string) string {
	trimmed := strings.TrimSpace(content)
	trimmed = strings.TrimPrefix(trimmed, "```json")
	trimmed = strings.TrimPrefix(trimmed, "```")
	trimmed = strings.TrimSuffix(strings.TrimSpace(trimmed), "```")

	start := strings.Index(trimmed, "{")
	end := strings.LastIndex(trimmed, "}")
	if start < 0 || end <= start {
		return ""
	}
	return trimmed[start : end+1]
}

func (c *ReviewClient) repairQueryReviewPayload(
	ctx context.Context,
	provider string,
	apiKey string,
	modelName string,
	invalidContent string,
	cause error,
	options copilotInfrastructure.ChatOptions,
) (*queryReviewPayload, error) {
	repairPrompt := fmt.Sprintf(
		"Your previous answer was rejected: %v\n\nRewrite it as exactly the JSON object required by the system prompt. Keep every query that the added lines support, add no query that is not in the added lines, and return the JSON object only.\n\n<<<REJECTED_ANSWER\n%s\nREJECTED_ANSWER>>>",
		cause,
		boundedQueryReviewText(invalidContent, queryReviewRepairContentSize),
	)

	repaired, _, err := c.client.ChatCompletion(ctx, provider, apiKey, modelName, repairPrompt, queryReviewSystemPrompt, false, copilotDomain.ReasoningEffortMedium, options)
	if err != nil {
		return nil, err
	}

	payload, err := parseQueryReviewPayload(repaired)
	if err != nil {
		return nil, fmt.Errorf("%w: the query review could not be parsed after one repair attempt: %v", sharedErrors.ErrBadGateway, err)
	}
	return payload, nil
}

func boundedQueryReviewText(value string, limit int) string {
	if len(value) <= limit {
		return value
	}
	return value[:limit] + "\n...[truncated for the repair request]..."
}

func normalizeQueries(items []queryReviewItemPayload) []domain.QueryItem {
	queries := make([]domain.QueryItem, 0, len(items))
	seen := make(map[string]bool)

	for _, item := range items {
		statement := strings.TrimSpace(item.QueryStatement)
		if statement == "" {
			continue
		}
		statement = forceExplainPrefix(statement)
		key := normalizeQueryStatementKey(statement)
		if seen[key] {
			continue
		}
		seen[key] = true

		queries = append(queries, domain.QueryItem{
			No:             len(queries) + 1,
			Feature:        strings.TrimSpace(item.Feature),
			Table:          strings.TrimSpace(item.Table),
			ActionType:     strings.ToUpper(strings.TrimSpace(item.ActionType)),
			QueryStatement: statement,
			Index:          strings.TrimSpace(item.Index),
			CreateIndex:    strings.TrimSpace(item.CreateIndex),
			File:           strings.TrimSpace(item.File),
			Line:           item.Line,
			Notes:          strings.TrimSpace(item.Notes),
		})
	}

	return queries
}

func forceExplainPrefix(statement string) string {
	if hasExplainPrefix(statement) {
		return statement
	}
	return explainPrefix + statement
}

func hasExplainPrefix(statement string) bool {
	trimmed := strings.TrimSpace(statement)
	if len(trimmed) < len(explainKeyword) {
		return false
	}
	if !strings.EqualFold(trimmed[:len(explainKeyword)], explainKeyword) {
		return false
	}
	return len(trimmed) == len(explainKeyword) || strings.ContainsRune(" \t\r\n", rune(trimmed[len(explainKeyword)]))
}

func normalizeQueryStatementKey(statement string) string {
	return strings.ToLower(strings.Join(strings.Fields(statement), " "))
}
