package infrastructure

import (
	"context"
	"testing"
)

func TestDefaultBaseBranchPrefersALocalBranch(t *testing.T) {
	_, repositoryPath := newScannerRepository(t)

	branch, err := DefaultBaseBranch(context.Background(), repositoryPath)
	if err != nil {
		t.Fatalf("expected a base branch, got %v", err)
	}
	if branch != "development" {
		t.Fatalf("expected development, got %q", branch)
	}
}

func TestDefaultBaseBranchFallsBackToTheRemoteBranch(t *testing.T) {
	_, repositoryPath := newScannerRepository(t)
	runScannerGit(t, repositoryPath, "branch", "-D", "development")
	runScannerGit(t, repositoryPath, "update-ref", "refs/remotes/origin/development", "HEAD")

	branch, err := DefaultBaseBranch(context.Background(), repositoryPath)
	if err != nil {
		t.Fatalf("expected a base branch, got %v", err)
	}
	if branch != "origin/development" {
		t.Fatalf("expected origin/development, got %q", branch)
	}
}

func TestDefaultBaseBranchFallsBackToMain(t *testing.T) {
	_, repositoryPath := newScannerRepository(t)
	runScannerGit(t, repositoryPath, "branch", "-D", "development")
	runScannerGit(t, repositoryPath, "branch", "main")

	branch, err := DefaultBaseBranch(context.Background(), repositoryPath)
	if err != nil {
		t.Fatalf("expected a base branch, got %v", err)
	}
	if branch != "main" {
		t.Fatalf("expected main, got %q", branch)
	}
}

func TestDefaultBaseBranchReturnsAnErrorWhenNoBaseExists(t *testing.T) {
	_, repositoryPath := newScannerRepository(t)
	runScannerGit(t, repositoryPath, "branch", "-D", "development")

	if _, err := DefaultBaseBranch(context.Background(), repositoryPath); err == nil {
		t.Fatal("expected an error when no default base branch exists")
	}
}
