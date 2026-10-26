package application

import (
	"context"
	"fmt"
	"strings"

	"lunar/backend/internal/queryreview/domain"
	sharedErrors "lunar/backend/internal/shared/errors"
)

func (s *QueryReviewService) ReadMigRepositories(ctx context.Context, userID string, documentID string) ([]domain.MigRepository, error) {
	trimmedID := strings.TrimSpace(documentID)
	if trimmedID == "" {
		return nil, fmt.Errorf("%w: document id is required", sharedErrors.ErrBadRequest)
	}
	if strings.TrimSpace(userID) == "" {
		return nil, fmt.Errorf("%w: authenticated user is required", sharedErrors.ErrUnauthorized)
	}
	if s.documents == nil {
		return nil, fmt.Errorf("%w: the document source is not configured", sharedErrors.ErrBadGateway)
	}

	document, err := s.documents.GetDocumentDetail(ctx, userID, trimmedID)
	if err != nil {
		return nil, err
	}
	if document == nil {
		return nil, fmt.Errorf("%w: document %s was not found", sharedErrors.ErrNotFound, trimmedID)
	}

	resolution, err := s.readMig(ctx, userID, document)
	if err != nil {
		return nil, err
	}

	return migRepositories(resolution.rows), nil
}

func migRepositories(rows []migTableRow) []domain.MigRepository {
	repositories := make([]domain.MigRepository, 0, len(rows))
	for _, row := range rows {
		repository := strings.TrimSpace(row.repoSlug)
		if repository == "" {
			repository = strings.TrimSpace(row.repository)
		}
		if repository == "" {
			continue
		}
		repositories = append(repositories, domain.MigRepository{
			Repository:   repository,
			Branch:       strings.TrimSpace(row.sourceBranch),
			TargetBranch: strings.TrimSpace(row.targetBranch),
			CompareURL:   strings.TrimSpace(row.compareURL),
		})
	}
	return repositories
}

func (s *QueryReviewService) MigTargets(ctx context.Context, userID string, documentID string) (*domain.MigTargetsResult, error) {
	trimmedID := strings.TrimSpace(documentID)
	if trimmedID == "" {
		return nil, fmt.Errorf("%w: document id is required", sharedErrors.ErrBadRequest)
	}
	if strings.TrimSpace(userID) == "" {
		return nil, fmt.Errorf("%w: authenticated user is required", sharedErrors.ErrUnauthorized)
	}
	if s.documents == nil {
		return nil, fmt.Errorf("%w: the document source is not configured", sharedErrors.ErrBadGateway)
	}

	document, err := s.documents.GetDocumentDetail(ctx, userID, trimmedID)
	if err != nil {
		return nil, err
	}
	if document == nil {
		return nil, fmt.Errorf("%w: document %s was not found", sharedErrors.ErrNotFound, trimmedID)
	}

	resolved, err := s.resolveMig(ctx, userID, document)
	if err != nil {
		return nil, err
	}

	targets := make([]domain.MigTarget, 0, len(resolved.rows))
	for _, row := range resolved.rows {
		repository := strings.TrimSpace(row.repoSlug)
		if repository == "" {
			repository = strings.TrimSpace(row.repository)
		}
		branch := strings.TrimSpace(row.sourceBranch)
		if repository == "" || branch == "" {
			continue
		}
		targets = append(targets, domain.MigTarget{
			Repository: repository,
			Branch:     branch,
			BaseRef:    strings.TrimSpace(row.targetBranch),
		})
	}

	return &domain.MigTargetsResult{
		MigPageTitle:        resolved.pageTitle,
		Targets:             targets,
		SkippedRepositories: nonNilSkippedRepositories(resolved.skipped),
	}, nil
}
