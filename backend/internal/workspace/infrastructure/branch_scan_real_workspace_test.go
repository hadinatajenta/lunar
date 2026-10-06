package infrastructure

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	jiraDomain "lunar/backend/internal/jira/domain"
)

const realBranchScanWorkspaceRootEnvironment = "LUNAR_TEST_WORKSPACE_ROOT"

func realBranchScanWorkspaceRoot(t *testing.T) string {
	t.Helper()

	workspaceRoot := strings.TrimSpace(os.Getenv(realBranchScanWorkspaceRootEnvironment))
	if workspaceRoot == "" {
		t.Skipf("set %s to run the real workspace checks", realBranchScanWorkspaceRootEnvironment)
	}
	if _, err := os.Stat(workspaceRoot); err != nil {
		t.Skipf("the workspace root is not available: %v", err)
	}

	return workspaceRoot
}

func branchCarriesIssueKey(branchName string, issueKey string) bool {
	upperBranchName := strings.ToUpper(branchName)
	searchFrom := 0

	for {
		offset := strings.Index(upperBranchName[searchFrom:], issueKey)
		if offset < 0 {
			return false
		}

		end := searchFrom + offset + len(issueKey)
		if end >= len(upperBranchName) || upperBranchName[end] < '0' || upperBranchName[end] > '9' {
			return true
		}
		searchFrom = end
	}
}

func TestRealWorkspaceBranchScanFindsTheKeyedBranches(t *testing.T) {
	requireGit(t)

	workspaceRoot := realBranchScanWorkspaceRoot(t)
	startedAt := time.Now()
	branchesByRepository, err := NewGitInspector([]string{workspaceRoot}).ScanRepositoryBranches(context.Background(), workspaceRoot)
	scanDuration := time.Since(startedAt)
	if err != nil {
		t.Fatalf("the real workspace scan failed: %v", err)
	}

	totalBranchCount := 0
	keyedBranchCount := 0
	keyOccurrenceCount := 0
	matchesByKey := make(map[string][]string)

	for repositoryName, branches := range branchesByRepository {
		for _, branch := range branches {
			totalBranchCount++

			issueKeys := jiraDomain.IssueKeysIn(branch.Name)
			if len(issueKeys) == 0 {
				continue
			}

			keyedBranchCount++
			keyOccurrenceCount += len(issueKeys)
			for _, issueKey := range issueKeys {
				matchesByKey[issueKey] = append(matchesByKey[issueKey], repositoryName+"/"+branch.Name)
			}
		}
	}

	t.Logf("real workspace: repositories=%d branches=%d keyed_branches=%d key_occurrences=%d distinct_keys=%d duration=%s",
		len(branchesByRepository), totalBranchCount, keyedBranchCount, keyOccurrenceCount, len(matchesByKey), scanDuration)

	if len(branchesByRepository) == 0 {
		t.Fatalf("the real workspace %s contains no repository", workspaceRoot)
	}
	if keyedBranchCount == 0 {
		t.Fatal("the real workspace contains no branch carrying a Jira key")
	}
	if scanDuration > 10*time.Second {
		t.Fatalf("a single workspace scan took %s", scanDuration)
	}

	for issueKey, matches := range matchesByKey {
		for _, match := range matches {
			branchName := match[strings.Index(match, "/")+1:]
			if !branchCarriesIssueKey(branchName, issueKey) {
				t.Fatalf("branch %s must not carry key %s", match, issueKey)
			}
		}
	}

	t.Logf("CRMMS-6397 matches: %v", matchesByKey["CRMMS-6397"])
	t.Logf("CRMMS-63970 matches: %v", matchesByKey["CRMMS-63970"])
}
