package application

import (
	"context"
	"errors"
	"sort"
	"strings"
	"time"

	authDomain "lunar/backend/internal/auth/domain"
	bitbucketDomain "lunar/backend/internal/bitbucket/domain"
	copilotDomain "lunar/backend/internal/copilot/domain"
	"lunar/backend/internal/dashboard/domain"
	jiraDomain "lunar/backend/internal/jira/domain"
	sharedErrors "lunar/backend/internal/shared/errors"
)

const technicalDocumentLabelPrefix = "DEV Document"
const openPullRequestStatus = "open"
const assignedPullRequestFilter = "mine"

type JiraSummaryProvider interface {
	GetMyIssues(ctx context.Context, userID string) ([]jiraDomain.JiraIssue, error)
}

type BitbucketSummaryProvider interface {
	ListPullRequests(ctx context.Context, userID string, filter string) ([]bitbucketDomain.PullRequest, error)
}

type CopilotSummaryProvider interface {
	ListModels(ctx context.Context, userID string) ([]copilotDomain.ModelInfo, error)
}

type CredentialSummaryProvider interface {
	GetRedactedSecrets(ctx context.Context, userID string) (*authDomain.RedactedSecrets, error)
}

type DashboardService struct {
	jira        JiraSummaryProvider
	bitbucket   BitbucketSummaryProvider
	copilot     CopilotSummaryProvider
	credentials CredentialSummaryProvider
}

func NewDashboardService(
	jira JiraSummaryProvider,
	bitbucket BitbucketSummaryProvider,
	copilot CopilotSummaryProvider,
	credentials CredentialSummaryProvider,
) *DashboardService {
	return &DashboardService{
		jira:        jira,
		bitbucket:   bitbucket,
		copilot:     copilot,
		credentials: credentials,
	}
}

func (s *DashboardService) GetSummary(ctx context.Context, userID string) domain.Summary {
	return domain.Summary{
		SyncedAt:  time.Now().UTC().Format(time.RFC3339),
		Jira:      s.buildJiraSummary(ctx, userID),
		Bitbucket: s.buildBitbucketSummary(ctx, userID),
		Copilot:   s.buildCopilotSummary(ctx, userID),
	}
}

func (s *DashboardService) buildJiraSummary(ctx context.Context, userID string) domain.JiraSummary {
	issues, err := s.jira.GetMyIssues(ctx, userID)
	if err != nil {
		return jiraSummaryFromError(err)
	}
	return domain.JiraSummary{
		Status:             domain.StatusOK,
		IsConfigured:       true,
		AssignedTickets:    len(issues),
		ActiveSprint:       mostFrequentSprintName(issues),
		TechnicalDocuments: countTechnicalDocuments(issues),
	}
}

func jiraSummaryFromError(err error) domain.JiraSummary {
	if errors.Is(err, sharedErrors.ErrUnauthorized) {
		return domain.JiraSummary{Status: domain.StatusUnconfigured}
	}
	return domain.JiraSummary{Status: domain.StatusError, IsConfigured: true}
}

func mostFrequentSprintName(issues []jiraDomain.JiraIssue) string {
	sprintCounts := make(map[string]int)
	mostFrequent := ""
	for _, issue := range issues {
		sprintName := strings.TrimSpace(issue.SprintName)
		if sprintName == "" {
			continue
		}
		sprintCounts[sprintName]++
		if sprintCounts[sprintName] > sprintCounts[mostFrequent] {
			mostFrequent = sprintName
		}
	}
	return mostFrequent
}

func countTechnicalDocuments(issues []jiraDomain.JiraIssue) int {
	documentCount := 0
	for _, issue := range issues {
		if strings.HasPrefix(issue.SubLabel, technicalDocumentLabelPrefix) {
			documentCount++
		}
	}
	return documentCount
}

func (s *DashboardService) buildBitbucketSummary(ctx context.Context, userID string) domain.BitbucketSummary {
	isConfigured := s.hasBitbucketPAT(ctx, userID)
	openPullRequests, err := s.bitbucket.ListPullRequests(ctx, userID, "")
	if err != nil {
		return failedBitbucketSummary(isConfigured)
	}
	assignedPullRequests, err := s.bitbucket.ListPullRequests(ctx, userID, assignedPullRequestFilter)
	if err != nil {
		return failedBitbucketSummary(isConfigured)
	}
	return domain.BitbucketSummary{
		Status:           bitbucketStatusFor(isConfigured),
		IsConfigured:     isConfigured,
		OpenPullRequests: countOpenPullRequests(openPullRequests),
		ReviewRequested:  countAssignedPullRequests(assignedPullRequests),
	}
}

func failedBitbucketSummary(isConfigured bool) domain.BitbucketSummary {
	return domain.BitbucketSummary{Status: domain.StatusError, IsConfigured: isConfigured}
}

func bitbucketStatusFor(isConfigured bool) string {
	if !isConfigured {
		return domain.StatusUnconfigured
	}
	return domain.StatusOK
}

func (s *DashboardService) hasBitbucketPAT(ctx context.Context, userID string) bool {
	secrets, err := s.credentials.GetRedactedSecrets(ctx, userID)
	if err != nil || secrets == nil {
		return false
	}
	return secrets.HasBitbucketPAT
}

func countOpenPullRequests(pullRequests []bitbucketDomain.PullRequest) int {
	openCount := 0
	for _, pullRequest := range pullRequests {
		if pullRequest.Status == openPullRequestStatus {
			openCount++
		}
	}
	return openCount
}

func countAssignedPullRequests(pullRequests []bitbucketDomain.PullRequest) int {
	assignedCount := 0
	for _, pullRequest := range pullRequests {
		if pullRequest.IsAssignedToMe {
			assignedCount++
		}
	}
	return assignedCount
}

func (s *DashboardService) buildCopilotSummary(ctx context.Context, userID string) domain.CopilotSummary {
	models, err := s.copilot.ListModels(ctx, userID)
	if err != nil {
		return domain.CopilotSummary{
			Status:              domain.StatusError,
			ConfiguredProviders: []string{},
		}
	}
	providerCount, configuredProviders := collectConfiguredProviders(models)
	return domain.CopilotSummary{
		Status:              domain.StatusOK,
		ActiveTools:         len(configuredProviders),
		TotalTools:          providerCount,
		ConfiguredProviders: configuredProviders,
	}
}

func collectConfiguredProviders(models []copilotDomain.ModelInfo) (providerCount int, configuredProviders []string) {
	providerSet := make(map[string]bool)
	configuredSet := make(map[string]bool)
	for _, model := range models {
		providerSet[model.Provider] = true
		if model.IsConfigured {
			configuredSet[model.Provider] = true
		}
	}
	configuredProviders = make([]string, 0, len(configuredSet))
	for provider := range configuredSet {
		configuredProviders = append(configuredProviders, provider)
	}
	sort.Strings(configuredProviders)
	return len(providerSet), configuredProviders
}
