package application

import (
	"context"
	"fmt"
	"strings"

	"lunar/backend/internal/queryreview/domain"
	sharedErrors "lunar/backend/internal/shared/errors"
)

func (s *QueryReviewService) ApplyQueryReviewTable(
	ctx context.Context,
	userID string,
	documentID string,
	request domain.ApplyQueryReviewRequest,
) (*domain.ApplyQueryReviewResult, error) {
	trimmedDocumentID := strings.TrimSpace(documentID)
	if trimmedDocumentID == "" {
		return nil, fmt.Errorf("%w: document id is required", sharedErrors.ErrBadRequest)
	}
	if strings.TrimSpace(userID) == "" {
		return nil, fmt.Errorf("%w: authenticated user is required", sharedErrors.ErrUnauthorized)
	}
	if s.documents == nil {
		return nil, fmt.Errorf("%w: the document source is not configured", sharedErrors.ErrBadGateway)
	}

	repositories, err := validateApplyRequest(request)
	if err != nil {
		return nil, err
	}

	storage, err := s.documents.GetPageStorage(ctx, userID, trimmedDocumentID)
	if err != nil {
		return nil, err
	}

	updatedStorage := ReplaceCaptureExplainTable(storage, BuildCaptureExplainTable(repositories))
	version, url, err := s.documents.UpdatePageStorage(ctx, userID, trimmedDocumentID, updatedStorage)
	if err != nil {
		return nil, err
	}

	return &domain.ApplyQueryReviewResult{
		PageID:  trimmedDocumentID,
		Version: version,
		URL:     url,
	}, nil
}

func validateApplyRequest(request domain.ApplyQueryReviewRequest) ([]domain.RepositoryQueryReview, error) {
	if len(request.Repositories) == 0 {
		return nil, fmt.Errorf("%w: at least one repository is required", sharedErrors.ErrBadRequest)
	}
	if len(request.Repositories) > maxCaptureRepositories {
		return nil, fmt.Errorf("%w: at most %d repositories are allowed", sharedErrors.ErrBadRequest, maxCaptureRepositories)
	}

	repositories := make([]domain.RepositoryQueryReview, 0, len(request.Repositories))
	for _, repository := range request.Repositories {
		name := strings.TrimSpace(repository.Repository)
		if exceedsCaptureLimit(name, maxCaptureRepositoryLength) {
			return nil, fmt.Errorf("%w: the repository name is too long", sharedErrors.ErrBadRequest)
		}
		if len(repository.Queries) > maxCaptureQueriesPerRepository {
			return nil, fmt.Errorf("%w: at most %d queries per repository are allowed", sharedErrors.ErrBadRequest, maxCaptureQueriesPerRepository)
		}

		queries := make([]domain.QueryItem, 0, len(repository.Queries))
		for _, query := range repository.Queries {
			feature := strings.TrimSpace(query.Feature)
			statement := strings.TrimSpace(query.QueryStatement)
			indexName := strings.TrimSpace(query.Index)
			createIndex := strings.TrimSpace(query.CreateIndex)
			if exceedsCaptureLimit(feature, maxCaptureFeatureLength) ||
				exceedsCaptureLimit(statement, maxCaptureStatementLength) ||
				exceedsCaptureLimit(indexName, maxCaptureIndexLength) ||
				exceedsCaptureLimit(createIndex, maxCaptureCreateIndexLength) {
				return nil, fmt.Errorf("%w: a query field exceeds the allowed length", sharedErrors.ErrBadRequest)
			}

			queries = append(queries, domain.QueryItem{
				No:             query.No,
				Feature:        feature,
				Repository:     name,
				QueryStatement: statement,
				Index:          indexName,
				CreateIndex:    createIndex,
			})
		}
		repositories = append(repositories, domain.RepositoryQueryReview{
			Repository: name,
			Queries:    queries,
		})
	}

	return repositories, nil
}

func exceedsCaptureLimit(value string, maxLength int) bool {
	return len([]rune(value)) > maxLength
}
