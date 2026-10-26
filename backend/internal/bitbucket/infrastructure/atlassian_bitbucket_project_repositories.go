package infrastructure

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"lunar/backend/internal/bitbucket/domain"
	sharedErrors "lunar/backend/internal/shared/errors"
)

const (
	projectRepositoryPageLimit = 100
	maxProjectRepositoryPages  = 50
	projectNotFoundMessage     = "the Bitbucket project does not exist"
)

type rawProjectRepositoryPage struct {
	IsLastPage    bool               `json:"isLastPage"`
	NextPageStart int                `json:"nextPageStart"`
	Values        []rawBitbucketRepo `json:"values"`
}

type projectRepositoryPage struct {
	repositories []domain.ProjectRepository
	nextStart    int
	isLastPage   bool
}

func (c *AtlassianBitbucketClient) ListProjectRepositories(ctx context.Context, pat string, projectKey string) ([]domain.ProjectRepository, error) {
	if strings.TrimSpace(pat) == "" {
		return nil, fmt.Errorf("%w: Bitbucket PAT is not configured", sharedErrors.ErrUnauthorized)
	}

	trimmedProject := strings.TrimSpace(projectKey)
	if trimmedProject == "" {
		return nil, fmt.Errorf("%w: Bitbucket project key is required", sharedErrors.ErrBadRequest)
	}

	repositories := make([]domain.ProjectRepository, 0)
	start := 0

	for pageIndex := 0; pageIndex < maxProjectRepositoryPages; pageIndex++ {
		page, err := c.fetchProjectRepositoryPage(ctx, pat, trimmedProject, start)
		if err != nil {
			return nil, err
		}
		repositories = append(repositories, page.repositories...)

		if page.isLastPage {
			return repositories, nil
		}
		if page.nextStart <= start {
			return nil, fmt.Errorf("bitbucket project repository listing did not advance beyond start %d", start)
		}
		start = page.nextStart
	}

	return nil, fmt.Errorf("bitbucket project repository listing exceeded %d pages", maxProjectRepositoryPages)
}

func (c *AtlassianBitbucketClient) fetchProjectRepositoryPage(ctx context.Context, pat string, projectKey string, start int) (projectRepositoryPage, error) {
	endpoint := fmt.Sprintf(
		"%s/rest/api/1.0/projects/%s/repos?limit=%d&start=%d",
		c.baseURL,
		url.PathEscape(projectKey),
		projectRepositoryPageLimit,
		start,
	)

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return projectRepositoryPage{}, err
	}
	request.Header.Set("Authorization", "Bearer "+pat)
	request.Header.Set("Accept", "application/json")
	request.Header.Set("User-Agent", "Lunar-Workspace/1.0")

	response, err := c.httpClient.Do(request)
	if err != nil {
		return projectRepositoryPage{}, fmt.Errorf("failed to list repositories in Bitbucket: %w", err)
	}
	defer response.Body.Close()

	if err := branchStatusError(response.StatusCode, projectNotFoundMessage); err != nil {
		return projectRepositoryPage{}, err
	}

	var rawPage rawProjectRepositoryPage
	if err := json.NewDecoder(response.Body).Decode(&rawPage); err != nil {
		return projectRepositoryPage{}, fmt.Errorf("failed to decode the Bitbucket project repository response: %w", err)
	}

	repositories := make([]domain.ProjectRepository, 0, len(rawPage.Values))
	for _, value := range rawPage.Values {
		repository, usable := projectRepositoryFromRawValue(value, projectKey)
		if !usable {
			continue
		}
		repositories = append(repositories, repository)
	}

	return projectRepositoryPage{
		repositories: repositories,
		nextStart:    rawPage.NextPageStart,
		isLastPage:   rawPage.IsLastPage,
	}, nil
}

func projectRepositoryFromRawValue(value rawBitbucketRepo, fallbackProjectKey string) (domain.ProjectRepository, bool) {
	slug := strings.TrimSpace(value.Slug)
	if slug == "" {
		return domain.ProjectRepository{}, false
	}

	projectKey := strings.TrimSpace(value.Project.Key)
	if projectKey == "" {
		projectKey = strings.TrimSpace(fallbackProjectKey)
	}

	return domain.ProjectRepository{Slug: slug, ProjectKey: projectKey}, true
}
