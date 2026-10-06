package infrastructure

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	sharedErrors "lunar/backend/internal/shared/errors"
)

const (
	branchListPageLimit          = 100
	maxBranchListPages           = 50
	branchReferencePrefix        = "refs/heads/"
	repositoryNotFoundMessage    = "the repository does not exist"
	defaultBranchNotFoundMessage = "the repository does not exist or has no default branch"
)

type rawBranchListPage struct {
	IsLastPage    bool                   `json:"isLastPage"`
	NextPageStart int                    `json:"nextPageStart"`
	Values        []rawBranchSearchValue `json:"values"`
}

type rawDefaultBranch struct {
	ID        string `json:"id"`
	DisplayID string `json:"displayId"`
}

type branchListPage struct {
	branchNames []string
	nextStart   int
	isLastPage  bool
}

func branchNameFromRawValue(value rawBranchSearchValue) string {
	name := strings.TrimSpace(value.DisplayID)
	if name == "" {
		name = strings.TrimPrefix(strings.TrimSpace(value.ID), branchReferencePrefix)
	}

	return name
}

func branchStatusError(statusCode int, notFoundMessage string) error {
	switch statusCode {
	case http.StatusOK:
		return nil
	case http.StatusUnauthorized:
		return fmt.Errorf("%w: invalid or expired Bitbucket PAT", sharedErrors.ErrUnauthorized)
	case http.StatusForbidden:
		return fmt.Errorf("%w: access forbidden: your PAT does not have permission for this repository", sharedErrors.ErrForbidden)
	case http.StatusNotFound:
		return fmt.Errorf("%w: %s", sharedErrors.ErrNotFound, notFoundMessage)
	default:
		return fmt.Errorf("bitbucket returned status %d", statusCode)
	}
}

func (c *AtlassianBitbucketClient) ListBranches(ctx context.Context, pat string, repo string) ([]string, error) {
	if strings.TrimSpace(pat) == "" {
		return nil, fmt.Errorf("%w: Bitbucket PAT is not configured", sharedErrors.ErrUnauthorized)
	}

	projectKey, repoSlug := parseRepoProject(repo)
	branchNames := make([]string, 0)
	start := 0

	for pageIndex := 0; pageIndex < maxBranchListPages; pageIndex++ {
		page, err := c.fetchBranchListPage(ctx, pat, projectKey, repoSlug, start)
		if err != nil {
			return nil, err
		}
		branchNames = append(branchNames, page.branchNames...)

		if page.isLastPage {
			return branchNames, nil
		}
		if page.nextStart <= start {
			return nil, fmt.Errorf("bitbucket branch listing did not advance beyond start %d", start)
		}
		start = page.nextStart
	}

	return nil, fmt.Errorf("bitbucket branch listing exceeded %d pages", maxBranchListPages)
}

func (c *AtlassianBitbucketClient) fetchBranchListPage(ctx context.Context, pat string, projectKey string, repoSlug string, start int) (branchListPage, error) {
	endpoint := fmt.Sprintf(
		"%s/rest/api/1.0/projects/%s/repos/%s/branches?limit=%d&start=%d",
		c.baseURL,
		url.PathEscape(projectKey),
		url.PathEscape(repoSlug),
		branchListPageLimit,
		start,
	)

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return branchListPage{}, err
	}
	request.Header.Set("Authorization", "Bearer "+pat)
	request.Header.Set("Accept", "application/json")
	request.Header.Set("User-Agent", "Lunar-Workspace/1.0")

	response, err := c.httpClient.Do(request)
	if err != nil {
		return branchListPage{}, fmt.Errorf("failed to list branches in Bitbucket: %w", err)
	}
	defer response.Body.Close()

	if err := branchStatusError(response.StatusCode, repositoryNotFoundMessage); err != nil {
		return branchListPage{}, err
	}

	var rawPage rawBranchListPage
	if err := json.NewDecoder(response.Body).Decode(&rawPage); err != nil {
		return branchListPage{}, fmt.Errorf("failed to decode Bitbucket branch response: %w", err)
	}

	branchNames := make([]string, 0, len(rawPage.Values))
	for _, value := range rawPage.Values {
		if name := branchNameFromRawValue(value); name != "" {
			branchNames = append(branchNames, name)
		}
	}

	return branchListPage{
		branchNames: branchNames,
		nextStart:   rawPage.NextPageStart,
		isLastPage:  rawPage.IsLastPage,
	}, nil
}

func (c *AtlassianBitbucketClient) DefaultBranch(ctx context.Context, pat string, repo string) (string, error) {
	if strings.TrimSpace(pat) == "" {
		return "", fmt.Errorf("%w: Bitbucket PAT is not configured", sharedErrors.ErrUnauthorized)
	}

	projectKey, repoSlug := parseRepoProject(repo)
	endpoint := fmt.Sprintf(
		"%s/rest/api/1.0/projects/%s/repos/%s/default-branch",
		c.baseURL,
		url.PathEscape(projectKey),
		url.PathEscape(repoSlug),
	)

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "", err
	}
	request.Header.Set("Authorization", "Bearer "+pat)
	request.Header.Set("Accept", "application/json")
	request.Header.Set("User-Agent", "Lunar-Workspace/1.0")

	response, err := c.httpClient.Do(request)
	if err != nil {
		return "", fmt.Errorf("failed to read the default branch in Bitbucket: %w", err)
	}
	defer response.Body.Close()

	if err := branchStatusError(response.StatusCode, defaultBranchNotFoundMessage); err != nil {
		return "", err
	}

	var rawBranch rawDefaultBranch
	if err := json.NewDecoder(response.Body).Decode(&rawBranch); err != nil {
		return "", fmt.Errorf("failed to decode the Bitbucket default branch response: %w", err)
	}

	name := strings.TrimSpace(rawBranch.DisplayID)
	if name == "" {
		name = strings.TrimPrefix(strings.TrimSpace(rawBranch.ID), branchReferencePrefix)
	}
	if name == "" {
		return "", fmt.Errorf("bitbucket answered without a default branch name")
	}

	return name, nil
}
