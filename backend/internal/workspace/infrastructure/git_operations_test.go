package infrastructure

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"lunar/backend/internal/workspace/domain"
)

func newScratchWorkspace(t *testing.T) (string, string) {
	t.Helper()
	requireGit(t)

	rootPath := t.TempDir()
	repositoryPath := filepath.Join(rootPath, "scratch")
	initFixtureRepository(t, repositoryPath)
	writeFixtureFile(t, filepath.Join(repositoryPath, "README.md"), "initial\n")
	commitFixture(t, repositoryPath, "initial commit")
	runGitFixture(t, repositoryPath, "checkout", "-q", "-b", "feature/second")

	return rootPath, repositoryPath
}

func commitFixtureAt(t *testing.T, repositoryPath string, message string, timestamp string) {
	t.Helper()

	runGitFixture(t, repositoryPath, "add", "-A")

	commandArguments := append([]string{"--no-optional-locks", "-C", repositoryPath}, "commit", "-q", "-m", message)
	command := exec.Command("git", commandArguments...)
	command.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=Lunar Test",
		"GIT_AUTHOR_EMAIL=lunar@test.local",
		"GIT_COMMITTER_NAME=Lunar Test",
		"GIT_COMMITTER_EMAIL=lunar@test.local",
		"GIT_CONFIG_NOSYSTEM=1",
		"GIT_AUTHOR_DATE="+timestamp,
		"GIT_COMMITTER_DATE="+timestamp,
	)

	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("git commit at %s failed: %v: %s", timestamp, err, output)
	}
}

func TestGitInspectorListBranchesMarksCurrentBranchFirstThenMostRecent(t *testing.T) {
	requireGit(t)

	rootPath := t.TempDir()
	repositoryPath := filepath.Join(rootPath, "scratch")
	initFixtureRepository(t, repositoryPath)
	writeFixtureFile(t, filepath.Join(repositoryPath, "README.md"), "initial\n")
	commitFixtureAt(t, repositoryPath, "oldest commit", "2024-01-01T00:00:00+00:00")
	runGitFixture(t, repositoryPath, "checkout", "-q", "-b", "release")
	writeFixtureFile(t, filepath.Join(repositoryPath, "release.txt"), "release\n")
	commitFixtureAt(t, repositoryPath, "newest commit", "2024-01-03T00:00:00+00:00")
	runGitFixture(t, repositoryPath, "checkout", "-q", "main")
	runGitFixture(t, repositoryPath, "checkout", "-q", "-b", "feature/second")
	writeFixtureFile(t, filepath.Join(repositoryPath, "feature.txt"), "feature\n")
	commitFixtureAt(t, repositoryPath, "middle commit", "2024-01-02T00:00:00+00:00")

	inspector := NewGitInspector([]string{rootPath})

	branches, err := inspector.ListBranches(context.Background(), rootPath, "scratch")
	if err != nil {
		t.Fatalf("cannot list branches: %v", err)
	}

	expectedNames := []string{"feature/second", "release", "main"}
	if len(branches) != len(expectedNames) {
		t.Fatalf("expected %d branches, got %d: %+v", len(expectedNames), len(branches), branches)
	}

	for index, expectedName := range expectedNames {
		if branches[index].Name != expectedName {
			t.Fatalf("expected branch %d to be %q, got %q", index, expectedName, branches[index].Name)
		}
		if branches[index].UpdatedRelative == "" {
			t.Fatalf("expected a relative update time for %q", expectedName)
		}
		if expectedName == "feature/second" && !branches[index].IsCurrent {
			t.Fatal("expected the checked out branch to be marked as current")
		}
		if expectedName != "feature/second" && branches[index].IsCurrent {
			t.Fatalf("expected %q to not be marked as current", expectedName)
		}
	}
}

func TestGitInspectorCheckoutBranchSwitchesAndRefusesDirtyTree(t *testing.T) {
	rootPath, repositoryPath := newScratchWorkspace(t)
	inspector := NewGitInspector([]string{rootPath})
	ctx := context.Background()

	checkedOut, err := inspector.CheckoutBranch(ctx, rootPath, "scratch", "main")
	if err != nil {
		t.Fatalf("cannot switch to main: %v", err)
	}
	if checkedOut != "main" {
		t.Fatalf("expected main, got %q", checkedOut)
	}

	writeFixtureFile(t, filepath.Join(repositoryPath, "README.md"), "dirty\n")

	if _, err := inspector.CheckoutBranch(ctx, rootPath, "scratch", "feature/second"); !errors.Is(err, domain.ErrWorkingTreeDirty) {
		t.Fatalf("expected ErrWorkingTreeDirty for a dirty tree, got %v", err)
	}

	currentBranch, err := readBranch(ctx, repositoryPath)
	if err != nil {
		t.Fatalf("cannot read current branch: %v", err)
	}
	if currentBranch != "main" {
		t.Fatalf("expected the dirty checkout to stay on main, got %q", currentBranch)
	}
}

