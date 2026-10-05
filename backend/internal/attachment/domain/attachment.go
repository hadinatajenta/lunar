package domain

import (
	"context"
	"time"
)

type Kind string

const (
	KindImage    Kind = "image"
	KindDocument Kind = "document"
)

type Attachment struct {
	ID          string
	UserID      string
	Name        string
	Kind        Kind
	ContentType string
	Size        int64
	Preview     string
	Text        string
	Content     []byte
	CreatedAt   time.Time
}

type Store interface {
	Create(ctx context.Context, attachment *Attachment) error
	GetByID(ctx context.Context, userID string, attachmentID string) (*Attachment, error)
}
