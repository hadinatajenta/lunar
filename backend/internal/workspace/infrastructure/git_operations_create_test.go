package infrastructure

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"lunar/backend/internal/workspace/domain"
)

func TestGitInspectorCreateBranchChecksOutTheBase(t *testing.T) {
	rootPath, repositoryPath := newScratchWorkspace(t)
	inspector := NewGitInspector([]string{rootPath})
	ctx := context.Background()

	baseHead := strings.TrimSpace(runGitFixture(t, repositoryPath, "rev-parse", "main"))

	outcome, err := inspector.CreateBranch(ctx, rootPath, "scratch", "feature/CRMMS-77911-Enhancement-Manage-User", "main")
	if err != nil {
		t.Fatalf("cannot create the branch: %v", err)
	}
	if outcome.Branch != "feature/CRMMS-77911-Enhancement-Manage-User" {
		t.Fatalf("unexpected branch name %q", outcome.Branch)
	}
	if outcome.BaseBranch != "main" {
		t.Fatalf("unexpected base branch %q", outcome.BaseBranch)
	}
	if outcome.HeadCommit != baseHead {
		t.Fatalf("expected the head commit of main %q, got %q", baseHead, outcome.HeadCommit)
	}

	currentBranch, err := readBranch(ctx, repositoryPath)
	if err != nil {
		t.Fatalf("cannot read the current branch: %v", err)
	}
	if currentBranch != outcome.Branch {
		t.Fatalf("expected the created branch to be checked out, got %q", currentBranch)
	}

	if head := strings.TrimSpace(runGitFixture(t, repositoryPath, "rev-parse", "main")); head != baseHead {
		t.Fatalf("expected main to stay at %q, got %q", baseHead, head)
	}
}

func TestGitInspectorCreateBranchRefusesAnExistingBranch(t *testing.T) {
	rootPath, repositoryPath := newScratchWorkspace(t)
	inspector := NewGitInspector([]string{rootPath})
	ctx := context.Background()

	currentHead := strings.TrimSpace(runGitFixture(t, repositoryPath, "rev-parse", "HEAD"))

	_, err := inspector.CreateBranch(ctx, rootPath, "scratch", "feature/second", "main")
	if !errors.Is(err, domain.ErrBranchAlreadyExists) {
		t.Fatalf("expected a branch already exists error, got %v", err)
	}

	if head := strings.TrimSpace(runGitFixture(t, repositoryPath, "rev-parse", "HEAD")); head != currentHead {
		t.Fatalf("expected the repository head to stay at %q, got %q", currentHead, head)
	}
}

func TestGitInspectorCreateBranchRefusesAMissingBaseBranch(t *testing.T) {
	rootPath, repositoryPath := newScratchWorkspace(t)
	inspector := NewGitInspector([]string{rootPath})
	ctx := context.Background()

	_, err := inspector.CreateBranch(ctx, rootPath, "scratch", "feature/CRMMS-1-new", "release/nonexistent")
	if !errors.Is(err, domain.ErrBranchNotFound) {
		t.Fatalf("expected a missing base branch error, got %v", err)
	}

	branches := runGitFixture(t, repositoryPath, "branch", "--format=%(refname:short)")
	for _, branchName := range strings.Split(branches, "\n") {
		if strings.TrimSpace(branchName) == "feature/CRMMS-1-new" {
			t.Fatal("expected no branch to be created from a missing base branch")
		}
	}
}

func TestGitInspectorCreateBranchRefusesADirtyTreeWithTheCheckoutGuardMessage(t *testing.T) {
	rootPath, repositoryPath := newScratchWorkspace(t)
	inspector := NewGitInspector([]string{rootPath})
	ctx := context.Background()

	writeFixtureFile(t, filepath.Join(repositoryPath, "README.md"), "modified\n")

	_, createErr := inspector.CreateBranch(ctx, rootPath, "scratch", "feature/CRMMS-1-new", "main")
	if !errors.Is(createErr, domain.ErrWorkingTreeDirty) {
		t.Fatalf("expected a dirty working tree error, got %v", createErr)
	}

	_, checkoutErr := inspector.CheckoutBranch(ctx, rootPath, "scratch", "main")
	if !errors.Is(checkoutErr, domain.ErrWorkingTreeDirty) {
		t.Fatalf("expected the checkout guard to refuse the same tree, got %v", checkoutErr)
	}
	if createErr.Error() != checkoutErr.Error() {
		t.Fatalf("expected the checkout guard message %q, got %q", checkoutErr.Error(), createErr.Error())
	}

	branches := runGitFixture(t, repositoryPath, "branch", "--format=%(refname:short)")
	if strings.Contains(branches, "feature/CRMMS-1-new") {
		t.Fatalf("expected no branch to be created on a dirty tree, got %q", branches)
	}
}

