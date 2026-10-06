package transport

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	jiraDomain "lunar/backend/internal/jira/domain"
	sharedAuth "lunar/backend/internal/shared/auth"
	sharedErrors "lunar/backend/internal/shared/errors"
	workspaceApp "lunar/backend/internal/workspace/application"
	"lunar/backend/internal/workspace/domain"
)

type stubRouteRoots struct {
	rootPath string
	err      error
}

func (s stubRouteRoots) WorkspaceRootFor(_ context.Context, _ string) (string, error) {
	if s.err != nil {
		return "", s.err
	}

	return s.rootPath, nil
}

type stubRouteLocal struct {
	repositories []domain.RepositorySummary
}

func (s stubRouteLocal) ListRepositories(_ context.Context, _ string) ([]domain.RepositorySummary, error) {
	return s.repositories, nil
}

type stubRouteIssues struct {
	issue   *jiraDomain.JiraIssue
	backlog *jiraDomain.JiraBacklogResponse
	err     error
}

func (s stubRouteIssues) GetIssueDetail(_ context.Context, _ string, _ string) (*jiraDomain.JiraIssue, error) {
	if s.err != nil {
		return nil, s.err
	}

	return s.issue, nil
}

func (s stubRouteIssues) GetBacklog(_ context.Context, _ string, _ string) (*jiraDomain.JiraBacklogResponse, error) {
	if s.err != nil {
		return nil, s.err
	}

	return s.backlog, nil
}

type stubRouteBranches struct {
	matches map[string][]workspaceApp.BranchMatch
}

func (s stubRouteBranches) SprintBranchesAtRoot(_ context.Context, _ string, issueKeys []string) (*workspaceApp.SprintBranches, error) {
	matches := make(map[string][]workspaceApp.BranchMatch, len(issueKeys))
	for _, key := range issueKeys {
		matches[key] = s.matches[key]
		if matches[key] == nil {
			matches[key] = []workspaceApp.BranchMatch{}
		}
	}

	return &workspaceApp.SprintBranches{Matches: matches}, nil
}

func authenticatedRoute(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := &sharedAuth.UserContext{UserID: "user-1"}
		next.ServeHTTP(w, r.WithContext(sharedAuth.ContextWithUser(r.Context(), user)))
	})
}

func anonymousRoute(next http.Handler) http.Handler {
	return next
}

func routeFixture(t *testing.T, roots stubRouteRoots, auth func(http.Handler) http.Handler) (*http.ServeMux, stubRouteIssues) {
	t.Helper()

	issues := stubRouteIssues{
		issue: &jiraDomain.JiraIssue{
			Key: "CRMMS-77911",
			Fields: jiraDomain.JiraIssueFields{
				Summary: "Enhancement Manage User",
				Parent:  &jiraDomain.JiraParent{Key: "CRMMS-100"},
			},
		},
		backlog: &jiraDomain.JiraBacklogResponse{Sprints: []jiraDomain.JiraSprint{{Name: "Sprint 21", Issues: []jiraDomain.JiraIssue{
			{Key: "CRMMS-77912", Fields: jiraDomain.JiraIssueFields{Parent: &jiraDomain.JiraParent{Key: "CRMMS-100"}}},
		}}}},
	}
	branches := stubRouteBranches{matches: map[string][]workspaceApp.BranchMatch{
		"CRMMS-77912": {{Repository: "mms-api", Branch: "feature/CRMMS-77912-first"}},
	}}
	local := stubRouteLocal{repositories: []domain.RepositorySummary{{Name: "mms-api", IsGit: true}}}

	mux := http.NewServeMux()
	RegisterRepositoryRoutes(mux, auth, roots, local, issues, branches, nil)
	return mux, issues
}

func TestRepositorySuggestionsRouteReturnsTheRankedCandidates(t *testing.T) {
	mux, _ := routeFixture(t, stubRouteRoots{rootPath: "/workspace"}, authenticatedRoute)

	response := httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/jira/issues/CRMMS-77911/repository-suggestions", nil))

	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.Code, response.Body.String())
	}
	body := response.Body.String()
	for _, expected := range []string{"\"confidence\":\"high\"", "\"repo\":\"mms-api\"", "\"signal\":\"epic_siblings\"", "\"local_repository_count\":1"} {
		if !strings.Contains(body, expected) {
			t.Fatalf("expected %s in %s", expected, body)
		}
	}
}

func TestRepositorySuggestionsRouteRefusesAnInvalidKey(t *testing.T) {
	mux, _ := routeFixture(t, stubRouteRoots{rootPath: "/workspace"}, authenticatedRoute)

	response := httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/jira/issues/not-a-key/repository-suggestions", nil))

	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", response.Code, response.Body.String())
	}
}

