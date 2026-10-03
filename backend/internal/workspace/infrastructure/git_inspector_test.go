package infrastructure

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"lunar/backend/internal/workspace/domain"
)

func requireGit(t *testing.T) {
	t.Helper()

	if _, err := exec.LookPath("git"); err != nil {
		t.Skipf("git executable is not available: %v", err)
	}
}

func runGitFixture(t *testing.T, repositoryPath string, args ...string) string {
	t.Helper()

	commandArguments := append([]string{"--no-optional-locks", "-C", repositoryPath}, args...)
	command := exec.Command("git", commandArguments...)
	command.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=Lunar Test",
		"GIT_AUTHOR_EMAIL=lunar@test.local",
		"GIT_COMMITTER_NAME=Lunar Test",
		"GIT_COMMITTER_EMAIL=lunar@test.local",
		"GIT_CONFIG_NOSYSTEM=1",
	)

	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v: %s", args, err, output)
	}
	return string(output)
}

func initFixtureRepository(t *testing.T, repositoryPath string) {
	t.Helper()

	if err := os.MkdirAll(repositoryPath, 0o755); err != nil {
		t.Fatalf("create repository directory: %v", err)
	}
	runGitFixture(t, repositoryPath, "init", "-q", "-b", "main", ".")
}

func writeFixtureFile(t *testing.T, path string, content string) {
	t.Helper()

	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write fixture file %s: %v", path, err)
	}
}

func commitFixture(t *testing.T, repositoryPath string, message string) {
	t.Helper()

	runGitFixture(t, repositoryPath, "add", "-A")
	runGitFixture(t, repositoryPath, "commit", "-q", "-m", message)
}

func TestParsePorcelainStatusMapsEveryStatusCode(t *testing.T) {
	output := strings.Join([]string{
		"M  staged.txt",
		" M modified.txt",
		"MM both.txt",
		"A  added.txt",
		"D  deleted.txt",
		"R  before.txt -> after.txt",
		"AD staged-then-deleted.txt",
		"?? untracked.txt",
		"",
	}, "\n")

	entries := parsePorcelainStatus(output)

	expected := []porcelainEntry{
		{Status: domain.StatusModified, Path: "staged.txt"},
		{Status: domain.StatusModified, Path: "modified.txt"},
		{Status: domain.StatusModified, Path: "both.txt"},
		{Status: domain.StatusAdded, Path: "added.txt"},
		{Status: domain.StatusDeleted, Path: "deleted.txt"},
		{Status: domain.StatusRenamed, Path: "after.txt", OriginalPath: "before.txt"},
		{Status: domain.StatusAdded, Path: "staged-then-deleted.txt"},
		{Status: domain.StatusUntracked, Path: "untracked.txt"},
	}

	if len(entries) != len(expected) {
		t.Fatalf("expected %d entries, got %d: %+v", len(expected), len(entries), entries)
	}

	for index, want := range expected {
		if entries[index] != want {
			t.Fatalf("entry %d: expected %+v, got %+v", index, want, entries[index])
		}
	}
}

func TestParsePorcelainStatusHandlesRenameWithTwoPaths(t *testing.T) {
	entries := parsePorcelainStatus("R  internal/auth/token.go -> internal/session/token.go\n")

	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].Status != domain.StatusRenamed {
		t.Fatalf("expected renamed status, got %q", entries[0].Status)
	}
	if entries[0].Path != "internal/session/token.go" {
		t.Fatalf("expected destination path, got %q", entries[0].Path)
	}
	if entries[0].OriginalPath != "internal/auth/token.go" {
		t.Fatalf("expected original path, got %q", entries[0].OriginalPath)
	}
}

func TestParsePorcelainStatusUnquotesPathsAndKeepsArrowInsideQuotes(t *testing.T) {
	output := " M \"caf\\303\\251.txt\"\nR  \"before -> name.txt\" -> \"after -> name.txt\"\n"

	entries := parsePorcelainStatus(output)

	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	if entries[0].Path != "café.txt" {
		t.Fatalf("expected unquoted path, got %q", entries[0].Path)
	}
	if entries[1].Status != domain.StatusRenamed {
		t.Fatalf("expected renamed status, got %q", entries[1].Status)
	}
	if entries[1].Path != "after -> name.txt" {
		t.Fatalf("expected destination path, got %q", entries[1].Path)
	}
	if entries[1].OriginalPath != "before -> name.txt" {
		t.Fatalf("expected original path, got %q", entries[1].OriginalPath)
	}
}

