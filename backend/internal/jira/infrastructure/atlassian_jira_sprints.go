package infrastructure

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"lunar/backend/internal/jira/domain"
	sharedErrors "lunar/backend/internal/shared/errors"
)

var errSprintListUnavailable = errors.New("jira sprint list unavailable")

func (c *AtlassianJiraClient) GetBacklogSprints(ctx context.Context, pat string, boardID int) ([]domain.JiraSprint, error) {
	if strings.TrimSpace(pat) == "" {
		return nil, fmt.Errorf("%w: Jira PAT is not configured", sharedErrors.ErrUnauthorized)
	}

	if pat == "invalid" {
		return nil, fmt.Errorf("%w: invalid or expired Jira PAT", sharedErrors.ErrUnauthorized)
	}

	if pat == "mock" || c.baseURL == "mock" {
		return c.seedSprints, nil
	}

	sprintList, err := c.fetchSprintList(ctx, pat, resolveBacklogBoardID(boardID))
	if err != nil {
		if errors.Is(err, errSprintListUnavailable) {
			return c.seedSprints, nil
		}
		return nil, err
	}

	filteredSprints := filterBacklogSprints(sprintList)
	prioritySprints := selectPrioritySprints(filteredSprints)

	if err := c.enrichBacklogSprints(ctx, pat, prioritySprints); err != nil {
		return nil, err
	}

	if len(prioritySprints) == 0 && c.fallbackEnabled {
		return c.seedSprints, nil
	}

	return prioritySprints, nil
}

func resolveBacklogBoardID(boardID int) int {
	if boardID > 0 {
		return boardID
	}
	return JIRA_FALLBACK_BOARD_ID
}

func (c *AtlassianJiraClient) fetchSprintList(ctx context.Context, pat string, boardID int) ([]domain.JiraSprint, error) {
	sprintURL := fmt.Sprintf("%s/rest/agile/1.0/board/%d/sprint?state=active", c.baseURL, boardID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, sprintURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+pat)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Lunar-Workspace/1.0")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, c.sprintListFallbackOr(fmt.Errorf("unable to reach Jira BRI API. Please check your VPN connection: %w", err))
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return nil, fmt.Errorf("%w: invalid or expired Jira PAT", sharedErrors.ErrUnauthorized)
	}

	if resp.StatusCode >= http.StatusInternalServerError {
		return nil, c.sprintListFallbackOr(fmt.Errorf("unable to reach Jira BRI API. Please check your VPN connection: upstream returned %d", resp.StatusCode))
	}

	if resp.StatusCode != http.StatusOK {
		return nil, c.sprintListFallbackOr(fmt.Errorf("jira returned status %d", resp.StatusCode))
	}

	var sprintResp struct {
		Values []domain.JiraSprint `json:"values"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&sprintResp); err != nil {
		return nil, c.sprintListFallbackOr(fmt.Errorf("failed to parse Jira sprint list: %w", err))
	}

	return sprintResp.Values, nil
}

func (c *AtlassianJiraClient) sprintListFallbackOr(fetchErr error) error {
	if c.fallbackEnabled {
		return errSprintListUnavailable
	}
	return fetchErr
}

func (c *AtlassianJiraClient) enrichBacklogSprints(ctx context.Context, pat string, sprints []domain.JiraSprint) error {
	var wg sync.WaitGroup
	inFlight := make(chan struct{}, BACKLOG_SPRINT_FETCH_LIMIT)
	failures := make(chan error, len(sprints))

	for sprintIndex := range sprints {
		inFlight <- struct{}{}
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer func() { <-inFlight }()
			if err := c.enrichSprintIssues(ctx, pat, &sprints[sprintIndex]); err != nil {
				failures <- err
			}
		}()
	}

	wg.Wait()
	close(failures)

	for failure := range failures {
		return fmt.Errorf("load sprint issues: %w", failure)
	}
	return nil
}

func (c *AtlassianJiraClient) enrichSprintIssues(ctx context.Context, pat string, sprint *domain.JiraSprint) error {
	issues, err := c.fetchSprintIssues(ctx, pat, *sprint)
	if err != nil {
		return fmt.Errorf("sprint %d: %w", sprint.ID, err)
	}
	sprint.Issues = issues
	return nil
}

func (c *AtlassianJiraClient) fetchSprintIssues(ctx context.Context, pat string, sprint domain.JiraSprint) ([]domain.JiraIssue, error) {
	issueURL := fmt.Sprintf("%s/rest/agile/1.0/sprint/%d/issue?fields=summary,status,assignee,issuetype,priority,parent,"+JIRA_SPRINT_FIELD+"&maxResults=%d&jql=%s",
		c.baseURL, sprint.ID, JIRA_SPRINT_ISSUE_LIMIT, url.QueryEscape(JIRA_PARENT_ISSUES_ONLY_JQL))
	ireq, err := http.NewRequestWithContext(ctx, http.MethodGet, issueURL, nil)
	if err != nil {
		return nil, fmt.Errorf("build sprint issue request: %w", err)
	}
	ireq.Header.Set("Authorization", "Bearer "+pat)
	ireq.Header.Set("Accept", "application/json")
	ireq.Header.Set("User-Agent", "Lunar-Workspace/1.0")

	iresp, err := c.httpClient.Do(ireq)
	if err != nil {
		return nil, fmt.Errorf("fetch sprint issues: %w", err)
	}
	defer iresp.Body.Close()

	if iresp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch sprint issues: upstream returned %d", iresp.StatusCode)
	}

	var searchResp domain.JiraSearchResponse
	if err := json.NewDecoder(iresp.Body).Decode(&searchResp); err != nil {
		return nil, fmt.Errorf("parse sprint issues: %w", err)
	}

	for i := range searchResp.Issues {
		searchResp.Issues[i].SprintName = sprint.Name
		searchResp.Issues[i].SprintBoardID = sprint.OriginBoardID
		enrichIssueMetadata(&searchResp.Issues[i])
	}

	return searchResp.Issues, nil
}
