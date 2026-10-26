package infrastructure

import (
	"context"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"lunar/backend/internal/utdocs/domain"
	workspacedomain "lunar/backend/internal/workspace/domain"
)

const maxScannedFileBytes = 256 * 1024

var goTestFunctionPattern = regexp.MustCompile(`(?m)^func (Test[A-Za-z0-9_]+)\(`)

var goFunctionNamePattern = regexp.MustCompile(`(?m)^func (?:\([^)]*\) )?([A-Za-z0-9_]+)\(`)

var nodeFileExtensions = map[string]bool{
	".js":  true,
	".jsx": true,
	".ts":  true,
	".tsx": true,
}

type Scanner struct{}

func NewScanner() *Scanner {
	return &Scanner{}
}

func (s *Scanner) Scan(ctx context.Context, request domain.ScanRequest) (*domain.ScanResult, error) {
	repositoryPath := strings.TrimSpace(request.RepositoryPath)
	if repositoryPath == "" {
		return nil, fmt.Errorf("the repository path is required to scan the branch")
	}

	branch, err := workspacedomain.ValidateBranchName(strings.TrimSpace(request.Branch))
	if err != nil {
		return nil, fmt.Errorf("validate the branch: %w", err)
	}
	baseRef, err := workspacedomain.ValidateBranchName(strings.TrimSpace(request.BaseRef))
	if err != nil {
		return nil, fmt.Errorf("validate the base ref: %w", err)
	}

	output, err := runUTDocsGit(ctx, repositoryPath, "diff", "--name-only", "--no-color", baseRef+".."+branch)
	if err != nil {
		return nil, fmt.Errorf("list the changed files: %w", err)
	}

	changedFiles := splitChangedFiles(output)
	result := &domain.ScanResult{
		ChangedFiles: changedFiles,
		Candidates:   make([]domain.CandidateTestFile, 0),
		NonTestFiles: make([]domain.NonTestFile, 0),
	}

	seenCandidates := make(map[string]bool)

	for _, changedPath := range changedFiles {
		if !isSafeRepositoryPath(changedPath) {
			continue
		}

		for _, candidatePath := range candidateTestPaths(repositoryPath, changedPath) {
			if seenCandidates[candidatePath] {
				continue
			}
			content, readErr := readScannedFile(repositoryPath, candidatePath)
			if readErr != nil {
				continue
			}
			seenCandidates[candidatePath] = true
			language := languageForPath(candidatePath)
			result.Candidates = append(result.Candidates, domain.CandidateTestFile{
				Path:          candidatePath,
				Package:       goPackagePath(candidatePath),
				Language:      language,
				Content:       content,
				TestFunctions: goTestFunctions(language, content),
			})
		}

		if isTestPath(changedPath) {
			continue
		}
		content, readErr := readScannedFile(repositoryPath, changedPath)
		if readErr != nil {
			continue
		}
		result.NonTestFiles = append(result.NonTestFiles, domain.NonTestFile{Path: changedPath, Content: content})
	}

	result.UntestedFunctions = extractUntestedFunctions(result.Candidates, result.NonTestFiles)

	return result, nil
}

func candidateTestPaths(repositoryPath string, changedPath string) []string {
	baseName := path.Base(changedPath)

	switch {
	case strings.HasSuffix(baseName, "_test.go"):
		return []string{changedPath}
	case isPHPTestName(baseName):
		return []string{changedPath}
	case isNodeTestName(baseName):
		return []string{changedPath}
	case strings.HasSuffix(baseName, ".go"):
		sibling := path.Join(path.Dir(changedPath), strings.TrimSuffix(baseName, ".go")+"_test.go")
		return existingRepositoryPaths(repositoryPath, []string{sibling})
	case isNodeSourceName(baseName):
		extension := path.Ext(baseName)
		stem := strings.TrimSuffix(baseName, extension)
		siblings := make([]string, 0, 4)
		for _, infix := range []string{".test", ".spec"} {
			siblings = append(siblings, path.Join(path.Dir(changedPath), stem+infix+extension))
		}
		return existingRepositoryPaths(repositoryPath, siblings)
	default:
		return nil
	}
}

func existingRepositoryPaths(repositoryPath string, candidates []string) []string {
	existing := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		info, err := os.Stat(filepath.Join(repositoryPath, filepath.FromSlash(candidate)))
		if err != nil || info.IsDir() {
			continue
		}
		existing = append(existing, candidate)
	}
	return existing
}

