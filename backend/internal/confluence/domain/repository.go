package domain

import "context"

type ConfluenceRepository interface {
	ListDocuments(ctx context.Context, pat string) ([]ConfluenceDocument, error)
	GetDocumentDetail(ctx context.Context, pat string, documentID string) (*ConfluenceDocument, error)
}
