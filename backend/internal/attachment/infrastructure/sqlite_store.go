package infrastructure

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"lunar/backend/internal/attachment/domain"
	sharedErrors "lunar/backend/internal/shared/errors"
)

const maxAttachmentsPerUser = 50

type SQLiteStore struct {
	db *sql.DB
}

func NewSQLiteStore(db *sql.DB) *SQLiteStore {
	return &SQLiteStore{db: db}
}

func (s *SQLiteStore) Create(ctx context.Context, attachment *domain.Attachment) error {
	insert := `
		INSERT INTO copilot_attachments (
			id, user_id, name, kind, content_type, size, preview, text_content, content, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err := s.db.ExecContext(
		ctx,
		insert,
		attachment.ID,
		attachment.UserID,
		attachment.Name,
		string(attachment.Kind),
		attachment.ContentType,
		attachment.Size,
		attachment.Preview,
		attachment.Text,
		attachment.Content,
		attachment.CreatedAt.Format(time.RFC3339Nano),
	)
	if err != nil {
		return fmt.Errorf("insert attachment: %w", err)
	}

	if err := s.pruneExcess(ctx, attachment.UserID); err != nil {
		return fmt.Errorf("prune attachments: %w", err)
	}
	return nil
}

func (s *SQLiteStore) GetByID(ctx context.Context, userID string, attachmentID string) (*domain.Attachment, error) {
	query := `
		SELECT id, user_id, name, kind, content_type, size, preview, text_content, content, created_at
		FROM copilot_attachments
		WHERE id = ? AND user_id = ?
	`
	row := s.db.QueryRowContext(ctx, query, attachmentID, userID)

	var (
		attachment  domain.Attachment
		kind        string
		createdAt   string
		contentCopy []byte
	)
	if err := row.Scan(
		&attachment.ID,
		&attachment.UserID,
		&attachment.Name,
		&kind,
		&attachment.ContentType,
		&attachment.Size,
		&attachment.Preview,
		&attachment.Text,
		&contentCopy,
		&createdAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, sharedErrors.ErrNotFound
		}
		return nil, fmt.Errorf("select attachment: %w", err)
	}

	attachment.Kind = domain.Kind(kind)
	attachment.Content = contentCopy
	parsedCreatedAt, parseErr := time.Parse(time.RFC3339Nano, createdAt)
	if parseErr != nil {
		return nil, fmt.Errorf("parse attachment created_at: %w", parseErr)
	}
	attachment.CreatedAt = parsedCreatedAt

	return &attachment, nil
}

func (s *SQLiteStore) pruneExcess(ctx context.Context, userID string) error {
	prune := `
		DELETE FROM copilot_attachments
		WHERE user_id = ?
		  AND id NOT IN (
			SELECT id FROM copilot_attachments
			WHERE user_id = ?
			ORDER BY created_at DESC, id DESC
			LIMIT ?
		  )
	`
	if _, err := s.db.ExecContext(ctx, prune, userID, userID, maxAttachmentsPerUser); err != nil {
		return fmt.Errorf("delete expired attachments: %w", err)
	}
	return nil
}
