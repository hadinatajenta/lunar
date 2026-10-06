package application

import "context"

type RemoteBranchScanner interface {
	SearchBranches(ctx context.Context, userID string, repo string, filterText string) ([]string, error)
	ListBranches(ctx context.Context, userID string, repo string) ([]string, error)
	DefaultBranch(ctx context.Context, userID string, repo string) (string, error)
}

var _ RemoteBranchScanner = (*BitbucketService)(nil)
