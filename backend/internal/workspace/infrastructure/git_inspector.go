package infrastructure

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"lunar/backend/internal/shared/cache"
	"lunar/backend/internal/workspace/domain"
)

const (
	messageNotGitRepository      = "not a git repository"
	messageRepositoryReadFailed  = "failed to read repository"
	messageRepositoryNameInvalid = "invalid repository name"
	messageRepositoryOutsideRoot = "repository is outside the workspace root"
	messageRepositoryMissing     = "repository does not exist"
	messageScanCancelled         = "repository scan was cancelled"
)

var _ domain.GitReader = (*GitInspector)(nil)

type GitInspector struct {
	allowedRoots []string
	cache        *cache.Store
}

func NewGitInspector(allowedRoots []string) *GitInspector {
	return newGitInspector(allowedRoots, domain.RepositoryListCacheTTL)
}

func newGitInspector(allowedRoots []string, cacheTTL time.Duration) *GitInspector {
	return &GitInspector{
		allowedRoots: append([]string(nil), allowedRoots...),
		cache:        cache.New(cacheTTL),
	}
}

func (i *GitInspector) ListRepositories(ctx context.Context, rootPath string) ([]domain.RepositorySummary, error) {
	resolvedRoot, err := domain.ResolveWorkspaceRoot(rootPath, i.allowedRoots)
	if err != nil {
		return nil, err
	}

	if cached, found := i.cache.Get(resolvedRoot); found {
		if summaries, valid := cached.([]domain.RepositorySummary); valid {
			return cloneSummaries(summaries), nil
		}
	}

	names, err := listRepositoryNames(resolvedRoot)
	if err != nil {
		return nil, err
	}

	summaries := i.scanRepositorySummaries(ctx, resolvedRoot, names)
	i.cache.Set(resolvedRoot, summaries)
	return cloneSummaries(summaries), nil
}

func (i *GitInspector) ReadRepositories(ctx context.Context, rootPath string, names []string) ([]domain.RepositoryDetail, error) {
	resolvedRoot, err := domain.ResolveWorkspaceRoot(rootPath, i.allowedRoots)
	if err != nil {
		return nil, err
	}

	details := make([]domain.RepositoryDetail, len(names))
	waitGroup := &sync.WaitGroup{}
	semaphore := make(chan struct{}, domain.MaxParallelScans)

	for index, name := range names {
		waitGroup.Add(1)
		go func(index int, name string) {
			defer waitGroup.Done()

			if !acquireScanSlot(ctx, semaphore) {
				details[index] = failedRepositoryDetail(name, messageScanCancelled)
				return
			}
			defer releaseScanSlot(semaphore)

			details[index] = i.readRepositoryDetail(ctx, resolvedRoot, name)
		}(index, name)
	}

	waitGroup.Wait()
	return details, nil
}

func (i *GitInspector) ValidateRoot(_ context.Context, rootPath string) error {
	_, err := domain.ResolveWorkspaceRoot(rootPath, i.allowedRoots)
	return err
}

func (i *GitInspector) scanRepositorySummaries(ctx context.Context, resolvedRoot string, names []string) []domain.RepositorySummary {
	summaries := make([]domain.RepositorySummary, len(names))
	waitGroup := &sync.WaitGroup{}
	semaphore := make(chan struct{}, domain.MaxParallelScans)

	for index, name := range names {
		waitGroup.Add(1)
		go func(index int, name string) {
			defer waitGroup.Done()

			if !acquireScanSlot(ctx, semaphore) {
				summaries[index] = domain.RepositorySummary{Name: name, Error: messageScanCancelled}
				return
			}
			defer releaseScanSlot(semaphore)

			summaries[index] = i.readRepositorySummary(ctx, resolvedRoot, name)
		}(index, name)
	}

	waitGroup.Wait()
	return summaries
}

