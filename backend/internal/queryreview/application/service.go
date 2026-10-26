package application

import (
	"context"
	"fmt"
	"strings"
	"sync"

	authDomain "lunar/backend/internal/auth/domain"
	confluenceDomain "lunar/backend/internal/confluence/domain"
	"lunar/backend/internal/queryreview/domain"
	sharedErrors "lunar/backend/internal/shared/errors"
)

const DefaultModel = "DeepSeek-V4 Pro (Thinking)"

const queryReviewRepositoryWorkers = 3

const emptyAnalysisSummary = "No added lines in the compared branches matched a database relevant file or query pattern."

type DocumentProvider interface {
	GetDocumentDetail(ctx context.Context, userID string, documentID string) (*confluenceDomain.ConfluenceDocument, error)
	GetPageAncestors(ctx context.Context, userID string, pageID string) ([]confluenceDomain.ConfluencePageRef, error)
	GetChildPages(ctx context.Context, userID string, pageID string) ([]confluenceDomain.ConfluencePageRef, error)
	GetPageStorage(ctx context.Context, userID string, pageID string) (string, error)
	UpdatePageStorage(ctx context.Context, userID string, pageID string, storage string) (int, string, error)
}

type PATAccess interface {
	GetDecryptedSecrets(ctx context.Context, userID string) (*authDomain.DecryptedSecrets, error)
}

type DiffSource interface {
	GetCompareDiff(ctx context.Context, pat string, projectKey string, repoSlug string, sourceBranch string, targetBranch string) (string, error)
	GetPullRequestBranches(ctx context.Context, pat string, projectKey string, repoSlug string, pullRequestID string) (domain.PullRequestBranches, error)
}

type DiffExtractor interface {
	Extract(diffText string) domain.DiffAnalysis
}

type QueryReviewer interface {
	ReviewQueries(ctx context.Context, userID string, modelName string, files []domain.PromptFile) (domain.QueryExtraction, error)
}

type QueryReviewService struct {
	documents  DocumentProvider
	patAccess  PATAccess
	diffSource DiffSource
	extractor  DiffExtractor
	reviewer   QueryReviewer
}

func NewQueryReviewService(
	documents DocumentProvider,
	patAccess PATAccess,
	diffSource DiffSource,
	extractor DiffExtractor,
	reviewer QueryReviewer,
) *QueryReviewService {
	return &QueryReviewService{
		documents:  documents,
		patAccess:  patAccess,
		diffSource: diffSource,
		extractor:  extractor,
		reviewer:   reviewer,
	}
}

func (s *QueryReviewService) GenerateQueryReview(
	ctx context.Context,
	userID string,
	documentID string,
	request domain.QueryReviewRequest,
) (*domain.QueryReviewResult, error) {
	trimmedDocumentID := strings.TrimSpace(documentID)
	if trimmedDocumentID == "" {
		return nil, fmt.Errorf("%w: document id is required", sharedErrors.ErrBadRequest)
	}
	if strings.TrimSpace(userID) == "" {
		return nil, fmt.Errorf("%w: authenticated user is required", sharedErrors.ErrUnauthorized)
	}
	if s.diffSource == nil || s.extractor == nil || s.reviewer == nil {
		return nil, fmt.Errorf("%w: the query review is not configured", sharedErrors.ErrBadGateway)
	}

	resolved, err := s.resolveSources(ctx, userID, trimmedDocumentID, request.Source)
	if err != nil {
		return nil, err
	}

	pat, err := s.resolvePAT(ctx, userID)
	if err != nil {
		return nil, err
	}

	modelName := strings.TrimSpace(request.Model)
	if modelName == "" {
		modelName = DefaultModel
	}

	completed := make([]selectedSource, 0, len(resolved.sources))
	for _, source := range resolved.sources {
		completedSource, err := s.completeSource(ctx, pat, source)
		if err != nil {
			return nil, err
		}
		completed = append(completed, completedSource)
	}

	return &domain.QueryReviewResult{
		DocumentID:          trimmedDocumentID,
		Model:               modelName,
		PromptVersion:       domain.PromptVersion,
		MigPageID:           resolved.migPageID,
		MigPageTitle:        resolved.migPageTitle,
		SkippedRepositories: nonNilSkippedRepositories(resolved.skipped),
		Repositories:        s.processRepositories(ctx, pat, userID, modelName, completed),
	}, nil
}

