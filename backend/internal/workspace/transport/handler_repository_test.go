package transport

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	workspaceDomain "lunar/backend/internal/workspace/domain"
)

type stubGitReader struct {
	summaries []workspaceDomain.RepositorySummary
	details   []workspaceDomain.RepositoryDetail
}

func (s *stubGitReader) ListRepositories(ctx context.Context, rootPath string) ([]workspaceDomain.RepositorySummary, error) {
	return s.summaries, nil
}

func (s *stubGitReader) ReadRepositories(ctx context.Context, rootPath string, names []string) ([]workspaceDomain.RepositoryDetail, error) {
	return s.details, nil
}

func (s *stubGitReader) ValidateRoot(ctx context.Context, rootPath string) error {
	return nil
}

func decodeRepositoriesResponse(t *testing.T, recorder *httptest.ResponseRecorder) repositoriesResponse {
	t.Helper()

	var response repositoriesResponse
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("cannot decode repositories response: %v", err)
	}
	return response
}

func decodeServicesResponse(t *testing.T, recorder *httptest.ResponseRecorder) servicesResponse {
	t.Helper()

	var response servicesResponse
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("cannot decode services response: %v", err)
	}
	return response
}

func TestWorkspaceHandler_RepositoriesAndServicesReturnClientSideMessageForHelperSource(t *testing.T) {
	handler, _ := setupWorkspaceHandler(t, nil)

	configureBody := `{"source":"helper","root_path":"/Users/erendt/BRI","helper_url":"http://127.0.0.1:5199"}`
	configureRecorder := performRequest(handler, authorizedRequest(t, http.MethodPut, "/api/workspace", configureBody, handlerTestUserA))
	if configureRecorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", configureRecorder.Code, configureRecorder.Body.String())
	}

	for _, path := range []string{"/api/workspace/repositories", "/api/workspace/services"} {
		recorder := performRequest(handler, authorizedRequest(t, http.MethodGet, path, "", handlerTestUserA))

		if recorder.Code != http.StatusBadRequest {
			t.Errorf("expected %s to return a 4xx, got %d", path, recorder.Code)
			continue
		}
		responseBody := recorder.Body.String()
		errorResponse := decodeErrorResponse(t, recorder)
		if !strings.Contains(errorResponse.Error, "client-side") {
			t.Errorf("expected %s to explain client-side reading, got %q", path, errorResponse.Error)
		}
		if !strings.Contains(errorResponse.Error, "local helper") {
			t.Errorf("expected %s to mention the local helper, got %q", path, errorResponse.Error)
		}
		if strings.Contains(responseBody, `"repositories":[]`) || strings.Contains(responseBody, `"services":[]`) {
			t.Errorf("expected %s to never send a fabricated success payload, got %s", path, responseBody)
		}
	}
}

func TestWorkspaceHandler_RepositoriesAndServicesReturnClientSideMessageWhenNoWorkspaceExists(t *testing.T) {
	handler, _ := setupWorkspaceHandler(t, nil)

	for _, path := range []string{"/api/workspace/repositories", "/api/workspace/services"} {
		recorder := performRequest(handler, authorizedRequest(t, http.MethodGet, path, "", handlerTestUserA))

		if recorder.Code != http.StatusBadRequest {
			t.Errorf("expected %s to return 400 for a fresh user, got %d", path, recorder.Code)
			continue
		}
		responseBody := recorder.Body.String()
		if !strings.Contains(decodeErrorResponse(t, recorder).Error, "client-side") {
			t.Errorf("expected %s to explain client-side reading, got %s", path, responseBody)
		}
	}
}

func TestWorkspaceHandler_ServerModeServicesFilterByStoredSelections(t *testing.T) {
	gitReader := &stubGitReader{
		details: []workspaceDomain.RepositoryDetail{
			{Name: "everest", Branch: "development"},
			{Name: "way4", Branch: "master"},
		},
	}
	handler, _ := setupWorkspaceHandler(t, gitReader)
	rootPath := t.TempDir()

	configureBody := `{"source":"server","root_path":"` + rootPath + `"}`
	if recorder := performRequest(handler, authorizedRequest(t, http.MethodPut, "/api/workspace", configureBody, handlerTestUserA)); recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", recorder.Code, recorder.Body.String())
	}
	if recorder := performRequest(handler, authorizedRequest(t, http.MethodPut, "/api/workspace/selections", `{"repos":["everest","missing-on-disk"]}`, handlerTestUserA)); recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", recorder.Code, recorder.Body.String())
	}

	recorder := performRequest(handler, authorizedRequest(t, http.MethodGet, "/api/workspace/services", "", handlerTestUserA))
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", recorder.Code, recorder.Body.String())
	}

	response := decodeServicesResponse(t, recorder)
	if len(response.Services) != 1 {
		t.Fatalf("expected only the selected repository on disk, got %+v", response.Services)
	}
	if response.Services[0].Name != "everest" {
		t.Errorf("expected everest, got %q", response.Services[0].Name)
	}
}

