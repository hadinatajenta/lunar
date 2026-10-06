package application

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	sharedErrors "lunar/backend/internal/shared/errors"
	"lunar/backend/internal/workspace/domain"
)

type stubBranchScanner struct {
	calls                int
	branchesByRepository map[string][]domain.Branch
	err                  error
}

func (s *stubBranchScanner) ScanRepositoryBranches(_ context.Context, _ string) (map[string][]domain.Branch, error) {
	s.calls++
	if s.err != nil {
		return nil, s.err
	}

	return s.branchesByRepository, nil
}

func keyedBranchScanner() *stubBranchScanner {
	return &stubBranchScanner{
		branchesByRepository: map[string][]domain.Branch{
			"alpha": {
				{Name: "feature/CRMMS-6397-first", IsCurrent: true},
				{Name: "feature/CRMMS-63970-second"},
				{Name: "main"},
			},
			"beta": {
				{Name: "feature/crmms-6397-lower"},
			},
		},
	}
}

func matchedBranches(matches []BranchMatch) []string {
	branches := make([]string, 0, len(matches))
	for _, match := range matches {
		branches = append(branches, match.Repository+"/"+match.Branch)
	}

	return branches
}

func TestIssueBranchServiceServesEverySprintKeyFromOneScan(t *testing.T) {
	scanner := keyedBranchScanner()
	service := NewIssueBranchService(scanner)
	issueKeys := []string{"CRMMS-6397", "CRMMS-77745", "CRMMS-77740", "CRMMS-77911", "CRMMS-77741", "CRMMS-77742", "CRMMS-77743"}

	branches, err := service.SprintBranchesAtRoot(context.Background(), "/workspace", issueKeys)
	if err != nil {
		t.Fatalf("sprint branches failed: %v", err)
	}

	if scanner.calls != 1 {
		t.Fatalf("expected one workspace scan for seven keys, got %d", scanner.calls)
	}
	if len(branches.Matches) != len(issueKeys) {
		t.Fatalf("expected %d keys in the response, got %d", len(issueKeys), len(branches.Matches))
	}
	for _, issueKey := range issueKeys {
		if _, found := branches.Matches[issueKey]; !found {
			t.Fatalf("every requested key must be present, %s is missing", issueKey)
		}
	}
	if branches.Matches["CRMMS-77745"] == nil {
		t.Fatal("a key without a match must carry an empty array, not null")
	}
}

func TestIssueBranchServiceCachesTheScanForTheCacheTTL(t *testing.T) {
	scanner := keyedBranchScanner()
	service := NewIssueBranchService(scanner)

	if _, err := service.IssueBranchesAtRoot(context.Background(), "/workspace", "CRMMS-6397"); err != nil {
		t.Fatalf("first read failed: %v", err)
	}
	if _, err := service.IssueBranchesAtRoot(context.Background(), "/workspace", "CRMMS-77745"); err != nil {
		t.Fatalf("second read failed: %v", err)
	}

	if scanner.calls != 1 {
		t.Fatalf("expected the cached scan to serve both reads, got %d scans", scanner.calls)
	}
}

func TestIssueBranchServiceScansAgainAfterTheCacheExpires(t *testing.T) {
	scanner := keyedBranchScanner()
	service := newIssueBranchService(scanner, time.Millisecond)

	if _, err := service.IssueBranchesAtRoot(context.Background(), "/workspace", "CRMMS-6397"); err != nil {
		t.Fatalf("first read failed: %v", err)
	}
	time.Sleep(10 * time.Millisecond)
	if _, err := service.IssueBranchesAtRoot(context.Background(), "/workspace", "CRMMS-6397"); err != nil {
		t.Fatalf("second read failed: %v", err)
	}

	if scanner.calls != 2 {
		t.Fatalf("expected an expired cache to scan again, got %d scans", scanner.calls)
	}
}

func TestIssueBranchServiceKeepsALongerKeySeparateFromItsPrefix(t *testing.T) {
	service := NewIssueBranchService(keyedBranchScanner())

	shortKeyBranches, err := service.IssueBranchesAtRoot(context.Background(), "/workspace", "CRMMS-6397")
	if err != nil {
		t.Fatalf("short key read failed: %v", err)
	}

	expectedShortKey := []string{"alpha/feature/CRMMS-6397-first", "beta/feature/crmms-6397-lower"}
	shortKeyMatches := matchedBranches(shortKeyBranches.Matches)
	if len(shortKeyMatches) != len(expectedShortKey) {
		t.Fatalf("expected %v, got %v", expectedShortKey, shortKeyMatches)
	}
	for index, expectedMatch := range expectedShortKey {
		if shortKeyMatches[index] != expectedMatch {
			t.Fatalf("expected %v, got %v", expectedShortKey, shortKeyMatches)
		}
	}

	longKeyBranches, err := service.IssueBranchesAtRoot(context.Background(), "/workspace", "CRMMS-63970")
	if err != nil {
		t.Fatalf("long key read failed: %v", err)
	}

	expectedLongKey := []string{"alpha/feature/CRMMS-63970-second"}
	longKeyMatches := matchedBranches(longKeyBranches.Matches)
	if len(longKeyMatches) != 1 || longKeyMatches[0] != expectedLongKey[0] {
		t.Fatalf("expected %v, got %v", expectedLongKey, longKeyMatches)
	}
}