func TestRepositorySuggestionsRouteRequiresASession(t *testing.T) {
	mux, _ := routeFixture(t, stubRouteRoots{rootPath: "/workspace"}, anonymousRoute)

	response := httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/jira/issues/CRMMS-77911/repository-suggestions", nil))

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", response.Code)
	}
}

func TestRepositorySuggestionsRouteReportsAnUnusableWorkspace(t *testing.T) {
	mux, _ := routeFixture(t, stubRouteRoots{err: workspaceApp.ErrWorkspaceNotUsable}, authenticatedRoute)

	response := httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/jira/issues/CRMMS-77911/repository-suggestions", nil))

	if response.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d: %s", response.Code, response.Body.String())
	}
}

func TestRepositorySuggestionsRouteReportsJiraAsUnreachable(t *testing.T) {
	mux, issues := routeFixture(t, stubRouteRoots{rootPath: "/workspace"}, authenticatedRoute)
	issues.err = sharedErrors.ErrBadGateway
	branches := stubRouteBranches{}
	local := stubRouteLocal{repositories: []domain.RepositorySummary{{Name: "mms-api"}}}
	mux = http.NewServeMux()
	RegisterRepositoryRoutes(mux, authenticatedRoute, stubRouteRoots{rootPath: "/workspace"}, local, issues, branches, nil)

	response := httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/jira/issues/CRMMS-77911/repository-suggestions", nil))

	if response.Code != http.StatusBadGateway {
		t.Fatalf("expected 502, got %d: %s", response.Code, response.Body.String())
	}
}

func TestRepositoryCatalogueRouteListsAndFilters(t *testing.T) {
	mux, _ := routeFixture(t, stubRouteRoots{rootPath: "/workspace"}, authenticatedRoute)

	response := httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/jira/repositories?search=mms", nil))

	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.Code, response.Body.String())
	}
	body := response.Body.String()
	if !strings.Contains(body, "\"repo\":\"mms-api\"") || !strings.Contains(body, "\"local_repository_count\":1") {
		t.Fatalf("unexpected catalogue body: %s", body)
	}

	empty := httptest.NewRecorder()
	mux.ServeHTTP(empty, httptest.NewRequest(http.MethodGet, "/api/jira/repositories?search=missing", nil))
	if empty.Code != http.StatusOK || !strings.Contains(empty.Body.String(), "\"repositories\":[]") {
		t.Fatalf("an unmatched search must return an empty list, got %d: %s", empty.Code, empty.Body.String())
	}
	if !strings.Contains(empty.Body.String(), "\"local_repository_count\":1") {
		t.Fatalf("the counts must survive filtering: %s", empty.Body.String())
	}
}

func TestRepositoryFeedbackRouteRecordsAnAddition(t *testing.T) {
	mux, _ := routeFixture(t, stubRouteRoots{rootPath: "/workspace"}, authenticatedRoute)

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/jira/issues/CRMMS-77911/repository-feedback", strings.NewReader(`{"repo":"mms-api","action":"added"}`))
	mux.ServeHTTP(response, request)

	if response.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d: %s", response.Code, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), "\"recorded\":true") {
		t.Fatalf("unexpected body: %s", response.Body.String())
	}
}

func TestRepositoryFeedbackRouteRefusesAnUnknownAction(t *testing.T) {
	mux, _ := routeFixture(t, stubRouteRoots{rootPath: "/workspace"}, authenticatedRoute)

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/jira/issues/CRMMS-77911/repository-feedback", strings.NewReader(`{"repo":"mms-api","action":"guessed"}`))
	mux.ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", response.Code, response.Body.String())
	}
}

func TestRepositoryFeedbackRouteRefusesARepositoryOutsideTheCatalogue(t *testing.T) {
	mux, _ := routeFixture(t, stubRouteRoots{rootPath: "/workspace"}, authenticatedRoute)

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/jira/issues/CRMMS-77911/repository-feedback", strings.NewReader(`{"repo":"unknown-repo","action":"added"}`))
	mux.ServeHTTP(response, request)

	if response.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", response.Code, response.Body.String())
	}
}

func TestRepositoryFeedbackRouteRefusesAMalformedBody(t *testing.T) {
	mux, _ := routeFixture(t, stubRouteRoots{rootPath: "/workspace"}, authenticatedRoute)

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/jira/issues/CRMMS-77911/repository-feedback", strings.NewReader(`{"repository":"mms-api"}`))
	mux.ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", response.Code, response.Body.String())
	}
}
