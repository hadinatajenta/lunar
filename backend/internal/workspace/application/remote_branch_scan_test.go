package application

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	sharedErrors "lunar/backend/internal/shared/errors"
)

func TestScanIssueKeyKeepsOnlyTheBranchesThatCarryTheKey(t *testing.T) {
	scanner := &stubRemoteBranchScanner{branches: map[string][]string{
		"NME/mms-api": {"feature/CRMMS-63970-longer", "feature/CRMMS-6397-existing", "bugfix_crmms-6397-lower", "main"},
		"NME/atlas":   {"release/production"},
	}}

	result, err := NewRemoteCandidateScanner(scanner).ScanIssueKey(context.Background(), "usr-1", "CRMMS-6397", []string{"NME/mms-api", "NME/atlas"})
	if err != nil {
		t.Fatalf("cannot scan the candidate repositories: %v", err)
	}

	expectedBranches := []string{"bugfix_crmms-6397-lower", "feature/CRMMS-6397-existing"}
	if !reflect.DeepEqual(result.Matches["NME/mms-api"], expectedBranches) {
		t.Fatalf("expected %+v, got %+v", expectedBranches, result.Matches["NME/mms-api"])
	}
	if len(result.Matches) != 2 || result.Matches["NME/atlas"] == nil || len(result.Matches["NME/atlas"]) != 0 {
		t.Fatalf("expected every checked repository to be reported, got %+v", result.Matches)
	}
	if result.RepositoriesChecked != 2 {
		t.Fatalf("expected two repositories checked, got %d", result.RepositoriesChecked)
	}
	if len(result.RepositoriesNotChecked) != 0 || len(result.Failures) != 0 || result.TimedOut {
		t.Fatalf("unexpected scan outcome %+v", result)
	}
}

func TestScanIssueKeyLeavesTheKeyedMatchesUnpopulated(t *testing.T) {
	scanner := &stubRemoteBranchScanner{branches: map[string][]string{"NME/a": {"feature/CRMMS-6397-existing"}}}

	result, err := NewRemoteCandidateScanner(scanner).ScanIssueKey(context.Background(), "usr-1", "CRMMS-6397", []string{"NME/a"})
	if err != nil {
		t.Fatalf("cannot scan the candidate repository: %v", err)
	}
	if result.MatchesByRepositoryAndKey != nil {
		t.Fatalf("expected the single key scan to leave the keyed matches unpopulated, got %+v", result.MatchesByRepositoryAndKey)
	}
}

func TestScanIssueKeyChecksAtMostTheCandidateLimit(t *testing.T) {
	repositories := []string{"NME/a", "NME/b", "NME/c", "NME/d", "NME/e", "NME/f", "NME/g"}
	scanner := &stubRemoteBranchScanner{}

	result, err := NewRemoteCandidateScanner(scanner).ScanIssueKey(context.Background(), "usr-1", "CRMMS-6397", repositories)
	if err != nil {
		t.Fatalf("cannot scan the candidate repositories: %v", err)
	}

	if result.RepositoriesChecked != RemoteCandidateLimit {
		t.Fatalf("expected %d repositories checked, got %d", RemoteCandidateLimit, result.RepositoriesChecked)
	}
	if len(scanner.calledRepositories()) != RemoteCandidateLimit {
		t.Fatalf("expected %d remote calls, got %+v", RemoteCandidateLimit, scanner.calledRepositories())
	}
	if !reflect.DeepEqual(result.RepositoriesNotChecked, []string{"NME/f", "NME/g"}) {
		t.Fatalf("expected the repositories beyond the limit to be reported, got %+v", result.RepositoriesNotChecked)
	}
}

func TestScanIssueKeyIgnoresBlankAndDuplicateRepositories(t *testing.T) {
	scanner := &stubRemoteBranchScanner{}

	result, err := NewRemoteCandidateScanner(scanner).ScanIssueKey(context.Background(), "usr-1", "CRMMS-6397", []string{" NME/a ", "", "NME/a", "NME/b"})
	if err != nil {
		t.Fatalf("cannot scan the candidate repositories: %v", err)
	}

	if !reflect.DeepEqual(scanner.calledRepositories(), []string{"NME/a", "NME/b"}) {
		t.Fatalf("unexpected remote calls %+v", scanner.calledRepositories())
	}
	if result.RepositoriesChecked != 2 {
		t.Fatalf("expected two repositories checked, got %d", result.RepositoriesChecked)
	}
}

func TestScanIssueKeyReturnsAnEmptyResultWithoutCandidates(t *testing.T) {
	scanner := &stubRemoteBranchScanner{}

	result, err := NewRemoteCandidateScanner(scanner).ScanIssueKey(context.Background(), "usr-1", "CRMMS-6397", nil)
	if err != nil {
		t.Fatalf("cannot scan an empty candidate list: %v", err)
	}
	if result.RepositoriesChecked != 0 || len(result.RepositoriesNotChecked) != 0 || result.TimedOut {
		t.Fatalf("unexpected scan outcome %+v", result)
	}
	if len(scanner.calledRepositories()) != 0 {
		t.Fatalf("expected no remote calls, got %+v", scanner.calledRepositories())
	}
}