func TestSplitNumstatPathResolvesRenameForms(t *testing.T) {
	cases := []struct {
		name             string
		input            string
		expectedPath     string
		expectedOriginal string
	}{
		{name: "plain", input: "internal/auth/refresh.go", expectedPath: "internal/auth/refresh.go"},
		{name: "full rename", input: "before.txt => after.txt", expectedPath: "after.txt", expectedOriginal: "before.txt"},
		{name: "brace rename", input: "internal/{auth => session}/token.go", expectedPath: "internal/session/token.go", expectedOriginal: "internal/auth/token.go"},
		{name: "quoted path", input: "\"caf\\303\\251.txt\"", expectedPath: "café.txt"},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			path, originalPath := splitNumstatPath(testCase.input)
			if path != testCase.expectedPath {
				t.Fatalf("expected path %q, got %q", testCase.expectedPath, path)
			}
			if originalPath != testCase.expectedOriginal {
				t.Fatalf("expected original path %q, got %q", testCase.expectedOriginal, originalPath)
			}
		})
	}
}

func TestParseNumstatReadsLineCounts(t *testing.T) {
	output := strings.Join([]string{
		"24\t6\tinternal/auth/refresh.go",
		"-\t-\tassets/logo.png",
		"0\t0\tbefore.txt => after.txt",
		"",
	}, "\n")

	deltas := parseNumstat(output)

	if deltas["internal/auth/refresh.go"] != (lineDelta{Added: 24, Deleted: 6}) {
		t.Fatalf("expected 24/6 for refresh.go, got %+v", deltas["internal/auth/refresh.go"])
	}
	if deltas["assets/logo.png"] != (lineDelta{Added: 0, Deleted: 0}) {
		t.Fatalf("expected 0/0 for binary file, got %+v", deltas["assets/logo.png"])
	}
	if deltas["after.txt"] != (lineDelta{Added: 0, Deleted: 0}) {
		t.Fatalf("expected 0/0 for rename destination, got %+v", deltas["after.txt"])
	}
	if deltas["before.txt"] != (lineDelta{Added: 0, Deleted: 0}) {
		t.Fatalf("expected 0/0 for rename source, got %+v", deltas["before.txt"])
	}
}

func TestParseCommitInfoReadsAllFields(t *testing.T) {
	output := "88b4ca3f0e1d2c3b4a5968778695a4b3c2d1e0f1\x1fHadinata Jenta\x1f37 minutes ago\x1ffix: guard nil session\n"

	commit, err := parseCommitInfo(output)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if commit.Hash != "88b4ca3" {
		t.Fatalf("expected abbreviated hash, got %q", commit.Hash)
	}
	if commit.Author != "Hadinata Jenta" {
		t.Fatalf("expected author, got %q", commit.Author)
	}
	if commit.RelativeTime != "37 minutes ago" {
		t.Fatalf("expected relative time, got %q", commit.RelativeTime)
	}
	if commit.Subject != "fix: guard nil session" {
		t.Fatalf("expected subject, got %q", commit.Subject)
	}
}

func TestParseCommitInfoRejectsUnexpectedOutput(t *testing.T) {
	if _, err := parseCommitInfo("hash-only"); err == nil {
		t.Fatal("expected an error for incomplete commit output")
	}
}

func TestHardenedGitEnvironmentOverridesInheritedValues(t *testing.T) {
	t.Setenv("GIT_TERMINAL_PROMPT", "1")
	t.Setenv("GIT_OPTIONAL_LOCKS", "1")
	t.Setenv("LC_ALL", "id_ID.UTF-8")

	environment := hardenedGitEnvironment()
	counts := map[string]int{}
	values := map[string]string{}

	for _, variable := range environment {
		key, value, found := strings.Cut(variable, "=")
		if !found {
			continue
		}
		counts[key]++
		values[key] = value
	}

	for key, expectedValue := range map[string]string{
		"GIT_TERMINAL_PROMPT": "0",
		"GIT_OPTIONAL_LOCKS":  "0",
		"GIT_CONFIG_NOSYSTEM": "1",
		"LC_ALL":              "C",
	} {
		if counts[key] != 1 {
			t.Fatalf("expected exactly one %s entry, got %d", key, counts[key])
		}
		if values[key] != expectedValue {
			t.Fatalf("expected %s=%s, got %s=%s", key, expectedValue, key, values[key])
		}
	}
}

