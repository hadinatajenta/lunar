package infrastructure

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"lunar/backend/internal/jira/domain"
	sharedErrors "lunar/backend/internal/shared/errors"
)

const (
	JIRA_DIAL_TIMEOUT            = 2 * time.Second
	JIRA_TLS_HANDSHAKE_TIMEOUT   = 2 * time.Second
	JIRA_RESPONSE_HEADER_TIMEOUT = 4 * time.Second
	JIRA_IDLE_CONN_TIMEOUT       = 30 * time.Second
	JIRA_CLIENT_TIMEOUT          = 6 * time.Second
	JIRA_BACKLOG_BOARD_ID        = 2646
	BACKLOG_SPRINT_FETCH_LIMIT   = 4
)

var errSprintListUnavailable = errors.New("jira sprint list unavailable")

type AtlassianJiraClient struct {
	baseURL         string
	httpClient      *http.Client
	fallbackEnabled bool
	seedIssues      []domain.JiraIssue
	seedSprints     []domain.JiraSprint
}

func NewAtlassianJiraClient(baseURL string) *AtlassianJiraClient {
	cleanURL := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if cleanURL == "" {
		cleanURL = "https://jira.bri.co.id"
	}

	issues := buildSeedIssues()
	sprints := buildSeedSprints(issues)

	return &AtlassianJiraClient{
		baseURL: cleanURL,
		httpClient: &http.Client{
			Timeout: JIRA_CLIENT_TIMEOUT,
			Transport: &http.Transport{
				DialContext: (&net.Dialer{
					Timeout: JIRA_DIAL_TIMEOUT,
				}).DialContext,
				TLSHandshakeTimeout:   JIRA_TLS_HANDSHAKE_TIMEOUT,
				ResponseHeaderTimeout: JIRA_RESPONSE_HEADER_TIMEOUT,
				IdleConnTimeout:       JIRA_IDLE_CONN_TIMEOUT,
				ForceAttemptHTTP2:     true,
			},
		},
		fallbackEnabled: true,
		seedIssues:      issues,
		seedSprints:     sprints,
	}
}

