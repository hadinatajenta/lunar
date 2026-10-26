package application

import (
	"context"
	"fmt"
	"strings"

	"lunar/backend/internal/bitbucket/domain"
	sharedErrors "lunar/backend/internal/shared/errors"
)

type ProjectRepositoryLister interface {
	ListProjectRepositories(ctx context.Context, pat string, projectKey string) ([]domain.ProjectRepository, error)
}

type RemoteRepositoryLister interface {
	ListRemoteRepositories(ctx context.Context, userID string, projectKey string) ([]domain.ProjectRepository, error)
}

var _ RemoteRepositoryLister = (*BitbucketService)(nil)

func (s *BitbucketService) ListRemoteRepositories(ctx context.Context, userID string, projectKey string) ([]domain.ProjectRepository, error) {
	trimmedProject := strings.TrimSpace(projectKey)
	if trimmedProject == "" {
		return nil, fmt.Errorf("%w: Bitbucket project key is required", sharedErrors.ErrBadRequest)
	}

	lister, supported := s.projectRepositoryLister()
	if !supported {
		return nil, fmt.Errorf("%w: the configured Bitbucket client cannot list project repositories", sharedErrors.ErrBadGateway)
	}

	pat, _ := s.resolveUserPAT(ctx, userID)
	return lister.ListProjectRepositories(ctx, pat, trimmedProject)
}

func (s *BitbucketService) projectRepositoryLister() (ProjectRepositoryLister, bool) {
	if s == nil {
		return nil, false
	}

	lister, supported := s.repo.(ProjectRepositoryLister)
	if !supported || lister == nil {
		return nil, false
	}

	return lister, true
}
