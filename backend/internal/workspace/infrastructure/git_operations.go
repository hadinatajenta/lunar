package infrastructure

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"lunar/backend/internal/workspace/domain"
)

const (
	branchRefFormat      = "%(refname:short)\x1f%(committerdate:unix)\x1f%(committerdate:relative)"
	branchFieldDelimiter = "\x1f"
	branchFieldCount     = 3
	stashMessage         = "lunar: stash local changes"
)

var _ domain.GitOperator = (*GitInspector)(nil)

type branchRef struct {
	summary  domain.Branch
	commitAt int64
}

func (i *GitInspector) ListBranches(ctx context.Context, rootPath string, repoName string) ([]domain.Branch, error) {
	repositoryPath, err := i.resolveOperationRepository(ctx, rootPath, repoName)
	if err != nil {
		return nil, err
	}

	output, err := runGit(ctx, repositoryPath, "for-each-ref", "--format="+branchRefFormat, "refs/heads")
	if err != nil {
		return nil, fmt.Errorf("list branches: %w", err)
	}

	currentBranch, err := readBranch(ctx, repositoryPath)
	if err != nil {
		return nil, fmt.Errorf("read current branch: %w", err)
	}

	branchRefs := parseBranchRefs(output, currentBranch)
	sortBranchRefs(branchRefs)

	branches := make([]domain.Branch, 0, len(branchRefs))
	for _, ref := range branchRefs {
		branches = append(branches, ref.summary)
	}

	return branches, nil
}

func (i *GitInspector) CheckoutBranch(ctx context.Context, rootPath string, repoName string, branchName string) (string, error) {
	validBranchName, err := domain.ValidateBranchName(branchName)
	if err != nil {
		return "", err
	}

	repositoryPath, err := i.resolveWritableRepository(ctx, rootPath, repoName)
	if err != nil {
		return "", err
	}

	if err := requireLocalBranch(ctx, repositoryPath, validBranchName); err != nil {
		return "", err
	}

	trackedChangeCount, err := readTrackedChangeCount(ctx, repositoryPath)
	if err != nil {
		return "", fmt.Errorf("read working tree state: %w", err)
	}
	if trackedChangeCount > 0 {
		return "", fmt.Errorf("%w: %d tracked paths have uncommitted changes", domain.ErrWorkingTreeDirty, trackedChangeCount)
	}

	if _, err := runGit(ctx, repositoryPath, "checkout", validBranchName, "--"); err != nil {
		return "", fmt.Errorf("%w: cannot switch branches: %w", domain.ErrGitOperationFailed, err)
	}

	return validBranchName, nil
}

func requireLocalBranch(ctx context.Context, repositoryPath string, branchName string) error {
	output, err := runGit(ctx, repositoryPath, "for-each-ref", "--format=%(refname:short)", "refs/heads")
	if err != nil {
		return fmt.Errorf("read local branches: %w", err)
	}

	for _, line := range strings.Split(output, "\n") {
		if strings.TrimSpace(line) == branchName {
			return nil
		}
	}

	return domain.ErrBranchNotFound
}

func (i *GitInspector) FetchRepository(ctx context.Context, rootPath string, repoName string) (domain.SyncStatus, error) {
	repositoryPath, err := i.resolveWritableRepository(ctx, rootPath, repoName)
	if err != nil {
		return domain.SyncStatus{}, err
	}

	if !hasAnyRemote(ctx, repositoryPath) {
		return domain.SyncStatus{}, fmt.Errorf("%w: the repository has no configured remote", domain.ErrGitOperationFailed)
	}

	if _, err := runGitWithTimeout(ctx, repositoryPath, domain.GitNetworkCommandTimeout, "fetch"); err != nil {
		return domain.SyncStatus{}, networkOperationError("fetch", err)
	}

	ahead, behind, err := readAheadBehind(ctx, repositoryPath)
	if err != nil {
		return domain.SyncStatus{}, fmt.Errorf("read ahead and behind counts: %w", err)
	}

	return domain.SyncStatus{Ahead: ahead, Behind: behind}, nil
}

func (i *GitInspector) PullRepository(ctx context.Context, rootPath string, repoName string) (domain.PullResult, error) {
	repositoryPath, err := i.resolveWritableRepository(ctx, rootPath, repoName)
	if err != nil {
		return domain.PullResult{}, err
	}

	if !hasUpstream(ctx, repositoryPath) {
		return domain.PullResult{}, fmt.Errorf("%w: the current branch has no upstream branch", domain.ErrGitOperationFailed)
	}

	if _, err := runGitWithTimeout(ctx, repositoryPath, domain.GitNetworkCommandTimeout, "pull"); err != nil {
		return domain.PullResult{}, networkOperationError("pull", err)
	}

	branch, err := readBranch(ctx, repositoryPath)
	if err != nil {
		return domain.PullResult{}, fmt.Errorf("read current branch: %w", err)
	}

	commit := readLastCommitOrEmpty(ctx, repositoryPath)
	return domain.PullResult{Branch: branch, UpdatedRelative: commit.RelativeTime}, nil
}