func TestGitInspectorListsGitAndPlainDirectories(t *testing.T) {
	requireGit(t)

	root := t.TempDir()
	repositoryPath := filepath.Join(root, "everest")
	initFixtureRepository(t, repositoryPath)
	writeFixtureFile(t, filepath.Join(repositoryPath, "main.go"), "package main\n")
	commitFixture(t, repositoryPath, "feat: initial")
	writeFixtureFile(t, filepath.Join(repositoryPath, "main.go"), "package main\n\nfunc main() {}\n")

	if err := os.MkdirAll(filepath.Join(root, "way4"), 0o755); err != nil {
		t.Fatalf("create plain directory: %v", err)
	}

	summaries, err := NewGitInspector(nil).ListRepositories(context.Background(), root)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(summaries) != 2 {
		t.Fatalf("expected 2 summaries, got %d: %+v", len(summaries), summaries)
	}

	everest := summaries[0]
	if everest.Name != "everest" {
		t.Fatalf("expected alphabetical order, got %q first", everest.Name)
	}
	if !everest.IsGit {
		t.Fatal("expected everest to be detected as a git repository")
	}
	if everest.Branch != "main" {
		t.Fatalf("expected branch main, got %q", everest.Branch)
	}
	if everest.DirtyCount != 1 {
		t.Fatalf("expected 1 dirty file, got %d", everest.DirtyCount)
	}
	if everest.UpdatedRelative == "" {
		t.Fatal("expected a relative update time")
	}
	if everest.Error != "" {
		t.Fatalf("expected no error for everest, got %q", everest.Error)
	}

	way4 := summaries[1]
	if way4.Name != "way4" {
		t.Fatalf("expected way4 second, got %q", way4.Name)
	}
	if way4.IsGit {
		t.Fatal("expected way4 to be reported as a plain folder")
	}
	if !strings.Contains(way4.Error, "not a git repository") {
		t.Fatalf("expected a not-a-git-repository error, got %q", way4.Error)
	}
}