func isTestPath(value string) bool {
	baseName := path.Base(value)
	return strings.HasSuffix(baseName, "_test.go") || isNodeTestName(baseName) || isPHPTestName(baseName)
}

func isPHPTestName(baseName string) bool {
	return strings.HasSuffix(baseName, "Test.php") || strings.HasSuffix(baseName, "_test.php")
}

func isNodeTestName(baseName string) bool {
	if !nodeFileExtensions[path.Ext(baseName)] {
		return false
	}
	return strings.Contains(baseName, ".test.") || strings.Contains(baseName, ".spec.")
}

func isNodeSourceName(baseName string) bool {
	return nodeFileExtensions[path.Ext(baseName)]
}

func goTestFunctions(language string, content string) []string {
	if language != domain.LanguageGo {
		return nil
	}
	matches := goTestFunctionPattern.FindAllStringSubmatch(content, -1)
	if len(matches) == 0 {
		return nil
	}
	functions := make([]string, 0, len(matches))
	seen := make(map[string]struct{}, len(matches))
	for _, match := range matches {
		name := match[1]
		if _, exists := seen[name]; exists {
			continue
		}
		seen[name] = struct{}{}
		functions = append(functions, name)
	}
	return functions
}

func goFunctionNames(content string) []string {
	matches := goFunctionNamePattern.FindAllStringSubmatch(content, -1)
	if len(matches) == 0 {
		return nil
	}
	names := make([]string, 0, len(matches))
	seen := make(map[string]struct{}, len(matches))
	for _, match := range matches {
		name := match[1]
		if _, exists := seen[name]; exists {
			continue
		}
		seen[name] = struct{}{}
		names = append(names, name)
	}
	return names
}

func extractUntestedFunctions(candidates []domain.CandidateTestFile, nonTestFiles []domain.NonTestFile) []domain.UntestedFunction {
	testFunctions := make(map[string]struct{})
	for _, candidate := range candidates {
		if candidate.Language != domain.LanguageGo {
			continue
		}
		for _, name := range candidate.TestFunctions {
			testFunctions[strings.TrimSpace(name)] = struct{}{}
		}
	}

	untested := make([]domain.UntestedFunction, 0)
	seen := make(map[string]struct{})
	for _, file := range nonTestFiles {
		if !strings.HasSuffix(file.Path, ".go") || isTestPath(file.Path) {
			continue
		}
		for _, name := range goFunctionNames(file.Content) {
			if _, isTest := testFunctions[name]; isTest {
				continue
			}
			key := file.Path + "\x1f" + name
			if _, exists := seen[key]; exists {
				continue
			}
			seen[key] = struct{}{}
			untested = append(untested, domain.UntestedFunction{
				Name:    name,
				File:    file.Path,
				Package: goPackagePath(file.Path),
			})
		}
	}
	return untested
}

func languageForPath(candidatePath string) string {
	switch {
	case strings.HasSuffix(candidatePath, ".go"):
		return domain.LanguageGo
	case isPHPTestName(path.Base(candidatePath)):
		return domain.LanguagePHP
	default:
		return domain.LanguageNode
	}
}

func goPackagePath(candidatePath string) string {
	directory := path.Dir(candidatePath)
	if directory == "." {
		return "."
	}
	return "./" + directory
}

func isSafeRepositoryPath(value string) bool {
	if value == "" || strings.HasPrefix(value, "/") || strings.Contains(value, "\\") || strings.ContainsRune(value, 0) {
		return false
	}
	cleaned := path.Clean(value)
	if cleaned != value || cleaned == "." || cleaned == ".." || strings.HasPrefix(cleaned, "../") {
		return false
	}
	return true
}

func readScannedFile(repositoryPath string, relativePath string) (string, error) {
	fullPath := filepath.Join(repositoryPath, filepath.FromSlash(relativePath))
	file, err := os.Open(fullPath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	content, err := io.ReadAll(io.LimitReader(file, maxScannedFileBytes))
	if err != nil {
		return "", err
	}
	return string(content), nil
}

func splitChangedFiles(output string) []string {
	changedFiles := make([]string, 0)
	for _, line := range strings.Split(output, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		changedFiles = append(changedFiles, trimmed)
	}
	return changedFiles
}
