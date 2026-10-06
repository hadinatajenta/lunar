package application

import (
	"context"
	"fmt"
	"strings"

	"lunar/backend/internal/workspace/domain"
)

type BranchCreationResult struct {
	Key     string
	Branch  string
	Created []CreatedBranch
	Failed  []FailedBranch
	Skipped []SkippedBranch
}

func (s *WorkspaceService) CreateBranches(ctx context.Context, userID string, request BranchCreationRequest) (*BranchCreationResult, error) {
	validBranchName, err := ValidateBranchCreationRequest(request)
	if err != nil {
		return nil, err
	}

	workspace, err := s.requireGitWorkspace(ctx, userID)
	if err != nil {
		return nil, err
	}

	if err := s.requireWritableWorkspaceRoot(workspace.RootPath); err != nil {
		return nil, err
	}

	outcome := RunBranchCreation(ctx, BranchCreationDependencies{
		Operations: s.branchOperations,
		Finder:     s.remoteBranchFinder,
		Memory:     s.baseBranchMemory,
		UserID:     userID,
		RootPath:   workspace.RootPath,
	}, request, validBranchName)

	return &BranchCreationResult{
		Key:     strings.TrimSpace(request.Key),
		Branch:  validBranchName,
		Created: outcome.Created,
		Failed:  outcome.Failed,
		Skipped: outcome.Skipped,
	}, nil
}

func (s *WorkspaceService) BaseBranchOptions(ctx context.Context, userID string, repoName string) (*domain.BaseBranchOptions, error) {
	workspace, err := s.requireGitWorkspace(ctx, userID)
	if err != nil {
		return nil, err
	}
	if s.branchOperations == nil {
		return nil, ErrServerReaderUnavailable
	}

	candidates, err := s.branchOperations.ReadBaseBranchCandidates(ctx, workspace.RootPath, repoName)
	if err != nil {
		return nil, fmt.Errorf("read base branch candidates: %w", err)
	}

	options := BuildBaseBranchOptions(repoName, candidates, s.baseBranchMemory.Recall(userID, repoName))
	return &options, nil
}

func ValidateWritableWorkspaceRoot(rootPath string, allowedRoots []string) error {
	if len(allowedRoots) == 0 {
		return domain.ErrWritesRequireAllowlist
	}

	_, err := domain.ResolveWorkspaceRoot(rootPath, allowedRoots)
	return err
}

func (s *WorkspaceService) requireWritableWorkspaceRoot(rootPath string) error {
	return ValidateWritableWorkspaceRoot(rootPath, s.allowedRoots)
}
