package transport

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	authDomain "lunar/backend/internal/auth/domain"
	confluenceDomain "lunar/backend/internal/confluence/domain"
	"lunar/backend/internal/queryreview/application"
	"lunar/backend/internal/queryreview/domain"
	sharedAuth "lunar/backend/internal/shared/auth"
	sharedErrors "lunar/backend/internal/shared/errors"
)

type handlerDocumentProvider struct {
	document      *confluenceDomain.ConfluenceDocument
	err           error
	ancestors     []confluenceDomain.ConfluencePageRef
	ancestorsErr  error
	children      []confluenceDomain.ConfluencePageRef
	childrenErr   error
	storage       string
	storageErr    error
	updateVersion int
	updateURL     string
	updateErr     error
	updatedValue  string
}

func (s *handlerDocumentProvider) GetDocumentDetail(ctx context.Context, userID string, documentID string) (*confluenceDomain.ConfluenceDocument, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.document, nil
}

func (s *handlerDocumentProvider) GetPageAncestors(ctx context.Context, userID string, pageID string) ([]confluenceDomain.ConfluencePageRef, error) {
	if s.ancestorsErr != nil {
		return nil, s.ancestorsErr
	}
	return s.ancestors, nil
}

func (s *handlerDocumentProvider) GetChildPages(ctx context.Context, userID string, pageID string) ([]confluenceDomain.ConfluencePageRef, error) {
	if s.childrenErr != nil {
		return nil, s.childrenErr
	}
	return s.children, nil
}

func (s *handlerDocumentProvider) GetPageStorage(ctx context.Context, userID string, pageID string) (string, error) {
	if s.storageErr != nil {
		return "", s.storageErr
	}
	return s.storage, nil
}

func (s *handlerDocumentProvider) UpdatePageStorage(ctx context.Context, userID string, pageID string, storage string) (int, string, error) {
	s.updatedValue = storage
	if s.updateErr != nil {
		return 0, "", s.updateErr
	}
	return s.updateVersion, s.updateURL, nil
}

type handlerPATAccess struct {
	secrets *authDomain.DecryptedSecrets
	err     error
}

func (s *handlerPATAccess) GetDecryptedSecrets(ctx context.Context, userID string) (*authDomain.DecryptedSecrets, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.secrets, nil
}

type handlerDiffSource struct {
	diff        string
	err         error
	compareHits int
}

func (s *handlerDiffSource) GetCompareDiff(ctx context.Context, pat string, projectKey string, repoSlug string, sourceBranch string, targetBranch string) (string, error) {
	s.compareHits++
	if s.err != nil {
		return "", s.err
	}
	return s.diff, nil
}

func (s *handlerDiffSource) GetPullRequestBranches(ctx context.Context, pat string, projectKey string, repoSlug string, pullRequestID string) (domain.PullRequestBranches, error) {
	return domain.PullRequestBranches{}, nil
}

type handlerExtractor struct {
	analysis domain.DiffAnalysis
}

func (s *handlerExtractor) Extract(diffText string) domain.DiffAnalysis {
	return s.analysis
}

type handlerReviewer struct {
	extraction domain.QueryExtraction
	err        error
}

func (s *handlerReviewer) ReviewQueries(ctx context.Context, userID string, modelName string, files []domain.PromptFile) (domain.QueryExtraction, error) {
	if s.err != nil {
		return domain.QueryExtraction{}, s.err
	}
	return s.extraction, nil
}

type handlerFixture struct {
	handler    *QueryReviewHandler
	documents  *handlerDocumentProvider
	pat        *handlerPATAccess
	diffSource *handlerDiffSource
	extractor  *handlerExtractor
	reviewer   *handlerReviewer
}

const handlerMigStorage = `<table><tbody>
<tr><th>Repository</th><th>Branch</th><th>Compare</th></tr>
<tr><td>micro-payment</td><td>feat/x</td><td>https://bitbucket.example.com/projects/NME/repos/micro-payment/compare/diff?sourceBranch=feat%2Fx&amp;targetBranch=development</td></tr>
</tbody></table>`