func TestIssueBranchServiceCanonicalisesTheRequestedKeyCase(t *testing.T) {
	service := NewIssueBranchService(keyedBranchScanner())

	branches, err := service.IssueBranchesAtRoot(context.Background(), "/workspace", "crmms-6397")
	if err != nil {
		t.Fatalf("lower case key read failed: %v", err)
	}

	if branches.Key != "CRMMS-6397" {
		t.Fatalf("expected the canonical key CRMMS-6397, got %q", branches.Key)
	}
	if len(branches.Matches) != 2 {
		t.Fatalf("expected the lower case key to match, got %v", matchedBranches(branches.Matches))
	}
}

func TestIssueBranchServiceReportsLocalMatchesAndTheScope(t *testing.T) {
	service := NewIssueBranchService(keyedBranchScanner())

	branches, err := service.IssueBranchesAtRoot(context.Background(), "/workspace", "CRMMS-6397")
	if err != nil {
		t.Fatalf("read failed: %v", err)
	}

	currentMatch := branches.Matches[0]
	if currentMatch.Source != "local" || !currentMatch.IsCloned || !currentMatch.IsCurrent {
		t.Fatalf("expected a current local match, got %+v", currentMatch)
	}
	if currentMatch.BitbucketProject != "" {
		t.Fatalf("the Bitbucket project is unknown in this phase, got %q", currentMatch.BitbucketProject)
	}
	if currentMatch.LastCommit != nil || currentMatch.LastCommitRelative != nil || currentMatch.Ahead != nil || currentMatch.Behind != nil {
		t.Fatalf("commit data requires the commit tier and must stay null, got %+v", currentMatch)
	}

	scope := branches.Scope
	if scope.LocalRepositoryCount != 2 || scope.RemoteRepositoryCount != 0 || scope.CoveragePercent != 100 {
		t.Fatalf("unexpected repository counts in the scope: %+v", scope)
	}
	if scope.VerifiedRemotely || scope.RemoteRepositoriesChecked != 0 {
		t.Fatalf("this phase must not claim remote verification: %+v", scope)
	}
}

func TestIssueBranchServiceMarshalsAnUnmatchedKeyAsAnEmptyArray(t *testing.T) {
	service := NewIssueBranchService(keyedBranchScanner())

	branches, err := service.IssueBranchesAtRoot(context.Background(), "/workspace", "CRMMS-77745")
	if err != nil {
		t.Fatalf("read failed: %v", err)
	}

	encoded, err := json.Marshal(branches)
	if err != nil {
		t.Fatalf("cannot encode the response: %v", err)
	}
	if !strings.Contains(string(encoded), `"matches":[]`) {
		t.Fatalf("an unmatched key must encode as an empty array, got %s", encoded)
	}
}

func TestIssueBranchServiceRefusesAnInvalidKey(t *testing.T) {
	service := NewIssueBranchService(keyedBranchScanner())

	if _, err := service.IssueBranchesAtRoot(context.Background(), "/workspace", "not-a-key"); !errors.Is(err, sharedErrors.ErrBadRequest) {
		t.Fatalf("expected a bad request for an invalid key, got %v", err)
	}
	if _, err := service.SprintBranchesAtRoot(context.Background(), "/workspace", []string{"CRMMS-6397", "not-a-key"}); !errors.Is(err, sharedErrors.ErrBadRequest) {
		t.Fatalf("expected a bad request for an invalid sprint key, got %v", err)
	}
}

func TestIssueBranchServiceReportsAScannerFailure(t *testing.T) {
	scanner := keyedBranchScanner()
	scanner.err = errors.New("git is unavailable")
	service := NewIssueBranchService(scanner)

	_, err := service.IssueBranchesAtRoot(context.Background(), "/workspace", "CRMMS-6397")
	if err == nil {
		t.Fatal("expected the scan failure to be reported")
	}
	if !strings.Contains(err.Error(), "git is unavailable") {
		t.Fatalf("the scan failure must be wrapped with its cause, got %v", err)
	}
}

func TestIssueBranchServiceRequiresAScanner(t *testing.T) {
	service := NewIssueBranchService(nil)

	if _, err := service.IssueBranchesAtRoot(context.Background(), "/workspace", "CRMMS-6397"); !errors.Is(err, ErrServerReaderUnavailable) {
		t.Fatalf("expected an unavailable reader, got %v", err)
	}
}
