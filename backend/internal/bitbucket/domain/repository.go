package domain

import "context"

type BitbucketRepository interface {
	ListPushes(ctx context.Context, pat string, filter string) ([]PushBranch, error)
	ListPullRequests(ctx context.Context, pat string, filter string) ([]PullRequest, error)
	GetPullRequest(ctx context.Context, pat string, repo string, prID string) (*PullRequest, error)
	GetPullRequestDiff(ctx context.Context, pat string, repo string, prID string) (*PRDiff, error)
	CreatePullRequest(ctx context.Context, pat string, req CreatePRRequest, author string) (*PullRequest, error)
	SearchBranches(ctx context.Context, pat string, repo string, filterText string) ([]string, error)
	ListBranches(ctx context.Context, pat string, repo string) ([]string, error)
	DefaultBranch(ctx context.Context, pat string, repo string) (string, error)
	PostPRComment(ctx context.Context, pat string, repo string, prID string, comment string, author string) (*PRComment, error)
	ApplyReviewAction(ctx context.Context, pat string, repo string, prID string, action string) error
	ResetSeedData()
}
