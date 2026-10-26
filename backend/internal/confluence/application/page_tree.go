package application

import (
	"context"
	"fmt"
	"strings"

	"lunar/backend/internal/confluence/domain"
	sharedErrors "lunar/backend/internal/shared/errors"
)

func (s *ConfluenceService) GetPageAncestors(ctx context.Context, userID string, pageID string) ([]domain.ConfluencePageRef, error) {
	trimmedID, err := validatePageTreePageID(pageID)
	if err != nil {
		return nil, err
	}

	pat, err := s.resolvePAT(ctx, userID)
	if err != nil {
		return nil, err
	}

	return s.repo.GetPageAncestors(ctx, pat, trimmedID)
}

func (s *ConfluenceService) GetChildPages(ctx context.Context, userID string, pageID string) ([]domain.ConfluencePageRef, error) {
	trimmedID, err := validatePageTreePageID(pageID)
	if err != nil {
		return nil, err
	}

	pat, err := s.resolvePAT(ctx, userID)
	if err != nil {
		return nil, err
	}

	return s.repo.GetChildPages(ctx, pat, trimmedID)
}

func (s *ConfluenceService) GetPageStorage(ctx context.Context, userID string, pageID string) (string, error) {
	trimmedID, err := validatePageTreePageID(pageID)
	if err != nil {
		return "", err
	}

	pat, err := s.resolvePAT(ctx, userID)
	if err != nil {
		return "", err
	}

	return s.repo.GetPageStorage(ctx, pat, trimmedID)
}

func (s *ConfluenceService) UpdatePageStorage(ctx context.Context, userID string, pageID string, storage string) (int, string, error) {
	trimmedID, err := validatePageTreePageID(pageID)
	if err != nil {
		return 0, "", err
	}
	if strings.TrimSpace(storage) == "" {
		return 0, "", fmt.Errorf("%w: storage value is required", sharedErrors.ErrBadRequest)
	}

	pat, err := s.resolvePAT(ctx, userID)
	if err != nil {
		return 0, "", err
	}

	return s.repo.UpdatePageStorage(ctx, pat, trimmedID, storage)
}

func validatePageTreePageID(pageID string) (string, error) {
	trimmedID := strings.TrimSpace(pageID)
	if trimmedID == "" {
		return "", fmt.Errorf("%w: page id is required", sharedErrors.ErrBadRequest)
	}
	return trimmedID, nil
}
