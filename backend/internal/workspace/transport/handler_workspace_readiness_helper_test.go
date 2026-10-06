package transport

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"testing"

	"lunar/backend/internal/workspace/application"
	workspaceDomain "lunar/backend/internal/workspace/domain"
)

func TestGetReadinessReportsPartialCoverageFromTheCatalogue(t *testing.T) {
	root := t.TempDir()
	scanner := &readinessBranchScanner{branchesByRepository: map[string][]workspaceDomain.Branch{
		"service-a": {{Name: "feature/CRMMS-77911-enhancement"}},
	}}
	catalogue := &readinessCatalogueSource{catalogue: &application.RepositoryCatalogue{
		Repositories: []application.CataloguedRepository{
			{Repo: "service-a", IsCloned: true},
			{Repo: "service-c"},
			{Repo: "  "},
			{Repo: "service-b"},
		},
		LocalRepositoryCount:  1,
		RemoteRepositoryCount: 3,
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
	if readiness.State != application.ReadinessPartial {
		t.Fatalf("expected partial, got %q", readiness.State)
	}
	if readiness.MissingRepositoryCount != 2 {
		t.Fatalf("expected 2 missing repositories, got %d", readiness.MissingRepositoryCount)
	}
	if len(readiness.MissingRepositories) != 2 || readiness.MissingRepositories[0] != "service-b" {
		t.Fatalf("expected the missing repositories to be sorted, got %+v", readiness.MissingRepositories)
	}
}

func TestGetReadinessReportsAScanFailure(t *testing.T) {
	handler := buildReadinessHandler(t, &application.WorkspaceConfig{
		Source:   workspaceDomain.SourceServer,
		RootPath: t.TempDir(),
	}, &readinessBranchScanner{err: errors.New("git is unavailable")}, nil)

	recorder := serveReadinessRequest(handler, "/api/workspace/readiness", "usr-1")
	if recorder.Code != http.StatusBadGateway {
		t.Fatalf("expected 502, got %d with body %s", recorder.Code, recorder.Body.String())
	}
	if message := decodeReadinessError(t, recorder); message != "the workspace repositories could not be read" {
		t.Fatalf("unexpected error message %q", message)
	}
}

func TestGetReadinessReportsAScanTimeout(t *testing.T) {
	handler := buildReadinessHandler(t, &application.WorkspaceConfig{
		Source:   workspaceDomain.SourceServer,
		RootPath: t.TempDir(),
	}, &readinessBranchScanner{err: fmt.Errorf("scan: %w", context.DeadlineExceeded)}, nil)

	recorder := serveReadinessRequest(handler, "/api/workspace/readiness", "usr-1")
	if recorder.Code != http.StatusGatewayTimeout {
		t.Fatalf("expected 504, got %d with body %s", recorder.Code, recorder.Body.String())
	}
}

func TestGetReadinessWithoutAWorkspaceSourceIsRefused(t *testing.T) {
	handler := buildReadinessHandler(t, nil, &readinessBranchScanner{}, nil)

	recorder := serveReadinessRequest(handler, "/api/workspace/readiness", "usr-1")
	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d with body %s", recorder.Code, recorder.Body.String())
	}
}

func TestGetHelperReadinessReadsTheRootFromTheQuery(t *testing.T) {
	root := t.TempDir()
	service := application.NewWorkspaceReadinessService(&readinessBranchScanner{branchesByRepository: map[string][]workspaceDomain.Branch{}}, nil)
	handler := NewWorkspaceReadinessHandler(service, nil, nil)

	recorder := serveReadinessRequest(http.HandlerFunc(handler.GetHelperReadiness), "/readiness?root="+root, "")
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d with body %s", recorder.Code, recorder.Body.String())
	}

	readiness := decodeReadiness(t, recorder)
	if readiness.State != application.ReadinessEmpty {
		t.Fatalf("expected empty, got %q", readiness.State)
	}
	if readiness.Message != fmt.Sprintf("No repositories were found in %s.", resolvedRoot(t, root)) {
		t.Fatalf("unexpected message %q", readiness.Message)
	}
}

func TestGetHelperReadinessTreatsAMissingRootAsNotConfigured(t *testing.T) {
	service := application.NewWorkspaceReadinessService(&readinessBranchScanner{}, nil)
	handler := NewWorkspaceReadinessHandler(service, nil, nil)

	recorder := serveReadinessRequest(http.HandlerFunc(handler.GetHelperReadiness), "/readiness", "")
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d with body %s", recorder.Code, recorder.Body.String())
	}

	if readiness := decodeReadiness(t, recorder); readiness.State != application.ReadinessNotConfigured {
		t.Fatalf("expected not-configured, got %q", readiness.State)
	}
}