func TestGitInspectorCreateBranchRefusesATraversalName(t *testing.T) {
	rootPath, repositoryPath := newScratchWorkspace(t)
	inspector := NewGitInspector([]string{rootPath})
	ctx := context.Background()

	_, err := inspector.CreateBranch(ctx, rootPath, "scratch", "main; rm -rf /", "main")
	if !errors.Is(err, domain.ErrInvalidBranchName) {
		t.Fatalf("expected an invalid branch name error, got %v", err)
	}

	_, err = inspector.CreateBranch(ctx, rootPath, "scratch", "feature/CRMMS-1-new", "main; rm -rf /")
	if !errors.Is(err, domain.ErrInvalidBranchName) {
		t.Fatalf("expected an invalid base branch name error, got %v", err)
	}

	branches := runGitFixture(t, repositoryPath, "branch", "--format=%(refname:short)")
	if strings.Contains(branches, "feature/CRMMS-1-new") {
		t.Fatalf("expected no branch to be created for an invalid name, got %q", branches)
	}
}

func TestGitInspectorCreateBranchRefusesARootOutsideTheAllowList(t *testing.T) {
	rootPath, repositoryPath := newScratchWorkspace(t)
	ctx := context.Background()

	outsideInspector := NewGitInspector([]string{filepath.Join(t.TempDir(), "elsewhere")})
	_, err := outsideInspector.CreateBranch(ctx, rootPath, "scratch", "feature/CRMMS-1-new", "main")
	if !errors.Is(err, domain.ErrRootOutsideScope) {
		t.Fatalf("expected a root outside the allow list error, got %v", err)
	}

	unlistedInspector := NewGitInspector(nil)
	_, err = unlistedInspector.CreateBranch(ctx, rootPath, "scratch", "feature/CRMMS-1-new", "main")
	if !errors.Is(err, domain.ErrWritesRequireAllowlist) {
		t.Fatalf("expected a writes require allow list error, got %v", err)
	}

	branches := runGitFixture(t, repositoryPath, "branch", "--format=%(refname:short)")
	if strings.Contains(branches, "feature/CRMMS-1-new") {
		t.Fatalf("expected no branch to be created outside the allow list, got %q", branches)
	}
}

func TestGitInspectorCreateBranchRefusesAMissingRepository(t *testing.T) {
	rootPath, _ := newScratchWorkspace(t)
	inspector := NewGitInspector([]string{rootPath})
	ctx := context.Background()

	_, err := inspector.CreateBranch(ctx, rootPath, "missing-repository", "feature/CRMMS-1-new", "main")
	if !errors.Is(err, domain.ErrRootNotFound) {
		t.Fatalf("expected a missing repository error, got %v", err)
	}

	_, err = inspector.CreateBranch(ctx, rootPath, "../outside", "feature/CRMMS-1-new", "main")
	if !errors.Is(err, domain.ErrInvalidRepoName) {
		t.Fatalf("expected an invalid repository name error, got %v", err)
	}
}

func TestGitInspectorReadRepositoryCoordinatesReadsTheOriginRemote(t *testing.T) {
	rootPath, repositoryPath := newScratchWorkspace(t)
	inspector := NewGitInspector([]string{rootPath})
	ctx := context.Background()

	coordinates, err := inspector.ReadRepositoryCoordinates(ctx, rootPath, "scratch")
	if err != nil {
		t.Fatalf("cannot read repository coordinates: %v", err)
	}
	if coordinates.IsKnown() {
		t.Fatalf("expected no coordinates without an origin remote, got %s", coordinates.String())
	}

	runGitFixture(t, repositoryPath, "remote", "add", "origin", "ssh://git@bitbucket.bri.co.id:7999/nme/mms-api.git")

	coordinates, err = inspector.ReadRepositoryCoordinates(ctx, rootPath, "scratch")
	if err != nil {
		t.Fatalf("cannot read repository coordinates: %v", err)
	}
	if coordinates.String() != "NME/mms-api" {
		t.Fatalf("expected NME/mms-api, got %q", coordinates.String())
	}

	_, err = inspector.ReadRepositoryCoordinates(ctx, rootPath, "..")
	if !errors.Is(err, domain.ErrInvalidRepoName) {
		t.Fatalf("expected an invalid repository name error, got %v", err)
	}
}