func TestWorkspaceHandler_ServerModeReportsMissingReader(t *testing.T) {
	handler, _ := setupWorkspaceHandler(t, nil)
	rootPath := t.TempDir()

	configureBody := `{"source":"server","root_path":"` + rootPath + `"}`
	configureRecorder := performRequest(handler, authorizedRequest(t, http.MethodPut, "/api/workspace", configureBody, handlerTestUserA))
	if configureRecorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", configureRecorder.Code, configureRecorder.Body.String())
	}

	for _, path := range []string{"/api/workspace/repositories", "/api/workspace/services"} {
		recorder := performRequest(handler, authorizedRequest(t, http.MethodGet, path, "", handlerTestUserA))

		if recorder.Code != http.StatusServiceUnavailable {
			t.Errorf("expected %s to return 503, got %d: %s", path, recorder.Code, recorder.Body.String())
			continue
		}
		errorResponse := decodeErrorResponse(t, recorder)
		if errorResponse.Error != "server-side repository reader is not configured" {
			t.Errorf("expected a clear reader message for %s, got %q", path, errorResponse.Error)
		}
	}
}

func TestWorkspaceHandler_ServerModeReturnsContractShapedRepositories(t *testing.T) {
	gitReader := &stubGitReader{
		summaries: []workspaceDomain.RepositorySummary{
			{Name: "everest", IsGit: true, Branch: "development", DirtyCount: 0, UpdatedRelative: "37 minutes ago"},
			{Name: "way4", IsGit: false, Error: "not a git repository"},
		},
	}
	handler, _ := setupWorkspaceHandler(t, gitReader)
	rootPath := t.TempDir()

	configureBody := `{"source":"server","root_path":"` + rootPath + `"}`
	if recorder := performRequest(handler, authorizedRequest(t, http.MethodPut, "/api/workspace", configureBody, handlerTestUserA)); recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", recorder.Code, recorder.Body.String())
	}

	recorder := performRequest(handler, authorizedRequest(t, http.MethodGet, "/api/workspace/repositories", "", handlerTestUserA))
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", recorder.Code, recorder.Body.String())
	}

	body := recorder.Body.String()
	for _, expectedKey := range []string{`"root_path"`, `"repositories"`, `"name"`, `"is_git"`, `"branch"`, `"dirty_count"`, `"updated_relative"`, `"error"`} {
		if !strings.Contains(body, expectedKey) {
			t.Errorf("expected the payload to contain %s, got %s", expectedKey, body)
		}
	}
	if strings.Contains(body, "isGit") || strings.Contains(body, "dirtyCount") {
		t.Errorf("expected snake_case json keys only, got %s", body)
	}

	response := decodeRepositoriesResponse(t, recorder)
	if len(response.Repositories) != 2 {
		t.Fatalf("expected 2 repositories, got %+v", response.Repositories)
	}
	if response.Repositories[0].Name != "everest" || !response.Repositories[0].IsGit || response.Repositories[0].Branch != "development" {
		t.Errorf("unexpected first repository: %+v", response.Repositories[0])
	}
	if response.Repositories[1].IsGit || response.Repositories[1].Error != "not a git repository" {
		t.Errorf("unexpected second repository: %+v", response.Repositories[1])
	}
}

func TestWorkspaceHandler_ServerModeReturnsContractShapedServices(t *testing.T) {
	gitReader := &stubGitReader{
		details: []workspaceDomain.RepositoryDetail{
			{
				Name:            "everest",
				Branch:          "development",
				UpdatedRelative: "37 minutes ago",
				Commit: workspaceDomain.CommitInfo{
					Hash:         "88b4ca3",
					Author:       "Hadinata Jenta",
					RelativeTime: "37 minutes ago",
					Subject:      "fix: guard nil session",
				},
				Ahead:          0,
				Behind:         2,
				Files:          []workspaceDomain.ChangedFile{{Status: workspaceDomain.StatusModified, Path: "internal/auth/refresh.go", Added: 24, Deleted: 6}},
				FilesTruncated: false,
			},
		},
	}
	handler, _ := setupWorkspaceHandler(t, gitReader)
	rootPath := t.TempDir()

	configureBody := `{"source":"server","root_path":"` + rootPath + `"}`
	if recorder := performRequest(handler, authorizedRequest(t, http.MethodPut, "/api/workspace", configureBody, handlerTestUserA)); recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", recorder.Code, recorder.Body.String())
	}
	if recorder := performRequest(handler, authorizedRequest(t, http.MethodPut, "/api/workspace/selections", `{"repos":["everest"]}`, handlerTestUserA)); recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", recorder.Code, recorder.Body.String())
	}

	recorder := performRequest(handler, authorizedRequest(t, http.MethodGet, "/api/workspace/services", "", handlerTestUserA))
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", recorder.Code, recorder.Body.String())
	}

	body := recorder.Body.String()
	for _, expectedKey := range []string{`"services"`, `"updated_relative"`, `"commit"`, `"hash"`, `"author"`, `"relative_time"`, `"subject"`, `"ahead"`, `"behind"`, `"files"`, `"status"`, `"path"`, `"added"`, `"deleted"`, `"files_truncated"`} {
		if !strings.Contains(body, expectedKey) {
			t.Errorf("expected the payload to contain %s, got %s", expectedKey, body)
		}
	}
	if strings.Contains(body, "relativeTime") || strings.Contains(body, "filesTruncated") {
		t.Errorf("expected snake_case json keys only, got %s", body)
	}

	response := decodeServicesResponse(t, recorder)
	if len(response.Services) != 1 {
		t.Fatalf("expected 1 service, got %+v", response.Services)
	}
	service := response.Services[0]
	if service.Name != "everest" || service.Behind != 2 || service.Commit.Hash != "88b4ca3" {
		t.Errorf("unexpected service detail: %+v", service)
	}
	if len(service.Files) != 1 || service.Files[0].Status != string(workspaceDomain.StatusModified) {
		t.Errorf("unexpected changed files: %+v", service.Files)
	}
}
