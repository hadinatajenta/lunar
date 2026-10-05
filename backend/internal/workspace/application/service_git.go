package application

import (
	"context"
	"fmt"

	"lunar/backend/internal/workspace/domain"
)

func (s *WorkspaceService) ListBranches(ctx context.Context, userID string, repoName string) ([]domain.Branch, error) {
	workspace, err := s.requireGitWorkspace(ctx, userID)
	if err != nil {
		return nil, err
	}

	branches, err := s.gitOperator.ListBranches(ctx, workspace.RootPath, repoName)
	if err != nil {
		return nil, fmt.Errorf("list branches: %w", err)
	}

	return branches, nil
}

func (s *WorkspaceService) CheckoutBranch(ctx context.Context, userID string, repoName string, branchName string) (string, error) {
	workspace, err := s.requireGitWorkspace(ctx, userID)
	if err != nil {
		return "", err
	}

	branch, err := s.gitOperator.CheckoutBranch(ctx, workspace.RootPath, repoName, branchName)
	if err != nil {
		return "", fmt.Errorf("checkout branch: %w", err)
	}

	return branch, nil
}

func (s *WorkspaceService) FetchRepository(ctx context.Context, userID string, repoName string) (domain.SyncStatus, error) {
	workspace, err := s.requireGitWorkspace(ctx, userID)
	if err != nil {
		return domain.SyncStatus{}, err
	}

	status, err := s.gitOperator.FetchRepository(ctx, workspace.RootPath, repoName)
	if err != nil {
		return domain.SyncStatus{}, fmt.Errorf("fetch repository: %w", err)
	}

	return status, nil
}

func (s *WorkspaceService) PullRepository(ctx context.Context, userID string, repoName string) (domain.PullResult, error) {
	workspace, err := s.requireGitWorkspace(ctx, userID)
	if err != nil {
		return domain.PullResult{}, err
	}

	pullResult, err := s.gitOperator.PullRepository(ctx, workspace.RootPath, repoName)
	if err != nil {
		return domain.PullResult{}, fmt.Errorf("pull repository: %w", err)
	}

	return pullResult, nil
}

func (s *WorkspaceService) StashRepository(ctx context.Context, userID string, repoName string) (int, error) {
	workspace, err := s.requireGitWorkspace(ctx, userID)
	if err != nil {
		return 0, err
	}

	stashCount, err := s.gitOperator.StashRepository(ctx, workspace.RootPath, repoName)
	if err != nil {
		return 0, fmt.Errorf("stash repository: %w", err)
	}

	return stashCount, nil
}

func (s *WorkspaceService) DiscardRepository(ctx context.Context, userID string, repoName string) (int, error) {
	workspace, err := s.requireGitWorkspace(ctx, userID)
	if err != nil {
		return 0, err
	}

	restoredCount, err := s.gitOperator.DiscardRepository(ctx, workspace.RootPath, repoName)
	if err != nil {
		return 0, fmt.Errorf("discard repository changes: %w", err)
	}

	return restoredCount, nil
}

func (s *WorkspaceService) requireGitWorkspace(ctx context.Context, userID string) (*domain.Workspace, error) {
	workspace, err := s.requireServerWorkspace(ctx, userID)
	if err != nil {
		return nil, err
	}

	if s.gitOperator == nil {
		return nil, ErrServerReaderUnavailable
	}

	return workspace, nil
}
