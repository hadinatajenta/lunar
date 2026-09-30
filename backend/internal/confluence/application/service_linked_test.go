package application

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	authApp "lunar/backend/internal/auth/application"
	authDomain "lunar/backend/internal/auth/domain"
	"lunar/backend/internal/confluence/domain"
	confluenceInfra "lunar/backend/internal/confluence/infrastructure"
	jiraDomain "lunar/backend/internal/jira/domain"
	"lunar/backend/internal/shared/crypto"
	sharedErrors "lunar/backend/internal/shared/errors"
)

type stubLinkedIssueProvider struct {
	issues       []jiraDomain.JiraIssue
	issuesErr    error
	linksByIssue map[string][]jiraDomain.JiraRemoteLink
	issuesCalls  atomic.Int64
	linksCalls   atomic.Int64
}

func (s *stubLinkedIssueProvider) GetMyIssues(ctx context.Context, userID string) ([]jiraDomain.JiraIssue, error) {
	s.issuesCalls.Add(1)
	if s.issuesErr != nil {
		return nil, s.issuesErr
	}
	return s.issues, nil
}

func (s *stubLinkedIssueProvider) GetIssueRemoteLinks(ctx context.Context, userID string, issueKey string) ([]jiraDomain.JiraRemoteLink, error) {
	s.linksCalls.Add(1)
	return s.linksByIssue[issueKey], nil
}

type spyRepository struct {
	inner              domain.ConfluenceRepository
	listDocumentsCalls atomic.Int64
	getByIDsCalls      atomic.Int64
	lastRequestedIDs   []string
	getByIDsErr        error
	getByIDsDocuments  []domain.ConfluenceDocument
}

func (r *spyRepository) ListDocuments(ctx context.Context, pat string) ([]domain.ConfluenceDocument, error) {
	r.listDocumentsCalls.Add(1)
	return r.inner.ListDocuments(ctx, pat)
}

func (r *spyRepository) GetDocumentDetail(ctx context.Context, pat string, documentID string) (*domain.ConfluenceDocument, error) {
	return r.inner.GetDocumentDetail(ctx, pat, documentID)
}

func (r *spyRepository) GetDocumentsByIDs(ctx context.Context, pat string, documentIDs []string) ([]domain.ConfluenceDocument, error) {
	r.getByIDsCalls.Add(1)
	r.lastRequestedIDs = append([]string{}, documentIDs...)
	if r.getByIDsErr != nil || r.getByIDsDocuments != nil {
		return r.getByIDsDocuments, r.getByIDsErr
	}
	return r.inner.GetDocumentsByIDs(ctx, pat, documentIDs)
}

type stubVaultRepo struct {
	secrets map[string]*authDomain.UserSecrets
}

func (s *stubVaultRepo) SaveSecrets(ctx context.Context, secrets *authDomain.UserSecrets) error {
	s.secrets[secrets.UserID] = secrets
	return nil
}

func (s *stubVaultRepo) GetSecretsByUserID(ctx context.Context, userID string) (*authDomain.UserSecrets, error) {
	secret, exists := s.secrets[userID]
	if !exists || secret == nil {
		return nil, sharedErrors.ErrNotFound
	}
	return secret, nil
}

func setupLinkedService(t *testing.T, upstreamURL string, provider LinkedIssueProvider, repo domain.ConfluenceRepository) *ConfluenceService {
	t.Helper()

	encKey := "0123456789abcdef0123456789abcdef"
	vaultRepo := &stubVaultRepo{secrets: make(map[string]*authDomain.UserSecrets)}

	validPATEnc, _ := crypto.EncryptSecret(validConfluencePAT, encKey)
	vaultRepo.secrets[configuredUserID] = &authDomain.UserSecrets{
		UserID:           configuredUserID,
		ConfluencePATEnc: validPATEnc,
	}

	authService := authApp.NewAuthService(nil, vaultRepo, encKey, "jwt-secret", time.Hour)
	if repo == nil {
		repo = confluenceInfra.NewConfluenceClient(upstreamURL)
	}
	return NewConfluenceService(repo, authService, provider)
}

