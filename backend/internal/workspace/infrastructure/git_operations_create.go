package infrastructure

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"lunar/backend/internal/workspace/domain"
)

const originRemoteName = "origin"

var _ domain.BranchOperations = (*GitInspector)(nil)

func (i *GitInspector) CreateBranch(ctx context.Context, rootPath string, repoName string, branchName string, baseBranch string) (domain.CreateBranchOutcome, error) {
	validBranchName, err := domain.ValidateBranchName(branchName)
	if err != nil {
		return domain.CreateBranchOutcome{}, err
	}

	validBaseBranch, err := domain.ValidateBranchName(baseBranch)
	if err != nil {
		return domain.CreateBranchOutcome{}, fmt.Errorf("base branch: %w", err)
	}

	repositoryPath, err := i.resolveWritableRepository(ctx, rootPath, repoName)
	if err != nil {
		return domain.CreateBranchOutcome{}, err
	}

	if err := rejectExistingLocalBranch(ctx, repositoryPath, validBranchName); err != nil {
		return domain.CreateBranchOutcome{}, err
	}

	if err := requireLocalBranch(ctx, repositoryPath, validBaseBranch); err != nil {
		return domain.CreateBranchOutcome{}, err
	}

	if err := requireCleanTrackedWorkingTree(ctx, repositoryPath); err != nil {
		return domain.CreateBranchOutcome{}, err
	}

	if _, err := runGit(ctx, repositoryPath, "checkout", "-b", validBranchName, validBaseBranch); err != nil {
		return domain.CreateBranchOutcome{}, fmt.Errorf("%w: cannot create the branch: %w", domain.ErrGitOperationFailed, err)
	}

	headCommit, err := readRepositoryHeadCommit(ctx, repositoryPath)
	if err != nil {
		return domain.CreateBranchOutcome{}, fmt.Errorf("read the created branch commit: %w", err)
	}

	return domain.CreateBranchOutcome{
		Branch:     validBranchName,
		BaseBranch: validBaseBranch,
		HeadCommit: headCommit,
	}, nil
}

func (i *GitInspector) ReadRepositoryCoordinates(ctx context.Context, rootPath string, repoName string) (domain.RepositoryCoordinates, error) {
	repositoryPath, err := i.resolveOperationRepository(ctx, rootPath, repoName)
	if err != nil {
		return domain.RepositoryCoordinates{}, err
	}

	remotes, err := runGit(ctx, repositoryPath, "remote")
	if err != nil {
		return domain.RepositoryCoordinates{}, fmt.Errorf("read the repository remotes: %w", err)
	}
	if !hasRemoteName(remotes, originRemoteName) {
		return domain.RepositoryCoordinates{}, nil
	}

	originURL, err := runGit(ctx, repositoryPath, "remote", "get-url", originRemoteName)
	if err != nil {
		return domain.RepositoryCoordinates{}, fmt.Errorf("read the origin remote: %w", err)
	}

	coordinates, found := domain.ParseRepositoryCoordinates(originURL)
	if !found {
		return domain.RepositoryCoordinates{}, nil
	}
	return coordinates, nil
}

func rejectExistingLocalBranch(ctx context.Context, repositoryPath string, branchName string) error {
	err := requireLocalBranch(ctx, repositoryPath, branchName)
	if err == nil {
		return fmt.Errorf("%w: %s", domain.ErrBranchAlreadyExists, branchName)
	}
	if errors.Is(err, domain.ErrBranchNotFound) {
		return nil
	}
	return err
}

func requireCleanTrackedWorkingTree(ctx context.Context, repositoryPath string) error {
	trackedChangeCount, err := readTrackedChangeCount(ctx, repositoryPath)
	if err != nil {
		return fmt.Errorf("read working tree state: %w", err)
	}
	if trackedChangeCount > 0 {
		return fmt.Errorf("%w: %d tracked paths have uncommitted changes", domain.ErrWorkingTreeDirty, trackedChangeCount)
	}
	return nil
}

func readRepositoryHeadCommit(ctx context.Context, repositoryPath string) (string, error) {
	output, err := runGit(ctx, repositoryPath, "rev-parse", "HEAD")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(output), nil
}

func hasRemoteName(output string, remoteName string) bool {
	for _, line := range strings.Split(output, "\n") {
		if strings.TrimSpace(line) == remoteName {
			return true
		}
	}
	return false
}
