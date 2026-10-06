package application

import (
	"context"
	"fmt"
	"strings"

	sharedErrors "lunar/backend/internal/shared/errors"
)

func (s *BitbucketService) SearchBranches(ctx context.Context, userID string, repo string, filterText string) ([]string, error) {
	if strings.TrimSpace(repo) == "" {
		return nil, fmt.Errorf("%w: repo is required", sharedErrors.ErrBadRequest)
	}

	pat, _ := s.resolveUserPAT(ctx, userID)
	return s.repo.SearchBranches(ctx, pat, repo, filterText)
}

func (s *BitbucketService) ListBranches(ctx context.Context, userID string, repo string) ([]string, error) {
	if strings.TrimSpace(repo) == "" {
		return nil, fmt.Errorf("%w: repo is required", sharedErrors.ErrBadRequest)
	}

	pat, _ := s.resolveUserPAT(ctx, userID)
	return s.repo.ListBranches(ctx, pat, repo)
}

func (s *BitbucketService) DefaultBranch(ctx context.Context, userID string, repo string) (string, error) {
	if strings.TrimSpace(repo) == "" {
		return "", fmt.Errorf("%w: repo is required", sharedErrors.ErrBadRequest)
	}

	pat, _ := s.resolveUserPAT(ctx, userID)
	return s.repo.DefaultBranch(ctx, pat, repo)
}
