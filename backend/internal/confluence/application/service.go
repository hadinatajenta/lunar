package application

import (
	"context"
	"fmt"
	"strings"
	"time"

	authApp "lunar/backend/internal/auth/application"
	"lunar/backend/internal/confluence/domain"
	"lunar/backend/internal/shared/cache"
	sharedErrors "lunar/backend/internal/shared/errors"
)

const CONFLUENCE_CACHE_TTL = 60 * time.Second

type ConfluenceService struct {
	repo        domain.ConfluenceRepository
	authService *authApp.AuthService
	cacheStore  *cache.Store
}

func NewConfluenceService(repo domain.ConfluenceRepository, authService *authApp.AuthService) *ConfluenceService {
	return &ConfluenceService{
		repo:        repo,
		authService: authService,
		cacheStore:  cache.New(CONFLUENCE_CACHE_TTL),
	}
}

func (s *ConfluenceService) resolvePAT(ctx context.Context, userID string) (string, error) {
	if s.authService == nil || strings.TrimSpace(userID) == "" {
		return "", fmt.Errorf("%w: Confluence PAT is not configured", sharedErrors.ErrUnauthorized)
	}

	decrypted, err := s.authService.GetDecryptedSecrets(ctx, userID)
	if err != nil || decrypted == nil || strings.TrimSpace(decrypted.ConfluencePAT) == "" {
		return "", fmt.Errorf("%w: Confluence PAT is not configured", sharedErrors.ErrUnauthorized)
	}

	return decrypted.ConfluencePAT, nil
}

func (s *ConfluenceService) GetDocuments(ctx context.Context, userID string) (*domain.ConfluenceDocumentsResponse, error) {
	pat, err := s.resolvePAT(ctx, userID)
	if err != nil {
		return nil, err
	}

	cacheKey := documentsCacheKey(userID)
	if cached, exists := s.cacheStore.Get(cacheKey); exists {
		if response, ok := cached.(*domain.ConfluenceDocumentsResponse); ok {
			return response, nil
		}
	}

	documents, err := s.repo.ListDocuments(ctx, pat)
	if err != nil {
		return nil, err
	}
	if documents == nil {
		documents = []domain.ConfluenceDocument{}
	}

	response := &domain.ConfluenceDocumentsResponse{
		Documents: documents,
		Total:     len(documents),
	}
	s.cacheStore.Set(cacheKey, response)
	return response, nil
}

func (s *ConfluenceService) GetDocumentDetail(ctx context.Context, userID string, documentID string) (*domain.ConfluenceDocument, error) {
	trimmedID := strings.TrimSpace(documentID)
	if trimmedID == "" {
		return nil, fmt.Errorf("%w: document id is required", sharedErrors.ErrBadRequest)
	}

	pat, err := s.resolvePAT(ctx, userID)
	if err != nil {
		return nil, err
	}

	cacheKey := documentCacheKey(userID, trimmedID)
	if cached, exists := s.cacheStore.Get(cacheKey); exists {
		if document, ok := cached.(*domain.ConfluenceDocument); ok {
			return document, nil
		}
	}

	document, err := s.repo.GetDocumentDetail(ctx, pat, trimmedID)
	if err != nil {
		return nil, err
	}

	s.cacheStore.Set(cacheKey, document)
	return document, nil
}

func documentsCacheKey(userID string) string {
	return "documents|" + userID
}

func documentCacheKey(userID string, documentID string) string {
	return "document|" + userID + "|" + documentID
}
