package application

import (
	"context"
	"fmt"
	"strings"

	sharedErrors "lunar/backend/internal/shared/errors"
)

func (s *ConfluenceService) UploadAttachment(ctx context.Context, userID string, pageID string, filename string, content []byte, mediaType string) (string, error) {
	trimmedPageID := strings.TrimSpace(pageID)
	if trimmedPageID == "" {
		return "", fmt.Errorf("%w: page id is required", sharedErrors.ErrBadRequest)
	}
	if strings.TrimSpace(filename) == "" {
		return "", fmt.Errorf("%w: attachment filename is required", sharedErrors.ErrBadRequest)
	}
	if len(content) == 0 {
		return "", fmt.Errorf("%w: attachment content is required", sharedErrors.ErrBadRequest)
	}

	pat, err := s.resolvePAT(ctx, userID)
	if err != nil {
		return "", err
	}

	return s.repo.UploadAttachment(ctx, pat, trimmedPageID, filename, content, mediaType)
}
