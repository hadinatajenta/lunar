package infrastructure

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	jiraDomain "lunar/backend/internal/jira/domain"
	"lunar/backend/internal/workspace/domain"
)

func newKeyedBranchWorkspace(t *testing.T) string {
	t.Helper()
	requireGit(t)

	rootPath := t.TempDir()
	createKeyedRepository(t, filepath.Join(rootPath, "alpha"), []string{"feature/CRMMS-6397-first", "feature/CRMMS-63970-second"})
	createKeyedRepository(t, filepath.Join(rootPath, "beta"), []string{"feature/crmms-6397-lower"})

	notesPath := filepath.Join(rootPath, "notes")
	if err := os.MkdirAll(notesPath, 0o755); err != nil {
		t.Fatalf("cannot create the notes directory: %v", err)
	}

	return rootPath
}

func createKeyedRepository(t *testing.T, repositoryPath string, branchNames []string) {
	t.Helper()

	initFixtureRepository(t, repositoryPath)
	writeFixtureFile(t, filepath.Join(repositoryPath, "README.md"), "initial\n")
	commitFixture(t, repositoryPath, "initial commit")

	for _, branchName := range branchNames {
		runGitFixture(t, repositoryPath, "checkout", "-q", "-b", branchName)
	}
}

func recordGitInvocations(t *testing.T) string {
	t.Helper()

	realGitPath, err := exec.LookPath("git")
	if err != nil {
		t.Skipf("git executable is not available: %v", err)
	}

	shimDirectory := t.TempDir()
	logPath := filepath.Join(shimDirectory, "git-invocations.log")
	script := "#!/bin/sh\nprintf '%s\n' \"$*\" >> \"" + logPath + "\"\nexec \"" + realGitPath + "\" \"$@\"\n"
	if err := os.WriteFile(filepath.Join(shimDirectory, "git"), []byte(script), 0o755); err != nil {
		t.Fatalf("cannot write the git recorder: %v", err)
	}
	t.Setenv("PATH", shimDirectory+string(os.PathListSeparator)+os.Getenv("PATH"))

	return logPath
}

func readGitInvocations(t *testing.T, logPath string) []string {
	t.Helper()

	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("cannot read the git recorder log: %v", err)
	}

	invocations := make([]string, 0)
	for _, line := range strings.Split(string(content), "\n") {
		if strings.TrimSpace(line) != "" {
			invocations = append(invocations, line)
		}
	}

	return invocations
}

func TestScanRepositoryBranchesReadsLocalBranchesOfEveryRepository(t *testing.T) {
	rootPath := newKeyedBranchWorkspace(t)

	branchesByRepository, err := NewGitInspector([]string{rootPath}).ScanRepositoryBranches(context.Background(), rootPath)
	if err != nil {
		t.Fatalf("scan failed: %v", err)
	}

	if len(branchesByRepository) != 2 {
		t.Fatalf("expected 2 repositories, got %d: %v", len(branchesByRepository), branchesByRepository)
	}
	if _, found := branchesByRepository["notes"]; found {
		t.Fatal("a directory that is not a repository must not be scanned")
	}

	alphaBranches := branchesByRepository["alpha"]
	expectedNames := []string{"feature/CRMMS-6397-first", "feature/CRMMS-63970-second", "main"}
	if len(alphaBranches) != len(expectedNames) {
		t.Fatalf("expected %v, got %+v", expectedNames, alphaBranches)
	}
	for index, expectedName := range expectedNames {
		if alphaBranches[index].Name != expectedName {
			t.Fatalf("expected %v, got %+v", expectedNames, alphaBranches)
		}
	}

	if !alphaBranches[1].IsCurrent {
		t.Fatalf("expected feature/CRMMS-63970-second to be the current branch, got %+v", alphaBranches)
	}
	if alphaBranches[2].IsCurrent {
		t.Fatalf("expected main not to be the current branch, got %+v", alphaBranches)
	}

	betaBranches := branchesByRepository["beta"]
	if len(betaBranches) != 2 || betaBranches[0].Name != "feature/crmms-6397-lower" {
		t.Fatalf("expected the keyed lower case branch, got %+v", betaBranches)
	}
}

