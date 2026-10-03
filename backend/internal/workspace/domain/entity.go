package domain

import (
	"context"
	"time"
)

type WorkspaceSource string

const (
	SourceHelper WorkspaceSource = "helper"
	SourceServer WorkspaceSource = "server"
)

const (
	MaxParallelScans       = 4
	MaxChangedFiles        = 200
	GitCommandTimeout      = 5 * time.Second
	RepositoryListCacheTTL = 5 * time.Second
)

type ChangeStatus string

const (
	StatusModified  ChangeStatus = "modified"
	StatusAdded     ChangeStatus = "added"
	StatusDeleted   ChangeStatus = "deleted"
	StatusRenamed   ChangeStatus = "renamed"
	StatusUntracked ChangeStatus = "untracked"
)

type Workspace struct {
	UserID         string
	Source         WorkspaceSource
	RootPath       string
	HelperURL      string
	HelperTokenEnc string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type RepositorySelection struct {
	UserID     string
	RepoName   string
	IsSelected bool
	UpdatedAt  time.Time
}

type RepositorySummary struct {
	Name            string
	IsGit           bool
	Branch          string
	DirtyCount      int
	UpdatedRelative string
	Error           string
}

type ChangedFile struct {
	Status  ChangeStatus
	Path    string
	Added   int
	Deleted int
}

type CommitInfo struct {
	Hash         string
	Author       string
	RelativeTime string
	Subject      string
}

type RepositoryDetail struct {
	Name            string
	Branch          string
	UpdatedRelative string
	Commit          CommitInfo
	Ahead           int
	Behind          int
	Files           []ChangedFile
	FilesTruncated  bool
	Error           string
}

type GitReader interface {
	ListRepositories(ctx context.Context, rootPath string) ([]RepositorySummary, error)
	ReadRepositories(ctx context.Context, rootPath string, names []string) ([]RepositoryDetail, error)
	ValidateRoot(ctx context.Context, rootPath string) error
}

type WorkspaceRepository interface {
	GetWorkspace(ctx context.Context, userID string) (*Workspace, error)
	UpsertWorkspace(ctx context.Context, workspace *Workspace) error
}

type SelectionRepository interface {
	ListSelected(ctx context.Context, userID string) ([]string, error)
	ReplaceSelections(ctx context.Context, userID string, repoNames []string) error
}
