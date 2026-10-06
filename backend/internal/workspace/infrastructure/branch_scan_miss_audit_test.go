package infrastructure

import (
	"context"
	"regexp"
	"sort"
	"strings"
	"testing"

	jiraDomain "lunar/backend/internal/jira/domain"
)

var adversarialLooseKeyPattern = regexp.MustCompile("(?i)[A-Z][A-Z0-9_]{0,9}-[0-9]{1,9}")

func TestRealWorkspaceAuditsEveryKeyLikeTokenForAMissedKey(t *testing.T) {
	requireGit(t)

	workspaceRoot := realBranchScanWorkspaceRoot(t)
	branchesByRepository, err := NewGitInspector([]string{workspaceRoot}).ScanRepositoryBranches(context.Background(), workspaceRoot)
	if err != nil {
		t.Fatalf("the real workspace scan failed: %v", err)
	}

	strictKeys := make(map[string]struct{})
	missedKeys := make(map[string][]string)
	looseTokenCount := 0
	branchCount := 0

	for repositoryName, branches := range branchesByRepository {
		for _, branch := range branches {
			branchCount++

			branchStrictKeys := make(map[string]struct{})
			for _, issueKey := range jiraDomain.IssueKeysIn(branch.Name) {
				branchStrictKeys[issueKey] = struct{}{}
				strictKeys[issueKey] = struct{}{}
			}

			for _, looseToken := range adversarialLooseKeyPattern.FindAllString(branch.Name, -1) {
				looseTokenCount++
				canonicalToken := strings.ToUpper(looseToken)
				if _, found := branchStrictKeys[canonicalToken]; found {
					continue
				}
				missedKeys[canonicalToken] = append(missedKeys[canonicalToken], repositoryName+"/"+branch.Name)
			}
		}
	}

	distinctStrictKeys := make([]string, 0, len(strictKeys))
	for issueKey := range strictKeys {
		distinctStrictKeys = append(distinctStrictKeys, issueKey)
	}
	sort.Strings(distinctStrictKeys)

	missedTokens := make([]string, 0, len(missedKeys))
	for token := range missedKeys {
		missedTokens = append(missedTokens, token)
	}
	sort.Strings(missedTokens)

	t.Logf("audit: branches=%d loose_tokens=%d distinct_strict_keys=%d distinct_missed_tokens=%d", branchCount, looseTokenCount, len(distinctStrictKeys), len(missedTokens))
	t.Logf("audit strict keys: %v", distinctStrictKeys)

	for _, token := range missedTokens {
		samples := missedKeys[token]
		sort.Strings(samples)
		if len(samples) > 3 {
			samples = samples[:3]
		}
		t.Logf("audit MISSED token %q on %v", token, samples)
	}

	if len(distinctStrictKeys) == 0 {
		t.Fatal("the real workspace yielded no key at all")
	}
}
func TestRealWorkspaceScanIssuesOnlyReadOnlyGitCommands(t *testing.T) {
	requireGit(t)

	workspaceRoot := realBranchScanWorkspaceRoot(t)
	logPath := recordGitInvocations(t)

	branchesByRepository, err := NewGitInspector([]string{workspaceRoot}).ScanRepositoryBranches(context.Background(), workspaceRoot)
	if err != nil {
		t.Fatalf("the real workspace scan failed: %v", err)
	}
	if len(branchesByRepository) == 0 {
		t.Fatal("the real workspace contains no repository")
	}

	invocations := readGitInvocations(t, logPath)
	if len(invocations) != len(branchesByRepository) {
		t.Fatalf("expected one git read per repository, got %d reads for %d repositories", len(invocations), len(branchesByRepository))
	}

	mutatingVerbs := []string{"checkout", "stash", "reset", "push", "pull", "fetch", "clean", "commit", "update-ref", "merge", "rebase", "branch -d", "branch -D", "branch -m", "worktree", "gc", "prune"}
	for _, invocation := range invocations {
		if !strings.Contains(invocation, "for-each-ref") {
			t.Fatalf("the real workspace scan must only list refs, got %q", invocation)
		}
		for _, verb := range mutatingVerbs {
			if strings.Contains(invocation, verb) {
				t.Fatalf("the real workspace scan must never run %q, got %q", verb, invocation)
			}
		}
	}

	t.Logf("real workspace scan: %d repositories, %d read-only git calls", len(branchesByRepository), len(invocations))
}
