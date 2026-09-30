package domain

import "context"

type JiraRepository interface {
	SearchMyIssues(ctx context.Context, pat string) ([]JiraIssue, error)
	GetBacklogSprints(ctx context.Context, pat string) ([]JiraSprint, error)
	GetIssueDetail(ctx context.Context, pat string, issueKey string) (*JiraIssue, error)
	VerifyPAT(ctx context.Context, pat string) (*JiraUser, error)
}
