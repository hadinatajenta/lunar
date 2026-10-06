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
	Matches                   map[string][]string
	MatchesByRepositoryAndKey map[string]map[string][]string
	RepositoriesChecked       int
	RepositoriesNotChecked    []string
	Failures                  []RemoteScanFailure
	TimedOut                  bool
}

type RemoteCandidateScanner struct {
	scanner bitbucketApp.RemoteBranchScanner
	timeout time.Duration
}

type remoteScanOutcome struct {
	checked       bool
	branches      []string
	branchesByKey map[string][]string
	err           error
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

	outcomes := s.scanCandidates(scanCtx, candidates, func(ctx context.Context, repository string) remoteScanOutcome {
		branches, err := s.scanner.SearchBranches(ctx, userID, repository, canonicalKey)
		if err != nil {
			return remoteScanOutcome{checked: true, err: fmt.Errorf("search branches in %s: %w", repository, err)}
		}

		return remoteScanOutcome{checked: true, branches: branchesCarryingKey(branches, canonicalKey)}
	})

	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("scan remote candidate repositories: %w", err)
	}

	result := &RemoteCandidateScanResult{
		Matches:                make(map[string][]string, len(candidates)),
		RepositoriesNotChecked: notChecked,
		Failures:               make([]RemoteScanFailure, 0),
		TimedOut:               scanCtx.Err() != nil,
	}
	result.collectOutcomes(candidates, outcomes)

	return result, nil
}

func (r *RemoteCandidateScanResult) collectOutcomes(candidates []string, outcomes []remoteScanOutcome) {
	for index, outcome := range outcomes {
		switch {
		case outcome.err != nil:
			r.RepositoriesChecked++
			r.Failures = append(r.Failures, RemoteScanFailure{Repository: candidates[index], Err: outcome.err})
		case outcome.checked:
			r.RepositoriesChecked++
			r.Matches[candidates[index]] = outcome.branches
			if outcome.branchesByKey != nil {
				if r.MatchesByRepositoryAndKey == nil {
					r.MatchesByRepositoryAndKey = make(map[string]map[string][]string, len(candidates))
				}
				r.MatchesByRepositoryAndKey[candidates[index]] = outcome.branchesByKey
			}
		default:
			r.RepositoriesNotChecked = append(r.RepositoriesNotChecked, candidates[index])
		}
	}
}

func (s *RemoteCandidateScanner) scanCandidates(ctx context.Context, candidates []string, scan func(ctx context.Context, repository string) remoteScanOutcome) []remoteScanOutcome {
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
			for index := range jobs {
				if ctx.Err() != nil {
					return
				}
				outcomes[index] = scan(ctx, candidates[index])
			}
		}()
	}
	waitGroup.Wait()

	return outcomes
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