func (c *AtlassianJiraClient) SetFallbackEnabled(enabled bool) {
	c.fallbackEnabled = enabled
}

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
	searchURL := fmt.Sprintf("%s/rest/api/2/search?jql=%s&maxResults=100&fields=summary,status,issuetype,priority,project,updated,created,assignee,parent,duedate,customfield_15310",
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

func (c *AtlassianJiraClient) GetBacklogSprints(ctx context.Context, pat string) ([]domain.JiraSprint, error) {
	if strings.TrimSpace(pat) == "" {
		return nil, fmt.Errorf("%w: Jira PAT is not configured", sharedErrors.ErrUnauthorized)
	}

	if pat == "invalid" {
		return nil, fmt.Errorf("%w: invalid or expired Jira PAT", sharedErrors.ErrUnauthorized)
	}

	if pat == "mock" || c.baseURL == "mock" {
		return c.seedSprints, nil
	}

	sprintList, err := c.fetchSprintList(ctx, pat)
	if err != nil {
		if errors.Is(err, errSprintListUnavailable) {
			return c.seedSprints, nil
		}
		return nil, err
	}

	filteredSprints := filterBacklogSprints(sprintList)
	prioritySprints := selectPrioritySprints(filteredSprints)
	c.enrichBacklogSprints(ctx, pat, prioritySprints)

	if len(prioritySprints) == 0 && c.fallbackEnabled {
		return c.seedSprints, nil
	}

	return prioritySprints, nil
}

func (c *AtlassianJiraClient) fetchSprintList(ctx context.Context, pat string) ([]domain.JiraSprint, error) {
	sprintURL := fmt.Sprintf("%s/rest/agile/1.0/board/%d/sprint?state=active,future", c.baseURL, JIRA_BACKLOG_BOARD_ID)
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

func (c *AtlassianJiraClient) enrichBacklogSprints(ctx context.Context, pat string, sprints []domain.JiraSprint) {
	var wg sync.WaitGroup
	inFlight := make(chan struct{}, BACKLOG_SPRINT_FETCH_LIMIT)

	for sprintIndex := range sprints {
		inFlight <- struct{}{}
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer func() { <-inFlight }()
			c.enrichSprintIssues(ctx, pat, &sprints[sprintIndex])
		}()
	}

	wg.Wait()
}

func (c *AtlassianJiraClient) enrichSprintIssues(ctx context.Context, pat string, sprint *domain.JiraSprint) {
	issues, err := c.fetchSprintIssues(ctx, pat, sprint.ID, sprint.Name)
	if err != nil {
		return
	}
	sprint.Issues = issues
}

func (c *AtlassianJiraClient) fetchSprintIssues(ctx context.Context, pat string, sprintID int, sprintName string) ([]domain.JiraIssue, error) {
	issueURL := fmt.Sprintf("%s/rest/agile/1.0/sprint/%d/issue?fields=summary,status,assignee,issuetype,priority,parent,customfield_15310&maxResults=100",
		c.baseURL, sprintID)
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
		searchResp.Issues[i].SprintName = sprintName
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

	targetURL := fmt.Sprintf("%s/rest/api/2/issue/%s?fields=summary,status,issuetype,priority,project,updated,created,assignee,parent,description,duedate,customfield_15310",
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

func enrichIssueMetadata(issue *domain.JiraIssue) {
	if issue.SprintName == "" && issue.Fields.AcceptanceCriteria != nil {
		rawSprint := *issue.Fields.AcceptanceCriteria
		if idx := strings.Index(rawSprint, "name="); idx != -1 {
			after := rawSprint[idx+5:]
			endIdx := strings.IndexAny(after, ",]")
			if endIdx != -1 {
				issue.SprintName = after[:endIdx]
			} else {
				issue.SprintName = after
			}
		} else if strings.Contains(rawSprint, "Sprint") {
			issue.SprintName = strings.TrimSpace(rawSprint)
		}
	}

	key := strings.ToUpper(issue.Key)
	summary := strings.ToLower(issue.Fields.Summary)
	var typeName string
	if issue.Fields.IssueType != nil {
		typeName = strings.ToLower(issue.Fields.IssueType.Name)
	}

	if strings.HasPrefix(key, "SOP-") ||
		strings.Contains(summary, "dokumen sop") ||
		strings.Contains(summary, "sop operasional") ||
		strings.Contains(summary, "[sop]") ||
		strings.Contains(summary, " sop ") ||
		strings.HasPrefix(summary, "sop ") ||
		strings.HasSuffix(summary, " sop") {
		issue.Kind = "sop"
		issue.SubLabel = "DEV Document · SOP"
		if issue.Points == 0 {
			issue.Points = 1
		}
		return
	}

	if strings.HasPrefix(key, "UT-") ||
		strings.Contains(summary, "dokumen ut") ||
		strings.Contains(summary, "unit test") ||
		strings.Contains(summary, "[ut]") ||
		strings.Contains(summary, " ut ") ||
		strings.HasPrefix(summary, "ut ") ||
		strings.HasSuffix(summary, " ut") ||
		typeName == "ut" {
		issue.Kind = "ut"
		issue.SubLabel = "DEV Document · UT"
		if issue.Points == 0 {
			issue.Points = 2
		}
		return
	}

	if strings.HasPrefix(key, "QR-") ||
		strings.Contains(summary, "query review") ||
		strings.Contains(summary, "review query") ||
		strings.Contains(summary, "[query]") ||
		strings.Contains(summary, "[qr]") ||
		strings.Contains(summary, "query ") ||
		strings.HasPrefix(summary, "query") {
		issue.Kind = "query"
		issue.SubLabel = "DEV Document · Query"
		if issue.Points == 0 {
			issue.Points = 3
		}
		return
	}

	if strings.Contains(typeName, "documentation") ||
		strings.Contains(summary, "dokumentasi") ||
		strings.Contains(summary, "dokumen ") ||
		strings.HasPrefix(summary, "dokumen") {
		issue.Kind = "ut"
		issue.SubLabel = "DEV Document"
		if issue.Points == 0 {
			issue.Points = 2
		}
		return
	}

	if strings.HasPrefix(key, "BUG-") ||
		strings.Contains(typeName, "bug") ||
		strings.Contains(typeName, "defect") ||
		strings.Contains(summary, "bug ") ||
		strings.Contains(summary, "defect ") {
		issue.Kind = "bug"
		issue.SubLabel = "Bug Fix"
		if issue.Points == 0 {
			issue.Points = 3
		}
		return
	}

	if strings.HasPrefix(key, "SUB-") ||
		(issue.Fields.IssueType != nil && issue.Fields.IssueType.Subtask) ||
		strings.Contains(typeName, "sub") {
		issue.Kind = "subtask"
		issue.SubLabel = "Subtask"
		if issue.Points == 0 {
			issue.Points = 2
		}
		return
	}

	if strings.Contains(typeName, "story") {
		issue.Kind = "story"
		issue.SubLabel = "MMS Story"
		return
	}

	issue.Kind = "task"
	issue.SubLabel = "MMS Task"
	if issue.Points == 0 {
		issue.Points = 3
	}
}

func (c *AtlassianJiraClient) VerifyPAT(ctx context.Context, pat string) (*domain.JiraUser, error) {
	if strings.TrimSpace(pat) == "" {
		return nil, fmt.Errorf("%w: Jira PAT is not configured", sharedErrors.ErrUnauthorized)
	}

	if pat == "invalid" {
		return nil, fmt.Errorf("%w: invalid or expired Jira PAT", sharedErrors.ErrUnauthorized)
	}

	if pat == "mock" || c.baseURL == "mock" {
		return &domain.JiraUser{
			DisplayName:  "Lunar Developer (BRI MMS)",
			EmailAddress: "developer@lunar.dev",
			Name:         "developer",
		}, nil
	}

	reqURL := fmt.Sprintf("%s/rest/api/2/myself", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+pat)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Lunar-Workspace/1.0")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		if c.fallbackEnabled {
			return &domain.JiraUser{
				DisplayName:  "Lunar Developer (BRI MMS)",
				EmailAddress: "developer@lunar.dev",
				Name:         "developer",
			}, nil
		}
		return nil, fmt.Errorf("%w: unable to reach Jira BRI API. Please check your VPN connection: %v", sharedErrors.ErrBadGateway, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return nil, fmt.Errorf("%w: invalid or expired Jira PAT", sharedErrors.ErrUnauthorized)
	}

	if resp.StatusCode >= http.StatusInternalServerError {
		if c.fallbackEnabled {
			return &domain.JiraUser{
				DisplayName:  "Lunar Developer (BRI MMS)",
				EmailAddress: "developer@lunar.dev",
				Name:         "developer",
			}, nil
		}
		return nil, fmt.Errorf("%w: unable to reach Jira BRI API. Please check your VPN connection: upstream returned %d", sharedErrors.ErrBadGateway, resp.StatusCode)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("jira returned status %d", resp.StatusCode)
	}

	var user domain.JiraUser
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		if c.fallbackEnabled {
			return &domain.JiraUser{
				DisplayName:  "Lunar Developer (BRI MMS)",
				EmailAddress: "developer@lunar.dev",
				Name:         "developer",
			}, nil
		}
		return nil, fmt.Errorf("failed to parse Jira myself response: %w", err)
	}

	return &user, nil
}
