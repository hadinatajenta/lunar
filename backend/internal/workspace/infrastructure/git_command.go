package infrastructure

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"lunar/backend/internal/workspace/domain"
)

const (
	upstreamArgument      = "HEAD...@{upstream}"
	upstreamReferenceName = "@{upstream}"
	commitLogFormat       = "%H%x1f%an%x1f%ar%x1f%s"
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

func runGit(ctx context.Context, repositoryPath string, args ...string) (string, error) {
	commandContext, cancel := context.WithTimeout(ctx, domain.GitCommandTimeout)
	defer cancel()

	commandArguments := append([]string{"--no-optional-locks", "-C", repositoryPath}, args...)
	command := exec.CommandContext(commandContext, "git", commandArguments...)
	command.Env = hardenedGitEnvironment()

	output, err := command.Output()
	if err != nil {
		return "", fmt.Errorf("git %s: %w", args[0], err)
	}
	return string(output), nil
}

func hardenedGitEnvironment() []string {
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

func isGitRepository(ctx context.Context, repositoryPath string) bool {
	_, err := runGit(ctx, repositoryPath, "rev-parse", "--git-dir")
	return err == nil
}

func readBranch(ctx context.Context, repositoryPath string) (string, error) {
	output, err := runGit(ctx, repositoryPath, "rev-parse", "--abbrev-ref", "HEAD")
	if err == nil {
		return strings.TrimSpace(output), nil
	}

	fallback, fallbackErr := runGit(ctx, repositoryPath, "symbolic-ref", "--short", "HEAD")
	if fallbackErr != nil {
		return "", err
	}
	return strings.TrimSpace(fallback), nil
}

func readDirtyCount(ctx context.Context, repositoryPath string) (int, error) {
	output, err := runGit(ctx, repositoryPath, "status", "--porcelain=v1")
	if err != nil {
		return 0, err
	}
	return len(parsePorcelainStatus(output)), nil
}

func readLastCommitOrEmpty(ctx context.Context, repositoryPath string) domain.CommitInfo {
	output, err := runGit(ctx, repositoryPath, "log", "-1", "--format="+commitLogFormat)
	if err != nil {
		return domain.CommitInfo{}
	}

	commit, err := parseCommitInfo(output)
	if err != nil {
		return domain.CommitInfo{}
	}
	return commit
}

func readChangedFiles(ctx context.Context, repositoryPath string) ([]domain.ChangedFile, bool, error) {
	output, err := runGit(ctx, repositoryPath, "status", "--porcelain=v1")
	if err != nil {
		return nil, false, err
	}

	entries := parsePorcelainStatus(output)
	lineDeltas := readLineDeltasOrEmpty(ctx, repositoryPath)

	files := make([]domain.ChangedFile, 0, len(entries))
	truncated := false

	for _, entry := range entries {
		if len(files) >= domain.MaxChangedFiles {
			truncated = true
			break
		}

		delta, found := lineDeltas[entry.Path]
		if !found && entry.OriginalPath != "" {
			delta = lineDeltas[entry.OriginalPath]
		}

		files = append(files, domain.ChangedFile{
			Status:  entry.Status,
			Path:    entry.Path,
			Added:   delta.Added,
			Deleted: delta.Deleted,
		})
	}

	return files, truncated, nil
}

func readLineDeltasOrEmpty(ctx context.Context, repositoryPath string) map[string]lineDelta {
	output, err := runGit(ctx, repositoryPath, "diff", "--numstat", "HEAD")
	if err != nil {
		return map[string]lineDelta{}
	}
	return parseNumstat(output)
}

func readAheadBehind(ctx context.Context, repositoryPath string) (int, int, error) {
	output, err := runGit(ctx, repositoryPath, "rev-list", "--left-right", "--count", upstreamArgument)
	if err != nil {
		if hasUpstream(ctx, repositoryPath) {
			return 0, 0, err
		}
		return 0, 0, nil
	}

	fields := strings.Fields(output)
	if len(fields) != 2 {
		return 0, 0, fmt.Errorf("unexpected ahead and behind output")
	}

	ahead, err := strconv.Atoi(fields[0])
	if err != nil {
		return 0, 0, fmt.Errorf("parse ahead count: %w", err)
	}

	behind, err := strconv.Atoi(fields[1])
	if err != nil {
		return 0, 0, fmt.Errorf("parse behind count: %w", err)
	}

	return ahead, behind, nil
}

func hasUpstream(ctx context.Context, repositoryPath string) bool {
	_, err := runGit(ctx, repositoryPath, "rev-parse", "--abbrev-ref", "--symbolic-full-name", upstreamReferenceName)
	return err == nil
}
