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

func adversarialGitInvocations(t *testing.T, logPath string) []string {
	t.Helper()

	if _, err := os.Stat(logPath); errors.Is(err, os.ErrNotExist) {
		return []string{}
	}

	return readGitInvocations(t, logPath)
}

func adversarialRefusedGitCalls(t *testing.T, logPath string) []string {
	t.Helper()

	invocations := adversarialGitInvocations(t, logPath)
	refused := make([]string, 0)
	for _, invocation := range invocations {
		if strings.Contains(invocation, "checkout") || strings.Contains(invocation, "stash") ||
			strings.Contains(invocation, "reset") || strings.Contains(invocation, "push") ||
			strings.Contains(invocation, "clean") || strings.Contains(invocation, "branch -D") {
			refused = append(refused, invocation)
		}
	}

	return refused
}

func TestAdversarialCreateBranchRefusesEveryCraftedRepositoryName(t *testing.T) {
	rootPath, repositoryPath := newScratchWorkspace(t)
	decoyRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(decoyRoot, "outside-repository"), 0o755); err != nil {
		t.Fatalf("cannot create the decoy root: %v", err)
	}
	runGitFixture(t, filepath.Join(decoyRoot, "outside-repository"), "init", "-q", "-b", "main", ".")

	inspector := NewGitInspector([]string{rootPath, decoyRoot})
	craftedNames := []string{
		"..",
		"../outside-repository",
		"scratch/../outside-repository",
		"scratch/../../etc",
		"/etc",
		"../../../../../../etc",
		"scratch" + string(os.PathSeparator) + ".." + string(os.PathSeparator) + "..",
		".",
		"",
		"   ",
		"scratch two",
		"~",
		string([]byte{'s', 'c', 'r', 0x00, 'a'}),
		"..%2f..%2fetc",
	}

	for _, craftedName := range craftedNames {
		outcome, err := inspector.CreateBranch(context.Background(), rootPath, craftedName, "feature/CRMMS-1-attack", "main")
		if err == nil {
			t.Fatalf("the crafted repository name %q must be refused, got %+v", craftedName, outcome)
		}
		if !errors.Is(err, domain.ErrInvalidRepoName) && !errors.Is(err, domain.ErrRootNotFound) &&
			!errors.Is(err, domain.ErrRepoOutsideRoot) && !errors.Is(err, domain.ErrNotGitRepository) {
			t.Fatalf("the crafted repository name %q must be refused as a path error, got %v", craftedName, err)
		}
	}

	for _, targetPath := range []string{repositoryPath, filepath.Join(decoyRoot, "outside-repository")} {
		for _, branchName := range adversarialBranchNames(t, targetPath) {
			if branchName == "feature/CRMMS-1-attack" {
				t.Fatalf("no repository may gain a branch from a crafted repository name, %s has %q", targetPath, branchName)
			}
		}
	}
}

func TestAdversarialCreateBranchRefusesASymlinkedRepositoryThatEscapesTheRoot(t *testing.T) {
	rootPath, _ := newScratchWorkspace(t)

	outsideRoot := t.TempDir()
	outsideRepositoryPath := filepath.Join(outsideRoot, "outside-repository")
	initFixtureRepository(t, outsideRepositoryPath)
	writeFixtureFile(t, filepath.Join(outsideRepositoryPath, "README.md"), "initial\n")
	commitFixture(t, outsideRepositoryPath, "initial commit")

	escapePath := filepath.Join(rootPath, "escape")
	if err := os.Symlink(outsideRepositoryPath, escapePath); err != nil {
		t.Fatalf("cannot create the escaping symlink: %v", err)
	}

	inspector := NewGitInspector([]string{rootPath, outsideRoot})
	_, err := inspector.CreateBranch(context.Background(), rootPath, "escape", "feature/CRMMS-1-via-symlink", "main")
	if !errors.Is(err, domain.ErrRepoOutsideRoot) {
		t.Fatalf("a symlinked repository that resolves outside the root must be refused, got %v", err)
	}

	for _, branchName := range adversarialBranchNames(t, outsideRepositoryPath) {
		if branchName == "feature/CRMMS-1-via-symlink" {
			t.Fatal("a symlinked repository outside the root must not gain a branch")
		}
	}

	if _, err := inspector.ReadRepositoryCoordinates(context.Background(), rootPath, "escape"); !errors.Is(err, domain.ErrRepoOutsideRoot) {
		t.Fatalf("the coordinate read must refuse the escaping symlink too, got %v", err)
	}
}

