package application

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"lunar/backend/internal/shared/cache"
	"lunar/backend/internal/workspace/domain"
)

const (
	RemoteCatalogueCacheTTL      = 10 * time.Minute
	maximumCatalogueRepositories = 2000
)

type RemoteRepository struct {
	Repo    string
	Project string
}

type CataloguedRepository struct {
	Repo     string `json:"repo"`
	Project  string `json:"project"`
	IsCloned bool   `json:"is_cloned"`
}

type RepositoryCatalogue struct {
	Repositories          []CataloguedRepository `json:"repositories"`
	LocalRepositoryCount  int                    `json:"local_repository_count"`
	RemoteRepositoryCount int                    `json:"remote_repository_count"`
	RemoteUnavailable     bool                   `json:"-"`
}

type CatalogueCounts struct {
	LocalRepositoryCount  int `json:"local_repository_count"`
	RemoteRepositoryCount int `json:"remote_repository_count"`
}

type RemoteRepositorySource interface {
	ListRemoteRepositories(ctx context.Context, userID string) ([]RemoteRepository, error)
}

type LocalRepositorySource interface {
	ListRepositories(ctx context.Context, rootPath string) ([]domain.RepositorySummary, error)
}

type RepositoryCatalogueService struct {
	workspaceRoots WorkspaceRootSource
	local          LocalRepositorySource
	remote         RemoteRepositorySource
	localCache     *cache.Store
	remoteCache    *cache.Store
}

func NewRepositoryCatalogueService(workspaceRoots WorkspaceRootSource, local LocalRepositorySource, remote RemoteRepositorySource) *RepositoryCatalogueService {
	return &RepositoryCatalogueService{
		workspaceRoots: workspaceRoots,
		local:          local,
		remote:         remote,
		localCache:     cache.New(LocalBranchScanCacheTTL),
		remoteCache:    cache.New(RemoteCatalogueCacheTTL),
	}
}

func (s *RepositoryCatalogueService) Catalogue(ctx context.Context, userID string) (*RepositoryCatalogue, error) {
	localNames, err := s.localRepositoryNames(ctx, userID)
	if err != nil {
		return nil, err
	}

	remoteRepositories, remoteErr := s.remoteRepositories(ctx, userID)
	catalogue := mergeCatalogue(localNames, remoteRepositories)
	catalogue.RemoteUnavailable = remoteErr != nil
	return catalogue, nil
}

func (s *RepositoryCatalogueService) Search(ctx context.Context, userID string, query string) (*RepositoryCatalogue, error) {
	catalogue, err := s.Catalogue(ctx, userID)
	if err != nil {
		return nil, err
	}

	return filterCatalogue(catalogue, query), nil
}

func (s *RepositoryCatalogueService) Counts(ctx context.Context, userID string) (CatalogueCounts, error) {
	catalogue, err := s.Catalogue(ctx, userID)
	if err != nil {
		return CatalogueCounts{}, err
	}

	return CatalogueCounts{
		LocalRepositoryCount:  catalogue.LocalRepositoryCount,
		RemoteRepositoryCount: catalogue.RemoteRepositoryCount,
	}, nil
}

func (s *RepositoryCatalogueService) localRepositoryNames(ctx context.Context, userID string) ([]string, error) {
	cacheKey := "local:" + strings.TrimSpace(userID)
	if cached, found := s.localCache.Get(cacheKey); found {
		if names, valid := cached.([]string); valid {
			return names, nil
		}
	}

	if s.local == nil {
		return nil, ErrServerReaderUnavailable
	}

	rootPath, err := s.workspaceRoots.WorkspaceRootFor(ctx, userID)
	if err != nil && !errors.Is(err, ErrWorkspaceNotUsable) {
		return nil, fmt.Errorf("resolve workspace root: %w", err)
	}

	names := []string{}
	if rootPath != "" {
		repositories, listErr := s.local.ListRepositories(ctx, rootPath)
		if listErr != nil {
			return nil, fmt.Errorf("list local repositories: %w", listErr)
		}
		for _, repository := range repositories {
			name := strings.TrimSpace(repository.Name)
			if name != "" {
				names = append(names, name)
			}
		}
	}

	s.localCache.Set(cacheKey, names)
	return names, nil
}

func (s *RepositoryCatalogueService) remoteRepositories(ctx context.Context, userID string) ([]RemoteRepository, error) {
	if s.remote == nil {
		return nil, nil
	}

	cacheKey := "remote:" + strings.TrimSpace(userID)
	if cached, found := s.remoteCache.Get(cacheKey); found {
		if repositories, valid := cached.([]RemoteRepository); valid {
			return repositories, nil
		}
	}

	requestCtx, cancel := context.WithTimeout(ctx, remoteCandidateScanTimeout)
	defer cancel()

	repositories, err := s.remote.ListRemoteRepositories(requestCtx, userID)
	if err != nil {
		return nil, fmt.Errorf("list remote repositories: %w", err)
	}

	s.remoteCache.Set(cacheKey, repositories)
	return repositories, nil
}