func TestGitInspectorReadsRepositoryDetail(t *testing.T) {
	requireGit(t)

	root := t.TempDir()
	repositoryPath := filepath.Join(root, "everest")
	initFixtureRepository(t, repositoryPath)
	writeFixtureFile(t, filepath.Join(repositoryPath, "main.go"), "package main\n\nfunc main() {}\n")
	writeFixtureFile(t, filepath.Join(repositoryPath, "notes.txt"), "alpha\nbeta\n")
	commitFixture(t, repositoryPath, "feat: initial commit")

	writeFixtureFile(t, filepath.Join(repositoryPath, "main.go"), "package main\n\nfunc main() {}\n\nfunc helper() {}\n")
	writeFixtureFile(t, filepath.Join(repositoryPath, "notes.txt"), "alpha\n")
	writeFixtureFile(t, filepath.Join(repositoryPath, "staged.txt"), "staged\n")
	runGitFixture(t, repositoryPath, "add", "staged.txt")
	writeFixtureFile(t, filepath.Join(repositoryPath, "untracked.txt"), "untracked\n")

	details, err := NewGitInspector(nil).ReadRepositories(context.Background(), root, []string{"everest"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(details) != 1 {
		t.Fatalf("expected 1 detail, got %d", len(details))
	}

	detail := details[0]
	if detail.Name != "everest" {
		t.Fatalf("expected name everest, got %q", detail.Name)
	}
	if detail.Branch != "main" {
		t.Fatalf("expected branch main, got %q", detail.Branch)
	}
	if detail.Error != "" {
		t.Fatalf("expected no error, got %q", detail.Error)
	}
	if detail.Ahead != 0 || detail.Behind != 0 {
		t.Fatalf("expected 0/0 ahead/behind, got %d/%d", detail.Ahead, detail.Behind)
	}
	if detail.FilesTruncated {
		t.Fatal("expected files_truncated to be false")
	}
	if detail.Commit.Hash != "7" && len(detail.Commit.Hash) != shortHashLength {
		t.Fatalf("expected a %d character hash, got %q", shortHashLength, detail.Commit.Hash)
	}
	if detail.Commit.Author != "Lunar Test" {
		t.Fatalf("expected author Lunar Test, got %q", detail.Commit.Author)
	}
	if detail.Commit.Subject != "feat: initial commit" {
		t.Fatalf("expected commit subject, got %q", detail.Commit.Subject)
	}
	if detail.Commit.RelativeTime == "" {
		t.Fatal("expected a relative commit time")
	}
	if detail.UpdatedRelative != detail.Commit.RelativeTime {
		t.Fatalf("expected updated_relative to match the commit time, got %q", detail.UpdatedRelative)
	}

	filesByPath := map[string]domain.ChangedFile{}
	for _, changedFile := range detail.Files {
		filesByPath[changedFile.Path] = changedFile
	}

	assertChangedFile(t, filesByPath, "main.go", domain.StatusModified, 2, 0)
	assertChangedFile(t, filesByPath, "notes.txt", domain.StatusModified, 0, 1)
	assertChangedFile(t, filesByPath, "staged.txt", domain.StatusAdded, 1, 0)
	assertChangedFile(t, filesByPath, "untracked.txt", domain.StatusUntracked, 0, 0)
}

func TestGitInspectorReadsRepositoryWithoutUpstream(t *testing.T) {
	requireGit(t)

	root := t.TempDir()
	repositoryPath := filepath.Join(root, "everest")
	initFixtureRepository(t, repositoryPath)
	writeFixtureFile(t, filepath.Join(repositoryPath, "main.go"), "package main\n")
	commitFixture(t, repositoryPath, "feat: initial commit")

	details, err := NewGitInspector(nil).ReadRepositories(context.Background(), root, []string{"everest"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if details[0].Ahead != 0 {
		t.Fatalf("expected ahead 0 without upstream, got %d", details[0].Ahead)
	}
	if details[0].Behind != 0 {
		t.Fatalf("expected behind 0 without upstream, got %d", details[0].Behind)
	}
	if details[0].Error != "" {
		t.Fatalf("expected no error without upstream, got %q", details[0].Error)
	}
}

func TestGitInspectorHandlesDetachedHead(t *testing.T) {
	requireGit(t)

	root := t.TempDir()
	repositoryPath := filepath.Join(root, "everest")
	initFixtureRepository(t, repositoryPath)
	writeFixtureFile(t, filepath.Join(repositoryPath, "main.go"), "package main\n")
	commitFixture(t, repositoryPath, "feat: initial commit")
	runGitFixture(t, repositoryPath, "checkout", "-q", "--detach", "HEAD")

	details, err := NewGitInspector(nil).ReadRepositories(context.Background(), root, []string{"everest"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if details[0].Branch != "HEAD" {
		t.Fatalf("expected detached branch HEAD, got %q", details[0].Branch)
	}
	if details[0].Error != "" {
		t.Fatalf("expected no error for detached HEAD, got %q", details[0].Error)
	}
}

func TestGitInspectorTruncatesChangedFiles(t *testing.T) {
	requireGit(t)

	root := t.TempDir()
	repositoryPath := filepath.Join(root, "big")
	initFixtureRepository(t, repositoryPath)
	writeFixtureFile(t, filepath.Join(repositoryPath, "seed.txt"), "seed\n")
	commitFixture(t, repositoryPath, "feat: initial commit")

	for index := 0; index < domain.MaxChangedFiles+5; index++ {
		writeFixtureFile(t, filepath.Join(repositoryPath, fmt.Sprintf("file-%03d.txt", index)), "content\n")
	}

	details, err := NewGitInspector(nil).ReadRepositories(context.Background(), root, []string{"big"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(details[0].Files) != domain.MaxChangedFiles {
		t.Fatalf("expected %d files, got %d", domain.MaxChangedFiles, len(details[0].Files))
	}
	if !details[0].FilesTruncated {
		t.Fatal("expected files_truncated to be true")
	}
}

func TestGitInspectorIsolatesRepositoryFailures(t *testing.T) {
	requireGit(t)

	root := t.TempDir()
	repositoryPath := filepath.Join(root, "everest")
	initFixtureRepository(t, repositoryPath)
	writeFixtureFile(t, filepath.Join(repositoryPath, "main.go"), "package main\n")
	commitFixture(t, repositoryPath, "feat: initial commit")
	if err := os.MkdirAll(filepath.Join(root, "way4"), 0o755); err != nil {
		t.Fatalf("create plain directory: %v", err)
	}

	inspector := NewGitInspector(nil)
	details, err := inspector.ReadRepositories(context.Background(), root, []string{"everest", "way4", "../outside", "missing-repo"})
	if err != nil {
		t.Fatalf("one failing repository must not fail the response: %v", err)
	}
	if len(details) != 4 {
		t.Fatalf("expected 4 details, got %d", len(details))
	}

	if details[0].Error != "" || details[0].Branch != "main" {
		t.Fatalf("expected everest to be readable, got %+v", details[0])
	}
	if !strings.Contains(details[1].Error, "not a git repository") {
		t.Fatalf("expected a not-a-git-repository error, got %q", details[1].Error)
	}
	if details[2].Error != messageRepositoryNameInvalid {
		t.Fatalf("expected invalid name error, got %q", details[2].Error)
	}
	if details[3].Error != messageRepositoryMissing {
		t.Fatalf("expected missing repository error, got %q", details[3].Error)
	}

	for _, detail := range details {
		if strings.ContainsAny(detail.Error, `/\`) {
			t.Fatalf("repository error must not leak a path: %q", detail.Error)
		}
		if detail.Files == nil {
			t.Fatalf("expected an empty files slice for %q, got nil", detail.Name)
		}
	}
}

func TestGitInspectorCachesRepositoryListing(t *testing.T) {
	requireGit(t)

	root := t.TempDir()
	repositoryPath := filepath.Join(root, "everest")
	initFixtureRepository(t, repositoryPath)
	writeFixtureFile(t, filepath.Join(repositoryPath, "main.go"), "package main\n")
	commitFixture(t, repositoryPath, "feat: initial commit")

	inspector := newGitInspector(nil, time.Second)
	ctx := context.Background()

	first, err := inspector.ListRepositories(ctx, root)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(first) != 1 {
		t.Fatalf("expected 1 repository, got %d", len(first))
	}

	if err := os.MkdirAll(filepath.Join(root, "way4"), 0o755); err != nil {
		t.Fatalf("create plain directory: %v", err)
	}

	cached, err := inspector.ListRepositories(ctx, root)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(cached) != 1 {
		t.Fatalf("expected the cached listing with 1 repository, got %d", len(cached))
	}

	first[0].Branch = "tampered"
	fromCache, err := inspector.ListRepositories(ctx, root)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if fromCache[0].Branch == "tampered" {
		t.Fatal("expected the cache to return a copy of the stored listing")
	}

	time.Sleep(1200 * time.Millisecond)

	fresh, err := inspector.ListRepositories(ctx, root)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(fresh) != 2 {
		t.Fatalf("expected the cache to expire and list 2 entries, got %d", len(fresh))
	}
}

func TestGitInspectorIgnoresUnusableCacheEntry(t *testing.T) {
	requireGit(t)

	root := t.TempDir()
	repositoryPath := filepath.Join(root, "everest")
	initFixtureRepository(t, repositoryPath)
	writeFixtureFile(t, filepath.Join(repositoryPath, "main.go"), "package main\n")
	commitFixture(t, repositoryPath, "feat: initial commit")

	resolvedRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatalf("resolve root: %v", err)
	}

	inspector := NewGitInspector(nil)
	inspector.cache.Set(resolvedRoot, "not a repository listing")

	summaries, err := inspector.ListRepositories(context.Background(), root)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(summaries) != 1 || !summaries[0].IsGit {
		t.Fatalf("expected a fresh listing, got %+v", summaries)
	}
}

func TestGitInspectorRejectsUnsafeRoots(t *testing.T) {
	ctx := context.Background()
	inspector := NewGitInspector(nil)

	if err := inspector.ValidateRoot(ctx, ""); !errors.Is(err, domain.ErrInvalidRoot) {
		t.Fatalf("expected ErrInvalidRoot for an empty path, got %v", err)
	}

	if err := inspector.ValidateRoot(ctx, "/"); !errors.Is(err, domain.ErrRootNotAllowed) {
		t.Fatalf("expected ErrRootNotAllowed for the filesystem root, got %v", err)
	}

	missingRoot := filepath.Join(t.TempDir(), "missing")
	if err := inspector.ValidateRoot(ctx, missingRoot); !errors.Is(err, domain.ErrRootNotFound) {
		t.Fatalf("expected ErrRootNotFound for a missing directory, got %v", err)
	}
	if _, err := inspector.ListRepositories(ctx, missingRoot); !errors.Is(err, domain.ErrRootNotFound) {
		t.Fatalf("expected ErrRootNotFound from ListRepositories, got %v", err)
	}

	filePath := filepath.Join(t.TempDir(), "file.txt")
	writeFixtureFile(t, filePath, "not a directory\n")
	if err := inspector.ValidateRoot(ctx, filePath); !errors.Is(err, domain.ErrRootNotDirectory) {
		t.Fatalf("expected ErrRootNotDirectory for a file, got %v", err)
	}
}

func TestGitInspectorRejectsMetacharacterRepositoryNameWithoutSpawn(t *testing.T) {
	requireGit(t)

	root := t.TempDir()
	repositoryName := "repo-$(touch injected)"
	repositoryPath := filepath.Join(root, repositoryName)
	initFixtureRepository(t, repositoryPath)
	writeFixtureFile(t, filepath.Join(repositoryPath, "main.go"), "package main\n")
	commitFixture(t, repositoryPath, "feat: initial commit")

	workingDirectory, err := os.Getwd()
	if err != nil {
		t.Fatalf("resolve working directory: %v", err)
	}
	injectedPath := filepath.Join(workingDirectory, "injected")
	if _, err := os.Stat(injectedPath); err == nil {
		t.Fatalf("precondition failed: %s already exists", injectedPath)
	}

	summaries, err := NewGitInspector(nil).ListRepositories(context.Background(), root)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(summaries) != 1 {
		t.Fatalf("expected 1 summary, got %d", len(summaries))
	}

	if summaries[0].IsGit {
		t.Fatalf("expected the metacharacter name to be rejected by the name whitelist, got %+v", summaries[0])
	}
	if summaries[0].Error == "" {
		t.Fatal("expected an explicit error for the rejected name")
	}
	if _, err := os.Stat(injectedPath); err == nil {
		t.Fatalf("repository path was interpreted by a shell: %s was created", injectedPath)
	}
}

func TestGitInspectorRejectsLeadingDashRepositoryName(t *testing.T) {
	requireGit(t)

	root := t.TempDir()
	repositoryName := "--help"
	repositoryPath := filepath.Join(root, repositoryName)
	initFixtureRepository(t, repositoryPath)
	writeFixtureFile(t, filepath.Join(repositoryPath, "main.go"), "package main\n")
	commitFixture(t, repositoryPath, "feat: initial commit")

	summaries, err := NewGitInspector(nil).ListRepositories(context.Background(), root)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(summaries) != 1 {
		t.Fatalf("expected 1 summary, got %d", len(summaries))
	}
	if summaries[0].IsGit {
		t.Fatalf("expected a leading dash name to be rejected, got %+v", summaries[0])
	}

	detail, err := NewGitInspector(nil).ReadRepositories(context.Background(), root, []string{repositoryName})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(detail) != 1 {
		t.Fatalf("expected 1 detail, got %d", len(detail))
	}
	if detail[0].Error == "" {
		t.Fatalf("expected the leading dash name to be rejected, got %+v", detail[0])
	}
}

func assertChangedFile(t *testing.T, filesByPath map[string]domain.ChangedFile, path string, status domain.ChangeStatus, added int, deleted int) {
	t.Helper()

	changedFile, found := filesByPath[path]
	if !found {
		t.Fatalf("expected a changed file for %q, got %+v", path, filesByPath)
	}
	if changedFile.Status != status {
		t.Fatalf("expected status %q for %q, got %q", status, path, changedFile.Status)
	}
	if changedFile.Added != added || changedFile.Deleted != deleted {
		t.Fatalf("expected %d/%d for %q, got %d/%d", added, deleted, path, changedFile.Added, changedFile.Deleted)
	}
}
