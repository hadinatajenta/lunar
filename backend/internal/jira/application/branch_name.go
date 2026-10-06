package application

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	jiraDomain "lunar/backend/internal/jira/domain"
	"lunar/backend/internal/shared/cache"
	sharedErrors "lunar/backend/internal/shared/errors"
	workspaceDomain "lunar/backend/internal/workspace/domain"
)

const (
	branchNameCacheTTL     = 30 * time.Minute
	branchNameModelTimeout = 30 * time.Second
)

type BranchNameIssueSource interface {
	GetIssueDetail(ctx context.Context, userID string, issueKey string) (*jiraDomain.JiraIssue, error)
}

type ModelCompletionClient interface {
	CompleteJSON(ctx context.Context, userID string, systemPrompt string, prompt string) (string, error)
}

type BranchNameService struct {
	issues     BranchNameIssueSource
	model      ModelCompletionClient
	cacheStore *cache.Store
}

func NewBranchNameService(issues BranchNameIssueSource, model ModelCompletionClient) *BranchNameService {
	return &BranchNameService{
		issues:     issues,
		model:      model,
		cacheStore: cache.New(branchNameCacheTTL),
	}
}

func (s *BranchNameService) Suggest(ctx context.Context, userID string, issueKey string) (*jiraDomain.BranchNameSuggestion, error) {
	canonicalKey, err := jiraDomain.ValidateIssueKey(issueKey)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", sharedErrors.ErrBadRequest, err)
	}
	if s.issues == nil {
		return nil, fmt.Errorf("branch name service: no issue reader is configured")
	}

	cacheKey := branchNameCacheKey(userID, canonicalKey)
	if cached, found := s.cacheStore.Get(cacheKey); found {
		if suggestion, ok := cached.(jiraDomain.BranchNameSuggestion); ok {
			return &suggestion, nil
		}
	}

	issue, err := s.issues.GetIssueDetail(ctx, userID, canonicalKey)
	if err != nil {
		return nil, fmt.Errorf("read issue %s: %w", canonicalKey, err)
	}
	if issue == nil {
		return nil, fmt.Errorf("%w: Jira answered issue %s without any content", sharedErrors.ErrBadGateway, canonicalKey)
	}

	suggestion, err := s.generate(ctx, userID, issue)
	if err != nil {
		return nil, err
	}
	if suggestion.Source == jiraDomain.BranchNameSourceModel {
		s.cacheStore.Set(cacheKey, *suggestion)
	}

	return suggestion, nil
}

func (s *BranchNameService) generate(ctx context.Context, userID string, issue *jiraDomain.JiraIssue) (*jiraDomain.BranchNameSuggestion, error) {
	if s.model != nil {
		suggestion, err := s.modelSuggestion(ctx, userID, issue)
		if err == nil {
			return suggestion, nil
		}

		log.Printf("the model branch name for %s could not be produced: %v", issueKey(issue), err)
	}

	return fallbackSuggestion(issue)
}

func (s *BranchNameService) modelSuggestion(ctx context.Context, userID string, issue *jiraDomain.JiraIssue) (*jiraDomain.BranchNameSuggestion, error) {
	key := issueKey(issue)
	if strings.TrimSpace(issue.Fields.Summary) == "" && strings.TrimSpace(issue.Fields.Description) == "" {
		return nil, fmt.Errorf("issue %s carries neither a summary nor a description", key)
	}

	callContext, cancel := context.WithTimeout(ctx, branchNameModelTimeout)
	defer cancel()

	raw, err := s.model.CompleteJSON(callContext, userID, branchNameSystemPrompt, BranchNamePrompt(issue))
	if err != nil {
		return nil, fmt.Errorf("ask the model for a branch name for %s: %w", key, err)
	}

	category, core, evidence, err := ParseBranchNameResponse(raw)
	if err != nil {
		return nil, fmt.Errorf("model branch name for %s: %w", key, err)
	}
	if !containsEvidence(issueText(issue), evidence) {
		return nil, fmt.Errorf("model branch name for %s: the evidence %q does not appear in the issue", key, evidence)
	}

	branch, err := buildBranchName(key, category, core)
	if err != nil {
		return nil, fmt.Errorf("model branch name for %s: %w", key, err)
	}

	return &jiraDomain.BranchNameSuggestion{
		Key:      key,
		Branch:   branch,
		Category: category,
		Core:     core,
		Source:   jiraDomain.BranchNameSourceModel,
		Evidence: evidence,
	}, nil
}

func fallbackSuggestion(issue *jiraDomain.JiraIssue) (*jiraDomain.BranchNameSuggestion, error) {
	key := issueKey(issue)
	core, evidence := fallbackCore(issue.Fields.Summary)
	if strings.TrimSpace(evidence) == "" {
		return nil, fmt.Errorf("%w: issue %s carries no summary to name the branch from", sharedErrors.ErrBadGateway, key)
	}

	branch, err := buildBranchName(key, "", core)
	if err != nil {
		return nil, fmt.Errorf("fallback branch name for %s: %w", key, err)
	}

	return &jiraDomain.BranchNameSuggestion{
		Key:      key,
		Branch:   branch,
		Category: "",
		Core:     core,
		Source:   jiraDomain.BranchNameSourceFallback,
		Evidence: evidence,
	}, nil
}

func buildBranchName(key string, category string, core string) (string, error) {
	branch, err := workspaceDomain.ValidateBranchName(jiraDomain.BuildBranchName(key, category, core))
	if err != nil {
		return "", err
	}

	return branch, nil
}

func branchNameCacheKey(userID string, issueKey string) string {
	return "branch-name|" + strings.TrimSpace(userID) + "|" + issueKey
}

func issueKey(issue *jiraDomain.JiraIssue) string {
	return strings.ToUpper(strings.TrimSpace(issue.Key))
}

func issueText(issue *jiraDomain.JiraIssue) string {
	return strings.TrimSpace(issue.Fields.Summary + "\n" + issue.Fields.Description)
}
