package application

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"lunar/backend/internal/workspace/domain"
)

type stubRepositoryBranchScanner struct {
	branchesByRepository map[string][]domain.Branch
	err                  error
	roots                []string
}

var _ RepositoryBranchScanner = (*stubRepositoryBranchScanner)(nil)

func (s *stubRepositoryBranchScanner) ScanRepositoryBranches(ctx context.Context, rootPath string) (map[string][]domain.Branch, error) {
	s.roots = append(s.roots, rootPath)
	if s.err != nil {
		return nil, s.err
	}

	return s.branchesByRepository, nil
}

func resolvedRootFor(t *testing.T, rootPath string) string {
	t.Helper()

	resolvedRoot, err := domain.ResolveWorkspaceRoot(rootPath, nil)
	if err != nil {
		t.Fatalf("cannot resolve the test root: %v", err)
	}

	return resolvedRoot
}

func TestReadinessForWarnsWithoutAConfiguredWorkspace(t *testing.T) {
	service := NewWorkspaceReadinessService(nil, nil)

	readiness, err := service.ReadinessFor(context.Background(), WorkspaceReadinessRequest{RootPath: "  "})
	if err != nil {
		t.Fatalf("readiness must never fail on a missing workspace, got %v", err)
	}
	if readiness.State != ReadinessNotConfigured {
		t.Fatalf("expected not-configured, got %q", readiness.State)
	}
}

func TestReadinessForReportsARootThatDoesNotExist(t *testing.T) {
	scanner := &stubRepositoryBranchScanner{}
	service := NewWorkspaceReadinessService(scanner, nil)

	readiness, err := service.ReadinessFor(context.Background(), WorkspaceReadinessRequest{
		RootPath: filepath.Join(t.TempDir(), "missing"),
	})
	if err != nil {
		t.Fatalf("a refused root must not fail the request, got %v", err)
	}
	if readiness.State != ReadinessInvalid {
		t.Fatalf("expected invalid, got %q", readiness.State)
	}
	if len(scanner.roots) != 0 {
		t.Fatalf("expected no scan for a refused root, got %+v", scanner.roots)
	}
}

func TestReadinessForReportsARootOutsideTheAllowedRoots(t *testing.T) {
	root := t.TempDir()
	otherRoot := t.TempDir()
	scanner := &stubRepositoryBranchScanner{branchesByRepository: map[string][]domain.Branch{
		"mms-api": {{Name: "feature/CRMMS-77911-enhancement"}},
	}}
	service := newWorkspaceReadinessService(scanner, []string{otherRoot}, time.Minute)

	readiness, err := service.ReadinessFor(context.Background(), WorkspaceReadinessRequest{RootPath: root})
	if err != nil {
		t.Fatalf("a refused root must not fail the request, got %v", err)
	}
	if readiness.State != ReadinessInvalid {
		t.Fatalf("expected invalid, got %q", readiness.State)
	}
	if readiness.Message != "workspace root is outside the allowed roots" {
		t.Fatalf("unexpected message %q", readiness.Message)
	}
	if len(scanner.roots) != 0 {
		t.Fatalf("expected no scan for a refused root, got %+v", scanner.roots)
	}
}

func TestReadinessForReportsAWorkspaceWithoutRepositories(t *testing.T) {
	root := t.TempDir()
	scanner := &stubRepositoryBranchScanner{branchesByRepository: map[string][]domain.Branch{}}
	service := newWorkspaceReadinessService(scanner, nil, time.Minute)

	readiness, err := service.ReadinessFor(context.Background(), WorkspaceReadinessRequest{RootPath: root})
	if err != nil {
		t.Fatalf("cannot read readiness: %v", err)
	}
	if readiness.State != ReadinessEmpty {
		t.Fatalf("expected empty, got %q", readiness.State)
	}
	if len(scanner.roots) != 1 || scanner.roots[0] != resolvedRootFor(t, root) {
		t.Fatalf("expected the resolved root to be scanned, got %+v", scanner.roots)
	}
}

func TestReadinessForReportsARootThatIsItselfARepository(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, ".git"), 0o750); err != nil {
		t.Fatalf("cannot create the repository metadata: %v", err)
	}
	scanner := &stubRepositoryBranchScanner{branchesByRepository: map[string][]domain.Branch{}}
	service := newWorkspaceReadinessService(scanner, nil, time.Minute)

	readiness, err := service.ReadinessFor(context.Background(), WorkspaceReadinessRequest{RootPath: root})
	if err != nil {
		t.Fatalf("cannot read readiness: %v", err)
	}
	if readiness.State != ReadinessSingleRepo {
		t.Fatalf("expected single-repo, got %q", readiness.State)
	}
	if !strings.Contains(readiness.Message, "is itself a repository") {
		t.Fatalf("unexpected message %q", readiness.Message)
	}
}

