package application

import (
	"context"
	"fmt"
	"strings"

	sharedErrors "lunar/backend/internal/shared/errors"
)

type pageCreator interface {
	GetPageSpaceKey(ctx context.Context, pat string, pageID string) (string, error)
	GetPageVersion(ctx context.Context, pat string, pageID string) (int, error)
	CreatePage(ctx context.Context, pat string, spaceKey string, parentID string, title string, storage string) (string, int, string, error)
}

func (s *ConfluenceService) CreatePage(ctx context.Context, userID string, parentPageID string, title string, storage string) (string, int, string, error) {
	trimmedParentID := strings.TrimSpace(parentPageID)
	if trimmedParentID == "" {
		return "", 0, "", fmt.Errorf("%w: parent page id is required", sharedErrors.ErrBadRequest)
	}
	if strings.TrimSpace(title) == "" {
		return "", 0, "", fmt.Errorf("%w: page title is required", sharedErrors.ErrBadRequest)
	}
	if strings.TrimSpace(storage) == "" {
		return "", 0, "", fmt.Errorf("%w: page storage is required", sharedErrors.ErrBadRequest)
	}

	creator, ok := s.repo.(pageCreator)
	if !ok {
		return "", 0, "", fmt.Errorf("%w: the Confluence page creator is not available", sharedErrors.ErrBadGateway)
	}

	pat, err := s.resolvePAT(ctx, userID)
	if err != nil {
		return "", 0, "", err
	}

	spaceKey, err := creator.GetPageSpaceKey(ctx, pat, trimmedParentID)
	if err != nil {
		return "", 0, "", err
	}

	return creator.CreatePage(ctx, pat, spaceKey, trimmedParentID, title, storage)
}

func (s *ConfluenceService) GetPageVersion(ctx context.Context, userID string, pageID string) (int, error) {
	trimmedID, err := validatePageTreePageID(pageID)
	if err != nil {
		return 0, err
	}

	creator, ok := s.repo.(pageCreator)
	if !ok {
		return 0, fmt.Errorf("%w: the Confluence page version reader is not available", sharedErrors.ErrBadGateway)
	}

	pat, err := s.resolvePAT(ctx, userID)
	if err != nil {
		return 0, err
	}
	return creator.GetPageVersion(ctx, pat, trimmedID)
}
