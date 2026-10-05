package infrastructure

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestGitInspectorCheckoutBranchAllowsUntrackedOnlyChanges(t *testing.T) {
	rootPath, repositoryPath := newScratchWorkspace(t)
	inspector := NewGitInspector([]string{rootPath})

	untrackedPath := filepath.Join(repositoryPath, "untracked.txt")
	writeFixtureFile(t, untrackedPath, "keep me\n")

	checkedOut, err := inspector.CheckoutBranch(context.Background(), rootPath, "scratch", "main")
	if err != nil {
		t.Fatalf("expected an untracked-only tree to allow checkout, got %v", err)
	}
	if checkedOut != "main" {
		t.Fatalf("expected main, got %q", checkedOut)
	}

	currentBranch, err := readBranch(context.Background(), repositoryPath)
	if err != nil {
		t.Fatalf("cannot read current branch: %v", err)
	}
	if currentBranch != "main" {
		t.Fatalf("expected the repository to switch to main, got %q", currentBranch)
	}

	if _, err := os.Stat(untrackedPath); err != nil {
		t.Fatalf("expected the untracked file to survive the checkout: %v", err)
	}
}

func TestReadTrackedChangeCountIgnoresUntrackedEntries(t *testing.T) {
	rootPath, repositoryPath := newScratchWorkspace(t)
	ctx := context.Background()
	inspector := NewGitInspector([]string{rootPath})

	writeFixtureFile(t, filepath.Join(repositoryPath, "untracked.txt"), "keep me\n")

	dirtyCount, err := readDirtyCount(ctx, repositoryPath)
	if err != nil {
		t.Fatalf("cannot read dirty count: %v", err)
	}
	if dirtyCount != 1 {
		t.Fatalf("expected the dirty count to keep counting untracked entries, got %d", dirtyCount)
	}

	summaries, err := inspector.ListRepositories(ctx, rootPath)
	if err != nil {
		t.Fatalf("cannot list repositories: %v", err)
	}
	if len(summaries) != 1 || summaries[0].DirtyCount != 1 {
		t.Fatalf("expected the UI dirty column to keep reporting 1 for an untracked-only repository, got %+v", summaries)
	}

	trackedCount, err := readTrackedChangeCount(ctx, repositoryPath)
	if err != nil {
		t.Fatalf("cannot read tracked change count: %v", err)
	}
	if trackedCount != 0 {
		t.Fatalf("expected no tracked changes, got %d", trackedCount)
	}

	writeFixtureFile(t, filepath.Join(repositoryPath, "README.md"), "modified\n")

	dirtyCount, err = readDirtyCount(ctx, repositoryPath)
	if err != nil {
		t.Fatalf("cannot read dirty count: %v", err)
	}
	if dirtyCount != 2 {
		t.Fatalf("expected the dirty count to include tracked and untracked entries, got %d", dirtyCount)
	}

	trackedCount, err = readTrackedChangeCount(ctx, repositoryPath)
	if err != nil {
		t.Fatalf("cannot read tracked change count: %v", err)
	}
	if trackedCount != 1 {
		t.Fatalf("expected 1 tracked change, got %d", trackedCount)
	}
}

func TestGitInspectorCheckoutAfterStashWithUntrackedFileRemaining(t *testing.T) {
	rootPath, repositoryPath := newScratchWorkspace(t)
	inspector := NewGitInspector([]string{rootPath})
	ctx := context.Background()

	writeFixtureFile(t, filepath.Join(repositoryPath, "README.md"), "stashed\n")
	writeFixtureFile(t, filepath.Join(repositoryPath, "untracked.txt"), "keep me\n")

	stashCount, err := inspector.StashRepository(ctx, rootPath, "scratch")
	if err != nil {
		t.Fatalf("cannot stash changes: %v", err)
	}
	if stashCount != 1 {
		t.Fatalf("expected 1 stash entry, got %d", stashCount)
	}

	checkedOut, err := inspector.CheckoutBranch(ctx, rootPath, "scratch", "main")
	if err != nil {
		t.Fatalf("expected checkout to succeed once tracked changes are stashed, got %v", err)
	}
	if checkedOut != "main" {
		t.Fatalf("expected main, got %q", checkedOut)
	}
}