func TestReadinessForReadsTheProjectInViewFromTheLocalBranches(t *testing.T) {
	root := t.TempDir()
	scanner := &stubRepositoryBranchScanner{branchesByRepository: map[string][]domain.Branch{
		"mms-api": {{Name: "feature/CRMMS-77911-enhancement"}},
	}}
	service := newWorkspaceReadinessService(scanner, nil, time.Minute)

	readiness, err := service.ReadinessFor(context.Background(), WorkspaceReadinessRequest{
		RootPath:              root,
		ProjectKey:            "CRMMS",
		RemoteRepositoryCount: 1,
	})
	if err != nil {
		t.Fatalf("cannot read readiness: %v", err)
	}
	if readiness.State != ReadinessReady {
		t.Fatalf("expected ready, got %q", readiness.State)
	}
	if readiness.LocalRepositoryCount != 1 {
		t.Fatalf("expected one local repository, got %d", readiness.LocalRepositoryCount)
	}
}

func TestReadinessForReportsPartialCoverageWithTheMissingRepositories(t *testing.T) {
	root := t.TempDir()
	scanner := &stubRepositoryBranchScanner{branchesByRepository: map[string][]domain.Branch{
		"mms-api": {{Name: "feature/CRMMS-77911-enhancement"}},
	}}
	service := newWorkspaceReadinessService(scanner, nil, time.Minute)

	readiness, err := service.ReadinessFor(context.Background(), WorkspaceReadinessRequest{
		RootPath:              root,
		ProjectKey:            "CRMMS",
		RemoteRepositoryCount: 3,
		MissingRepositories:   []string{"elbrus", "administrasi-service"},
	})
	if err != nil {
		t.Fatalf("cannot read readiness: %v", err)
	}
	if readiness.State != ReadinessPartial {
		t.Fatalf("expected partial, got %q", readiness.State)
	}
	if readiness.MissingRepositoryCount != 2 {
		t.Fatalf("expected two missing repositories, got %d", readiness.MissingRepositoryCount)
	}
	if len(readiness.MissingRepositories) != 2 || readiness.MissingRepositories[0] != "administrasi-service" {
		t.Fatalf("expected the missing repositories to be sorted, got %+v", readiness.MissingRepositories)
	}
}

func TestReadinessForReusesTheCachedWorkspaceScan(t *testing.T) {
	root := t.TempDir()
	scanner := &stubRepositoryBranchScanner{branchesByRepository: map[string][]domain.Branch{
		"mms-api": {{Name: "feature/CRMMS-77911-enhancement"}},
	}}
	service := newWorkspaceReadinessService(scanner, nil, time.Minute)
	request := WorkspaceReadinessRequest{RootPath: root, ProjectKey: "CRMMS"}

	if _, err := service.ReadinessFor(context.Background(), request); err != nil {
		t.Fatalf("cannot read readiness: %v", err)
	}
	if _, err := service.ReadinessFor(context.Background(), request); err != nil {
		t.Fatalf("cannot read readiness: %v", err)
	}

	if len(scanner.roots) != 1 {
		t.Fatalf("expected one scan for repeated readiness reads, got %d", len(scanner.roots))
	}
}

func TestReadinessForReportsAScanFailure(t *testing.T) {
	root := t.TempDir()
	scanner := &stubRepositoryBranchScanner{err: errors.New("git is unavailable")}
	service := newWorkspaceReadinessService(scanner, nil, time.Minute)

	_, err := service.ReadinessFor(context.Background(), WorkspaceReadinessRequest{RootPath: root})
	if err == nil {
		t.Fatal("expected a scan failure to be reported")
	}
	if !strings.Contains(err.Error(), "git is unavailable") {
		t.Fatalf("expected the cause to be kept, got %q", err.Error())
	}
}

func TestReadinessForRequiresAScanner(t *testing.T) {
	service := NewWorkspaceReadinessService(nil, nil)

	_, err := service.ReadinessFor(context.Background(), WorkspaceReadinessRequest{RootPath: t.TempDir()})
	if !errors.Is(err, ErrServerReaderUnavailable) {
		t.Fatalf("expected an unavailable reader error, got %v", err)
	}
}
