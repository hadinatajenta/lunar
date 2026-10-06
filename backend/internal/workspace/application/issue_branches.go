package application

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	jiraDomain "lunar/backend/internal/jira/domain"
	"lunar/backend/internal/shared/cache"
	sharedErrors "lunar/backend/internal/shared/errors"
	"lunar/backend/internal/workspace/domain"
)

const (
	LocalBranchScanCacheTTL      = 30 * time.Second
	branchSourceLocal            = "local"
	unknownRemoteRepositoryCount = 0
	fullLocalCoveragePercent     = 100
)

var ErrWorkspaceNotUsable = errors.New("no usable workspace is configured")

type RepositoryBranchScanner interface {
	ScanRepositoryBranches(ctx context.Context, rootPath string) (map[string][]domain.Branch, error)
}

type BranchMatch struct {
	Repository         string  `json:"repo"`
	Branch             string  `json:"branch"`
	Source             string  `json:"source"`
	IsCurrent          bool    `json:"is_current"`
	IsCloned           bool    `json:"is_cloned"`
	BitbucketProject   string  `json:"bitbucket_project"`
	LastCommit         *string `json:"last_commit"`
	LastCommitRelative *string `json:"last_commit_relative"`
	Ahead              *int    `json:"ahead"`
	Behind             *int    `json:"behind"`
}

type WorkspaceScope struct {
	LocalRepositoryCount      int  `json:"local_repository_count"`
	RemoteRepositoryCount     int  `json:"remote_repository_count"`
	CoveragePercent           int  `json:"coverage_percent"`
	VerifiedRemotely          bool `json:"verified_remotely"`
	RemoteRepositoriesChecked int  `json:"remote_repositories_checked"`
}

type IssueBranches struct {
	Key     string         `json:"key"`
	Matches []BranchMatch  `json:"matches"`
	Scope   WorkspaceScope `json:"scope"`
}

type SprintBranches struct {
	Matches map[string][]BranchMatch `json:"matches"`
	Scope   WorkspaceScope           `json:"scope"`
}

type workspaceBranchIndex struct {
	matchesByKey map[string][]BranchMatch
	scope        WorkspaceScope
}

type IssueBranchService struct {
	scanner RepositoryBranchScanner
	cache   *cache.Store
}

func NewIssueBranchService(scanner RepositoryBranchScanner) *IssueBranchService {
	return newIssueBranchService(scanner, LocalBranchScanCacheTTL)
}

func newIssueBranchService(scanner RepositoryBranchScanner, cacheTTL time.Duration) *IssueBranchService {
	return &IssueBranchService{scanner: scanner, cache: cache.New(cacheTTL)}
}

func (s *IssueBranchService) IssueBranchesAtRoot(ctx context.Context, rootPath string, issueKey string) (*IssueBranches, error) {
	canonicalKey, err := canonicalIssueKey(issueKey)
	if err != nil {
		return nil, err
	}

	index, err := s.branchIndex(ctx, rootPath)
	if err != nil {
		return nil, err
	}

	return &IssueBranches{
		Key:     canonicalKey,
		Matches: index.matchesForKey(canonicalKey),
		Scope:   index.scope,
	}, nil
}

func (s *IssueBranchService) SprintBranchesAtRoot(ctx context.Context, rootPath string, issueKeys []string) (*SprintBranches, error) {
	index, err := s.branchIndex(ctx, rootPath)
	if err != nil {
		return nil, err
	}

	matchesByKey := make(map[string][]BranchMatch, len(issueKeys))
	for _, issueKey := range issueKeys {
		canonicalKey, err := canonicalIssueKey(issueKey)
		if err != nil {
			return nil, err
		}
		if _, duplicate := matchesByKey[canonicalKey]; duplicate {
			continue
		}

		matchesByKey[canonicalKey] = index.matchesForKey(canonicalKey)
	}

	return &SprintBranches{Matches: matchesByKey, Scope: index.scope}, nil
}

func (s *IssueBranchService) branchIndex(ctx context.Context, rootPath string) (*workspaceBranchIndex, error) {
	if s.scanner == nil {
		return nil, ErrServerReaderUnavailable
	}

	cacheKey := strings.TrimSpace(rootPath)
	if cached, found := s.cache.Get(cacheKey); found {
		if index, valid := cached.(*workspaceBranchIndex); valid {
			return index, nil
		}
	}

	branchesByRepository, err := s.scanner.ScanRepositoryBranches(ctx, rootPath)
	if err != nil {
		return nil, fmt.Errorf("scan workspace branches: %w", err)
	}

	index := buildWorkspaceBranchIndex(branchesByRepository)
	s.cache.Set(cacheKey, index)
	return index, nil
}

func canonicalIssueKey(issueKey string) (string, error) {
	canonicalKey, err := jiraDomain.ValidateIssueKey(issueKey)
	if err != nil {
		return "", fmt.Errorf("%w: %s", sharedErrors.ErrBadRequest, err)
	}

	return canonicalKey, nil
}

func buildWorkspaceBranchIndex(branchesByRepository map[string][]domain.Branch) *workspaceBranchIndex {
	matchesByKey := make(map[string][]BranchMatch)
	repositoryNames := make([]string, 0, len(branchesByRepository))
	for repositoryName := range branchesByRepository {
		repositoryNames = append(repositoryNames, repositoryName)
	}
	sort.Strings(repositoryNames)

	for _, repositoryName := range repositoryNames {
		for _, branch := range branchesByRepository[repositoryName] {
			for _, issueKey := range jiraDomain.IssueKeysIn(branch.Name) {
				matchesByKey[issueKey] = append(matchesByKey[issueKey], localBranchMatch(repositoryName, branch))
			}
		}
	}

	return &workspaceBranchIndex{
		matchesByKey: matchesByKey,
		scope:        localWorkspaceScope(len(repositoryNames)),
	}
}

func (i *workspaceBranchIndex) matchesForKey(issueKey string) []BranchMatch {
	matches := i.matchesByKey[issueKey]
	clonedMatches := make([]BranchMatch, len(matches))
	copy(clonedMatches, matches)
	return clonedMatches
}

func localBranchMatch(repositoryName string, branch domain.Branch) BranchMatch {
	return BranchMatch{
		Repository:       repositoryName,
		Branch:           branch.Name,
		Source:           branchSourceLocal,
		IsCurrent:        branch.IsCurrent,
		IsCloned:         true,
		BitbucketProject: "",
	}
}

func localWorkspaceScope(localRepositoryCount int) WorkspaceScope {
	coveragePercent := 0
	if localRepositoryCount > 0 {
		coveragePercent = fullLocalCoveragePercent
	}

	return WorkspaceScope{
		LocalRepositoryCount:      localRepositoryCount,
		RemoteRepositoryCount:     unknownRemoteRepositoryCount,
		CoveragePercent:           coveragePercent,
		VerifiedRemotely:          false,
		RemoteRepositoriesChecked: 0,
	}
}

func (s *WorkspaceService) WorkspaceRootFor(ctx context.Context, userID string) (string, error) {
	workspace, err := s.requireServerWorkspace(ctx, userID)
	if err != nil {
		if errors.Is(err, sharedErrors.ErrBadRequest) {
			return "", fmt.Errorf("%w: %w", ErrWorkspaceNotUsable, err)
		}
		return "", err
	}

	return workspace.RootPath, nil
}
