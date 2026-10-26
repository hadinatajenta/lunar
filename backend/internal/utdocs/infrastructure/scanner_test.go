package infrastructure

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"lunar/backend/internal/utdocs/domain"
)

func requireScannerGit(t *testing.T) {
	t.Helper()

	if _, err := exec.LookPath("git"); err != nil {
		t.Skipf("git executable is not available: %v", err)
	}
}

func runScannerGit(t *testing.T, repositoryPath string, args ...string) string {
	t.Helper()

	command := exec.Command("git", append([]string{"--no-optional-locks", "-C", repositoryPath}, args...)...)
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

func writeScannerFile(t *testing.T, rootPath string, relativePath string, content string) {
	t.Helper()

	fullPath := filepath.Join(rootPath, filepath.FromSlash(relativePath))
	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		t.Fatalf("cannot create the fixture directory for %s: %v", relativePath, err)
	}
	if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
		t.Fatalf("cannot write the fixture file %s: %v", relativePath, err)
	}
}

func newScannerRepository(t *testing.T) (string, string) {
	t.Helper()
	requireScannerGit(t)

	rootPath := t.TempDir()
	repositoryPath := filepath.Join(rootPath, "micro-payment")

	writeScannerFile(t, repositoryPath, "internal/repo/dao.go", "package repo\n\nfunc Find() {}\n")
	writeScannerFile(t, repositoryPath, "internal/repo/dao_test.go", "package repo\n\nfunc TestFind(t *testing.T) {}\n")
	writeScannerFile(t, repositoryPath, "web/util.ts", "export const util = 1\n")
	writeScannerFile(t, repositoryPath, "web/util.test.ts", "test('util', () => {})\n")
	writeScannerFile(t, repositoryPath, "tests/LegacyTest.php", "<?php\n")
	runScannerGit(t, repositoryPath, "init", "-q", "-b", "development")
	runScannerGit(t, repositoryPath, "add", "-A")
	runScannerGit(t, repositoryPath, "commit", "-q", "-m", "initial commit")
	runScannerGit(t, repositoryPath, "checkout", "-q", "-b", "feat/CRMMS-123")

	return rootPath, repositoryPath
}

func commitScannerChanges(t *testing.T, repositoryPath string) {
	t.Helper()

	runScannerGit(t, repositoryPath, "add", "-A")
	runScannerGit(t, repositoryPath, "commit", "-q", "-m", "branch changes")
}

func findCandidate(t *testing.T, candidates []domain.CandidateTestFile, candidatePath string) domain.CandidateTestFile {
	t.Helper()

	for _, candidate := range candidates {
		if candidate.Path == candidatePath {
			return candidate
		}
	}
	t.Fatalf("expected a candidate for %s, got %+v", candidatePath, candidates)
	return domain.CandidateTestFile{}
}

func TestScannerDiscoversTestFilesNextToChangedProductionFiles(t *testing.T) {
	_, repositoryPath := newScannerRepository(t)

	writeScannerFile(t, repositoryPath, "internal/repo/dao.go", "package repo\n\nfunc Find() int { return 1 }\n")
	writeScannerFile(t, repositoryPath, "web/util.ts", "export const util = 2\n")
	writeScannerFile(t, repositoryPath, "tests/LegacyTest.php", "<?php\n\n// changed\n")
	commitScannerChanges(t, repositoryPath)

	result, err := NewScanner().Scan(context.Background(), domain.ScanRequest{
		RepositoryPath: repositoryPath,
		Branch:         "feat/CRMMS-123",
		BaseRef:        "development",
	})
	if err != nil {
		t.Fatalf("expected the scan to succeed, got %v", err)
	}

	expectedChanged := []string{"internal/repo/dao.go", "tests/LegacyTest.php", "web/util.ts"}
	if strings.Join(result.ChangedFiles, ",") != strings.Join(expectedChanged, ",") {
		t.Fatalf("expected the changed files, got %+v", result.ChangedFiles)
	}

	if len(result.Candidates) != 3 {
		t.Fatalf("expected three candidates, got %+v", result.Candidates)
	}

	repositoryCandidate := findCandidate(t, result.Candidates, "internal/repo/dao_test.go")
	if repositoryCandidate.Language != domain.LanguageGo || repositoryCandidate.Package != "./internal/repo" {
		t.Fatalf("expected the go sibling candidate, got %+v", repositoryCandidate)
	}
	if !strings.Contains(repositoryCandidate.Content, "func TestFind") {
		t.Fatalf("expected the test file content, got %q", repositoryCandidate.Content)
	}

	nodeCandidate := findCandidate(t, result.Candidates, "web/util.test.ts")
	if nodeCandidate.Language != domain.LanguageNode {
		t.Fatalf("expected a node candidate, got %+v", nodeCandidate)
	}

	phpCandidate := findCandidate(t, result.Candidates, "tests/LegacyTest.php")
	if phpCandidate.Language != domain.LanguagePHP {
		t.Fatalf("expected a php candidate for the changed php test file, got %+v", phpCandidate)
	}

	if len(result.NonTestFiles) != 2 {
		t.Fatalf("expected the two changed production files, got %+v", result.NonTestFiles)
	}
	nonTestPaths := []string{result.NonTestFiles[0].Path, result.NonTestFiles[1].Path}
	if strings.Join(nonTestPaths, ",") != "internal/repo/dao.go,web/util.ts" {
		t.Fatalf("expected the changed non-test files, got %+v", nonTestPaths)
	}
	if !strings.Contains(result.NonTestFiles[0].Content, "func Find() int") {
		t.Fatalf("expected the production file content, got %q", result.NonTestFiles[0].Content)
	}
}