func (i *GitInspector) StashRepository(ctx context.Context, rootPath string, repoName string) (int, error) {
	repositoryPath, err := i.resolveWritableRepository(ctx, rootPath, repoName)
	if err != nil {
		return 0, err
	}

	if _, err := runGit(ctx, repositoryPath, "stash", "push", "-m", stashMessage); err != nil {
		return 0, fmt.Errorf("%w: cannot create a stash entry: %w", domain.ErrGitOperationFailed, err)
	}

	stashCount, err := readStashCount(ctx, repositoryPath)
	if err != nil {
		return 0, fmt.Errorf("count stash entries: %w", err)
	}

	return stashCount, nil
}

func (i *GitInspector) DiscardRepository(ctx context.Context, rootPath string, repoName string) (int, error) {
	repositoryPath, err := i.resolveWritableRepository(ctx, rootPath, repoName)
	if err != nil {
		return 0, err
	}

	restoredCount, err := countRestorablePaths(ctx, repositoryPath)
	if err != nil {
		return 0, fmt.Errorf("read tracked changes: %w", err)
	}

	if _, err := runGit(ctx, repositoryPath, "checkout", "--", "."); err != nil {
		return 0, fmt.Errorf("%w: cannot restore tracked files: %w", domain.ErrGitOperationFailed, err)
	}

	return restoredCount, nil
}

func (i *GitInspector) resolveWritableRepository(ctx context.Context, rootPath string, repoName string) (string, error) {
	if len(i.allowedRoots) == 0 {
		return "", domain.ErrWritesRequireAllowlist
	}

	return i.resolveOperationRepository(ctx, rootPath, repoName)
}

func (i *GitInspector) resolveOperationRepository(ctx context.Context, rootPath string, repoName string) (string, error) {
	resolvedRoot, err := domain.ResolveWorkspaceRoot(rootPath, i.allowedRoots)
	if err != nil {
		return "", err
	}

	repositoryPath, err := domain.ResolveRepoPath(resolvedRoot, repoName)
	if err != nil {
		return "", err
	}

	if !isGitRepository(ctx, repositoryPath) {
		return "", fmt.Errorf("%w: the repository path is not a git repository", domain.ErrNotGitRepository)
	}

	return repositoryPath, nil
}

func parseBranchRefs(output string, currentBranch string) []branchRef {
	refs := make([]branchRef, 0)

	for _, line := range strings.Split(output, "\n") {
		fields := strings.Split(strings.TrimRight(line, "\r"), branchFieldDelimiter)
		if len(fields) != branchFieldCount {
			continue
		}

		name := strings.TrimSpace(fields[0])
		if name == "" {
			continue
		}

		refs = append(refs, branchRef{
			summary: domain.Branch{
				Name:            name,
				IsCurrent:       name == currentBranch,
				UpdatedRelative: strings.TrimSpace(fields[2]),
			},
			commitAt: parseUnixTimestamp(fields[1]),
		})
	}

	return refs
}

func parseUnixTimestamp(value string) int64 {
	seconds, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
	if err != nil {
		return 0
	}
	return seconds
}

func sortBranchRefs(refs []branchRef) {
	sort.SliceStable(refs, func(left int, right int) bool {
		if refs[left].summary.IsCurrent != refs[right].summary.IsCurrent {
			return refs[left].summary.IsCurrent
		}
		if refs[left].commitAt != refs[right].commitAt {
			return refs[left].commitAt > refs[right].commitAt
		}
		return refs[left].summary.Name < refs[right].summary.Name
	})
}

func hasAnyRemote(ctx context.Context, repositoryPath string) bool {
	output, err := runGit(ctx, repositoryPath, "remote")
	if err != nil {
		return false
	}
	return strings.TrimSpace(output) != ""
}

func networkOperationError(operation string, err error) error {
	if errors.Is(err, domain.ErrGitTimeout) {
		return fmt.Errorf("%w: %s did not finish within the network timeout: %w", domain.ErrGitNetworkFailure, operation, err)
	}
	return fmt.Errorf("%w: %s failed, check remote connectivity and credentials: %w", domain.ErrGitNetworkFailure, operation, err)
}

func readStashCount(ctx context.Context, repositoryPath string) (int, error) {
	output, err := runGit(ctx, repositoryPath, "stash", "list", "--format=%gd")
	if err != nil {
		return 0, err
	}

	return countNonEmptyLines(output), nil
}

func countRestorablePaths(ctx context.Context, repositoryPath string) (int, error) {
	output, err := runGit(ctx, repositoryPath, "diff", "--name-only")
	if err != nil {
		return 0, err
	}
	return countNonEmptyLines(output), nil
}

func countNonEmptyLines(output string) int {
	count := 0
	for _, line := range strings.Split(output, "\n") {
		if strings.TrimSpace(line) != "" {
			count++
		}
	}
	return count
}
