package application

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	sharedErrors "lunar/backend/internal/shared/errors"
)

func sortedNames(names []string) []string {
	sorted := make([]string, len(names))
	copy(sorted, names)
	sort.Strings(sorted)
	return sorted
}

func TestScanIssueKeysAttributesEveryBranchToItsKey(t *testing.T) {
	scanner := &stubRemoteBranchScanner{listBranches: map[string][]string{
		"NME/mms-api": {"feature/CRMMS-6397-existing", "bugfix_crmms-6397-lower", "feature/CRMMS-63970-longer", "feature/CRMMS-77911-enhancement", "main"},
		"NME/atlas":   {"release/production"},
	}}

	result, err := NewRemoteCandidateScanner(scanner).ScanIssueKeys(context.Background(), "usr-1", []string{"CRMMS-6397", "CRMMS-77911"}, []string{"NME/mms-api", "NME/atlas"})
	if err != nil {
		t.Fatalf("cannot scan the candidate repositories: %v", err)
	}

	expectedMatches := []string{"bugfix_crmms-6397-lower", "feature/CRMMS-6397-existing", "feature/CRMMS-77911-enhancement"}
	if !reflect.DeepEqual(result.Matches["NME/mms-api"], expectedMatches) {
		t.Fatalf("expected %+v, got %+v", expectedMatches, result.Matches["NME/mms-api"])
	}

	expectedByKey := map[string][]string{
		"CRMMS-6397":  {"bugfix_crmms-6397-lower", "feature/CRMMS-6397-existing"},
		"CRMMS-77911": {"feature/CRMMS-77911-enhancement"},
	}
	if !reflect.DeepEqual(result.MatchesByRepositoryAndKey["NME/mms-api"], expectedByKey) {
		t.Fatalf("expected %+v, got %+v", expectedByKey, result.MatchesByRepositoryAndKey["NME/mms-api"])
	}

	atlasByKey := result.MatchesByRepositoryAndKey["NME/atlas"]
	if len(atlasByKey) != 2 || len(atlasByKey["CRMMS-6397"]) != 0 || len(atlasByKey["CRMMS-77911"]) != 0 {
		t.Fatalf("expected every requested key for a checked repository, got %+v", atlasByKey)
	}
	if result.Matches["NME/atlas"] == nil || len(result.Matches["NME/atlas"]) != 0 {
		t.Fatalf("expected an empty match list for a checked repository, got %+v", result.Matches["NME/atlas"])
	}
	if result.RepositoriesChecked != 2 || len(result.RepositoriesNotChecked) != 0 || len(result.Failures) != 0 || result.TimedOut {
		t.Fatalf("unexpected scan outcome %+v", result)
	}
	if !reflect.DeepEqual(sortedNames(scanner.calledListRepositories()), []string{"NME/atlas", "NME/mms-api"}) {
		t.Fatalf("expected one listing per repository, got %+v", scanner.calledListRepositories())
	}
}

func TestScanIssueKeysAttributesOneBranchToEveryKeyItCarries(t *testing.T) {
	scanner := &stubRemoteBranchScanner{listBranches: map[string][]string{
		"NME/a": {"feature/CRMMS-6397-and-CRMMS-77911"},
	}}

	result, err := NewRemoteCandidateScanner(scanner).ScanIssueKeys(context.Background(), "usr-1", []string{"CRMMS-6397", "CRMMS-77911"}, []string{"NME/a"})
	if err != nil {
		t.Fatalf("cannot scan the candidate repositories: %v", err)
	}

	if len(result.Matches["NME/a"]) != 1 || result.Matches["NME/a"][0] != "feature/CRMMS-6397-and-CRMMS-77911" {
		t.Fatalf("expected the branch once in the union, got %+v", result.Matches["NME/a"])
	}
	for _, issueKey := range []string{"CRMMS-6397", "CRMMS-77911"} {
		got := result.MatchesByRepositoryAndKey["NME/a"][issueKey]
		if len(got) != 1 || got[0] != "feature/CRMMS-6397-and-CRMMS-77911" {
			t.Fatalf("expected the branch under %s, got %+v", issueKey, got)
		}
	}
}

