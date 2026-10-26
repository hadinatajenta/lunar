package infrastructure

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

const (
	gitCommandTimeout = 30 * time.Second
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

func runUTDocsGit(ctx context.Context, repositoryPath string, args ...string) (string, error) {
	commandContext, cancel := context.WithTimeout(ctx, gitCommandTimeout)
	defer cancel()

	commandArguments := append([]string{"--no-optional-locks", "-C", repositoryPath}, args...)
	command := exec.CommandContext(commandContext, "git", commandArguments...)
	command.Env = hardenedUTDocsGitEnvironment()

	output, err := command.Output()
	if err != nil {
		if errors.Is(commandContext.Err(), context.DeadlineExceeded) {
			return "", fmt.Errorf("git %s timed out after %s", args[0], gitCommandTimeout)
		}
		return "", fmt.Errorf("git %s: %w", args[0], err)
	}
	return string(output), nil
}

func hardenedUTDocsGitEnvironment() []string {
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

func CurrentBranch(ctx context.Context, repositoryPath string) (string, error) {
	output, err := runUTDocsGit(ctx, repositoryPath, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return "", fmt.Errorf("read the current branch: %w", err)
	}
	return strings.TrimSpace(output), nil
}

func IsWorktreeDirty(ctx context.Context, repositoryPath string) (bool, error) {
	output, err := runUTDocsGit(ctx, repositoryPath, "status", "--porcelain", "--untracked-files=no")
	if err != nil {
		return false, fmt.Errorf("read the worktree status: %w", err)
	}
	return strings.TrimSpace(output) != "", nil
}

var defaultBaseBranchCandidates = []string{"development", "main", "master"}

func DefaultBaseBranch(ctx context.Context, repositoryPath string) (string, error) {
	candidates := make([]string, 0, len(defaultBaseBranchCandidates)+1)
	if output, err := runUTDocsGit(ctx, repositoryPath, "symbolic-ref", "--quiet", "--short", "refs/remotes/origin/HEAD"); err == nil {
		if branch := strings.TrimPrefix(strings.TrimSpace(output), "origin/"); branch != "" {
			candidates = append(candidates, branch)
		}
	}
	candidates = append(candidates, defaultBaseBranchCandidates...)

	for _, candidate := range candidates {
		if refExists(ctx, repositoryPath, "refs/heads/"+candidate) {
			return candidate, nil
		}
		if refExists(ctx, repositoryPath, "refs/remotes/origin/"+candidate) {
			return "origin/" + candidate, nil
		}
	}

	return "", fmt.Errorf("cannot resolve the default base branch for %s", repositoryPath)
}

func refExists(ctx context.Context, repositoryPath string, reference string) bool {
	_, err := runUTDocsGit(ctx, repositoryPath, "show-ref", "--verify", "--quiet", reference)
	return err == nil
}

func RestoreStash(ctx context.Context, repositoryPath string) error {
	if _, err := runUTDocsGit(ctx, repositoryPath, "stash", "pop"); err != nil {
		return fmt.Errorf("restore the stashed changes: %w", err)
	}
	return nil
}