func TestAdversarialCreateBranchRunsNoGitAtAllWhenTheRootIsOutsideTheAllowList(t *testing.T) {
	rootPath, repositoryPath := newScratchWorkspace(t)
	logPath := recordGitInvocations(t)

	outsideInspector := NewGitInspector([]string{filepath.Join(t.TempDir(), "allowed-elsewhere")})
	_, err := outsideInspector.CreateBranch(context.Background(), rootPath, "scratch", "feature/CRMMS-1-outside", "main")
	if !errors.Is(err, domain.ErrRootOutsideScope) {
		t.Fatalf("a root outside the allow list must be refused, got %v", err)
	}

	unlistedInspector := NewGitInspector(nil)
	_, err = unlistedInspector.CreateBranch(context.Background(), rootPath, "scratch", "feature/CRMMS-1-unlisted", "main")
	if !errors.Is(err, domain.ErrWritesRequireAllowlist) {
		t.Fatalf("an unlisted inspector must refuse writes, got %v", err)
	}

	if invocations := adversarialGitInvocations(t, logPath); len(invocations) != 0 {
		t.Fatalf("a refused root must not run a single git command, got %v", invocations)
	}

	for _, branchName := range adversarialBranchNames(t, repositoryPath) {
		if branchName == "feature/CRMMS-1-outside" || branchName == "feature/CRMMS-1-unlisted" {
			t.Fatalf("a refused root must not gain a branch, got %q", branchName)
		}
	}
}

func TestAdversarialCreateBranchRefusesEveryCraftedBaseBranchName(t *testing.T) {
	rootPath, repositoryPath := newScratchWorkspace(t)
	inspector := NewGitInspector([]string{rootPath})

	craftedBaseBranches := []string{
		"--upload-pack=touch /tmp/lunar-qa",
		"-b",
		"-",
		"--force",
		"HEAD",
		"FETCH_HEAD",
		"origin/main",
		"refs/heads/main",
		"main..HEAD",
		"@{upstream}",
		"main\n--force",
		"main; rm -rf /",
		"main main",
		"main\u0000",
		"",
		"   ",
		"../../etc/passwd",
		"main.lock",
		".main",
	}

	for _, craftedBase := range craftedBaseBranches {
		_, err := inspector.CreateBranch(context.Background(), rootPath, "scratch", "feature/CRMMS-1-crafted-base", craftedBase)
		if err == nil {
			t.Fatalf("the crafted base branch %q must be refused", craftedBase)
		}
	}

	for _, branchName := range adversarialBranchNames(t, repositoryPath) {
		if branchName == "feature/CRMMS-1-crafted-base" {
			t.Fatal("no branch may be created from a crafted base branch")
		}
	}
}

func TestAdversarialScanAndBaseCandidatesRunOnlyReadOnlyGitCommands(t *testing.T) {
	rootPath := newKeyedBranchWorkspace(t)
	logPath := recordGitInvocations(t)
	inspector := NewGitInspector([]string{rootPath})
	ctx := context.Background()

	if _, err := inspector.ScanRepositoryBranches(ctx, rootPath); err != nil {
		t.Fatalf("the scan failed: %v", err)
	}
	if _, err := inspector.ReadBaseBranchCandidates(ctx, rootPath, "alpha"); err != nil {
		t.Fatalf("the base branch read failed: %v", err)
	}

	invocations := adversarialGitInvocations(t, logPath)
	if len(invocations) == 0 {
		t.Fatal("expected the read paths to run git commands")
	}
	for _, invocation := range invocations {
		if !strings.Contains(invocation, "for-each-ref") && !strings.Contains(invocation, "rev-parse") && !strings.Contains(invocation, "symbolic-ref") {
			t.Fatalf("the read paths must only run read-only git commands, got %q", invocation)
		}
	}
	if refused := adversarialRefusedGitCalls(t, logPath); len(refused) != 0 {
		t.Fatalf("the read paths must never run a mutating git command, got %v", refused)
	}
}

func TestAdversarialScanAcceptsAnAllowedRootReachedThroughASymlink(t *testing.T) {
	realRootPath := newKeyedBranchWorkspace(t)
	linkRootPath := filepath.Join(t.TempDir(), "linked-root")
	if err := os.Symlink(realRootPath, linkRootPath); err != nil {
		t.Fatalf("cannot create the root symlink: %v", err)
	}

	expectedRootPath, err := filepath.EvalSymlinks(realRootPath)
	if err != nil {
		t.Fatalf("cannot resolve the scratch root: %v", err)
	}

	scanRoot, err := domain.ResolveWorkspaceRoot(linkRootPath, []string{realRootPath})
	if err != nil {
		t.Fatalf("an allowed root reached through a symlink must resolve: %v", err)
	}
	if scanRoot != expectedRootPath {
		t.Fatalf("the root must resolve to its real path %q, got %q", expectedRootPath, scanRoot)
	}

	branchesByRepository, err := NewGitInspector([]string{realRootPath}).ScanRepositoryBranches(context.Background(), linkRootPath)
	if err != nil {
		t.Fatalf("the scan failed: %v", err)
	}
	if len(branchesByRepository) != 2 {
		t.Fatalf("expected the two scratch repositories, got %d", len(branchesByRepository))
	}
}

func adversarialBranchNames(t *testing.T, repositoryPath string) []string {
	t.Helper()

	output := runGitFixture(t, repositoryPath, "branch", "--format=%(refname:short)")
	names := make([]string, 0)
	for _, line := range strings.Split(output, "\n") {
		if trimmed := strings.TrimSpace(line); trimmed != "" {
			names = append(names, trimmed)
		}
	}

	return names
}
