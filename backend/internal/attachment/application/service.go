package application

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"lunar/backend/internal/attachment/domain"
	sharedErrors "lunar/backend/internal/shared/errors"

	"github.com/google/uuid"
)

var attachmentIDPattern = regexp.MustCompile(`^[A-Za-z0-9-]{1,64}$`)

type Service struct {
	store domain.Store
}

func NewService(store domain.Store) *Service {
	return &Service{store: store}
}

func ValidateAttachmentID(attachmentID string) error {
	if !attachmentIDPattern.MatchString(strings.TrimSpace(attachmentID)) {
		return fmt.Errorf("%w: the attachment id is malformed", sharedErrors.ErrBadRequest)
	}
	return nil
}

func (s *Service) Upload(ctx context.Context, userID string, rawName string, content []byte) (*domain.Attachment, error) {
	if strings.TrimSpace(userID) == "" {
		return nil, fmt.Errorf("%w: an authenticated user is required", sharedErrors.ErrUnauthorized)
	}
	if len(content) == 0 {
		return nil, fmt.Errorf("%w: the uploaded file is empty", sharedErrors.ErrBadRequest)
	}

	accepted, err := AcceptUpload(rawName, content)
	if err != nil {
		return nil, err
	}

	attachment := &domain.Attachment{
		ID:          uuid.NewString(),
		UserID:      userID,
		Name:        accepted.Name,
		Kind:        accepted.Kind,
		ContentType: accepted.ContentType,
		Size:        int64(len(content)),
		Content:     content,
		CreatedAt:   time.Now().UTC(),
	}

	if accepted.Kind == domain.KindDocument {
		extracted, extractErr := ExtractDocumentText(accepted.Name, content)
		if extractErr != nil {
			return nil, fmt.Errorf("%w: %s", ErrUnreadableDocument, accepted.Name)
		}
		attachment.Text = extracted
		attachment.Preview = BuildDocumentPreview(extracted)
	} else {
		attachment.Preview = BuildImagePreview(accepted.ContentType, content)
	}

	if err := s.store.Create(ctx, attachment); err != nil {
		return nil, fmt.Errorf("store attachment: %w", err)
	}
	return attachment, nil
}

func (s *Service) Load(ctx context.Context, userID string, attachmentID string) (*domain.Attachment, error) {
	if strings.TrimSpace(userID) == "" {
		return nil, fmt.Errorf("%w: an authenticated user is required", sharedErrors.ErrUnauthorized)
	}
	if err := ValidateAttachmentID(attachmentID); err != nil {
		return nil, err
	}

	attachment, err := s.store.GetByID(ctx, userID, strings.TrimSpace(attachmentID))
	if err != nil {
		return nil, fmt.Errorf("load attachment: %w", err)
	}
	return attachment, nil
}