func TestExtractConfluencePageID(t *testing.T) {
	testCases := []struct {
		name     string
		rawURL   string
		expected string
	}{
		{"pages path form", "https://confluence.bri.co.id/pages/123", "123"},
		{"viewpage action with pageId", "https://confluence.bri.co.id/pages/viewpage.action?pageId=456", "456"},
		{"pageId mid query", "https://confluence.bri.co.id/x?a=1&pageId=789&b=2", "789"},
		{"pageId first param", "https://confluence.bri.co.id/x?pageId=1010", "1010"},
		{"no id returns empty", "https://confluence.bri.co.id/display/DEV/Some-Page", ""},
		{"non numeric pageId returns empty", "https://confluence.bri.co.id/x?pageId=abc", ""},
		{"empty url returns empty", "", ""},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			actual := extractConfluencePageID(testCase.rawURL)
			if actual != testCase.expected {
				t.Errorf("extractConfluencePageID(%q) = %q, want %q", testCase.rawURL, actual, testCase.expected)
			}
		})
	}
}

func TestConfluenceService_LinkedScopeReturnsNoLinksWhenNoRemoteLinks(t *testing.T) {
	provider := &stubLinkedIssueProvider{
		issues:       []jiraDomain.JiraIssue{{ID: "1", Key: "CRMMS-1"}},
		linksByIssue: map[string][]jiraDomain.JiraRemoteLink{"CRMMS-1": nil},
	}
	spy := &spyRepository{inner: confluenceInfra.NewConfluenceClient("https://confluence.example.com")}
	service := setupLinkedService(t, "", provider, spy)

	response, err := service.GetDocuments(context.Background(), configuredUserID, CONFLUENCE_SCOPE_LINKED)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if response.Source != domain.SourceNoLinks {
		t.Errorf("expected source no-links, got %q", response.Source)
	}
	if len(response.Documents) != 0 || response.Total != 0 {
		t.Errorf("expected empty documents, got %d", response.Total)
	}
	if spy.listDocumentsCalls.Load() != 0 {
		t.Errorf("expected ListDocuments to never be called in linked scope, got %d calls", spy.listDocumentsCalls.Load())
	}
}

func TestConfluenceService_LinkedScopeDegradesWhenIssuesUnavailable(t *testing.T) {
	provider := &stubLinkedIssueProvider{issuesErr: errors.New("jira unreachable")}
	spy := &spyRepository{inner: confluenceInfra.NewConfluenceClient("https://confluence.example.com")}
	service := setupLinkedService(t, "", provider, spy)

	response, err := service.GetDocuments(context.Background(), configuredUserID, CONFLUENCE_SCOPE_LINKED)
	if err != nil {
		t.Fatalf("expected degraded response instead of error, got %v", err)
	}
	if response.Source != domain.SourceUnavailable {
		t.Errorf("expected source unavailable, got %q", response.Source)
	}
	if !response.Degraded {
		t.Error("expected degraded flag to be set")
	}
	if !strings.Contains(response.DegradedReason, "jira unreachable") {
		t.Errorf("expected degraded reason to include cause, got %q", response.DegradedReason)
	}
	if spy.listDocumentsCalls.Load() != 0 {
		t.Errorf("expected fallback ListDocuments to never be called, got %d calls", spy.listDocumentsCalls.Load())
	}
}

func TestConfluenceService_LinkedScopeDegradesWithoutJiraIntegration(t *testing.T) {
	spy := &spyRepository{inner: confluenceInfra.NewConfluenceClient("https://confluence.example.com")}
	service := setupLinkedService(t, "", nil, spy)

	response, err := service.GetDocuments(context.Background(), configuredUserID, CONFLUENCE_SCOPE_LINKED)
	if err != nil {
		t.Fatalf("expected degraded response instead of error, got %v", err)
	}
	if response.Source != domain.SourceUnavailable || !response.Degraded {
		t.Errorf("expected unavailable degraded response, got source=%q degraded=%v", response.Source, response.Degraded)
	}
	if spy.listDocumentsCalls.Load() != 0 {
		t.Errorf("expected ListDocuments to never be called, got %d calls", spy.listDocumentsCalls.Load())
	}
}

