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
