package transport

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	sharedAuth "lunar/backend/internal/shared/auth"
	"lunar/backend/internal/workspace/application"
	workspaceDomain "lunar/backend/internal/workspace/domain"
)

type readinessConfigSource struct {
	config *application.WorkspaceConfig
	err    error
}

var _ WorkspaceConfigSource = (*readinessConfigSource)(nil)

func (s *readinessConfigSource) GetWorkspace(ctx context.Context, userID string) (*application.WorkspaceConfig, error) {
	if s.err != nil {
		return nil, s.err
	}

	return s.config, nil
}

type readinessBranchScanner struct {
	branchesByRepository map[string][]workspaceDomain.Branch
	err                  error
}

var _ application.RepositoryBranchScanner = (*readinessBranchScanner)(nil)

func (s *readinessBranchScanner) ScanRepositoryBranches(ctx context.Context, rootPath string) (map[string][]workspaceDomain.Branch, error) {
	if s.err != nil {
		return nil, s.err
	}

	return s.branchesByRepository, nil
}

type readinessCatalogueSource struct {
	catalogue *application.RepositoryCatalogue
	err       error
}

var _ ReadinessCatalogueSource = (*readinessCatalogueSource)(nil)

func (s *readinessCatalogueSource) Catalogue(ctx context.Context, userID string) (*application.RepositoryCatalogue, error) {
	if s.err != nil {
		return nil, s.err
	}

	return s.catalogue, nil
}

type readinessErrorBody struct {
	Error string `json:"error"`
}

func buildReadinessHandler(t *testing.T, config *application.WorkspaceConfig, scanner *readinessBranchScanner, catalogue ReadinessCatalogueSource) http.Handler {
	t.Helper()

	service := application.NewWorkspaceReadinessService(scanner, nil)
	var configSource WorkspaceConfigSource
	if config != nil {
		configSource = &readinessConfigSource{config: config}
	}
	handler := NewWorkspaceReadinessHandler(service, configSource, catalogue)

	return http.HandlerFunc(handler.GetReadiness)
}

func serveReadinessRequest(handler http.Handler, target string, userID string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(http.MethodGet, target, nil)
	if userID != "" {
		request = request.WithContext(sharedAuth.ContextWithUser(request.Context(), &sharedAuth.UserContext{UserID: userID}))
	}

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	return recorder
}

func decodeReadiness(t *testing.T, recorder *httptest.ResponseRecorder) application.WorkspaceReadiness {
	t.Helper()

	var readiness application.WorkspaceReadiness
	if err := json.Unmarshal(recorder.Body.Bytes(), &readiness); err != nil {
		t.Fatalf("cannot decode the readiness payload %q: %v", recorder.Body.String(), err)
	}

	return readiness
}

func decodeReadinessError(t *testing.T, recorder *httptest.ResponseRecorder) string {
	t.Helper()

	var body readinessErrorBody
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("cannot decode the error body %q: %v", recorder.Body.String(), err)
	}

	return body.Error
}

func resolvedRoot(t *testing.T, rootPath string) string {
	t.Helper()

	resolved, err := workspaceDomain.ResolveWorkspaceRoot(rootPath, nil)
	if err != nil {
		t.Fatalf("cannot resolve the test root: %v", err)
	}

	return resolved
}