func TestScannerSkipsChangedFilesWithoutTestSiblings(t *testing.T) {
	_, repositoryPath := newScannerRepository(t)

	writeScannerFile(t, repositoryPath, "internal/other/plain.go", "package other\n")
	commitScannerChanges(t, repositoryPath)

	result, err := NewScanner().Scan(context.Background(), domain.ScanRequest{
		RepositoryPath: repositoryPath,
		Branch:         "feat/CRMMS-123",
		BaseRef:        "development",
	})
	if err != nil {
		t.Fatalf("expected the scan to succeed, got %v", err)
	}
	if len(result.Candidates) != 0 {
		t.Fatalf("expected no candidates, got %+v", result.Candidates)
	}
	if len(result.NonTestFiles) != 1 || result.NonTestFiles[0].Path != "internal/other/plain.go" {
		t.Fatalf("expected the changed production file, got %+v", result.NonTestFiles)
	}
}

func TestScannerRejectsInvalidRefs(t *testing.T) {
	_, repositoryPath := newScannerRepository(t)

	_, err := NewScanner().Scan(context.Background(), domain.ScanRequest{
		RepositoryPath: repositoryPath,
		Branch:         "-bad-branch",
		BaseRef:        "development",
	})
	if err == nil {
		t.Fatal("expected an invalid branch to be rejected")
	}

	_, err = NewScanner().Scan(context.Background(), domain.ScanRequest{
		RepositoryPath: repositoryPath,
		Branch:         "feat/CRMMS-123",
		BaseRef:        "bad..ref",
	})
	if err == nil {
		t.Fatal("expected an invalid base ref to be rejected")
	}
}

func TestScannerSkipsDeletedTestFiles(t *testing.T) {
	_, repositoryPath := newScannerRepository(t)

	runScannerGit(t, repositoryPath, "rm", "-q", "internal/repo/dao_test.go")
	commitScannerChanges(t, repositoryPath)

	result, err := NewScanner().Scan(context.Background(), domain.ScanRequest{
		RepositoryPath: repositoryPath,
		Branch:         "feat/CRMMS-123",
		BaseRef:        "development",
	})
	if err != nil {
		t.Fatalf("expected the scan to succeed, got %v", err)
	}
	if len(result.Candidates) != 0 {
		t.Fatalf("expected no candidates for a deleted test file, got %+v", result.Candidates)
	}
	for _, changedFile := range result.ChangedFiles {
		if changedFile != "internal/repo/dao_test.go" {
			t.Fatalf("expected the deletion in the changed files, got %+v", result.ChangedFiles)
		}
	}
}

func TestScannerExtractsGoTestFunctionsFromChangedTestFiles(t *testing.T) {
	_, repositoryPath := newScannerRepository(t)

	writeScannerFile(t, repositoryPath, "internal/repo/dao_test.go", "package repo\n\nfunc TestFind(t *testing.T) {}\n\nfunc TestSave(t *testing.T) {}\n\nfunc helper() {}\n")
	commitScannerChanges(t, repositoryPath)

	result, err := NewScanner().Scan(context.Background(), domain.ScanRequest{
		RepositoryPath: repositoryPath,
		Branch:         "feat/CRMMS-123",
		BaseRef:        "development",
	})
	if err != nil {
		t.Fatalf("expected the scan to succeed, got %v", err)
	}

	candidate := findCandidate(t, result.Candidates, "internal/repo/dao_test.go")
	if strings.Join(candidate.TestFunctions, ",") != "TestFind,TestSave" {
		t.Fatalf("expected the changed test functions, got %+v", candidate.TestFunctions)
	}
}