func TestGitInspectorCheckoutBranchRejectsUnsafeBranchNames(t *testing.T) {
	rootPath, _ := newScratchWorkspace(t)
	inspector := NewGitInspector([]string{rootPath})

	cases := []string{"", "-x", "a b", "a;b", "a..b", "a\nb", "a$b", "../escape", "HEAD@{1}"}

	for _, testCase := range cases {
		t.Run(testCase, func(t *testing.T) {
			_, err := inspector.CheckoutBranch(context.Background(), rootPath, "scratch", testCase)
			if !errors.Is(err, domain.ErrInvalidBranchName) {
				t.Fatalf("expected ErrInvalidBranchName for %q, got %v", testCase, err)
			}
		})
	}
}

func TestGitInspectorCheckoutBranchRejectsNamesThatAreNotLocalBranches(t *testing.T) {
	rootPath, repositoryPath := newScratchWorkspace(t)
	inspector := NewGitInspector([]string{rootPath})
	ctx := context.Background()

	branchBefore, err := readBranch(ctx, repositoryPath)
	if err != nil {
		t.Fatalf("cannot read current branch: %v", err)
	}

	cases := []string{"HEAD", "refs/heads/main", "0000000000000000000000000000000000000000", "release-tag", "no-such-branch"}

	for _, testCase := range cases {
		t.Run(testCase, func(t *testing.T) {
			_, err := inspector.CheckoutBranch(ctx, rootPath, "scratch", testCase)
			if !errors.Is(err, domain.ErrBranchNotFound) {
				t.Fatalf("expected ErrBranchNotFound for %q, got %v", testCase, err)
			}
		})
	}

	branchAfter, err := readBranch(ctx, repositoryPath)
	if err != nil {
		t.Fatalf("cannot read current branch: %v", err)
	}
	if branchAfter != branchBefore {
		t.Fatalf("expected the branch to stay %q, got %q", branchBefore, branchAfter)
	}
}

func TestGitInspectorRefusesDestructiveOperationsWithoutAnAllowedRootList(t *testing.T) {
	rootPath, _ := newScratchWorkspace(t)
	inspector := NewGitInspector(nil)
	ctx := context.Background()

	operations := []struct {
		name string
		run  func() error
	}{
		{"checkout", func() error { _, err := inspector.CheckoutBranch(ctx, rootPath, "scratch", "main"); return err }},
		{"fetch", func() error { _, err := inspector.FetchRepository(ctx, rootPath, "scratch"); return err }},
		{"pull", func() error { _, err := inspector.PullRepository(ctx, rootPath, "scratch"); return err }},
		{"stash", func() error { _, err := inspector.StashRepository(ctx, rootPath, "scratch"); return err }},
		{"discard", func() error { _, err := inspector.DiscardRepository(ctx, rootPath, "scratch"); return err }},
	}

	for _, operation := range operations {
		t.Run(operation.name, func(t *testing.T) {
			if err := operation.run(); !errors.Is(err, domain.ErrWritesRequireAllowlist) {
				t.Fatalf("expected ErrWritesRequireAllowlist, got %v", err)
			}
		})
	}

	if _, err := inspector.ListBranches(ctx, rootPath, "scratch"); err != nil {
		t.Fatalf("expected reads to keep working without an allow list, got %v", err)
	}
}

func TestGitInspectorStashRepositoryCreatesEntryAndLeavesTreeClean(t *testing.T) {
	rootPath, repositoryPath := newScratchWorkspace(t)
	inspector := NewGitInspector([]string{rootPath})
	ctx := context.Background()

	writeFixtureFile(t, filepath.Join(repositoryPath, "README.md"), "stashed\n")

	stashCount, err := inspector.StashRepository(ctx, rootPath, "scratch")
	if err != nil {
		t.Fatalf("cannot stash changes: %v", err)
	}
	if stashCount != 1 {
		t.Fatalf("expected 1 stash entry, got %d", stashCount)
	}

	dirtyCount, err := readDirtyCount(ctx, repositoryPath)
	if err != nil {
		t.Fatalf("cannot read dirty count: %v", err)
	}
	if dirtyCount != 0 {
		t.Fatalf("expected a clean tree after stashing, got %d dirty paths", dirtyCount)
	}
}

func TestGitInspectorDiscardRepositoryRestoresTrackedFilesOnly(t *testing.T) {
	rootPath, repositoryPath := newScratchWorkspace(t)
	inspector := NewGitInspector([]string{rootPath})

	writeFixtureFile(t, filepath.Join(repositoryPath, "README.md"), "modified\n")
	untrackedPath := filepath.Join(repositoryPath, "untracked.txt")
	writeFixtureFile(t, untrackedPath, "keep me\n")

	restoredCount, err := inspector.DiscardRepository(context.Background(), rootPath, "scratch")
	if err != nil {
		t.Fatalf("cannot discard changes: %v", err)
	}
	if restoredCount != 1 {
		t.Fatalf("expected 1 restored tracked path, got %d", restoredCount)
	}

	contents, err := os.ReadFile(filepath.Join(repositoryPath, "README.md"))
	if err != nil {
		t.Fatalf("cannot read the restored file: %v", err)
	}
	if string(contents) != "initial\n" {
		t.Fatalf("expected the tracked file to be restored, got %q", contents)
	}

	if _, err := os.Stat(untrackedPath); err != nil {
		t.Fatalf("expected the untracked file to survive a discard: %v", err)
	}
}