func mergeCatalogue(localNames []string, remoteRepositories []RemoteRepository) *RepositoryCatalogue {
	localByName := make(map[string]struct{}, len(localNames))
	for _, name := range localNames {
		localByName[strings.ToLower(name)] = struct{}{}
	}

	entries := make([]CataloguedRepository, 0, len(localNames)+len(remoteRepositories))
	seenIdentities := make(map[string]struct{}, len(localNames)+len(remoteRepositories))
	remoteNames := make(map[string]struct{}, len(remoteRepositories))

	for _, remote := range sortedRemoteRepositories(remoteRepositories) {
		name := strings.TrimSpace(remote.Repo)
		if name == "" {
			continue
		}

		project := strings.TrimSpace(remote.Project)
		identity := repositoryIdentity(project, name)
		if _, duplicate := seenIdentities[identity]; duplicate {
			continue
		}

		_, isCloned := localByName[strings.ToLower(name)]
		entries = append(entries, CataloguedRepository{Repo: name, Project: project, IsCloned: isCloned})
		seenIdentities[identity] = struct{}{}
		remoteNames[strings.ToLower(name)] = struct{}{}
	}

	for _, name := range localNames {
		trimmedName := strings.TrimSpace(name)
		if trimmedName == "" {
			continue
		}
		if _, onRemote := remoteNames[strings.ToLower(trimmedName)]; onRemote {
			continue
		}

		identity := repositoryIdentity("", trimmedName)
		if _, duplicate := seenIdentities[identity]; duplicate {
			continue
		}

		entries = append(entries, CataloguedRepository{Repo: trimmedName, IsCloned: true})
		seenIdentities[identity] = struct{}{}
	}

	sortCataloguedRepositories(entries)
	if len(entries) > maximumCatalogueRepositories {
		entries = entries[:maximumCatalogueRepositories]
	}

	return &RepositoryCatalogue{
		Repositories:          entries,
		LocalRepositoryCount:  len(localNames),
		RemoteRepositoryCount: len(remoteRepositories),
	}
}

func filterCatalogue(catalogue *RepositoryCatalogue, query string) *RepositoryCatalogue {
	needle := strings.ToLower(strings.TrimSpace(query))
	filtered := make([]CataloguedRepository, 0, len(catalogue.Repositories))
	for _, repository := range catalogue.Repositories {
		if needle == "" || matchesCatalogueQuery(repository, needle) {
			filtered = append(filtered, repository)
		}
	}

	return &RepositoryCatalogue{
		Repositories:          filtered,
		LocalRepositoryCount:  catalogue.LocalRepositoryCount,
		RemoteRepositoryCount: catalogue.RemoteRepositoryCount,
		RemoteUnavailable:     catalogue.RemoteUnavailable,
	}
}

func matchesCatalogueQuery(repository CataloguedRepository, needle string) bool {
	if strings.Contains(strings.ToLower(repository.Repo), needle) {
		return true
	}

	return repository.Project != "" && strings.Contains(strings.ToLower(repository.Project), needle)
}

func sortedRemoteRepositories(repositories []RemoteRepository) []RemoteRepository {
	sorted := make([]RemoteRepository, len(repositories))
	copy(sorted, repositories)
	sort.SliceStable(sorted, func(first int, second int) bool {
		if !strings.EqualFold(sorted[first].Repo, sorted[second].Repo) {
			return strings.ToLower(sorted[first].Repo) < strings.ToLower(sorted[second].Repo)
		}
		return sorted[first].Project < sorted[second].Project
	})

	return sorted
}

func sortCataloguedRepositories(repositories []CataloguedRepository) {
	sort.SliceStable(repositories, func(first int, second int) bool {
		if !strings.EqualFold(repositories[first].Repo, repositories[second].Repo) {
			return strings.ToLower(repositories[first].Repo) < strings.ToLower(repositories[second].Repo)
		}
		return repositories[first].Project < repositories[second].Project
	})
}

func repositoryIdentity(project string, repo string) string {
	return strings.ToLower(strings.TrimSpace(project) + "/" + strings.TrimSpace(repo))
}

func (c *RepositoryCatalogue) contains(repo string) bool {
	if c == nil {
		return false
	}

	for _, entry := range c.Repositories {
		if strings.EqualFold(entry.Repo, strings.TrimSpace(repo)) {
			return true
		}
	}

	return false
}
