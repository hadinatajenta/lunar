package infrastructure

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"lunar/backend/internal/workspace/domain"
)

const (
	branchScanRefFormat      = "%(refname:short)\x1f%(HEAD)"
	branchScanFieldDelimiter = "\x1f"
	branchScanFieldCount     = 2
	branchScanCurrentMarker  = "*"
	gitDirectoryName         = ".git"
)

func (i *GitInspector) ScanRepositoryBranches(ctx context.Context, rootPath string) (map[string][]domain.Branch, error) {
	resolvedRoot, err := domain.ResolveWorkspaceRoot(rootPath, i.allowedRoots)
	if err != nil {
		return nil, err
	}

	repositoryNames, err := listRepositoryNames(resolvedRoot)
	if err != nil {
		return nil, err
	}

	branchesByRepository := make(map[string][]domain.Branch, len(repositoryNames))
	scanErrors := make([]error, 0)
	resultsMutex := &sync.Mutex{}
	waitGroup := &sync.WaitGroup{}
	semaphore := make(chan struct{}, domain.MaxParallelScans)

	for _, repositoryName := range repositoryNames {
		waitGroup.Add(1)
		go func(repositoryName string) {
			defer waitGroup.Done()

			if !acquireScanSlot(ctx, semaphore) {
				return
			}
			defer releaseScanSlot(semaphore)

			branches, isRepository, scanErr := i.scanRepositoryBranches(ctx, resolvedRoot, repositoryName)

			resultsMutex.Lock()
			defer resultsMutex.Unlock()

			if scanErr != nil {
				scanErrors = append(scanErrors, scanErr)
				return
			}
			if isRepository {
				branchesByRepository[repositoryName] = branches
			}
		}(repositoryName)
	}

	waitGroup.Wait()

	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("scan repository branches: %w", err)
	}
	if len(scanErrors) > 0 {
		return nil, errors.Join(scanErrors...)
	}

	return branchesByRepository, nil
}

func (i *GitInspector) scanRepositoryBranches(ctx context.Context, resolvedRoot string, repositoryName string) ([]domain.Branch, bool, error) {
	repositoryPath, err := domain.ResolveRepoPath(resolvedRoot, repositoryName)
	if err != nil {
		return nil, false, fmt.Errorf("resolve repository %s: %w", repositoryName, err)
	}

	isRepository, err := hasGitDirectory(repositoryPath)
	if err != nil {
		return nil, false, fmt.Errorf("read repository metadata of %s: %w", repositoryName, err)
	}
	if !isRepository {
		return nil, false, nil
	}

	output, err := runGit(ctx, repositoryPath, "for-each-ref", "--format="+branchScanRefFormat, "refs/heads")
	if err != nil {
		return nil, false, fmt.Errorf("scan branches of %s: %w", repositoryName, err)
	}

	return parseBranchScanOutput(output), true, nil
}

func hasGitDirectory(repositoryPath string) (bool, error) {
	_, err := os.Stat(filepath.Join(repositoryPath, gitDirectoryName))
	if err == nil {
		return true, nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}

	return false, fmt.Errorf("stat repository metadata: %w", err)
}

func parseBranchScanOutput(output string) []domain.Branch {
	branches := make([]domain.Branch, 0)

	for _, line := range strings.Split(output, "\n") {
		trimmedLine := strings.TrimRight(line, "\r")
		if strings.TrimSpace(trimmedLine) == "" {
			continue
		}

		fields := strings.Split(trimmedLine, branchScanFieldDelimiter)
		if len(fields) != branchScanFieldCount {
			continue
		}

		name := strings.TrimSpace(fields[0])
		if name == "" {
			continue
		}

		branches = append(branches, domain.Branch{
			Name:      name,
			IsCurrent: strings.TrimSpace(fields[1]) == branchScanCurrentMarker,
		})
	}

	return branches
}
