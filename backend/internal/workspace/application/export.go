package application

import (
	"context"
	"fmt"

	"lunar/backend/internal/workspace/domain"
)

func (s *WorkspaceService) ResolveExportRepositoryPath(ctx context.Context, userID string, repoName string) (string, error) {
	workspace, err := s.requireServerWorkspace(ctx, userID)
	if err != nil {
		return "", err
	}

	resolvedRoot, err := domain.ResolveWorkspaceRoot(workspace.RootPath, s.allowedRoots)
	if err != nil {
		return "", fmt.Errorf("resolve workspace root: %w", err)
	}

	repositoryPath, err := domain.ResolveRepoPath(resolvedRoot, repoName)
	if err != nil {
		return "", fmt.Errorf("resolve repository path: %w", err)
	}
	return repositoryPath, nil
}