func (s *QueryReviewService) processRepositories(
	ctx context.Context,
	pat string,
	userID string,
	modelName string,
	sources []selectedSource,
) []domain.RepositoryQueryReview {
	repositories := make([]domain.RepositoryQueryReview, len(sources))
	semaphore := make(chan struct{}, queryReviewRepositoryWorkers)
	var waitGroup sync.WaitGroup

	for index, source := range sources {
		waitGroup.Add(1)
		go func(position int, repositorySource selectedSource) {
			defer waitGroup.Done()

			select {
			case semaphore <- struct{}{}:
			case <-ctx.Done():
				repositories[position] = errorRepositoryReview(newRepositoryReview(repositorySource), ctx.Err())
				return
			}
			defer func() { <-semaphore }()

			repositories[position] = s.processRepository(ctx, pat, userID, modelName, repositorySource)
		}(index, source)
	}

	waitGroup.Wait()
	return repositories
}

func (s *QueryReviewService) processRepository(
	ctx context.Context,
	pat string,
	userID string,
	modelName string,
	source selectedSource,
) domain.RepositoryQueryReview {
	review := newRepositoryReview(source)

	diffText, err := s.diffSource.GetCompareDiff(ctx, pat, source.projectKey, source.repoSlug, source.sourceBranch, source.targetBranch)
	if err != nil {
		return errorRepositoryReview(review, err)
	}

	analysis := s.extractor.Extract(diffText)
	review.AnalyzedFiles = nonNilAnalyzedFiles(analysis.Analyzed)
	review.SkippedFiles = nonNilSkippedFiles(analysis.Skipped)
	review.Truncated = analysis.Truncated
	review.Summary = emptyAnalysisSummary

	if len(analysis.Files) == 0 {
		review.Status = domain.RepositoryStatusNoQueries
		return review
	}

	extraction, err := s.reviewer.ReviewQueries(ctx, userID, modelName, analysis.Files)
	if err != nil {
		return errorRepositoryReview(review, err)
	}

	review.Queries = prefixRepositoryQueries(source.repoSlug, extraction.Queries)
	if summary := strings.TrimSpace(extraction.Summary); summary != "" {
		review.Summary = summary
	}
	if len(review.Queries) == 0 {
		review.Status = domain.RepositoryStatusNoQueries
	}
	return review
}

func newRepositoryReview(source selectedSource) domain.RepositoryQueryReview {
	return domain.RepositoryQueryReview{
		Repository:    source.repoSlug,
		ProjectKey:    source.projectKey,
		SourceBranch:  source.sourceBranch,
		TargetBranch:  source.targetBranch,
		CompareURL:    source.compareURL,
		Status:        domain.RepositoryStatusSuccess,
		AnalyzedFiles: []domain.AnalyzedFile{},
		SkippedFiles:  []domain.SkippedFile{},
		Queries:       []domain.QueryItem{},
	}
}

func errorRepositoryReview(review domain.RepositoryQueryReview, err error) domain.RepositoryQueryReview {
	review.Status = domain.RepositoryStatusError
	review.Error = err.Error()
	review.Queries = []domain.QueryItem{}
	return review
}

func prefixRepositoryQueries(repository string, queries []domain.QueryItem) []domain.QueryItem {
	prefixed := make([]domain.QueryItem, 0, len(queries))
	for _, query := range queries {
		query.Repository = repository
		query.Feature = "[" + repository + "] " + strings.TrimSpace(query.Feature)
		prefixed = append(prefixed, query)
	}
	return prefixed
}

func (s *QueryReviewService) resolvePAT(ctx context.Context, userID string) (string, error) {
	if s.patAccess == nil {
		return "", fmt.Errorf("%w: Bitbucket PAT is not configured", sharedErrors.ErrUnauthorized)
	}

	decrypted, err := s.patAccess.GetDecryptedSecrets(ctx, userID)
	if err != nil {
		return "", fmt.Errorf("resolve the Bitbucket PAT: %w", err)
	}
	if decrypted == nil || strings.TrimSpace(decrypted.BitbucketPAT) == "" {
		return "", fmt.Errorf("%w: Bitbucket PAT is not configured", sharedErrors.ErrUnauthorized)
	}

	return strings.TrimSpace(decrypted.BitbucketPAT), nil
}

func nonNilAnalyzedFiles(files []domain.AnalyzedFile) []domain.AnalyzedFile {
	if files == nil {
		return []domain.AnalyzedFile{}
	}
	return files
}

func nonNilSkippedFiles(files []domain.SkippedFile) []domain.SkippedFile {
	if files == nil {
		return []domain.SkippedFile{}
	}
	return files
}

func nonNilSkippedRepositories(repositories []domain.SkippedRepository) []domain.SkippedRepository {
	if repositories == nil {
		return []domain.SkippedRepository{}
	}
	return repositories
}
