package domain

import (
	"context"
	"errors"
	"net/url"
	"strings"
)

var ErrBranchAlreadyExists = errors.New("the branch already exists in this repository")

type CreateBranchOutcome struct {
	Branch     string
	BaseBranch string
	HeadCommit string
}

type RepositoryCoordinates struct {
	Project string
	Slug    string
}

func (c RepositoryCoordinates) IsKnown() bool {
	return c.Project != "" && c.Slug != ""
}

func (c RepositoryCoordinates) String() string {
	return c.Project + "/" + c.Slug
}

type BranchOperations interface {
	CreateBranch(ctx context.Context, rootPath string, repoName string, branchName string, baseBranch string) (CreateBranchOutcome, error)
	ReadBaseBranchCandidates(ctx context.Context, rootPath string, repoName string) (BaseBranchCandidates, error)
	ReadRepositoryCoordinates(ctx context.Context, rootPath string, repoName string) (RepositoryCoordinates, error)
}

func ParseRepositoryCoordinates(remoteURL string) (RepositoryCoordinates, bool) {
	trimmedURL := strings.TrimSpace(remoteURL)
	if trimmedURL == "" {
		return RepositoryCoordinates{}, false
	}

	path := remotePath(trimmedURL)
	if path == "" {
		return RepositoryCoordinates{}, false
	}

	segments := strings.Split(strings.Trim(path, "/"), "/")
	if len(segments) < 2 {
		return RepositoryCoordinates{}, false
	}

	slug := strings.TrimSuffix(segments[len(segments)-1], ".git")
	project := segments[len(segments)-2]
	if slug == "" || project == "" {
		return RepositoryCoordinates{}, false
	}

	return RepositoryCoordinates{Project: strings.ToUpper(project), Slug: slug}, true
}

func remotePath(remoteURL string) string {
	parsed, err := url.Parse(remoteURL)
	if err == nil && parsed.Scheme != "" {
		switch parsed.Scheme {
		case "http", "https", "ssh":
			if parsed.Host == "" {
				return ""
			}
			return parsed.Path
		default:
			return ""
		}
	}

	_, afterHost, found := strings.Cut(remoteURL, ":")
	if !found {
		return ""
	}
	return afterHost
}