func TestConfluenceService_AllScopeUsesCqlListDocuments(t *testing.T) {
	var upstreamHits atomic.Int64
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upstreamHits.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"results":[{"id":"101","title":"UT Coverage Report","space":{"key":"DEV","name":"DEV Team"},"version":{"when":"2026-01-02T03:04:05.000+07:00","by":{"displayName":"Jane Doe"}},"_links":{"base":"https://confluence.example.com","webui":"/display/DEV/UT-Coverage-Report"}}]}`))
	}))
	defer upstream.Close()

	service := setupLinkedService(t, upstream.URL, nil, nil)

	response, err := service.GetDocuments(context.Background(), configuredUserID, CONFLUENCE_SCOPE_ALL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if response.Source != domain.SourceCqlAll {
		t.Errorf("expected source cql-all, got %q", response.Source)
	}
	if response.Total != 1 {
		t.Errorf("expected 1 document from CQL list, got %d", response.Total)
	}
	if upstreamHits.Load() != 1 {
		t.Errorf("expected 1 upstream hit, got %d", upstreamHits.Load())
	}
}

func TestConfluenceService_SuccessfulLinkedResponseIsCached(t *testing.T) {
	provider := &stubLinkedIssueProvider{
		issues: []jiraDomain.JiraIssue{{ID: "1", Key: "CRMMS-1"}},
		linksByIssue: map[string][]jiraDomain.JiraRemoteLink{
			"CRMMS-1": {{ID: 1, Object: jiraDomain.JiraRemoteLinkObject{URL: "https://confluence.bri.co.id/pages/110001"}}},
		},
	}
	spy := &spyRepository{
		inner:             confluenceInfra.NewConfluenceClient("https://confluence.example.com"),
		getByIDsDocuments: []domain.ConfluenceDocument{{ID: "110001", Title: "Wiki Page"}},
	}
	service := setupLinkedService(t, "", provider, spy)

	first, err := service.GetDocuments(context.Background(), configuredUserID, CONFLUENCE_SCOPE_LINKED)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	second, err := service.GetDocuments(context.Background(), configuredUserID, CONFLUENCE_SCOPE_LINKED)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if first.Source != domain.SourceJiraLinks || first.Total != 1 {
		t.Errorf("expected jira-links response with 1 document, got source=%q total=%d", first.Source, first.Total)
	}
	if spy.getByIDsCalls.Load() != 1 {
		t.Errorf("expected second call served from cache with 1 getByIDs call, got %d", spy.getByIDsCalls.Load())
	}
	if second.Total != first.Total {
		t.Errorf("expected cached response to match, got %d vs %d", second.Total, first.Total)
	}
}

func TestConfluenceService_CollectPageIDsCapsAtMaxLinkedDocs(t *testing.T) {
	issues := make([]jiraDomain.JiraIssue, 0, CONFLUENCE_MAX_LINKED_ISSUES)
	linksByIssue := make(map[string][]jiraDomain.JiraRemoteLink, CONFLUENCE_MAX_LINKED_ISSUES)
	for i := range CONFLUENCE_MAX_LINKED_ISSUES {
		issueKey := "CRMMS-" + strconv.Itoa(i)
		issues = append(issues, jiraDomain.JiraIssue{ID: issueKey, Key: issueKey})
		linksByIssue[issueKey] = []jiraDomain.JiraRemoteLink{
			{ID: i, Object: jiraDomain.JiraRemoteLinkObject{URL: "https://confluence.bri.co.id/pages/" + strconv.Itoa(1000+i)}},
			{ID: i + 1000, Object: jiraDomain.JiraRemoteLinkObject{URL: "https://confluence.bri.co.id/x?pageId=" + strconv.Itoa(5000+i)}},
		}
	}

	provider := &stubLinkedIssueProvider{issues: issues, linksByIssue: linksByIssue}
	spy := &spyRepository{
		inner:             confluenceInfra.NewConfluenceClient("https://confluence.example.com"),
		getByIDsDocuments: []domain.ConfluenceDocument{},
	}
	service := setupLinkedService(t, "", provider, spy)

	if _, err := service.GetDocuments(context.Background(), configuredUserID, CONFLUENCE_SCOPE_LINKED); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if spy.getByIDsCalls.Load() != 1 {
		t.Fatalf("expected 1 batch fetch, got %d", spy.getByIDsCalls.Load())
	}
	if len(spy.lastRequestedIDs) != CONFLUENCE_MAX_LINKED_DOCS {
		t.Errorf("expected exactly %d page IDs after cap, got %d", CONFLUENCE_MAX_LINKED_DOCS, len(spy.lastRequestedIDs))
	}
}

func TestConfluenceService_DegradedResponseIsNotCached(t *testing.T) {
	provider := &stubLinkedIssueProvider{issuesErr: errors.New("jira unreachable")}
	service := setupLinkedService(t, "", provider, nil)

	if _, err := service.GetDocuments(context.Background(), configuredUserID, CONFLUENCE_SCOPE_LINKED); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := service.GetDocuments(context.Background(), configuredUserID, CONFLUENCE_SCOPE_LINKED); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if calls := provider.issuesCalls.Load(); calls != 2 {
		t.Errorf("expected degraded response to be recomputed on every call, got %d upstream calls", calls)
	}
}

func TestConfluenceService_LinkedScopeDeduplicatesPageIDs(t *testing.T) {
	provider := &stubLinkedIssueProvider{
		issues: []jiraDomain.JiraIssue{
			{ID: "1", Key: "CRMMS-1"},
			{ID: "2", Key: "CRMMS-2"},
		},
		linksByIssue: map[string][]jiraDomain.JiraRemoteLink{
			"CRMMS-1": {{ID: 1, Object: jiraDomain.JiraRemoteLinkObject{URL: "https://confluence.bri.co.id/pages/110001"}}},
			"CRMMS-2": {{ID: 2, Object: jiraDomain.JiraRemoteLinkObject{URL: "https://confluence.bri.co.id/x?pageId=110001"}}},
		},
	}
	spy := &spyRepository{
		inner:             confluenceInfra.NewConfluenceClient("https://confluence.example.com"),
		getByIDsDocuments: []domain.ConfluenceDocument{},
	}
	service := setupLinkedService(t, "", provider, spy)

	if _, err := service.GetDocuments(context.Background(), configuredUserID, CONFLUENCE_SCOPE_LINKED); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(spy.lastRequestedIDs) != 1 {
		t.Errorf("expected duplicate page IDs to be deduplicated into 1, got %d", len(spy.lastRequestedIDs))
	}
}

func TestConfluenceService_PartialFetchFailureReturnsDegraded(t *testing.T) {
	provider := &stubLinkedIssueProvider{
		issues: []jiraDomain.JiraIssue{{ID: "1", Key: "CRMMS-1"}},
		linksByIssue: map[string][]jiraDomain.JiraRemoteLink{
			"CRMMS-1": {{ID: 1, Object: jiraDomain.JiraRemoteLinkObject{URL: "https://confluence.bri.co.id/pages/110001"}}},
		},
	}
	spy := &spyRepository{
		inner:             confluenceInfra.NewConfluenceClient("https://confluence.example.com"),
		getByIDsDocuments: []domain.ConfluenceDocument{{ID: "110001", Title: "Wiki Page"}},
		getByIDsErr:       errors.New("confluence returned status 500"),
	}
	service := setupLinkedService(t, "", provider, spy)

	response, err := service.GetDocuments(context.Background(), configuredUserID, CONFLUENCE_SCOPE_LINKED)
	if err != nil {
		t.Fatalf("expected partial success instead of hard error, got %v", err)
	}
	if response.Total != 1 {
		t.Errorf("expected partial documents to be returned, got %d", response.Total)
	}
	if !response.Degraded {
		t.Error("expected degraded flag for partial failure")
	}
	if !strings.Contains(response.DegradedReason, "500") {
		t.Errorf("expected failure reason preserved, got %q", response.DegradedReason)
	}
}

func TestConfluenceService_TotalFetchFailurePropagatesError(t *testing.T) {
	provider := &stubLinkedIssueProvider{
		issues: []jiraDomain.JiraIssue{{ID: "1", Key: "CRMMS-1"}},
		linksByIssue: map[string][]jiraDomain.JiraRemoteLink{
			"CRMMS-1": {{ID: 1, Object: jiraDomain.JiraRemoteLinkObject{URL: "https://confluence.bri.co.id/pages/110001"}}},
		},
	}
	spy := &spyRepository{
		inner:       confluenceInfra.NewConfluenceClient("https://confluence.example.com"),
		getByIDsErr: errors.New("unable to reach Confluence BRI API"),
	}
	service := setupLinkedService(t, "", provider, spy)

	_, err := service.GetDocuments(context.Background(), configuredUserID, CONFLUENCE_SCOPE_LINKED)
	if err == nil {
		t.Fatal("expected error when zero documents could be fetched")
	}
	if !strings.Contains(err.Error(), "unable to reach Confluence") {
		t.Errorf("expected upstream error propagated, got %v", err)
	}
}
