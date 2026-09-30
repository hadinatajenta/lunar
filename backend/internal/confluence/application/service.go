package application

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	authApp "lunar/backend/internal/auth/application"
	"lunar/backend/internal/confluence/domain"
	jiraDomain "lunar/backend/internal/jira/domain"
	"lunar/backend/internal/shared/cache"
	sharedErrors "lunar/backend/internal/shared/errors"
)

const (
	CONFLUENCE_CACHE_TTL     = 60 * time.Second
	CONFLUENCE_LINKED_BUDGET = 12 * time.Second

	CONFLUENCE_SCOPE_LINKED = "linked"
	CONFLUENCE_SCOPE_ALL    = "all"

	CONFLUENCE_REMOTE_LINK_WORKERS = 5
	CONFLUENCE_MAX_LINKED_ISSUES   = 100
	CONFLUENCE_MAX_LINKED_DOCS     = 60
)

var (
	confluencePagesIDPattern = regexp.MustCompile(`/pages/(\d+)`)
	confluencePageIDPattern  = regexp.MustCompile(`[?&]pageId=(\d+)`)
)

type LinkedIssueProvider interface {
	GetMyIssues(ctx context.Context, userID string) ([]jiraDomain.JiraIssue, error)
	GetIssueRemoteLinks(ctx context.Context, userID string, issueKey string) ([]jiraDomain.JiraRemoteLink, error)
}

type ConfluenceService struct {
	repo         domain.ConfluenceRepository
	authService  *authApp.AuthService
	linkedIssues LinkedIssueProvider
	cacheStore   *cache.Store
}

func NewConfluenceService(
	repo domain.ConfluenceRepository,
	authService *authApp.AuthService,
	linkedIssues LinkedIssueProvider,
) *ConfluenceService {
	return &ConfluenceService{
		repo:         repo,
		authService:  authService,
		linkedIssues: linkedIssues,
		cacheStore:   cache.New(CONFLUENCE_CACHE_TTL),
	}
}

func (s *ConfluenceService) resolvePAT(ctx context.Context, userID string) (string, error) {
	if s.authService == nil || strings.TrimSpace(userID) == "" {
		return "", fmt.Errorf("%w: Confluence PAT is not configured", sharedErrors.ErrUnauthorized)
	}

	decrypted, err := s.authService.GetDecryptedSecrets(ctx, userID)
	if err != nil || decrypted == nil || strings.TrimSpace(decrypted.ConfluencePAT) == "" {
		return "", fmt.Errorf("%w: Confluence PAT is not configured", sharedErrors.ErrUnauthorized)
	}

	return decrypted.ConfluencePAT, nil
}

func (s *ConfluenceService) GetDocuments(ctx context.Context, userID string, scope string) (*domain.ConfluenceDocumentsResponse, error) {
	pat, err := s.resolvePAT(ctx, userID)
	if err != nil {
		return nil, err
	}

	normalizedScope := normalizeScope(scope)

	cacheKey := documentsCacheKey(userID, normalizedScope)
	if !cache.IsForceRefresh(ctx) {
		if cached, exists := s.cacheStore.Get(cacheKey); exists {
			if response, ok := cached.(*domain.ConfluenceDocumentsResponse); ok {
				return response, nil
			}
		}
	}

	var response *domain.ConfluenceDocumentsResponse
	if normalizedScope == CONFLUENCE_SCOPE_ALL {
		response, err = s.fetchAllDocuments(ctx, pat)
	} else {
		response, err = s.fetchLinkedDocuments(ctx, userID, pat)
	}
	if err != nil {
		return nil, err
	}

	if response.Total > 0 && !response.Degraded {
		s.cacheStore.Set(cacheKey, response)
	}
	return response, nil
}

func (s *ConfluenceService) GetDocumentDetail(ctx context.Context, userID string, documentID string) (*domain.ConfluenceDocument, error) {
	trimmedID := strings.TrimSpace(documentID)
	if trimmedID == "" {
		return nil, fmt.Errorf("%w: document id is required", sharedErrors.ErrBadRequest)
	}

	pat, err := s.resolvePAT(ctx, userID)
	if err != nil {
		return nil, err
	}

	cacheKey := documentCacheKey(userID, trimmedID)
	if cached, exists := s.cacheStore.Get(cacheKey); exists {
		if document, ok := cached.(*domain.ConfluenceDocument); ok {
			return document, nil
		}
	}

	document, err := s.repo.GetDocumentDetail(ctx, pat, trimmedID)
	if err != nil {
		return nil, err
	}

	s.cacheStore.Set(cacheKey, document)
	return document, nil
}

func (s *ConfluenceService) fetchAllDocuments(ctx context.Context, pat string) (*domain.ConfluenceDocumentsResponse, error) {
	documents, err := s.repo.ListDocuments(ctx, pat)
	if err != nil {
		return nil, err
	}
	if documents == nil {
		documents = []domain.ConfluenceDocument{}
	}
	return &domain.ConfluenceDocumentsResponse{
		Documents: documents,
		Total:     len(documents),
		Source:    domain.SourceCqlAll,
	}, nil
}

