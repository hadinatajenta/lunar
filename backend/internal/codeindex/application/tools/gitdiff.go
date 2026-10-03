package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"unicode"

	agentdomain "lunar/backend/internal/agent/domain"
	codeindexinfrastructure "lunar/backend/internal/codeindex/infrastructure"
	workspacedomain "lunar/backend/internal/workspace/domain"
)

const (
	maxGitDiffCharacters   = 40000
	gitDiffTruncationNote  = "\n[the diff was truncated]\n"
	gitDiffBaselineDefault = "HEAD"
)

type GitDiffRequest struct {
	RepositoryName     string
	BaseRef            string
	IncludeUncommitted bool
	FilePath           string
}

type GitDiffResult struct {
	RepositoryName string
	Branch         string
	HeadSHA        string
	BaseRef        string
	ChangedFiles   []string
	UntrackedFiles []string
	Diff           string
	IsTruncated    bool
}

type CodeGitReader interface {
	GetDiff(ctx context.Context, request GitDiffRequest) (GitDiffResult, error)
}

type gitDiffArguments struct {
	RepoName           string `json:"repo_name"`
	BaseRef            string `json:"base_ref"`
	IncludeUncommitted *bool  `json:"include_uncommitted"`
	FilePath           string `json:"file_path"`
}

type gitDiffTool struct {
	dependencies toolDependencies
}

func (t gitDiffTool) Definition() agentdomain.ToolDefinition {
	return toolDefinition(ToolNameGetGitDiff, getGitDiffDescription, getGitDiffSchema, true)
}

func (t gitDiffTool) Execute(ctx context.Context, args json.RawMessage) (agentdomain.ToolResult, error) {
	var arguments gitDiffArguments
	if err := json.Unmarshal(args, &arguments); err != nil {
		return agentdomain.ToolResult{}, fmt.Errorf("get_git_diff: invalid arguments: %w", err)
	}

	repoName, err := requireArgument(arguments.RepoName, "repo_name")
	if err != nil {
		return agentdomain.ToolResult{}, fmt.Errorf("get_git_diff: %w", err)
	}
	if _, err := workspacedomain.ValidateRepoName(repoName); err != nil {
		return agentdomain.ToolResult{}, fmt.Errorf("get_git_diff: invalid repository name %q: %w", repoName, err)
	}

	baseRef := strings.TrimSpace(arguments.BaseRef)
	if err := validateGitReference(baseRef); err != nil {
		return agentdomain.ToolResult{}, fmt.Errorf("get_git_diff: %w", err)
	}
	filePath := strings.TrimSpace(arguments.FilePath)
	if err := validateRelativeFilePath(filePath); err != nil {
		return agentdomain.ToolResult{}, fmt.Errorf("get_git_diff: %w", err)
	}

	reader, err := t.dependencies.requireGitReader()
	if err != nil {
		return agentdomain.ToolResult{}, fmt.Errorf("get_git_diff: %w", err)
	}

	includeUncommitted := true
	if arguments.IncludeUncommitted != nil {
		includeUncommitted = *arguments.IncludeUncommitted
	}

	result, err := reader.GetDiff(ctx, GitDiffRequest{
		RepositoryName:     repoName,
		BaseRef:            baseRef,
		IncludeUncommitted: includeUncommitted,
		FilePath:           filePath,
	})
	if err != nil {
		return agentdomain.ToolResult{}, fmt.Errorf("get_git_diff: %w", err)
	}

	diff, isDiffTruncated := truncateText(result.Diff, maxGitDiffCharacters)
	baseline := strings.TrimSpace(result.BaseRef)
	if baseline == "" {
		baseline = baseRef
	}
	if baseline == "" {
		baseline = gitDiffBaselineDefault
	}

	builder := newBoundedResultBuilder(maxGitDiffCharacters)
	builder.write(fmt.Sprintf(
		"Git diff of repository %s (branch: %s, HEAD: %s, baseline: %s):\n",
		result.RepositoryName,
		result.Branch,
		result.HeadSHA,
		baseline,
	))
	if len(result.ChangedFiles) == 0 {
		builder.write("No tracked file changed in this scope.\n")
	} else {
		builder.write("Changed files:\n- " + strings.Join(result.ChangedFiles, "\n- ") + "\n")
	}
	if len(result.UntrackedFiles) > 0 {
		builder.write("Untracked files (content not shown):\n- " + strings.Join(result.UntrackedFiles, "\n- ") + "\n")
	}
	if strings.TrimSpace(diff) != "" {
		builder.write("\n```diff\n" + diff + "\n```\n")
	}
	if result.IsTruncated || isDiffTruncated {
		builder.write("\n[the diff was truncated: read the specific files with get_file_content before concluding]\n")
	}

	content := builder.finish(gitDiffTruncationNote)
	snippet, _ := truncateText(content, maxSnippetCharacters)
	sourceRepo := strings.TrimSpace(result.RepositoryName)
	if sourceRepo == "" {
		sourceRepo = repoName
	}

	return agentdomain.ToolResult{
		Content: content,
		Sources: []agentdomain.SourceReference{{
			Repo:    sourceRepo,
			File:    filePath,
			Type:    "git_diff",
			Snippet: snippet,
		}},
	}, nil
}

func validateGitReference(reference string) error {
	if reference == "" || reference == gitDiffBaselineDefault {
		return nil
	}
	if strings.HasPrefix(reference, "-") || strings.Contains(reference, "..") ||
		strings.ContainsAny(reference, "~^:\\?*[ ") {
		return fmt.Errorf("invalid base_ref %q", reference)
	}
	for _, character := range reference {
		if unicode.IsControl(character) {
			return fmt.Errorf("invalid base_ref %q", reference)
		}
	}
	return nil
}

func validateRelativeFilePath(filePath string) error {
	if filePath == "" {
		return nil
	}

	cleaned := filepath.Clean(filePath)
	if filepath.IsAbs(cleaned) || cleaned == "." || cleaned == ".." ||
		strings.HasPrefix(cleaned, ".."+string(filepath.Separator)) {
		return fmt.Errorf("invalid file_path %q", filePath)
	}
	if codeindexinfrastructure.IsExcludedFileName(cleaned) {
		return fmt.Errorf("file_path %q is excluded from reads", filePath)
	}
	return nil
}