func newHandlerFixture() *handlerFixture {
	documents := &handlerDocumentProvider{
		document:  &confluenceDomain.ConfluenceDocument{ID: "123", Title: "CRMMS-1 Query Review"},
		ancestors: []confluenceDomain.ConfluencePageRef{{ID: "100", Title: "Parent"}},
		children:  []confluenceDomain.ConfluencePageRef{{ID: "200", Title: "CRMMS-1-MIG"}},
		storage:   handlerMigStorage,
	}
	pat := &handlerPATAccess{secrets: &authDomain.DecryptedSecrets{BitbucketPAT: "pat-1"}}
	diffSource := &handlerDiffSource{diff: "raw diff"}
	extractor := &handlerExtractor{analysis: domain.DiffAnalysis{
		Files: []domain.PromptFile{
			{Path: "internal/repo/dao.go", Language: "go", AddedLines: []domain.PromptLine{{Number: 11, Content: `db.Raw("select id from user_data")`}}},
		},
		Analyzed: []domain.AnalyzedFile{{Path: "internal/repo/dao.go", AddedLines: 1, Language: "go"}},
		Skipped:  []domain.SkippedFile{},
	}}
	reviewer := &handlerReviewer{extraction: domain.QueryExtraction{
		Queries: []domain.QueryItem{
			{No: 1, Feature: "Get user by PN", Table: "user_data", ActionType: "SELECT", QueryStatement: "select id from user_data where pn = ?", Index: "pn", File: "internal/repo/dao.go", Line: 11},
		},
		Summary:       "One query was introduced.",
		PromptVersion: "queryreview-v1",
	}}

	service := application.NewQueryReviewService(documents, pat, diffSource, extractor, reviewer)
	return &handlerFixture{
		handler:    NewQueryReviewHandler(service),
		documents:  documents,
		pat:        pat,
		diffSource: diffSource,
		extractor:  extractor,
		reviewer:   reviewer,
	}
}

func performQueryReview(t *testing.T, handler *QueryReviewHandler, userID string, body string) *httptest.ResponseRecorder {
	t.Helper()

	var reader io.Reader
	if body == "" {
		reader = strings.NewReader("")
	} else {
		reader = strings.NewReader(body)
	}

	request := httptest.NewRequest(http.MethodPost, "/api/confluence/documents/123/query-review", reader)
	request.SetPathValue("id", "123")
	if userID != "" {
		ctx := sharedAuth.ContextWithUser(request.Context(), &sharedAuth.UserContext{UserID: userID, Email: "developer@example.com"})
		request = request.WithContext(ctx)
	}

	recorder := httptest.NewRecorder()
	handler.Generate(recorder, request)
	return recorder
}

const explicitSourceBody = `{"model":"model-a","source":{"compare_url":"https://bitbucket.example.com/projects/NME/repos/micro-payment/compare/diff?sourceBranch=feat%2Fx&targetBranch=development","project_key":"NME","repo_slug":"micro-payment","source_branch":"feat/x","target_branch":"development"}}`

type handlerErrorResponse struct {
	Error string `json:"error"`
	Code  string `json:"code"`
}

func decodeHandlerError(t *testing.T, recorder *httptest.ResponseRecorder) handlerErrorResponse {
	t.Helper()

	var payload handlerErrorResponse
	if err := json.NewDecoder(recorder.Body).Decode(&payload); err != nil {
		t.Fatalf("cannot decode the error body: %v", err)
	}
	return payload
}

func TestQueryReviewHandlerRejectsUnauthenticatedRequests(t *testing.T) {
	fixture := newHandlerFixture()

	recorder := performQueryReview(t, fixture.handler, "", explicitSourceBody)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", recorder.Code)
	}
	if fixture.diffSource.compareHits != 0 {
		t.Fatalf("expected no downstream call, got %d", fixture.diffSource.compareHits)
	}
}

func TestQueryReviewHandlerRejectsAnInvalidBody(t *testing.T) {
	fixture := newHandlerFixture()

	recorder := performQueryReview(t, fixture.handler, "usr-1", `{"model":"model-a","unexpected":true}`)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", recorder.Code)
	}
}

func TestQueryReviewHandlerReturns200WithTheContractShape(t *testing.T) {
	fixture := newHandlerFixture()

	recorder := performQueryReview(t, fixture.handler, "usr-1", explicitSourceBody)
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", recorder.Code, recorder.Body.String())
	}

	var payload map[string]json.RawMessage
	if err := json.NewDecoder(recorder.Body).Decode(&payload); err != nil {
		t.Fatalf("cannot decode the response: %v", err)
	}

	requiredKeys := []string{
		"document_id", "model", "prompt_version", "mig_page_id", "mig_page_title", "skipped_repositories", "repositories",
	}
	if len(payload) != len(requiredKeys) {
		t.Fatalf("expected exactly %d keys, got %v", len(requiredKeys), payload)
	}
	for _, key := range requiredKeys {
		if _, exists := payload[key]; !exists {
			t.Errorf("expected the %q key in the response", key)
		}
	}

	var promptVersion string
	if err := json.Unmarshal(payload["prompt_version"], &promptVersion); err != nil {
		t.Fatalf("cannot decode the prompt version: %v", err)
	}
	if promptVersion != "queryreview-v1" {
		t.Fatalf("expected queryreview-v1, got %q", promptVersion)
	}

	var repositories []map[string]json.RawMessage
	if err := json.Unmarshal(payload["repositories"], &repositories); err != nil {
		t.Fatalf("cannot decode the repositories: %v", err)
	}
	if len(repositories) != 1 {
		t.Fatalf("expected 1 repository, got %d", len(repositories))
	}

	var queries []map[string]json.RawMessage
	if err := json.Unmarshal(repositories[0]["queries"], &queries); err != nil {
		t.Fatalf("cannot decode the queries: %v", err)
	}
	if len(queries) != 1 {
		t.Fatalf("expected 1 query, got %d", len(queries))
	}
	if _, exists := queries[0]["query_statement"]; !exists {
		t.Fatal("expected the query statement in the query entry")
	}
	if _, exists := queries[0]["repository"]; !exists {
		t.Fatal("expected the repository in the query entry")
	}
}