func (s *ConfluenceService) fetchLinkedDocuments(ctx context.Context, userID string, pat string) (*domain.ConfluenceDocumentsResponse, error) {
	if s.linkedIssues == nil {
		return unavailableResponse("Jira integration is not configured"), nil
	}

	jiraIssues, err := s.linkedIssues.GetMyIssues(ctx, userID)
	if err != nil {
		return unavailableResponse("Jira issues are unavailable: " + err.Error()), nil
	}

	budgetCtx, cancel := context.WithTimeout(ctx, CONFLUENCE_LINKED_BUDGET)
	defer cancel()

	pageIDs, degradedReasons := s.collectConfluencePageIDs(budgetCtx, userID, jiraIssues)
	if len(pageIDs) == 0 {
		if len(degradedReasons) > 0 {
			return unavailableResponse(strings.Join(degradedReasons, "; ")), nil
		}
		return &domain.ConfluenceDocumentsResponse{
			Documents: []domain.ConfluenceDocument{},
			Source:    domain.SourceNoLinks,
		}, nil
	}

	documents, fetchErr := s.repo.GetDocumentsByIDs(budgetCtx, pat, pageIDs)
	if fetchErr != nil {
		if errors.Is(fetchErr, sharedErrors.ErrUnauthorized) || len(documents) == 0 {
			return nil, fetchErr
		}
		degradedReasons = append(degradedReasons, fetchErr.Error())
	}

	return &domain.ConfluenceDocumentsResponse{
		Documents:      documents,
		Total:          len(documents),
		Source:         domain.SourceJiraLinks,
		Degraded:       len(degradedReasons) > 0,
		DegradedReason: strings.Join(degradedReasons, "; "),
	}, nil
}

func documentsCacheKey(userID string, scope string) string {
	return "documents|" + userID + "|" + scope
}

func documentCacheKey(userID string, documentID string) string {
	return "document|" + userID + "|" + documentID
}

func (s *ConfluenceService) collectConfluencePageIDs(ctx context.Context, userID string, issues []jiraDomain.JiraIssue) ([]string, []string) {
	if len(issues) > CONFLUENCE_MAX_LINKED_ISSUES {
		issues = issues[:CONFLUENCE_MAX_LINKED_ISSUES]
	}

	linksByIssue := make([][]jiraDomain.JiraRemoteLink, len(issues))
	failures := make([]string, len(issues))
	semaphore := make(chan struct{}, CONFLUENCE_REMOTE_LINK_WORKERS)
	var wg sync.WaitGroup

	for i, issue := range issues {
		wg.Add(1)
		go func(index int, issueKey string) {
			defer wg.Done()

			select {
			case semaphore <- struct{}{}:
			case <-ctx.Done():
				failures[index] = ctx.Err().Error()
				return
			}
			defer func() { <-semaphore }()

			links, err := s.linkedIssues.GetIssueRemoteLinks(ctx, userID, issueKey)
			if err != nil {
				failures[index] = fmt.Sprintf("%s: %v", issueKey, err)
				return
			}
			linksByIssue[index] = links
		}(i, issue.Key)
	}
	wg.Wait()

	seen := make(map[string]bool)
	pageIDs := make([]string, 0, len(issues))
	for _, links := range linksByIssue {
		for _, link := range links {
			pageID := extractConfluencePageID(link.Object.URL)
			if pageID == "" || seen[pageID] {
				continue
			}
			seen[pageID] = true
			pageIDs = append(pageIDs, pageID)
			if len(pageIDs) >= CONFLUENCE_MAX_LINKED_DOCS {
				return pageIDs, collectFailures(failures)
			}
		}
	}
	return pageIDs, collectFailures(failures)
}

func collectFailures(failures []string) []string {
	degradedReasons := make([]string, 0, len(failures))
	for _, failure := range failures {
		if failure != "" {
			degradedReasons = append(degradedReasons, failure)
		}
	}
	return degradedReasons
}

func extractConfluencePageID(rawURL string) string {
	if matches := confluencePagesIDPattern.FindStringSubmatch(rawURL); len(matches) > 1 {
		return matches[1]
	}
	if matches := confluencePageIDPattern.FindStringSubmatch(rawURL); len(matches) > 1 {
		return matches[1]
	}
	return ""
}

func unavailableResponse(reason string) *domain.ConfluenceDocumentsResponse {
	return &domain.ConfluenceDocumentsResponse{
		Documents:      []domain.ConfluenceDocument{},
		Source:         domain.SourceUnavailable,
		Degraded:       true,
		DegradedReason: reason,
	}
}

func normalizeScope(scope string) string {
	if strings.EqualFold(strings.TrimSpace(scope), CONFLUENCE_SCOPE_ALL) {
		return CONFLUENCE_SCOPE_ALL
	}
	return CONFLUENCE_SCOPE_LINKED
}