func TestScanRepositoryBranchesKeepsALongerKeySeparateFromItsPrefix(t *testing.T) {
	rootPath := newKeyedBranchWorkspace(t)

	branchesByRepository, err := NewGitInspector([]string{rootPath}).ScanRepositoryBranches(context.Background(), rootPath)
	if err != nil {
		t.Fatalf("scan failed: %v", err)
	}

	matchesByKey := make(map[string][]string)
	for repositoryName, branches := range branchesByRepository {
		for _, branch := range branches {
			for _, issueKey := range jiraDomain.IssueKeysIn(branch.Name) {
				matchesByKey[issueKey] = append(matchesByKey[issueKey], repositoryName+"/"+branch.Name)
			}
		}
	}
	for issueKey := range matchesByKey {
		sort.Strings(matchesByKey[issueKey])
	}

	expectedShortKey := []string{"alpha/feature/CRMMS-6397-first", "beta/feature/crmms-6397-lower"}
	shortKeyMatches := matchesByKey["CRMMS-6397"]
	if len(shortKeyMatches) != len(expectedShortKey) {
		t.Fatalf("expected %v, got %v", expectedShortKey, shortKeyMatches)
	}
	for index, expectedMatch := range expectedShortKey {
		if shortKeyMatches[index] != expectedMatch {
			t.Fatalf("expected %v, got %v", expectedShortKey, shortKeyMatches)
		}
	}

	longKeyMatches := matchesByKey["CRMMS-63970"]
	if len(longKeyMatches) != 1 || longKeyMatches[0] != "alpha/feature/CRMMS-63970-second" {
		t.Fatalf("expected only the longer key branch, got %v", longKeyMatches)
	}
}

func TestScanRepositoryBranchesRunsOneGitCallPerRepository(t *testing.T) {
	rootPath := newKeyedBranchWorkspace(t)
	logPath := recordGitInvocations(t)

	if _, err := NewGitInspector([]string{rootPath}).ScanRepositoryBranches(context.Background(), rootPath); err != nil {
		t.Fatalf("scan failed: %v", err)
	}

	invocations := readGitInvocations(t, logPath)
	if len(invocations) != 2 {
		t.Fatalf("expected one git call per repository, got %d: %v", len(invocations), invocations)
	}
	for _, invocation := range invocations {
		if !strings.Contains(invocation, "for-each-ref") {
			t.Fatalf("the scan must only list refs, got %q", invocation)
		}
	}
}

func TestScanRepositoryBranchesRefusesRootsOutsideTheAllowedRoots(t *testing.T) {
	rootPath := newKeyedBranchWorkspace(t)

	inspector := NewGitInspector([]string{filepath.Join(rootPath, "elsewhere")})
	if _, err := inspector.ScanRepositoryBranches(context.Background(), rootPath); !errors.Is(err, domain.ErrRootOutsideScope) {
		t.Fatalf("expected a root outside the allowed roots to be refused, got %v", err)
	}

	if _, err := NewGitInspector([]string{rootPath}).ScanRepositoryBranches(context.Background(), ""); !errors.Is(err, domain.ErrInvalidRoot) {
		t.Fatalf("expected an empty root to be refused, got %v", err)
	}
}

func TestScanRepositoryBranchesReportsAGitFailureWithItsRepository(t *testing.T) {
	rootPath := t.TempDir()
	brokenPath := filepath.Join(rootPath, "broken")
	if err := os.MkdirAll(brokenPath, 0o755); err != nil {
		t.Fatalf("cannot create the broken repository directory: %v", err)
	}
	writeFixtureFile(t, filepath.Join(brokenPath, ".git"), "gitdir: /nonexistent-git-directory\n")

	_, err := NewGitInspector([]string{rootPath}).ScanRepositoryBranches(context.Background(), rootPath)
	if err == nil {
		t.Fatal("expected a broken repository to fail the scan")
	}
	if !strings.Contains(err.Error(), "broken") {
		t.Fatalf("the error must name the repository, got %v", err)
	}
}