func TestScanIssueKeyRecordsARepositoryFailureWithoutFailingTheScan(t *testing.T) {
	scanner := &stubRemoteBranchScanner{
		branches: map[string][]string{"NME/b": {"feature/CRMMS-6397-existing"}},
		errs:     map[string]error{"NME/a": errRemoteUnreachable},
	}

	result, err := NewRemoteCandidateScanner(scanner).ScanIssueKey(context.Background(), "usr-1", "CRMMS-6397", []string{"NME/a", "NME/b"})
	if err != nil {
		t.Fatalf("one unreachable repository must not fail the scan, got %v", err)
	}

	if result.RepositoriesChecked != 2 {
		t.Fatalf("expected two repositories asked, got %d", result.RepositoriesChecked)
	}
	if len(result.Failures) != 1 || result.Failures[0].Repository != "NME/a" {
		t.Fatalf("expected the failed repository to be named, got %+v", result.Failures)
	}
	if !errors.Is(result.Failures[0].Err, errRemoteUnreachable) {
		t.Fatalf("expected the cause to be kept, got %v", result.Failures[0].Err)
	}
	if !strings.Contains(result.Failures[0].Err.Error(), "NME/a") {
		t.Fatalf("expected the repository in the failure context, got %q", result.Failures[0].Err.Error())
	}
	if !reflect.DeepEqual(result.Matches["NME/b"], []string{"feature/CRMMS-6397-existing"}) {
		t.Fatalf("expected the reachable repository to answer, got %+v", result.Matches)
	}
}

func TestScanIssueKeyReportsTheRepositoriesSkippedWhenTheDeadlinePasses(t *testing.T) {
	repositories := []string{"NME/a", "NME/b", "NME/c", "NME/d", "NME/e"}
	scanner := &stubRemoteBranchScanner{searchDelay: 500 * time.Millisecond}
	service := newRemoteCandidateScanner(scanner, 20*time.Millisecond)

	result, err := service.ScanIssueKey(context.Background(), "usr-1", "CRMMS-6397", repositories)
	if err != nil {
		t.Fatalf("a passed deadline must not fail the request, got %v", err)
	}

	if !result.TimedOut {
		t.Fatalf("expected the deadline to be reported, got %+v", result)
	}
	if result.RepositoriesChecked != 4 {
		t.Fatalf("expected four repositories asked before the deadline, got %d", result.RepositoriesChecked)
	}
	if len(result.Failures) != 4 || !errors.Is(result.Failures[0].Err, context.DeadlineExceeded) {
		t.Fatalf("expected the deadline to be reported per repository, got %+v", result.Failures)
	}
	if !reflect.DeepEqual(result.RepositoriesNotChecked, []string{"NME/e"}) {
		t.Fatalf("expected the unasked repository to be reported, got %+v", result.RepositoriesNotChecked)
	}
	if len(result.Matches) != 0 {
		t.Fatalf("expected no verified matches, got %+v", result.Matches)
	}
}

func TestScanIssueKeyRunsTheConfiguredBoundedConcurrency(t *testing.T) {
	arrivals := make(chan struct{}, remoteScanConcurrency)
	release := make(chan struct{})
	scanner := &stubRemoteBranchScanner{beforeReturn: func() {
		arrivals <- struct{}{}
		<-release
	}}
	repositories := []string{"NME/a", "NME/b", "NME/c", "NME/d", "NME/e"}

	scanResults := make(chan *RemoteCandidateScanResult, 1)
	scanErrors := make(chan error, 1)
	go func() {
		result, err := NewRemoteCandidateScanner(scanner).ScanIssueKey(context.Background(), "usr-1", "CRMMS-6397", repositories)
		scanResults <- result
		scanErrors <- err
	}()

	for index := 0; index < remoteScanConcurrency; index++ {
		select {
		case <-arrivals:
		case <-time.After(2 * time.Second):
			t.Fatalf("expected %d repositories in flight, only %d arrived", remoteScanConcurrency, index)
		}
	}
	close(release)

	if err := <-scanErrors; err != nil {
		t.Fatalf("cannot scan the candidate repositories: %v", err)
	}
	result := <-scanResults
	if result.RepositoriesChecked != len(repositories) {
		t.Fatalf("expected every repository to be checked, got %d", result.RepositoriesChecked)
	}
	if scanner.peakInFlight() != remoteScanConcurrency {
		t.Fatalf("expected %d repositories in flight, saw %d", remoteScanConcurrency, scanner.peakInFlight())
	}
}

func TestScanIssueKeyFailsWhenTheCallerContextIsCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := NewRemoteCandidateScanner(&stubRemoteBranchScanner{}).ScanIssueKey(ctx, "usr-1", "CRMMS-6397", []string{"NME/a"})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected the cancellation to be reported, got %v", err)
	}
}

func TestScanIssueKeyRejectsAnInvalidIssueKey(t *testing.T) {
	_, err := NewRemoteCandidateScanner(&stubRemoteBranchScanner{}).ScanIssueKey(context.Background(), "usr-1", "not-a-key", []string{"NME/a"})
	if !errors.Is(err, sharedErrors.ErrBadRequest) {
		t.Fatalf("expected a bad request error, got %v", err)
	}
}

func TestScanIssueKeyRequiresAScanner(t *testing.T) {
	_, err := NewRemoteCandidateScanner(nil).ScanIssueKey(context.Background(), "usr-1", "CRMMS-6397", []string{"NME/a"})
	if !errors.Is(err, ErrServerReaderUnavailable) {
		t.Fatalf("expected an unavailable reader error, got %v", err)
	}
}
