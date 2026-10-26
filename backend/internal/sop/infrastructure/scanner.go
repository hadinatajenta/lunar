package infrastructure

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"
	"time"

	sopdomain "lunar/backend/internal/sop/domain"
)

const (
	sopGitCommandTimeout    = 30 * time.Second
	sopMaxScannedFileBytes  = 1 << 20
	sopMaxStatementsPerFile = 200
	sopMaxTotalStatements   = 1000
	sopMaxDatabaseFiles     = 200
)

var hardenedGitEnvironmentVariables = []string{
	"GIT_TERMINAL_PROMPT=0",
	"GIT_OPTIONAL_LOCKS=0",
	"GIT_CONFIG_NOSYSTEM=1",
	"LC_ALL=C",
}

var hardenedGitEnvironmentKeys = map[string]struct{}{
	"GIT_TERMINAL_PROMPT": {},
	"GIT_OPTIONAL_LOCKS":  {},
	"GIT_CONFIG_NOSYSTEM": {},
	"LC_ALL":              {},
}

type Scanner struct{}

func NewScanner() *Scanner {
	return &Scanner{}
}

func (s *Scanner) Scan(ctx context.Context, input sopdomain.ScanInput) (*sopdomain.RepositoryScan, error) {
	repositoryPath := strings.TrimSpace(input.RepositoryPath)
	if repositoryPath == "" {
		return nil, fmt.Errorf("the repository path is required to scan the branch")
	}

	branch := strings.TrimSpace(input.Branch)
	baseRef := strings.TrimSpace(input.BaseRef)
	if branch == "" || baseRef == "" {
		return nil, fmt.Errorf("the branch and base ref are required to scan the branch")
	}

	head, found, err := ResolveHeadRef(ctx, repositoryPath, branch)
	if err != nil {
		return nil, fmt.Errorf("resolve the branch %s: %w", branch, err)
	}
	if !found {
		return skippedScan(input.Repository, branch, baseRef), nil
	}

	resolvedBase, err := ResolveBaseRef(ctx, repositoryPath, baseRef)
	if err != nil {
		return nil, fmt.Errorf("resolve the base ref %s: %w", baseRef, err)
	}

	output, err := runSOPGit(ctx, repositoryPath, "diff", "--name-only", "--no-color", "--diff-filter=ACMR", resolvedBase+".."+head)
	if err != nil {
		return nil, fmt.Errorf("list the changed files: %w", err)
	}

	result := &sopdomain.RepositoryScan{
		Repository:    strings.TrimSpace(input.Repository),
		Branch:        branch,
		BaseRef:       baseRef,
		DatabaseFiles: []sopdomain.DatabaseFile{},
		Tables:        []sopdomain.DatabaseTable{},
		Warnings:      []string{},
	}

	totalStatements := 0
	databaseFileLimitReported := false
	totalLimitReported := false

	for _, changedPath := range splitChangedFiles(output) {
		if !isSafeRepositoryPath(changedPath) {
			continue
		}

		if sopdomain.IsDatabaseFilePath(changedPath) {
			if len(result.DatabaseFiles) >= sopMaxDatabaseFiles {
				if !databaseFileLimitReported {
					result.Warnings = append(result.Warnings, "more than "+strconv.Itoa(sopMaxDatabaseFiles)+" database files changed; the extra files were not read")
					databaseFileLimitReported = true
				}
				continue
			}
			content, readErr := readGitFile(ctx, repositoryPath, head, changedPath)
			if readErr != nil {
				return nil, fmt.Errorf("read %s: %w", changedPath, readErr)
			}
			statements := sopdomain.ExtractDatabaseStatements(content)
			if len(statements) > sopMaxStatementsPerFile {
				statements = statements[:sopMaxStatementsPerFile]
				result.Warnings = append(result.Warnings, changedPath+" has more than "+strconv.Itoa(sopMaxStatementsPerFile)+" statements; the extra statements were dropped")
			}
			remaining := sopMaxTotalStatements - totalStatements
			if remaining < 0 {
				remaining = 0
			}
			if len(statements) > remaining {
				statements = statements[:remaining]
				if !totalLimitReported {
					result.Warnings = append(result.Warnings, "the total statement limit of "+strconv.Itoa(sopMaxTotalStatements)+" was reached; the extra statements were dropped")
					totalLimitReported = true
				}
			}
			totalStatements += len(statements)
			result.DatabaseFiles = append(result.DatabaseFiles, sopdomain.DatabaseFile{Path: changedPath, Statements: statements})
			continue
		}

		if strings.HasSuffix(changedPath, ".go") {
			content, readErr := readGitFile(ctx, repositoryPath, head, changedPath)
			if readErr != nil {
				return nil, fmt.Errorf("read %s: %w", changedPath, readErr)
			}
			if strings.Contains(content, "AutoMigrate") {
				result.Warnings = append(result.Warnings, "GORM AutoMigrate found in "+changedPath+"; DDL is not read from ORM code")
			}
		}
	}

	result.Tables = sopdomain.MergeDatabaseTables(result.DatabaseFiles)
	return result, nil
}

