package application

import (
	"context"
	"fmt"
	"sort"
	"strings"

	jiraDomain "lunar/backend/internal/jira/domain"
	sharedErrors "lunar/backend/internal/shared/errors"
)

const RemoteKeyLimit = 50

func (s *RemoteCandidateScanner) ScanIssueKeys(ctx context.Context, userID string, issueKeys []string, repositories []string) (*RemoteCandidateScanResult, error) {
	if s.scanner == nil {
		return nil, ErrServerReaderUnavailable
	}

	canonicalKeys, err := canonicalIssueKeys(issueKeys)
	if err != nil {
		return nil, err
	}

	candidates, notChecked := candidateRepositories(repositories)
	if len(canonicalKeys) == 0 {
		return emptyCandidateScanResult(candidates, notChecked), nil
	}

	scanCtx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	outcomes := s.scanCandidates(scanCtx, candidates, func(ctx context.Context, repository string) remoteScanOutcome {
		branches, err := s.scanner.ListBranches(ctx, userID, repository)
		if err != nil {
			return remoteScanOutcome{checked: true, err: fmt.Errorf("list branches in %s: %w", repository, err)}
		}

		branchesByKey, matches := branchesCarryingKeys(branches, canonicalKeys)

		return remoteScanOutcome{checked: true, branches: matches, branchesByKey: branchesByKey}
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

func canonicalIssueKeys(issueKeys []string) ([]string, error) {
	keys := make([]string, 0, len(issueKeys))
	seen := make(map[string]struct{}, len(issueKeys))

	for _, issueKey := range issueKeys {
		canonicalKey, err := canonicalIssueKey(issueKey)
		if err != nil {
			return nil, err
		}
		if _, duplicate := seen[canonicalKey]; duplicate {
			continue
		}
		seen[canonicalKey] = struct{}{}
		keys = append(keys, canonicalKey)
	}

	if len(keys) > RemoteKeyLimit {
		return nil, fmt.Errorf("%w: at most %d issue keys can be scanned at once", sharedErrors.ErrBadRequest, RemoteKeyLimit)
	}

	return keys, nil
}

func emptyCandidateScanResult(candidates []string, notChecked []string) *RemoteCandidateScanResult {
	skipped := make([]string, 0, len(candidates)+len(notChecked))
	skipped = append(skipped, candidates...)
	skipped = append(skipped, notChecked...)

	return &RemoteCandidateScanResult{
		Matches:                make(map[string][]string, len(candidates)),
		RepositoriesNotChecked: skipped,
		Failures:               make([]RemoteScanFailure, 0),
	}
}

func branchesCarryingKeys(branchNames []string, issueKeys []string) (map[string][]string, []string) {
	branchesByKey := make(map[string][]string, len(issueKeys))
	for _, issueKey := range issueKeys {
		branchesByKey[issueKey] = make([]string, 0)
	}

	seenPairs := make(map[string]struct{}, len(branchNames))
	seenBranches := make(map[string]struct{}, len(branchNames))
	matches := make([]string, 0, len(branchNames))

	for _, branchName := range branchNames {
		trimmed := strings.TrimSpace(branchName)
		if trimmed == "" {
			continue
		}

		carriesKey := false
		for _, issueKey := range issueKeys {
			if !jiraDomain.ContainsIssueKey(trimmed, issueKey) {
				continue
			}
			carriesKey = true

			pair := issueKey + "\x00" + trimmed
			if _, duplicate := seenPairs[pair]; duplicate {
				continue
			}
			seenPairs[pair] = struct{}{}
			branchesByKey[issueKey] = append(branchesByKey[issueKey], trimmed)
		}

		if !carriesKey {
			continue
		}
		if _, duplicate := seenBranches[trimmed]; duplicate {
			continue
		}
		seenBranches[trimmed] = struct{}{}
		matches = append(matches, trimmed)
	}

	for issueKey := range branchesByKey {
		sort.Strings(branchesByKey[issueKey])
	}
	sort.Strings(matches)

	return branchesByKey, matches
}