func TestGetReadinessReportsAReadyWorkspace(t *testing.T) {
	root := t.TempDir()
	scanner := &readinessBranchScanner{branchesByRepository: map[string][]workspaceDomain.Branch{
		"mms-api": {{Name: "feature/CRMMS-77911-enhancement"}},
	}}
	catalogue := &readinessCatalogueSource{catalogue: &application.RepositoryCatalogue{
		Repositories:          []application.CataloguedRepository{{Repo: "mms-api", IsCloned: true}},
		LocalRepositoryCount:  1,
		RemoteRepositoryCount: 1,
	}}
	handler := buildReadinessHandler(t, &application.WorkspaceConfig{
		Source:   workspaceDomain.SourceServer,
		RootPath: root,
	}, scanner, catalogue)

	recorder := serveReadinessRequest(handler, "/api/workspace/readiness?project=CRMMS", "usr-1")
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d with body %s", recorder.Code, recorder.Body.String())
	}

	readiness := decodeReadiness(t, recorder)
	if readiness.State != application.ReadinessReady {
		t.Fatalf("expected ready, got %q", readiness.State)
	}
	if readiness.RootPath != resolvedRoot(t, root) {
		t.Fatalf("expected the resolved root %q, got %q", resolvedRoot(t, root), readiness.RootPath)
	}
	if readiness.LocalRepositoryCount != 1 || readiness.RemoteRepositoryCount != 1 {
		t.Fatalf("unexpected repository counts %+v", readiness)
	}
	if readiness.MissingRepositories == nil || len(readiness.MissingRepositories) != 0 {
		t.Fatalf("expected an empty missing repository list, got %+v", readiness.MissingRepositories)
	}
}

func TestGetReadinessReportsNotConfiguredWithoutAWorkspace(t *testing.T) {
	handler := buildReadinessHandler(t, &application.WorkspaceConfig{Source: workspaceDomain.SourceServer}, &readinessBranchScanner{}, nil)

	recorder := serveReadinessRequest(handler, "/api/workspace/readiness", "usr-1")
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d with body %s", recorder.Code, recorder.Body.String())
	}

	readiness := decodeReadiness(t, recorder)
	if readiness.State != application.ReadinessNotConfigured {
		t.Fatalf("expected not-configured, got %q", readiness.State)
	}
	if readiness.Message != "No workspace folder is configured. Choose the folder that holds your service repositories." {
		t.Fatalf("unexpected message %q", readiness.Message)
	}
}

func TestGetReadinessReportsAWorkspaceReadByTheHelper(t *testing.T) {
	handler := buildReadinessHandler(t, &application.WorkspaceConfig{
		Source:   workspaceDomain.SourceHelper,
		RootPath: "/Users/erendt/BRI",
	}, &readinessBranchScanner{}, nil)

	recorder := serveReadinessRequest(handler, "/api/workspace/readiness", "usr-1")
	if recorder.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d with body %s", recorder.Code, recorder.Body.String())
	}
	if message := decodeReadinessError(t, recorder); message != "the workspace is read by the local helper" {
		t.Fatalf("unexpected error message %q", message)
	}
}

func TestGetReadinessRequiresAnAuthenticatedUser(t *testing.T) {
	handler := buildReadinessHandler(t, &application.WorkspaceConfig{Source: workspaceDomain.SourceServer}, &readinessBranchScanner{}, nil)

	recorder := serveReadinessRequest(handler, "/api/workspace/readiness", "")
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d with body %s", recorder.Code, recorder.Body.String())
	}
}

func TestGetReadinessRejectsAProjectThatIsNotAProjectKey(t *testing.T) {
	handler := buildReadinessHandler(t, &application.WorkspaceConfig{Source: workspaceDomain.SourceServer}, &readinessBranchScanner{}, nil)

	recorder := serveReadinessRequest(handler, "/api/workspace/readiness?project=12-34", "usr-1")
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d with body %s", recorder.Code, recorder.Body.String())
	}
	if message := decodeReadinessError(t, recorder); message != "the project key is invalid" {
		t.Fatalf("unexpected error message %q", message)
	}
}

func TestGetReadinessReportsForeignRepositoriesForTheProjectInView(t *testing.T) {
	root := t.TempDir()
	scanner := &readinessBranchScanner{branchesByRepository: map[string][]workspaceDomain.Branch{
		"service-a": {{Name: "feature/OTHERPROJECT-14-login"}},
	}}
	handler := buildReadinessHandler(t, &application.WorkspaceConfig{
		Source:   workspaceDomain.SourceServer,
		RootPath: root,
	}, scanner, nil)

	recorder := serveReadinessRequest(handler, "/api/workspace/readiness?project=CRMMS", "usr-1")
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d with body %s", recorder.Code, recorder.Body.String())
	}

	if readiness := decodeReadiness(t, recorder); readiness.State != application.ReadinessForeign {
		t.Fatalf("expected foreign, got %q", readiness.State)
	}
}
