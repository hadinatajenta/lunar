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

const branchSearchLimit = 25

type rawBranchSearchPage struct {
	Values []rawBranchSearchValue `json:"values"`
}

type rawBranchSearchValue struct {
	ID        string `json:"id"`
	DisplayID string `json:"displayId"`
}

func (c *AtlassianBitbucketClient) SearchBranches(ctx context.Context, pat string, repo string, filterText string) ([]string, error) {
	if strings.TrimSpace(pat) == "" {
		return nil, fmt.Errorf("%w: Bitbucket PAT is not configured", sharedErrors.ErrUnauthorized)
	}

	projectKey, repoSlug := parseRepoProject(repo)
	endpoint := fmt.Sprintf(
		"%s/rest/api/1.0/projects/%s/repos/%s/branches?limit=%d&filterText=%s",
		c.baseURL,
		url.PathEscape(projectKey),
		url.PathEscape(repoSlug),
		branchSearchLimit,
		url.QueryEscape(strings.TrimSpace(filterText)),
	)

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Authorization", "Bearer "+pat)
	request.Header.Set("Accept", "application/json")
	request.Header.Set("User-Agent", "Lunar-Workspace/1.0")

	response, err := c.httpClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("failed to search branches in Bitbucket: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode == http.StatusUnauthorized {
		return nil, fmt.Errorf("%w: invalid or expired Bitbucket PAT", sharedErrors.ErrUnauthorized)
	}
	if response.StatusCode == http.StatusForbidden {
		return nil, fmt.Errorf("%w: access forbidden: your PAT does not have permission for this repository", sharedErrors.ErrForbidden)
	}
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bitbucket returned status %d", response.StatusCode)
	}

	var page rawBranchSearchPage
	if err := json.NewDecoder(response.Body).Decode(&page); err != nil {
		return nil, fmt.Errorf("failed to decode Bitbucket branch response: %w", err)
	}

	branchNames := make([]string, 0, len(page.Values))
	for _, value := range page.Values {
		name := strings.TrimSpace(value.DisplayID)
		if name == "" {
			name = strings.TrimPrefix(strings.TrimSpace(value.ID), "refs/heads/")
		}
		if name != "" {
			branchNames = append(branchNames, name)
		}
	}

	return branchNames, nil
}
