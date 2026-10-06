package infrastructure

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"lunar/backend/internal/jira/domain"
	sharedErrors "lunar/backend/internal/shared/errors"
)

func (c *AtlassianJiraClient) SearchMyIssues(ctx context.Context, pat string) ([]domain.JiraIssue, error) {
	if strings.TrimSpace(pat) == "" {
		return nil, fmt.Errorf("%w: Jira PAT is not configured", sharedErrors.ErrUnauthorized)
	}

	if pat == "invalid" {
		return nil, fmt.Errorf("%w: invalid or expired Jira PAT", sharedErrors.ErrUnauthorized)
	}

	if pat == "mock" || c.baseURL == "mock" {
		return c.filterMySeedIssues(), nil
	}

	jql := "assignee = currentUser() AND (statusCategory != Done OR updated >= -30d) ORDER BY updated DESC"
	searchURL := fmt.Sprintf("%s/rest/api/2/search?jql=%s&maxResults=100&fields=summary,status,issuetype,priority,project,updated,created,assignee,parent,duedate,"+JIRA_SPRINT_FIELD,
		c.baseURL, url.QueryEscape(jql))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, searchURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+pat)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Lunar-Workspace/1.0")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		if c.fallbackEnabled {
			return c.filterMySeedIssues(), nil
		}
		return nil, fmt.Errorf("unable to reach Jira BRI API. Please check your VPN connection: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return nil, fmt.Errorf("%w: invalid or expired Jira PAT", sharedErrors.ErrUnauthorized)
	}

	if resp.StatusCode >= http.StatusInternalServerError {
		if c.fallbackEnabled {
			return c.filterMySeedIssues(), nil
		}
		return nil, fmt.Errorf("unable to reach Jira BRI API. Please check your VPN connection: upstream returned %d", resp.StatusCode)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("jira returned status %d", resp.StatusCode)
	}

	var searchResp domain.JiraSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&searchResp); err != nil {
		if c.fallbackEnabled {
			return c.filterMySeedIssues(), nil
		}
		return nil, fmt.Errorf("failed to parse Jira search response: %w", err)
	}

	for i := range searchResp.Issues {
		enrichIssueMetadata(&searchResp.Issues[i])
	}

	return searchResp.Issues, nil
}

func (c *AtlassianJiraClient) GetIssueDetail(ctx context.Context, pat string, issueKey string) (*domain.JiraIssue, error) {
	if strings.TrimSpace(issueKey) == "" {
		return nil, fmt.Errorf("%w: issue key is required", sharedErrors.ErrBadRequest)
	}

	if strings.TrimSpace(pat) == "" {
		return nil, fmt.Errorf("%w: Jira PAT is not configured", sharedErrors.ErrUnauthorized)
	}

	if pat == "invalid" {
		return nil, fmt.Errorf("%w: invalid or expired Jira PAT", sharedErrors.ErrUnauthorized)
	}

	if pat == "mock" || c.baseURL == "mock" {
		return c.findSeedIssue(issueKey)
	}

	targetURL := fmt.Sprintf("%s/rest/api/2/issue/%s?fields=summary,status,issuetype,priority,project,updated,created,assignee,parent,description,duedate,"+JIRA_SPRINT_FIELD,
		c.baseURL, url.PathEscape(issueKey))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, targetURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+pat)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Lunar-Workspace/1.0")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		if c.fallbackEnabled {
			return c.findSeedIssue(issueKey)
		}
		return nil, fmt.Errorf("unable to reach Jira BRI API. Please check your VPN connection: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return nil, fmt.Errorf("%w: invalid or expired Jira PAT", sharedErrors.ErrUnauthorized)
	}

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("%w: issue %s not found", sharedErrors.ErrNotFound, issueKey)
	}

	if resp.StatusCode >= http.StatusInternalServerError {
		if c.fallbackEnabled {
			return c.findSeedIssue(issueKey)
		}
		return nil, fmt.Errorf("unable to reach Jira BRI API. Please check your VPN connection: upstream returned %d", resp.StatusCode)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("jira returned status %d", resp.StatusCode)
	}

	var issue domain.JiraIssue
	if err := json.NewDecoder(resp.Body).Decode(&issue); err != nil {
		if c.fallbackEnabled {
			return c.findSeedIssue(issueKey)
		}
		return nil, fmt.Errorf("failed to parse Jira issue detail: %w", err)
	}

	enrichIssueMetadata(&issue)
	return &issue, nil
}

func (c *AtlassianJiraClient) GetIssueRemoteLinks(ctx context.Context, pat string, issueKey string) ([]domain.JiraRemoteLink, error) {
	if strings.TrimSpace(issueKey) == "" {
		return nil, fmt.Errorf("%w: issue key is required", sharedErrors.ErrBadRequest)
	}

	if strings.TrimSpace(pat) == "" {
		return nil, fmt.Errorf("%w: Jira PAT is not configured", sharedErrors.ErrUnauthorized)
	}

	if pat == "invalid" {
		return nil, fmt.Errorf("%w: invalid or expired Jira PAT", sharedErrors.ErrUnauthorized)
	}

	if pat == "mock" || c.baseURL == "mock" {
		return buildSeedRemoteLinks(issueKey), nil
	}

	targetURL := fmt.Sprintf("%s/rest/api/2/issue/%s/remotelink", c.baseURL, url.PathEscape(issueKey))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, targetURL, nil)
	if err != nil {
		return nil, fmt.Errorf("build remote link request for %s: %w", issueKey, err)
	}
	req.Header.Set("Authorization", "Bearer "+pat)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Lunar-Workspace/1.0")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		if c.fallbackEnabled {
			return buildSeedRemoteLinks(issueKey), nil
		}
		return nil, fmt.Errorf("fetch remote links for %s: %w", issueKey, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return nil, fmt.Errorf("%w: invalid or expired Jira PAT", sharedErrors.ErrUnauthorized)
	}

	if resp.StatusCode == http.StatusNotFound {
		return []domain.JiraRemoteLink{}, nil
	}

	if resp.StatusCode >= http.StatusInternalServerError {
		if c.fallbackEnabled {
			return buildSeedRemoteLinks(issueKey), nil
		}
		return nil, fmt.Errorf("fetch remote links for %s: upstream returned %d", issueKey, resp.StatusCode)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch remote links for %s: upstream returned %d", issueKey, resp.StatusCode)
	}

	var links []domain.JiraRemoteLink
	if err := json.NewDecoder(resp.Body).Decode(&links); err != nil {
		return nil, fmt.Errorf("failed to parse remote links for %s: %w", issueKey, err)
	}
	if links == nil {
		return []domain.JiraRemoteLink{}, nil
	}
	return links, nil
}

func (c *AtlassianJiraClient) findSeedIssue(issueKey string) (*domain.JiraIssue, error) {
	for _, issue := range c.seedIssues {
		if strings.EqualFold(issue.Key, issueKey) {
			copied := issue
			return &copied, nil
		}
	}
	return nil, fmt.Errorf("%w: issue %s not found", sharedErrors.ErrNotFound, issueKey)
}

func (c *AtlassianJiraClient) filterMySeedIssues() []domain.JiraIssue {
	myIssues := make([]domain.JiraIssue, 0, len(c.seedIssues))
	for _, issue := range c.seedIssues {
		if issue.Fields.Assignee != nil {
			myIssues = append(myIssues, issue)
		}
	}
	return myIssues
}
