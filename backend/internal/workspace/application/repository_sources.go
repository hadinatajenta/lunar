package application

import (
	"context"

	jiraDomain "lunar/backend/internal/jira/domain"
)

type WorkspaceRootSource interface {
	WorkspaceRootFor(ctx context.Context, userID string) (string, error)
}

type IssueContextSource interface {
	GetIssueDetail(ctx context.Context, userID string, issueKey string) (*jiraDomain.JiraIssue, error)
	GetBacklog(ctx context.Context, userID string, requestedSquad string) (*jiraDomain.JiraBacklogResponse, error)
}

type BranchMatchSource interface {
	SprintBranchesAtRoot(ctx context.Context, rootPath string, issueKeys []string) (*SprintBranches, error)
}

type CodeMatch struct {
	Repo     string
	FilePath string
}

type CodeSearchSource interface {
	SearchCode(ctx context.Context, userID string, term string, limit int) ([]CodeMatch, error)
}

type LinkedPage struct {
	Title string
	Body  string
}

type LinkedPageSource interface {
	LinkedPages(ctx context.Context, userID string, issueKey string) ([]LinkedPage, error)
}

type RepositoryRankRequest struct {
	UserID      string
	IssueKey    string
	Summary     string
	Description string
	Candidates  []string
}

type RankedRepository struct {
	Repo   string
	Rank   int
	Reason string
}

type RepositoryRanker interface {
	RankRepositories(ctx context.Context, request RepositoryRankRequest) ([]RankedRepository, error)
}