func TestGitInspectorReadBaseBranchCandidatesListsEveryBranchWithItsCommit(t *testing.T) {
	requireGit(t)
	ctx := context.Background()

	rootPath := t.TempDir()
	remotePath := filepath.Join(t.TempDir(), "remote.git")
	if err := os.MkdirAll(remotePath, 0o755); err != nil {
		t.Fatalf("cannot create the bare remote: %v", err)
	}
	runGitFixture(t, remotePath, "init", "-q", "--bare", "-b", "main", ".")

	sourcePath := filepath.Join(t.TempDir(), "source")
	if err := os.MkdirAll(sourcePath, 0o755); err != nil {
		t.Fatalf("cannot create the source repository: %v", err)
	}
	runGitFixture(t, sourcePath, "init", "-q", "-b", "main", ".")
	writeFixtureFile(t, filepath.Join(sourcePath, "README.md"), "initial\n")
	commitFixtureAt(t, sourcePath, "main commit", "2024-01-01T00:00:00+00:00")
	runGitFixture(t, sourcePath, "remote", "add", "origin", remotePath)
	runGitFixture(t, sourcePath, "push", "-q", "-u", "origin", "main")
	runGitFixture(t, sourcePath, "checkout", "-q", "-b", "feature/CRMMS-77911-Other-Team")
	writeFixtureFile(t, filepath.Join(sourcePath, "feature.txt"), "feature\n")
	commitFixtureAt(t, sourcePath, "feature commit", "2024-03-01T00:00:00+00:00")
	runGitFixture(t, sourcePath, "push", "-q", "origin", "feature/CRMMS-77911-Other-Team")

	repositoryPath := filepath.Join(rootPath, "scratch")
	if err := os.MkdirAll(repositoryPath, 0o755); err != nil {
		t.Fatalf("cannot create the cloned repository: %v", err)
	}
	runGitFixture(t, repositoryPath, "clone", "-q", remotePath, ".")
	runGitFixture(t, repositoryPath, "branch", "feature/CRMMS-77911-Other-Team", "origin/feature/CRMMS-77911-Other-Team")

	inspector := NewGitInspector([]string{rootPath})
	candidates, err := inspector.ReadBaseBranchCandidates(ctx, rootPath, "scratch")
	if err != nil {
		t.Fatalf("cannot read base branch candidates: %v", err)
	}
	if !candidates.IsCloned {
		t.Fatal("expected the cloned repository to be reported as cloned")
	}
	if candidates.RemoteDefault != "main" {
		t.Fatalf("expected the remote default main, got %q", candidates.RemoteDefault)
	}
	if len(candidates.Branches) != 2 {
		t.Fatalf("expected every branch, got %d: %+v", len(candidates.Branches), candidates.Branches)
	}

	for _, candidate := range candidates.Branches {
		if candidate.LastCommitRelative == "" {
			t.Fatalf("expected %q to carry its last commit", candidate.Name)
		}
		if candidate.CommitAt <= 0 {
			t.Fatalf("expected %q to carry a commit timestamp, got %d", candidate.Name, candidate.CommitAt)
		}
	}

	options := domain.OrderBaseBranchOptions(candidates.Branches, "", candidates.RemoteDefault)
	if len(options) != 2 {
		t.Fatalf("expected every branch to be offered, got %d", len(options))
	}
	if options[0].Name != "main" || options[0].Source != domain.BaseBranchSourceBitbucketDefault {
		t.Fatalf("expected main first as the remote default, got %+v", options[0])
	}
	if options[1].Name != "feature/CRMMS-77911-Other-Team" || options[1].Source != domain.BaseBranchSourceOther {
		t.Fatalf("expected the feature branch to be offered as other, got %+v", options[1])
	}
}

func TestGitInspectorReadBaseBranchCandidatesReportsAMissingRepository(t *testing.T) {
	rootPath := t.TempDir()
	inspector := NewGitInspector([]string{rootPath})
	ctx := context.Background()

	candidates, err := inspector.ReadBaseBranchCandidates(ctx, rootPath, "missing-repository")
	if err != nil {
		t.Fatalf("expected a missing repository to be reported, got %v", err)
	}
	if candidates.IsCloned {
		t.Fatal("expected a missing repository to be reported as not cloned")
	}
	if len(candidates.Branches) != 0 {
		t.Fatalf("expected no branches for a missing repository, got %+v", candidates.Branches)
	}

	if err := os.MkdirAll(filepath.Join(rootPath, "plain-folder"), 0o755); err != nil {
		t.Fatalf("cannot create the plain folder: %v", err)
	}

	candidates, err = inspector.ReadBaseBranchCandidates(ctx, rootPath, "plain-folder")
	if err != nil {
		t.Fatalf("expected a non git folder to be reported, got %v", err)
	}
	if candidates.IsCloned {
		t.Fatal("expected a non git folder to be reported as not cloned")
	}
}
