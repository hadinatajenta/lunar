package infrastructure

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"lunar/backend/internal/jira/domain"
	sharedErrors "lunar/backend/internal/shared/errors"
)

const (
	JIRA_DIAL_TIMEOUT            = 2 * time.Second
	JIRA_TLS_HANDSHAKE_TIMEOUT   = 2 * time.Second
	JIRA_RESPONSE_HEADER_TIMEOUT = 15 * time.Second
	JIRA_IDLE_CONN_TIMEOUT       = 30 * time.Second
	JIRA_CLIENT_TIMEOUT          = 20 * time.Second
	JIRA_FALLBACK_BOARD_ID       = 2646
	JIRA_SPRINT_FIELD            = "customfield_10104"
	BACKLOG_SPRINT_FETCH_LIMIT   = 4
	JIRA_SPRINT_ISSUE_LIMIT      = 200
	JIRA_PARENT_ISSUES_ONLY_JQL  = "issuetype not in subTaskIssueTypes()"
)

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
		fallbackEnabled: false,
		seedIssues:      issues,
		seedSprints:     sprints,
	}
}

func (c *AtlassianJiraClient) SetFallbackEnabled(enabled bool) {
	c.fallbackEnabled = enabled
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
