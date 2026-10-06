package application

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"lunar/backend/internal/shared/cache"
	"lunar/backend/internal/workspace/domain"
)

const gitMetadataEntry = ".git"

type WorkspaceReadinessRequest struct {
	RootPath              string
	ProjectKey            string
	RemoteRepositoryCount int
	MissingRepositories   []string
}

type localWorkspaceFacts struct {
	repositoryCount  int
	branchNames      []string
	rootIsRepository bool
}

type WorkspaceReadinessService struct {
	scanner      RepositoryBranchScanner
	allowedRoots []string
	cache        *cache.Store
}

func NewWorkspaceReadinessService(scanner RepositoryBranchScanner, allowedRoots []string) *WorkspaceReadinessService {
	return newWorkspaceReadinessService(scanner, allowedRoots, LocalBranchScanCacheTTL)
}

func newWorkspaceReadinessService(scanner RepositoryBranchScanner, allowedRoots []string, cacheTTL time.Duration) *WorkspaceReadinessService {
	return &WorkspaceReadinessService{
		scanner:      scanner,
		allowedRoots: append([]string(nil), allowedRoots...),
		cache:        cache.New(cacheTTL),
	}
}

func (s *WorkspaceReadinessService) ReadinessFor(ctx context.Context, request WorkspaceReadinessRequest) (WorkspaceReadiness, error) {
	facts := WorkspaceReadinessFacts{
		RootPath:              strings.TrimSpace(request.RootPath),
		ProjectKey:            request.ProjectKey,
		RemoteRepositoryCount: request.RemoteRepositoryCount,
		MissingRepositories:   request.MissingRepositories,
	}

	if facts.RootPath == "" {
		return ClassifyWorkspaceReadiness(facts), nil
	}

	resolvedRoot, err := domain.ResolveWorkspaceRoot(facts.RootPath, s.allowedRoots)
	if err != nil {
		facts.RootError = err
		return ClassifyWorkspaceReadiness(facts), nil
	}
	facts.RootPath = resolvedRoot

	localFacts, err := s.localFacts(ctx, resolvedRoot)
	if err != nil {
		return WorkspaceReadiness{}, err
	}

	facts.LocalRepositoryCount = localFacts.repositoryCount
	facts.BranchNames = localFacts.branchNames
	facts.RootIsRepository = localFacts.rootIsRepository

	return ClassifyWorkspaceReadiness(facts), nil
}

func (s *WorkspaceReadinessService) localFacts(ctx context.Context, resolvedRoot string) (localWorkspaceFacts, error) {
	if cached, found := s.cache.Get(resolvedRoot); found {
		if facts, valid := cached.(localWorkspaceFacts); valid {
			return facts, nil
		}
	}

	isRepository, err := rootIsRepository(resolvedRoot)
	if err != nil {
		return localWorkspaceFacts{}, err
	}

	if s.scanner == nil {
		return localWorkspaceFacts{}, ErrServerReaderUnavailable
	}

	branchesByRepository, err := s.scanner.ScanRepositoryBranches(ctx, resolvedRoot)
	if err != nil {
		return localWorkspaceFacts{}, fmt.Errorf("scan repositories for readiness: %w", err)
	}

	facts := localWorkspaceFacts{
		repositoryCount:  len(branchesByRepository),
		branchNames:      workspaceBranchNames(branchesByRepository),
		rootIsRepository: isRepository,
	}
	s.cache.Set(resolvedRoot, facts)

	return facts, nil
}

func rootIsRepository(resolvedRoot string) (bool, error) {
	candidate := filepath.Join(resolvedRoot, gitMetadataEntry)
	if !strings.HasPrefix(candidate, resolvedRoot+string(os.PathSeparator)) {
		return false, fmt.Errorf("read workspace metadata: %s escapes the workspace root", candidate)
	}

	_, err := os.Stat(candidate)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}

	return false, fmt.Errorf("read workspace metadata: %w", err)
}

func workspaceBranchNames(branchesByRepository map[string][]domain.Branch) []string {
	names := make([]string, 0)
	for _, branches := range branchesByRepository {
		for _, branch := range branches {
			names = append(names, branch.Name)
		}
	}

	return names
}
