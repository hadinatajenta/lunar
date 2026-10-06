package application

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	bitbucketApp "lunar/backend/internal/bitbucket/application"
	jiraDomain "lunar/backend/internal/jira/domain"
)

const (
	RemoteCandidateLimit       = 5
	remoteScanConcurrency      = 4
	remoteCandidateScanTimeout = 20 * time.Second
)

type RemoteScanFailure struct {
	Repository string
	Err        error
}

type RemoteCandidateScanResult struct {
	Matches                map[string][]string
	RepositoriesChecked    int
	RepositoriesNotChecked []string
	Failures               []RemoteScanFailure
	TimedOut               bool
}

type RemoteCandidateScanner struct {
	scanner bitbucketApp.RemoteBranchScanner
	timeout time.Duration
}

type remoteScanOutcome struct {
	checked  bool
	branches []string
	err      error
}

func NewRemoteCandidateScanner(scanner bitbucketApp.RemoteBranchScanner) *RemoteCandidateScanner {
	return newRemoteCandidateScanner(scanner, remoteCandidateScanTimeout)
}

func newRemoteCandidateScanner(scanner bitbucketApp.RemoteBranchScanner, timeout time.Duration) *RemoteCandidateScanner {
	return &RemoteCandidateScanner{scanner: scanner, timeout: timeout}
}

func (s *RemoteCandidateScanner) ScanIssueKey(ctx context.Context, userID string, issueKey string, repositories []string) (*RemoteCandidateScanResult, error) {
	if s.scanner == nil {
		return nil, ErrServerReaderUnavailable
	}

	canonicalKey, err := canonicalIssueKey(issueKey)
	if err != nil {
		return nil, err
	}

	candidates, notChecked := candidateRepositories(repositories)
	scanCtx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	outcomes := s.scanCandidates(scanCtx, userID, canonicalKey, candidates)

	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("scan remote candidate repositories: %w", err)
	}

	result := &RemoteCandidateScanResult{
		Matches:                make(map[string][]string, len(candidates)),
		RepositoriesNotChecked: notChecked,
		Failures:               make([]RemoteScanFailure, 0),
		TimedOut:               scanCtx.Err() != nil,
	}

	for index, outcome := range outcomes {
		switch {
		case outcome.err != nil:
			result.RepositoriesChecked++
			result.Failures = append(result.Failures, RemoteScanFailure{Repository: candidates[index], Err: outcome.err})
		case outcome.checked:
			result.RepositoriesChecked++
			result.Matches[candidates[index]] = outcome.branches
		default:
			result.RepositoriesNotChecked = append(result.RepositoriesNotChecked, candidates[index])
		}
	}

	return result, nil
}

func (s *RemoteCandidateScanner) scanCandidates(ctx context.Context, userID string, issueKey string, candidates []string) []remoteScanOutcome {
	outcomes := make([]remoteScanOutcome, len(candidates))
	if len(candidates) == 0 {
		return outcomes
	}

	jobs := make(chan int, len(candidates))
	for index := range candidates {
		jobs <- index
	}
	close(jobs)

	workerCount := remoteScanConcurrency
	if len(candidates) < workerCount {
		workerCount = len(candidates)
	}

	waitGroup := &sync.WaitGroup{}
	for workerIndex := 0; workerIndex < workerCount; workerIndex++ {
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			s.scanWorker(ctx, userID, issueKey, candidates, jobs, outcomes)
		}()
	}
	waitGroup.Wait()

	return outcomes
}

func (s *RemoteCandidateScanner) scanWorker(ctx context.Context, userID string, issueKey string, candidates []string, jobs <-chan int, outcomes []remoteScanOutcome) {
	for index := range jobs {
		if ctx.Err() != nil {
			return
		}

		branches, err := s.scanner.SearchBranches(ctx, userID, candidates[index], issueKey)
		if err != nil {
			outcomes[index] = remoteScanOutcome{checked: true, err: fmt.Errorf("search branches in %s: %w", candidates[index], err)}
			continue
		}

		outcomes[index] = remoteScanOutcome{checked: true, branches: branchesCarryingKey(branches, issueKey)}
	}
}

func candidateRepositories(repositories []string) ([]string, []string) {
	candidates := make([]string, 0, RemoteCandidateLimit)
	notChecked := make([]string, 0)
	seen := make(map[string]struct{}, len(repositories))

	for _, repository := range repositories {
		trimmed := strings.TrimSpace(repository)
		if trimmed == "" {
			continue
		}
		if _, duplicate := seen[trimmed]; duplicate {
			continue
		}
		seen[trimmed] = struct{}{}

		if len(candidates) < RemoteCandidateLimit {
			candidates = append(candidates, trimmed)
			continue
		}
		notChecked = append(notChecked, trimmed)
	}

	return candidates, notChecked
}

func branchesCarryingKey(branchNames []string, issueKey string) []string {
	matches := make([]string, 0, len(branchNames))
	seen := make(map[string]struct{}, len(branchNames))

	for _, branchName := range branchNames {
		trimmed := strings.TrimSpace(branchName)
		if trimmed == "" || !jiraDomain.ContainsIssueKey(trimmed, issueKey) {
			continue
		}
		if _, duplicate := seen[trimmed]; duplicate {
			continue
		}
		seen[trimmed] = struct{}{}
		matches = append(matches, trimmed)
	}

	sort.Strings(matches)
	return matches
}