func (i *GitInspector) readRepositorySummary(ctx context.Context, resolvedRoot string, name string) domain.RepositorySummary {
	repositoryPath, err := domain.ResolveRepoPath(resolvedRoot, name)
	if err != nil {
		return domain.RepositorySummary{Name: name, Error: repositoryErrorMessage(err)}
	}

	if !isGitRepository(ctx, repositoryPath) {
		return domain.RepositorySummary{Name: name, Error: messageNotGitRepository}
	}

	branch, err := readBranch(ctx, repositoryPath)
	if err != nil {
		return domain.RepositorySummary{Name: name, IsGit: true, Error: messageRepositoryReadFailed}
	}

	dirtyCount, err := readDirtyCount(ctx, repositoryPath)
	if err != nil {
		return domain.RepositorySummary{Name: name, IsGit: true, Branch: branch, Error: messageRepositoryReadFailed}
	}

	commit := readLastCommitOrEmpty(ctx, repositoryPath)

	return domain.RepositorySummary{
		Name:            name,
		IsGit:           true,
		Branch:          branch,
		DirtyCount:      dirtyCount,
		UpdatedRelative: commit.RelativeTime,
	}
}

func (i *GitInspector) readRepositoryDetail(ctx context.Context, resolvedRoot string, name string) domain.RepositoryDetail {
	repositoryPath, err := domain.ResolveRepoPath(resolvedRoot, name)
	if err != nil {
		return failedRepositoryDetail(name, repositoryErrorMessage(err))
	}

	if !isGitRepository(ctx, repositoryPath) {
		return failedRepositoryDetail(name, messageNotGitRepository)
	}

	branch, err := readBranch(ctx, repositoryPath)
	if err != nil {
		return failedRepositoryDetail(name, messageRepositoryReadFailed)
	}

	files, truncated, err := readChangedFiles(ctx, repositoryPath)
	if err != nil {
		detail := failedRepositoryDetail(name, messageRepositoryReadFailed)
		detail.Branch = branch
		return detail
	}

	ahead, behind, err := readAheadBehind(ctx, repositoryPath)
	if err != nil {
		detail := failedRepositoryDetail(name, messageRepositoryReadFailed)
		detail.Branch = branch
		return detail
	}

	commit := readLastCommitOrEmpty(ctx, repositoryPath)

	return domain.RepositoryDetail{
		Name:            name,
		Branch:          branch,
		UpdatedRelative: commit.RelativeTime,
		Commit:          commit,
		Ahead:           ahead,
		Behind:          behind,
		Files:           files,
		FilesTruncated:  truncated,
	}
}

func failedRepositoryDetail(name string, message string) domain.RepositoryDetail {
	return domain.RepositoryDetail{
		Name:  name,
		Files: []domain.ChangedFile{},
		Error: message,
	}
}

func listRepositoryNames(resolvedRoot string) ([]string, error) {
	entries, err := os.ReadDir(resolvedRoot)
	if err != nil {
		return nil, fmt.Errorf("read workspace root: %w", err)
	}

	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		if !isDirectory(filepath.Join(resolvedRoot, entry.Name())) {
			continue
		}
		names = append(names, entry.Name())
	}

	sort.Strings(names)
	return names, nil
}

func isDirectory(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

func acquireScanSlot(ctx context.Context, semaphore chan struct{}) bool {
	select {
	case semaphore <- struct{}{}:
		return true
	case <-ctx.Done():
		return false
	}
}

func releaseScanSlot(semaphore chan struct{}) {
	<-semaphore
}

func cloneSummaries(summaries []domain.RepositorySummary) []domain.RepositorySummary {
	cloned := make([]domain.RepositorySummary, len(summaries))
	copy(cloned, summaries)
	return cloned
}

func repositoryErrorMessage(err error) string {
	switch {
	case errors.Is(err, domain.ErrRootNotFound):
		return messageRepositoryMissing
	case errors.Is(err, domain.ErrRepoOutsideRoot):
		return messageRepositoryOutsideRoot
	case errors.Is(err, domain.ErrInvalidRepoName):
		return messageRepositoryNameInvalid
	default:
		return messageRepositoryReadFailed
	}
}
