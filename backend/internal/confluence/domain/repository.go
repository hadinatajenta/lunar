package domain

import "context"

type ConfluenceRepository interface {
	ListDocuments(ctx context.Context, pat string) ([]ConfluenceDocument, error)
	GetDocumentDetail(ctx context.Context, pat string, documentID string) (*ConfluenceDocument, error)
	GetDocumentsByIDs(ctx context.Context, pat string, documentIDs []string) ([]ConfluenceDocument, error)
	GetPageAncestors(ctx context.Context, pat string, pageID string) ([]ConfluencePageRef, error)
	GetChildPages(ctx context.Context, pat string, pageID string) ([]ConfluencePageRef, error)
	GetPageStorage(ctx context.Context, pat string, pageID string) (string, error)
	UpdatePageStorage(ctx context.Context, pat string, pageID string, storage string) (int, string, error)
	UploadAttachment(ctx context.Context, pat string, pageID string, filename string, content []byte, mediaType string) (string, error)
}