func TestScannerDoesNotExtractGoFunctionsFromNonGoFiles(t *testing.T) {
	_, repositoryPath := newScannerRepository(t)

	writeScannerFile(t, repositoryPath, "web/util.test.ts", "test('TestFind', () => {})\n")
	commitScannerChanges(t, repositoryPath)

	result, err := NewScanner().Scan(context.Background(), domain.ScanRequest{
		RepositoryPath: repositoryPath,
		Branch:         "feat/CRMMS-123",
		BaseRef:        "development",
	})
	if err != nil {
		t.Fatalf("expected the scan to succeed, got %v", err)
	}

	candidate := findCandidate(t, result.Candidates, "web/util.test.ts")
	if len(candidate.TestFunctions) != 0 {
		t.Fatalf("expected no go functions for a node test, got %+v", candidate.TestFunctions)
	}
}

func TestCurrentBranchReadsTheCheckout(t *testing.T) {
	_, repositoryPath := newScannerRepository(t)

	branch, err := CurrentBranch(context.Background(), repositoryPath)
	if err != nil {
		t.Fatalf("expected the branch to be read, got %v", err)
	}
	if branch != "feat/CRMMS-123" {
		t.Fatalf("expected the feature branch, got %q", branch)
	}
}

func findUntestedFunction(t *testing.T, functions []domain.UntestedFunction, name string) domain.UntestedFunction {
	t.Helper()

	for _, function := range functions {
		if function.Name == name {
			return function
		}
	}
	t.Fatalf("expected an untested function %s, got %+v", name, functions)
	return domain.UntestedFunction{}
}

func TestScannerExtractsUntestedFunctionsFromChangedProductionFiles(t *testing.T) {
	_, repositoryPath := newScannerRepository(t)

	writeScannerFile(t, repositoryPath, "internal/repo/dao.go", "package repo\n\nfunc Find() int { return 1 }\n\nfunc (r *Repo) Save() {}\n\nfunc helper() {}\n\nfunc TestFind() {}\n")
	writeScannerFile(t, repositoryPath, "internal/repo/dao_test.go", "package repo\n\nfunc TestFind(t *testing.T) {}\n\nfunc TestSaveData(t *testing.T) {}\n")
	commitScannerChanges(t, repositoryPath)

	result, err := NewScanner().Scan(context.Background(), domain.ScanRequest{
		RepositoryPath: repositoryPath,
		Branch:         "feat/CRMMS-123",
		BaseRef:        "development",
	})
	if err != nil {
		t.Fatalf("expected the scan to succeed, got %v", err)
	}

	if len(result.UntestedFunctions) != 3 {
		t.Fatalf("expected three untested functions, got %+v", result.UntestedFunctions)
	}
	find := findUntestedFunction(t, result.UntestedFunctions, "Find")
	if find.File != "internal/repo/dao.go" || find.Package != "./internal/repo" {
		t.Fatalf("expected the production file and package, got %+v", find)
	}
	method := findUntestedFunction(t, result.UntestedFunctions, "Save")
	if method.File != "internal/repo/dao.go" {
		t.Fatalf("expected the method file, got %+v", method)
	}
	findUntestedFunction(t, result.UntestedFunctions, "helper")
	for _, function := range result.UntestedFunctions {
		if function.Name == "TestFind" {
			t.Fatalf("expected the test function to be subtracted, got %+v", result.UntestedFunctions)
		}
	}
}

func TestScannerReportsNoUntestedFunctionsWithoutChangedProductionFiles(t *testing.T) {
	_, repositoryPath := newScannerRepository(t)

	writeScannerFile(t, repositoryPath, "internal/repo/dao_test.go", "package repo\n\nfunc TestFind(t *testing.T) {}\n\nfunc TestSaveData(t *testing.T) {}\n")
	commitScannerChanges(t, repositoryPath)

	result, err := NewScanner().Scan(context.Background(), domain.ScanRequest{
		RepositoryPath: repositoryPath,
		Branch:         "feat/CRMMS-123",
		BaseRef:        "development",
	})
	if err != nil {
		t.Fatalf("expected the scan to succeed, got %v", err)
	}
	if len(result.UntestedFunctions) != 0 {
		t.Fatalf("expected no untested functions, got %+v", result.UntestedFunctions)
	}
}