func TestQueryReviewHandlerReturns200ForTheMigPath(t *testing.T) {
	fixture := newHandlerFixture()

	recorder := performQueryReview(t, fixture.handler, "usr-1", `{}`)
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", recorder.Code, recorder.Body.String())
	}

	var payload struct {
		MigPageID    string `json:"mig_page_id"`
		MigPageTitle string `json:"mig_page_title"`
		Repositories []struct {
			Repository string `json:"repository"`
		} `json:"repositories"`
	}
	if err := json.NewDecoder(recorder.Body).Decode(&payload); err != nil {
		t.Fatalf("cannot decode the response: %v", err)
	}
	if payload.MigPageID != "200" || payload.MigPageTitle != "CRMMS-1-MIG" {
		t.Fatalf("expected the MIG page identity, got %q/%q", payload.MigPageID, payload.MigPageTitle)
	}
	if len(payload.Repositories) != 1 || payload.Repositories[0].Repository != "micro-payment" {
		t.Fatalf("expected the MIG repository, got %+v", payload.Repositories)
	}
}

func TestQueryReviewHandlerReturns422WhenThereIsNoMigPage(t *testing.T) {
	fixture := newHandlerFixture()
	fixture.documents.children = []confluenceDomain.ConfluencePageRef{{ID: "200", Title: "Design"}}

	recorder := performQueryReview(t, fixture.handler, "usr-1", `{}`)

	if recorder.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d: %s", recorder.Code, recorder.Body.String())
	}
	payload := decodeHandlerError(t, recorder)
	if payload.Code != domain.ErrorCodeNoCompareLink {
		t.Fatalf("expected the %s code, got %q", domain.ErrorCodeNoCompareLink, payload.Code)
	}
}

func TestQueryReviewHandlerReturns404WhenTheDocumentIsMissing(t *testing.T) {
	fixture := newHandlerFixture()
	fixture.documents.err = sharedErrors.ErrNotFound

	recorder := performQueryReview(t, fixture.handler, "usr-1", `{}`)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", recorder.Code, recorder.Body.String())
	}
}

func TestQueryReviewHandlerReturns401WhenTheBitbucketPATIsMissing(t *testing.T) {
	fixture := newHandlerFixture()
	fixture.pat.secrets = &authDomain.DecryptedSecrets{}

	recorder := performQueryReview(t, fixture.handler, "usr-1", explicitSourceBody)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", recorder.Code, recorder.Body.String())
	}
	if fixture.diffSource.compareHits != 0 {
		t.Fatalf("expected no compare fetch, got %d", fixture.diffSource.compareHits)
	}
}

func TestQueryReviewHandlerReturns400WhenTheSourceIsIncomplete(t *testing.T) {
	fixture := newHandlerFixture()

	recorder := performQueryReview(t, fixture.handler, "usr-1", `{"source":{"project_key":"NME","repo_slug":"micro-payment","target_branch":"development"}}`)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", recorder.Code, recorder.Body.String())
	}
}

func TestQueryReviewHandlerReturns502WhenThePatResolutionFails(t *testing.T) {
	fixture := newHandlerFixture()
	fixture.pat.err = errors.New("the vault is unreachable")

	recorder := performQueryReview(t, fixture.handler, "usr-1", explicitSourceBody)

	if recorder.Code != http.StatusBadGateway {
		t.Fatalf("expected 502, got %d: %s", recorder.Code, recorder.Body.String())
	}
}

func TestQueryReviewHandlerKeepsARepositoryFailureInThe200Body(t *testing.T) {
	fixture := newHandlerFixture()
	fixture.diffSource.err = errors.New("bitbucket is unreachable")

	recorder := performQueryReview(t, fixture.handler, "usr-1", explicitSourceBody)
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", recorder.Code, recorder.Body.String())
	}

	var payload struct {
		Repositories []struct {
			Status string `json:"status"`
			Error  string `json:"error"`
		} `json:"repositories"`
	}
	if err := json.NewDecoder(recorder.Body).Decode(&payload); err != nil {
		t.Fatalf("cannot decode the response: %v", err)
	}
	if len(payload.Repositories) != 1 || payload.Repositories[0].Status != domain.RepositoryStatusError {
		t.Fatalf("expected a repository error status, got %+v", payload.Repositories)
	}
	if !strings.Contains(payload.Repositories[0].Error, "bitbucket is unreachable") {
		t.Fatalf("expected the failure message, got %q", payload.Repositories[0].Error)
	}
}