func TestScanIssueKeysChecksAtMostTheCandidateLimit(t *testing.T) {
	repositories := []string{"NME/a", "NME/b", "NME/c", "NME/d", "NME/e", "NME/f", "NME/g"}
	scanner := &stubRemoteBranchScanner{}

	result, err := NewRemoteCandidateScanner(scanner).ScanIssueKeys(context.Background(), "usr-1", []string{"CRMMS-6397"}, repositories)
	if err != nil {
		t.Fatalf("cannot scan the candidate repositories: %v", err)
	}

	if result.RepositoriesChecked != RemoteCandidateLimit {
		t.Fatalf("expected %d repositories checked, got %d", RemoteCandidateLimit, result.RepositoriesChecked)
	}
	if len(scanner.calledListRepositories()) != RemoteCandidateLimit {
		t.Fatalf("expected %d remote calls, got %+v", RemoteCandidateLimit, scanner.calledListRepositories())
	}
	if len(result.MatchesByRepositoryAndKey) != RemoteCandidateLimit {
		t.Fatalf("expected %d keyed repositories, got %+v", RemoteCandidateLimit, result.MatchesByRepositoryAndKey)
	}
	if !reflect.DeepEqual(result.RepositoriesNotChecked, []string{"NME/f", "NME/g"}) {
		t.Fatalf("expected the repositories beyond the limit to be reported, got %+v", result.RepositoriesNotChecked)
	}
}

func TestScanIssueKeysFailsOnAnyInvalidKey(t *testing.T) {
	scanner := &stubRemoteBranchScanner{listBranches: map[string][]string{"NME/a": {"feature/CRMMS-6397-existing"}}}

	_, err := NewRemoteCandidateScanner(scanner).ScanIssueKeys(context.Background(), "usr-1", []string{"CRMMS-6397", "not-a-key"}, []string{"NME/a"})
	if !errors.Is(err, sharedErrors.ErrBadRequest) {
		t.Fatalf("expected a bad request error, got %v", err)
	}
	if len(scanner.calledListRepositories()) != 0 {
		t.Fatalf("expected no remote call for an invalid key, got %+v", scanner.calledListRepositories())
	}
}

func TestScanIssueKeysDeduplicatesTheRequestedKeys(t *testing.T) {
	scanner := &stubRemoteBranchScanner{listBranches: map[string][]string{"NME/a": {"feature/CRMMS-6397-existing"}}}

	result, err := NewRemoteCandidateScanner(scanner).ScanIssueKeys(context.Background(), "usr-1", []string{"CRMMS-6397", "crmms-6397", " CRMMS-6397 "}, []string{"NME/a"})
	if err != nil {
		t.Fatalf("cannot scan the candidate repositories: %v", err)
	}

	if len(result.MatchesByRepositoryAndKey["NME/a"]) != 1 {
		t.Fatalf("expected one canonical key, got %+v", result.MatchesByRepositoryAndKey["NME/a"])
	}
	got := result.MatchesByRepositoryAndKey["NME/a"]["CRMMS-6397"]
	if len(got) != 1 || got[0] != "feature/CRMMS-6397-existing" {
		t.Fatalf("unexpected branches for the canonical key %+v", got)
	}
}

func TestScanIssueKeysRejectsMoreKeysThanTheBound(t *testing.T) {
	issueKeys := make([]string, 0, RemoteKeyLimit+1)
	for index := 0; index <= RemoteKeyLimit; index++ {
		issueKeys = append(issueKeys, fmt.Sprintf("CRMMS-%d", index+1))
	}
	scanner := &stubRemoteBranchScanner{}

	_, err := NewRemoteCandidateScanner(scanner).ScanIssueKeys(context.Background(), "usr-1", issueKeys, []string{"NME/a"})
	if !errors.Is(err, sharedErrors.ErrBadRequest) {
		t.Fatalf("expected a bad request error, got %v", err)
	}
	if !strings.Contains(err.Error(), fmt.Sprintf("%d", RemoteKeyLimit)) {
		t.Fatalf("expected the bound to be named, got %q", err.Error())
	}
	if len(scanner.calledListRepositories()) != 0 {
		t.Fatalf("expected no remote call beyond the bound, got %+v", scanner.calledListRepositories())
	}
}

