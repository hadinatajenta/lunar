package application

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"lunar/backend/internal/queryreview/domain"
	sharedErrors "lunar/backend/internal/shared/errors"
)

type selectedSource struct {
	compareURL    string
	origin        string
	projectKey    string
	repoSlug      string
	sourceBranch  string
	targetBranch  string
	pullRequestID string
}

type resolvedSources struct {
	sources      []selectedSource
	migPageID    string
	migPageTitle string
	skipped      []domain.SkippedRepository
}

func (s *QueryReviewService) resolveSources(ctx context.Context, userID string, documentID string, explicit *domain.CompareSource) (resolvedSources, error) {
	if explicit != nil {
		source, err := explicitSelectedSource(*explicit)
		if err != nil {
			return resolvedSources{}, err
		}
		return resolvedSources{sources: []selectedSource{source}, skipped: []domain.SkippedRepository{}}, nil
	}

	if s.documents == nil {
		return resolvedSources{}, fmt.Errorf("%w: the document source is not configured", sharedErrors.ErrBadGateway)
	}

	document, err := s.documents.GetDocumentDetail(ctx, userID, documentID)
	if err != nil {
		return resolvedSources{}, err
	}
	if document == nil {
		return resolvedSources{}, fmt.Errorf("%w: document %s was not found", sharedErrors.ErrNotFound, documentID)
	}

	resolution, err := s.resolveMig(ctx, userID, document)
	if err != nil {
		return resolvedSources{}, err
	}

	sources := make([]selectedSource, 0, len(resolution.rows))
	for _, row := range resolution.rows {
		sources = append(sources, migRowSelectedSource(row))
	}

	return resolvedSources{
		sources:      sources,
		migPageID:    resolution.pageID,
		migPageTitle: resolution.pageTitle,
		skipped:      resolution.skipped,
	}, nil
}

func (s *QueryReviewService) completeSource(ctx context.Context, pat string, source selectedSource) (selectedSource, error) {
	if source.pullRequestID == "" {
		return source, nil
	}

	branches, err := s.diffSource.GetPullRequestBranches(ctx, pat, source.projectKey, source.repoSlug, source.pullRequestID)
	if err != nil {
		return selectedSource{}, err
	}

	source.sourceBranch = branches.SourceBranch
	source.targetBranch = branches.TargetBranch
	source.compareURL = buildCompareURL(source.origin, source.projectKey, source.repoSlug, branches.SourceBranch, branches.TargetBranch)
	return source, nil
}

func explicitSelectedSource(source domain.CompareSource) (selectedSource, error) {
	compareURL := strings.TrimSpace(source.CompareURL)
	projectKey := strings.TrimSpace(source.ProjectKey)
	repoSlug := strings.TrimSpace(source.RepoSlug)
	sourceBranch := strings.TrimSpace(source.SourceBranch)
	targetBranch := strings.TrimSpace(source.TargetBranch)
	origin := ""
	pullRequestID := ""

	if projectKey == "" || repoSlug == "" || sourceBranch == "" || targetBranch == "" {
		if compareURL == "" {
			return selectedSource{}, fmt.Errorf(
				"%w: source.project_key, source.repo_slug, source.source_branch, and source.target_branch are required, or provide source.compare_url",
				sharedErrors.ErrBadRequest,
			)
		}
		candidate, ok := parseCompareCandidate(compareURL)
		if !ok {
			return selectedSource{}, fmt.Errorf("%w: source.compare_url is not a Bitbucket compare or pull request URL", sharedErrors.ErrBadRequest)
		}
		origin = candidate.Origin
		pullRequestID = candidate.PullRequestID
		if projectKey == "" {
			projectKey = candidate.ProjectKey
		}
		if repoSlug == "" {
			repoSlug = candidate.RepoSlug
		}
		if sourceBranch == "" {
			sourceBranch = candidate.SourceBranch
		}
		if targetBranch == "" {
			targetBranch = candidate.TargetBranch
		}
	}

	if projectKey == "" {
		return selectedSource{}, fmt.Errorf("%w: source.project_key is required", sharedErrors.ErrBadRequest)
	}
	if repoSlug == "" {
		return selectedSource{}, fmt.Errorf("%w: source.repo_slug is required", sharedErrors.ErrBadRequest)
	}
	if pullRequestID == "" && sourceBranch == "" {
		return selectedSource{}, fmt.Errorf("%w: source.source_branch is required", sharedErrors.ErrBadRequest)
	}
	if pullRequestID == "" && targetBranch == "" {
		return selectedSource{}, fmt.Errorf("%w: source.target_branch is required", sharedErrors.ErrBadRequest)
	}

	return selectedSource{
		compareURL:    compareURL,
		origin:        origin,
		projectKey:    projectKey,
		repoSlug:      repoSlug,
		sourceBranch:  sourceBranch,
		targetBranch:  targetBranch,
		pullRequestID: pullRequestID,
	}, nil
}

func migRowSelectedSource(row migTableRow) selectedSource {
	repoSlug := row.repoSlug
	if repoSlug == "" {
		repoSlug = row.repository
	}

	return selectedSource{
		compareURL:    row.compareURL,
		origin:        row.origin,
		projectKey:    row.projectKey,
		repoSlug:      repoSlug,
		sourceBranch:  row.sourceBranch,
		targetBranch:  row.targetBranch,
		pullRequestID: row.pullRequestID,
	}
}

func buildCompareURL(origin string, projectKey string, repoSlug string, sourceBranch string, targetBranch string) string {
	trimmedOrigin := strings.TrimRight(strings.TrimSpace(origin), "/")
	if trimmedOrigin == "" {
		return ""
	}

	query := url.Values{}
	query.Set(compareSourceBranchParameter, sourceBranch)
	query.Set(compareTargetBranchParameter, targetBranch)

	return fmt.Sprintf(
		"%s/projects/%s/repos/%s/compare/diff?%s",
		trimmedOrigin,
		url.PathEscape(projectKey),
		url.PathEscape(repoSlug),
		query.Encode(),
	)
}