func skippedScan(repo string, branch string, baseRef string) *sopdomain.RepositoryScan {
	return &sopdomain.RepositoryScan{
		Repository:    strings.TrimSpace(repo),
		Branch:        branch,
		BaseRef:       baseRef,
		DatabaseFiles: []sopdomain.DatabaseFile{},
		Tables:        []sopdomain.DatabaseTable{},
		Warnings:      []string{"the branch " + branch + " does not exist in this repository; the repository was skipped"},
	}
}

func runSOPGit(ctx context.Context, repositoryPath string, args ...string) (string, error) {
	output, _, err := runSOPGitCommand(ctx, repositoryPath, args...)
	return output, err
}

func runSOPGitCommand(ctx context.Context, repositoryPath string, args ...string) (string, int, error) {
	commandContext, cancel := context.WithTimeout(ctx, sopGitCommandTimeout)
	defer cancel()

	commandArguments := append([]string{"--no-optional-locks", "-C", repositoryPath}, args...)
	command := exec.CommandContext(commandContext, "git", commandArguments...)
	command.Env = hardenedSOPGitEnvironment()

	output, err := command.Output()
	if err != nil {
		if errors.Is(commandContext.Err(), context.DeadlineExceeded) {
			return string(output), -1, fmt.Errorf("git %s timed out after %s", args[0], sopGitCommandTimeout)
		}
		var exitError *exec.ExitError
		if errors.As(err, &exitError) {
			return string(output), exitError.ExitCode(), fmt.Errorf("git %s: %w", args[0], err)
		}
		return string(output), -1, fmt.Errorf("git %s: %w", args[0], err)
	}
	return string(output), 0, nil
}

func hardenedSOPGitEnvironment() []string {
	environment := make([]string, 0, len(os.Environ())+len(hardenedGitEnvironmentVariables))
	for _, variable := range os.Environ() {
		key, _, found := strings.Cut(variable, "=")
		if !found {
			continue
		}
		if _, overridden := hardenedGitEnvironmentKeys[key]; overridden {
			continue
		}
		environment = append(environment, variable)
	}
	return append(environment, hardenedGitEnvironmentVariables...)
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

func readGitFile(ctx context.Context, repositoryPath string, ref string, relativePath string) (string, error) {
	return runSOPGitLimited(ctx, repositoryPath, sopMaxScannedFileBytes, "show", ref+":"+relativePath)
}

func runSOPGitLimited(ctx context.Context, repositoryPath string, maxBytes int64, args ...string) (string, error) {
	commandContext, cancel := context.WithTimeout(ctx, sopGitCommandTimeout)
	defer cancel()

	commandArguments := append([]string{"--no-optional-locks", "-C", repositoryPath}, args...)
	command := exec.CommandContext(commandContext, "git", commandArguments...)
	command.Env = hardenedSOPGitEnvironment()

	stdout, err := command.StdoutPipe()
	if err != nil {
		return "", fmt.Errorf("git %s: %w", args[0], err)
	}
	if err := command.Start(); err != nil {
		if errors.Is(commandContext.Err(), context.DeadlineExceeded) {
			return "", fmt.Errorf("git %s timed out after %s", args[0], sopGitCommandTimeout)
		}
		return "", fmt.Errorf("git %s: %w", args[0], err)
	}

	content, readErr := io.ReadAll(io.LimitReader(stdout, maxBytes))
	_, _ = io.Copy(io.Discard, stdout)
	waitErr := command.Wait()

	if readErr != nil {
		return "", fmt.Errorf("git %s: %w", args[0], readErr)
	}
	if waitErr != nil {
		if errors.Is(commandContext.Err(), context.DeadlineExceeded) {
			return "", fmt.Errorf("git %s timed out after %s", args[0], sopGitCommandTimeout)
		}
		return "", fmt.Errorf("git %s: %w", args[0], waitErr)
	}
	return string(content), nil
}
