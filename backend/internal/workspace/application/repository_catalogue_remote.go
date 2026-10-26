package application

import (
	"context"
	"fmt"
	"strings"

	bitbucketApp "lunar/backend/internal/bitbucket/application"
	bitbucketDomain "lunar/backend/internal/bitbucket/domain"
)

const bitbucketProjectKey = "NME"

type RemoteRepositoryLister interface {
	ListRemoteRepositories(ctx context.Context, userID string, projectKey string) ([]bitbucketDomain.ProjectRepository, error)
}

type RemoteRepositoryProvider interface {
	RemoteRepositoryLister
	bitbucketApp.RemoteBranchScanner
}

type RemoteRepositoryCatalogueSource struct {
	lister RemoteRepositoryLister
}

var _ RemoteRepositorySource = (*RemoteRepositoryCatalogueSource)(nil)

func NewRemoteRepositoryCatalogueSource(lister RemoteRepositoryLister) *RemoteRepositoryCatalogueSource {
	return &RemoteRepositoryCatalogueSource{lister: lister}
}

func (s *RemoteRepositoryCatalogueSource) ListRemoteRepositories(ctx context.Context, userID string) ([]RemoteRepository, error) {
	if s == nil || s.lister == nil {
		return nil, ErrServerReaderUnavailable
	}

	repositories, err := s.lister.ListRemoteRepositories(ctx, userID, bitbucketProjectKey)
	if err != nil {
		return nil, fmt.Errorf("list the repositories of Bitbucket project %s: %w", bitbucketProjectKey, err)
	}

	catalogue := make([]RemoteRepository, 0, len(repositories))
	for _, repository := range repositories {
		name := strings.TrimSpace(repository.Slug)
		if name == "" {
			continue
		}
		catalogue = append(catalogue, RemoteRepository{
			Repo:    name,
			Project: strings.TrimSpace(repository.ProjectKey),
		})
	}

	return catalogue, nil
}
