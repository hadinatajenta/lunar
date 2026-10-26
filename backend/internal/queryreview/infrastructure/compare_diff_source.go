package infrastructure

import (
	"context"
	"fmt"
	"strings"

	bitbucketDomain "lunar/backend/internal/bitbucket/domain"
	"lunar/backend/internal/queryreview/application"
	"lunar/backend/internal/queryreview/domain"
	sharedErrors "lunar/backend/internal/shared/errors"
)

type CompareDiffClient interface {
	GetCompareDiff(ctx context.Context, pat string, projectKey string, repoSlug string, sourceBranch string, targetBranch string) (string, error)
	GetPullRequest(ctx context.Context, pat string, repo string, prID string) (*bitbucketDomain.PullRequest, error)
}

var _ application.DiffSource = (*CompareDiffSource)(nil)

type CompareDiffSource struct {
	client CompareDiffClient
}

func NewCompareDiffSource(client CompareDiffClient) *CompareDiffSource {
	return &CompareDiffSource{client: client}
}

func (s *CompareDiffSource) GetCompareDiff(ctx context.Context, pat string, projectKey string, repoSlug string, sourceBranch string, targetBranch string) (string, error) {
	if s.client == nil {
		return "", fmt.Errorf("%w: the Bitbucket client is not configured", sharedErrors.ErrBadGateway)
	}
	return s.client.GetCompareDiff(ctx, pat, projectKey, repoSlug, sourceBranch, targetBranch)
}

func (s *CompareDiffSource) GetPullRequestBranches(ctx context.Context, pat string, projectKey string, repoSlug string, pullRequestID string) (domain.PullRequestBranches, error) {
	if s.client == nil {
		return domain.PullRequestBranches{}, fmt.Errorf("%w: the Bitbucket client is not configured", sharedErrors.ErrBadGateway)
	}

	pullRequest, err := s.client.GetPullRequest(ctx, pat, projectKey+"/"+repoSlug, pullRequestID)
	if err != nil {
		return domain.PullRequestBranches{}, err
	}
	if pullRequest == nil || strings.TrimSpace(pullRequest.SourceBranch) == "" || strings.TrimSpace(pullRequest.TargetBranch) == "" {
		return domain.PullRequestBranches{}, fmt.Errorf("%w: pull request %s in %s/%s does not expose both branches", sharedErrors.ErrBadGateway, pullRequestID, projectKey, repoSlug)
	}

	return domain.PullRequestBranches{
		SourceBranch: pullRequest.SourceBranch,
		TargetBranch: pullRequest.TargetBranch,
	}, nil
}