func TestScanIssueKeysReportsTheRepositoriesSkippedWhenTheDeadlinePasses(t *testing.T) {
	repositories := []string{"NME/a", "NME/b", "NME/c", "NME/d", "NME/e"}
	scanner := &stubRemoteBranchScanner{listDelay: 500 * time.Millisecond}
	service := newRemoteCandidateScanner(scanner, 20*time.Millisecond)

	result, err := service.ScanIssueKeys(context.Background(), "usr-1", []string{"CRMMS-6397"}, repositories)
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
	if len(result.MatchesByRepositoryAndKey) != 0 {
		t.Fatalf("expected no keyed matches, got %+v", result.MatchesByRepositoryAndKey)
	}
}

func TestScanIssueKeysRecordsARepositoryFailureWithoutFailingTheScan(t *testing.T) {
	scanner := &stubRemoteBranchScanner{
		listBranches: map[string][]string{"NME/b": {"feature/CRMMS-6397-existing"}},
		listErrs:     map[string]error{"NME/a": errRemoteUnreachable},
	}

	result, err := NewRemoteCandidateScanner(scanner).ScanIssueKeys(context.Background(), "usr-1", []string{"CRMMS-6397"}, []string{"NME/a", "NME/b"})
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
	if len(result.MatchesByRepositoryAndKey) != 1 {
		t.Fatalf("expected the failed repository to stay out of the keyed matches, got %+v", result.MatchesByRepositoryAndKey)
	}
	got := result.MatchesByRepositoryAndKey["NME/b"]["CRMMS-6397"]
	if len(got) != 1 || got[0] != "feature/CRMMS-6397-existing" {
		t.Fatalf("expected the reachable repository to answer, got %+v", got)
	}
}

func TestScanIssueKeysReturnsAnEmptyResultWithoutKeys(t *testing.T) {
	scanner := &stubRemoteBranchScanner{listBranches: map[string][]string{"NME/a": {"feature/CRMMS-6397-existing"}}}

	result, err := NewRemoteCandidateScanner(scanner).ScanIssueKeys(context.Background(), "usr-1", nil, []string{"NME/a", "NME/b"})
	if err != nil {
		t.Fatalf("an empty key list must not fail the request, got %v", err)
	}

	if result.RepositoriesChecked != 0 || result.TimedOut {
		t.Fatalf("unexpected scan outcome %+v", result)
	}
	if !reflect.DeepEqual(result.RepositoriesNotChecked, []string{"NME/a", "NME/b"}) {
		t.Fatalf("expected every repository to be reported as not checked, got %+v", result.RepositoriesNotChecked)
	}
	if len(result.Matches) != 0 || len(scanner.calledListRepositories()) != 0 {
		t.Fatalf("expected no remote call and no matches, got %+v and %+v", result.Matches, scanner.calledListRepositories())
	}
}

func TestScanIssueKeysRunsTheConfiguredBoundedConcurrency(t *testing.T) {
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
		result, err := NewRemoteCandidateScanner(scanner).ScanIssueKeys(context.Background(), "usr-1", []string{"CRMMS-6397"}, repositories)
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
	if result := <-scanResults; result.RepositoriesChecked != len(repositories) {
		t.Fatalf("expected every repository to be checked, got %d", result.RepositoriesChecked)
	}
	if scanner.peakInFlight() != remoteScanConcurrency {
		t.Fatalf("expected %d repositories in flight, saw %d", remoteScanConcurrency, scanner.peakInFlight())
	}
}

func TestScanIssueKeysFailsWhenTheCallerContextIsCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := NewRemoteCandidateScanner(&stubRemoteBranchScanner{}).ScanIssueKeys(ctx, "usr-1", []string{"CRMMS-6397"}, []string{"NME/a"})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected the cancellation to be reported, got %v", err)
	}
}

func TestScanIssueKeysRequiresAScanner(t *testing.T) {
	_, err := NewRemoteCandidateScanner(nil).ScanIssueKeys(context.Background(), "usr-1", []string{"CRMMS-6397"}, []string{"NME/a"})
	if !errors.Is(err, ErrServerReaderUnavailable) {
		t.Fatalf("expected an unavailable reader error, got %v", err)
	}
}
