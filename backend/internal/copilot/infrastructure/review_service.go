package infrastructure

import (
	"context"
	"fmt"
	"strings"

	bitbucketDomain "lunar/backend/internal/bitbucket/domain"
	"lunar/backend/internal/copilot/domain"
	sharedErrors "lunar/backend/internal/shared/errors"
)

const (
	reviewNonThinkingOutputTokens = 8000
	reviewThinkingOutputTokens    = 32000
	reviewRepairContentLimit      = 12000
)

func (c *LLMClient) CodeReview(
	ctx context.Context,
	provider string,
	apiKey string,
	modelName string,
	pr *bitbucketDomain.PullRequest,
	diff *bitbucketDomain.PRDiff,
) (*bitbucketDomain.AICodeReview, error) {
	if pr == nil || diff == nil {
		return nil, fmt.Errorf("%w: a pull request and its diff are required for a review", sharedErrors.ErrBadRequest)
	}

	reviewContext := buildReviewContext(*pr, diff)
	userPrompt := buildReviewUserPrompt(reviewContext)
	options := ChatOptions{
		MaxOutputTokens:  reviewOutputTokenCap(c.modelThinks(modelName)),
		JSONOutput:       true,
		FailOnTruncation: true,
	}

	content, _, err := c.ChatCompletion(ctx, provider, apiKey, modelName, userPrompt, reviewSystemPrompt, false, domain.ReasoningEffortMedium, options)
	if err != nil {
		return nil, err
	}

	payload, err := parseReviewPayload(content)
	if err != nil {
		payload, err = c.repairReviewPayload(ctx, provider, apiKey, modelName, content, err, options)
		if err != nil {
			return nil, err
		}
	}

	return c.assembleReview(payload, reviewContext, pr, modelName), nil
}

func (c *LLMClient) modelThinks(modelName string) bool {
	_, _, hasThinking := c.ResolveProviderAndModel(modelName)
	return hasThinking
}

func reviewOutputTokenCap(modelThinks bool) int {
	if modelThinks {
		return reviewThinkingOutputTokens
	}
	return reviewNonThinkingOutputTokens
}

func (c *LLMClient) repairReviewPayload(
	ctx context.Context,
	provider string,
	apiKey string,
	modelName string,
	invalidContent string,
	cause error,
	options ChatOptions,
) (*aiReviewPayload, error) {
	repairPrompt := fmt.Sprintf(
		"Your previous answer was rejected: %v\n\nRewrite it as exactly the JSON object required by the system prompt. Keep every fact, add no new finding, and return the JSON object only.\n\n<<<REJECTED_ANSWER\n%s\nREJECTED_ANSWER>>>",
		cause,
		boundedReviewText(invalidContent, reviewRepairContentLimit),
	)

	repaired, _, err := c.ChatCompletion(ctx, provider, apiKey, modelName, repairPrompt, reviewSystemPrompt, false, domain.ReasoningEffortMedium, options)
	if err != nil {
		return nil, err
	}

	payload, err := parseReviewPayload(repaired)
	if err != nil {
		return nil, fmt.Errorf("%w: the review could not be parsed after one repair attempt: %v", sharedErrors.ErrBadGateway, err)
	}
	return payload, nil
}

func (c *LLMClient) assembleReview(
	payload *aiReviewPayload,
	reviewContext reviewContext,
	pr *bitbucketDomain.PullRequest,
	modelName string,
) *bitbucketDomain.AICodeReview {
	findings, assumptions := normalizeReviewFindings(payload, reviewContext)

	review := bitbucketDomain.AICodeReview{
		PRID:           pr.ID,
		Model:          modelName,
		PromptVersion:  ReviewPromptVersion,
		Summary:        strings.TrimSpace(payload.Summary),
		Findings:       findings,
		Assumptions:    assumptions,
		Strengths:      dedupeStrings(payload.Strengths),
		Recommendation: normalizeRecommendation(payload, findings),
		Truncated:      len(reviewContext.TruncatedFile) > 0,
	}
	if review.Summary == "" {
		review.Summary = composeFallbackSummary(findings, reviewContext)
	}

	review.GeneratedComment = composeReviewComment(review, reviewContext, payload.ClosingNote)
	return &review
}

func boundedReviewText(value string, limit int) string {
	if len(value) <= limit {
		return value
	}
	return value[:limit] + "\n...[truncated for the repair request]..."
}
