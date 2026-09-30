package application

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	authApp "lunar/backend/internal/auth/application"
	"lunar/backend/internal/jira/domain"
	sharedErrors "lunar/backend/internal/shared/errors"
)

type JiraService struct {
	repo         domain.JiraRepository
	authService  *authApp.AuthService
	cacheMu      sync.Mutex
	cacheEntries map[string]jiraCacheEntry
	cacheTTL     time.Duration
}

func NewJiraService(repo domain.JiraRepository, authService *authApp.AuthService) *JiraService {
	return &JiraService{
		repo:         repo,
		authService:  authService,
		cacheEntries: make(map[string]jiraCacheEntry),
		cacheTTL:     JIRA_CACHE_TTL,
	}
}

func (s *JiraService) resolvePAT(ctx context.Context, userID string) (string, error) {
	if s.authService == nil || strings.TrimSpace(userID) == "" {
		return "", fmt.Errorf("%w: Jira PAT is not configured", sharedErrors.ErrUnauthorized)
	}

	decrypted, err := s.authService.GetDecryptedSecrets(ctx, userID)
	if err != nil || decrypted == nil || strings.TrimSpace(decrypted.JiraPAT) == "" {
		return "", fmt.Errorf("%w: Jira PAT is not configured", sharedErrors.ErrUnauthorized)
	}

	return decrypted.JiraPAT, nil
}

func (s *JiraService) GetMyIssues(ctx context.Context, userID string) ([]domain.JiraIssue, error) {
	pat, err := s.resolvePAT(ctx, userID)
	if err != nil {
		return nil, err
	}

	if !isForceRefresh(ctx) {
		if cached, exists := s.loadCacheEntry(issuesCacheKey(userID)); exists {
			if issues, ok := cached.([]domain.JiraIssue); ok {
				return issues, nil
			}
		}
	}

	issues, err := s.repo.SearchMyIssues(ctx, pat)
	if err != nil {
		return nil, err
	}
	s.storeCacheEntry(issuesCacheKey(userID), issues)
	return issues, nil
}

func (s *JiraService) GetBacklog(ctx context.Context, userID string, requestedSquad string) (*domain.JiraBacklogResponse, error) {
	pat, err := s.resolvePAT(ctx, userID)
	if err != nil {
		return nil, err
	}

	cacheKey := backlogCacheKey(userID, requestedSquad)
	if !isForceRefresh(ctx) {
		if cached, exists := s.loadCacheEntry(cacheKey); exists {
			if backlog, ok := cached.(*domain.JiraBacklogResponse); ok {
				return backlog, nil
			}
		}
	}

	backlog, err := s.buildBacklogResponse(ctx, pat, requestedSquad)
	if err != nil {
		return nil, err
	}
	s.storeCacheEntry(cacheKey, backlog)
	return backlog, nil
}

func (s *JiraService) buildBacklogResponse(ctx context.Context, pat string, requestedSquad string) (*domain.JiraBacklogResponse, error) {
	allSprints, err := s.repo.GetBacklogSprints(ctx, pat)
	if err != nil {
		return nil, err
	}

	availableSquads := collectAvailableSquads(allSprints)

	detectedSquad := ""
	myIssues, searchErr := s.repo.SearchMyIssues(ctx, pat)
	if searchErr == nil {
		detectedSquad = detectSquadFromIssues(myIssues)
	}

	targetSquad := strings.TrimSpace(requestedSquad)
	if targetSquad == "" {
		targetSquad = detectedSquad
		if targetSquad == "" {
			targetSquad = "all"
		}
	}

	return buildFilteredBacklog(allSprints, targetSquad, detectedSquad, availableSquads), nil
}

func collectAvailableSquads(allSprints []domain.JiraSprint) []string {
	seenSquads := make(map[string]bool)
	var availableSquads []string
	for _, sprint := range allSprints {
		squad := domain.ExtractSquadName(sprint.Name)
		if squad != "" && !seenSquads[squad] {
			seenSquads[squad] = true
			availableSquads = append(availableSquads, squad)
		}
	}
	return availableSquads
}

func detectSquadFromIssues(myIssues []domain.JiraIssue) string {
	for _, issue := range myIssues {
		if !isIssueDone(issue) && issue.SprintName != "" {
			if squad := domain.ExtractSquadName(issue.SprintName); squad != "" {
				return squad
			}
		}
	}
	for _, issue := range myIssues {
		if issue.SprintName != "" {
			if squad := domain.ExtractSquadName(issue.SprintName); squad != "" {
				return squad
			}
		}
	}
	return ""
}

func isIssueDone(issue domain.JiraIssue) bool {
	return strings.EqualFold(issue.Fields.Status.StatusCategory.Key, "done") ||
		strings.EqualFold(issue.Fields.Status.Name, "done") ||
		strings.EqualFold(issue.Fields.Status.Name, "closed") ||
		strings.EqualFold(issue.Fields.Status.Name, "resolved")
}

func buildFilteredBacklog(allSprints []domain.JiraSprint, targetSquad string, detectedSquad string, availableSquads []string) *domain.JiraBacklogResponse {
	filteredSprints := make([]domain.JiraSprint, 0, len(allSprints))
	for _, sprint := range allSprints {
		if domain.MatchSquad(sprint.Name, targetSquad) {
			filteredSprints = append(filteredSprints, sprint)
		}
	}

	totalIssues := 0
	activeSprintName := ""
	for _, sprint := range filteredSprints {
		totalIssues += len(sprint.Issues)
		if (sprint.State == "active" || sprint.State == "") && activeSprintName == "" {
			activeSprintName = sprint.Name
		}
	}

	return &domain.JiraBacklogResponse{
		Sprints:          filteredSprints,
		TotalIssues:      totalIssues,
		ActiveSprintName: activeSprintName,
		DetectedSquad:    detectedSquad,
		AvailableSquads:  availableSquads,
	}
}

func (s *JiraService) GetIssueDetail(ctx context.Context, userID string, issueKey string) (*domain.JiraIssue, error) {
	trimmedKey := strings.TrimSpace(issueKey)
	if trimmedKey == "" {
		return nil, fmt.Errorf("%w: issue key is required", sharedErrors.ErrBadRequest)
	}

	pat, err := s.resolvePAT(ctx, userID)
	if err != nil {
		return nil, err
	}

	return s.repo.GetIssueDetail(ctx, pat, trimmedKey)
}
