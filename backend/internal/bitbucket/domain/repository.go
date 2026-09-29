package domain

import "context"

type BitbucketRepository interface {
	ListPushes(ctx context.Context, pat string, filter string) ([]PushBranch, error)
	ListPullRequests(ctx context.Context, pat string, filter string) ([]PullRequest, error)
	GetPullRequestDiff(ctx context.Context, pat string, repo string, prID string) (*PRDiff, error)
	CreatePullRequest(ctx context.Context, pat string, req CreatePRRequest, author string) (*PullRequest, error)
	PostPRComment(ctx context.Context, pat string, repo string, prID string, comment string, author string) (*PRComment, error)
	ApplyReviewAction(ctx context.Context, pat string, repo string, prID string, action string) error
	GenerateAIReview(ctx context.Context, prID string, repo string) (*AICodeReview, error)
	ResetSeedData()
}