func TestGitInspectorDiscardRepositoryLeavesStagedChangesAlone(t *testing.T) {
	rootPath, repositoryPath := newScratchWorkspace(t)
	inspector := NewGitInspector([]string{rootPath})

	writeFixtureFile(t, filepath.Join(repositoryPath, "README.md"), "staged\n")
	runGitFixture(t, repositoryPath, "add", "-A")

	restoredCount, err := inspector.DiscardRepository(context.Background(), rootPath, "scratch")
	if err != nil {
		t.Fatalf("cannot discard changes: %v", err)
	}
	if restoredCount != 0 {
		t.Fatalf("expected staged-only changes to restore nothing, got %d", restoredCount)
	}

	status := strings.TrimSpace(runGitFixture(t, repositoryPath, "status", "--porcelain=v1"))
	if status != "M  README.md" {
		t.Fatalf("expected the staged change to stay staged, got %q", status)
	}
}

func TestGitInspectorFetchAndPullAgainstLocalRemote(t *testing.T) {
	rootPath, repositoryPath := newScratchWorkspace(t)
	inspector := NewGitInspector([]string{rootPath})
	ctx := context.Background()

	remotePath := filepath.Join(t.TempDir(), "remote.git")
	if err := os.MkdirAll(remotePath, 0o755); err != nil {
		t.Fatalf("cannot create the bare remote directory: %v", err)
	}
	runGitFixture(t, remotePath, "init", "-q", "--bare", ".")
	runGitFixture(t, repositoryPath, "remote", "add", "origin", remotePath)
	runGitFixture(t, repositoryPath, "push", "-q", "-u", "origin", "feature/second")

	clonePath := filepath.Join(t.TempDir(), "clone")
	if err := os.MkdirAll(clonePath, 0o755); err != nil {
		t.Fatalf("cannot create the clone directory: %v", err)
	}
	runGitFixture(t, clonePath, "clone", "-q", "-b", "feature/second", remotePath, ".")
	writeFixtureFile(t, filepath.Join(clonePath, "pulled.txt"), "from the clone\n")
	commitFixture(t, clonePath, "remote commit")
	runGitFixture(t, clonePath, "push", "-q", "origin", "feature/second")

	status, err := inspector.FetchRepository(ctx, rootPath, "scratch")
	if err != nil {
		t.Fatalf("cannot fetch: %v", err)
	}
	if status.Ahead != 0 || status.Behind != 1 {
		t.Fatalf("expected ahead 0 and behind 1 after fetch, got %+v", status)
	}

	pullResult, err := inspector.PullRepository(ctx, rootPath, "scratch")
	if err != nil {
		t.Fatalf("cannot pull: %v", err)
	}
	if pullResult.Branch != "feature/second" {
		t.Fatalf("expected the current branch feature/second, got %q", pullResult.Branch)
	}
	if pullResult.UpdatedRelative == "" {
		t.Fatal("expected a relative update time after pulling")
	}

	if _, err := os.Stat(filepath.Join(repositoryPath, "pulled.txt")); err != nil {
		t.Fatalf("expected the pulled commit to land in the working tree: %v", err)
	}
}

func TestGitInspectorNetworkOperationsRequireConfiguredRemote(t *testing.T) {
	rootPath, _ := newScratchWorkspace(t)
	inspector := NewGitInspector([]string{rootPath})
	ctx := context.Background()

	if _, err := inspector.FetchRepository(ctx, rootPath, "scratch"); !errors.Is(err, domain.ErrGitOperationFailed) {
		t.Fatalf("expected ErrGitOperationFailed for a repository without a remote, got %v", err)
	}

	if _, err := inspector.PullRepository(ctx, rootPath, "scratch"); !errors.Is(err, domain.ErrGitOperationFailed) {
		t.Fatalf("expected ErrGitOperationFailed for a branch without an upstream, got %v", err)
	}
}

func TestRunGitWithTimeoutReportsTimeoutError(t *testing.T) {
	rootPath, repositoryPath := newScratchWorkspace(t)

	if _, err := runGitWithTimeout(context.Background(), repositoryPath, time.Nanosecond, "status"); !errors.Is(err, domain.ErrGitTimeout) {
		t.Fatalf("expected ErrGitTimeout, got %v", err)
	}

	if _, err := runGitWithTimeout(context.Background(), repositoryPath, domain.GitCommandTimeout, "status"); err != nil {
		t.Fatalf("expected inherited root path %q to be readable, got %v", rootPath, err)
	}
}
