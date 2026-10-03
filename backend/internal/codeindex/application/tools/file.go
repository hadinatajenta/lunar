package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	agentdomain "lunar/backend/internal/agent/domain"
	codeindexinfrastructure "lunar/backend/internal/codeindex/infrastructure"
	workspacedomain "lunar/backend/internal/workspace/domain"
)

const (
	defaultFileWindowLines = 80
	maxFileWindowLines     = 120
	maxFileReadBytes       = 1 << 20
	maxFileContentChars    = 20000
	fileTruncationNote     = "\n[output truncated]\n"
)

type fileContentArguments struct {
	RepoName  string `json:"repo_name"`
	FilePath  string `json:"file_path"`
	StartLine int    `json:"start_line"`
	EndLine   int    `json:"end_line"`
}

type fileExcerpt struct {
	RepoName    string
	FilePath    string
	StartLine   int
	EndLine     int
	TotalLines  int
	Lines       []string
	IsTruncated bool
}

type fileContentTool struct {
	dependencies toolDependencies
}

func (t fileContentTool) Definition() agentdomain.ToolDefinition {
	return toolDefinition(ToolNameGetFileContent, getFileContentDescription, getFileContentSchema, true)
}

func (t fileContentTool) Execute(ctx context.Context, args json.RawMessage) (agentdomain.ToolResult, error) {
	var arguments fileContentArguments
	if err := json.Unmarshal(args, &arguments); err != nil {
		return agentdomain.ToolResult{}, fmt.Errorf("get_file_content: invalid arguments: %w", err)
	}

	repoName, err := requireArgument(arguments.RepoName, "repo_name")
	if err != nil {
		return agentdomain.ToolResult{}, fmt.Errorf("get_file_content: %w", err)
	}
	filePath, err := requireArgument(arguments.FilePath, "file_path")
	if err != nil {
		return agentdomain.ToolResult{}, fmt.Errorf("get_file_content: %w", err)
	}
	workspaceRoot, err := t.dependencies.requireWorkspaceRoot()
	if err != nil {
		return agentdomain.ToolResult{}, fmt.Errorf("get_file_content: %w", err)
	}

	excerpt, err := readFileExcerpt(ctx, workspaceRoot, repoName, filePath, arguments.StartLine, arguments.EndLine)
	if err != nil {
		return agentdomain.ToolResult{}, fmt.Errorf("get_file_content: %w", err)
	}
	if excerpt.TotalLines == 0 {
		return emptyToolResult(fmt.Sprintf("File %s/%s is empty.", excerpt.RepoName, excerpt.FilePath)), nil
	}

	content := formatFileExcerpt(excerpt)
	source := agentdomain.SourceReference{
		Repo:      excerpt.RepoName,
		File:      excerpt.FilePath,
		Type:      "file_content",
		Snippet:   content,
		StartLine: excerpt.StartLine,
		EndLine:   excerpt.EndLine,
	}
	return agentdomain.ToolResult{Content: content, Sources: []agentdomain.SourceReference{source}}, nil
}

func readFileExcerpt(
	ctx context.Context,
	workspaceRoot string,
	repoName string,
	requestedPath string,
	startLine int,
	endLine int,
) (fileExcerpt, error) {
	if err := ctx.Err(); err != nil {
		return fileExcerpt{}, err
	}

	repoPath, err := workspacedomain.ResolveRepoPath(workspaceRoot, repoName)
	if err != nil {
		return fileExcerpt{}, fmt.Errorf("resolve repository %q: %w", repoName, err)
	}

	cleanPath := filepath.Clean(strings.TrimSpace(requestedPath))
	if cleanPath == "" || cleanPath == "." || cleanPath == ".." || filepath.IsAbs(cleanPath) ||
		strings.HasPrefix(cleanPath, ".."+string(filepath.Separator)) {
		return fileExcerpt{}, fmt.Errorf("path %q escapes the repository root", requestedPath)
	}
	if codeindexinfrastructure.IsExcludedFileName(cleanPath) {
		return fileExcerpt{}, fmt.Errorf("path %q is excluded from reads", requestedPath)
	}

	fullPath := filepath.Join(repoPath, cleanPath)
	if !workspacedomain.WithinAllowedRoots(fullPath, []string{repoPath}) {
		return fileExcerpt{}, fmt.Errorf("path %q escapes the repository root", requestedPath)
	}

	file, err := os.Open(fullPath)
	if err != nil {
		return fileExcerpt{}, fmt.Errorf("open %q: %w", requestedPath, err)
	}
	data, readErr := io.ReadAll(io.LimitReader(file, maxFileReadBytes+1))
	closeErr := file.Close()
	if readErr != nil {
		return fileExcerpt{}, fmt.Errorf("read %q: %w", requestedPath, readErr)
	}
	if closeErr != nil {
		return fileExcerpt{}, fmt.Errorf("close %q: %w", requestedPath, closeErr)
	}

	isTruncated := len(data) > maxFileReadBytes
	if isTruncated {
		data = data[:maxFileReadBytes]
	}
	if codeindexinfrastructure.IsBinaryContent(data) {
		return fileExcerpt{}, fmt.Errorf("path %q is not a text file", requestedPath)
	}

	lines := strings.Split(string(data), "\n")
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}

	excerpt := fileExcerpt{
		RepoName:    repoName,
		FilePath:    filepath.ToSlash(cleanPath),
		TotalLines:  len(lines),
		IsTruncated: isTruncated,
	}
	if len(lines) == 0 {
		return excerpt, nil
	}

	excerpt.StartLine = clampStartLine(startLine, len(lines))
	excerpt.EndLine = clampEndLine(endLine, excerpt.StartLine, len(lines))

	selected := lines[excerpt.StartLine-1 : excerpt.EndLine]
	redacted, _ := codeindexinfrastructure.RedactSecretsPerLine(strings.Join(selected, "\n"))
	excerpt.Lines = strings.Split(redacted, "\n")
	return excerpt, nil
}

func clampStartLine(startLine int, totalLines int) int {
	if startLine <= 0 {
		return 1
	}
	if startLine > totalLines {
		return totalLines
	}
	return startLine
}

func clampEndLine(endLine int, startLine int, totalLines int) int {
	if endLine <= 0 || endLine < startLine {
		endLine = startLine + defaultFileWindowLines
	}
	if endLine > startLine+maxFileWindowLines {
		endLine = startLine + maxFileWindowLines
	}
	if endLine > totalLines {
		endLine = totalLines
	}
	return endLine
}

func resolveSearchRoot(workspaceRoot string, repoFilter string) (string, string, error) {
	if repoFilter == "" {
		return workspaceRoot, workspaceRoot, nil
	}

	repoPath, err := workspacedomain.ResolveRepoPath(workspaceRoot, repoFilter)
	if err != nil {
		return "", "", fmt.Errorf("resolve repository %q: %w", repoFilter, err)
	}
	return repoPath, filepath.Dir(repoPath), nil
}

func splitRepositoryPath(relativePath string) (string, string) {
	segments := strings.SplitN(relativePath, "/", 2)
	if len(segments) == 1 {
		return segments[0], relativePath
	}
	return segments[0], segments[1]
}

func formatFileExcerpt(excerpt fileExcerpt) string {
	builder := newBoundedResultBuilder(maxFileContentChars)

	truncatedNote := ""
	if excerpt.IsTruncated {
		truncatedNote = fmt.Sprintf(", truncated at %d bytes", maxFileReadBytes)
	}
	builder.write(fmt.Sprintf(
		"File: %s/%s (lines %d-%d of %d%s):\n```\n",
		excerpt.RepoName,
		excerpt.FilePath,
		excerpt.StartLine,
		excerpt.EndLine,
		excerpt.TotalLines,
		truncatedNote,
	))

	for offset, line := range excerpt.Lines {
		if !builder.write(fmt.Sprintf("%4d | %s\n", excerpt.StartLine+offset, line)) {
			break
		}
	}
	return builder.finish(fileTruncationNote)
}

func writeRouterExcerpt(
	ctx context.Context,
	builder *boundedResultBuilder,
	workspaceRoot string,
	serviceName string,
	filePath string,
) []agentdomain.SourceReference {
	if workspaceRoot == "" {
		return nil
	}

	excerpt, err := readFileExcerpt(ctx, workspaceRoot, serviceName, filePath, 1, 60)
	if err != nil {
		builder.write(fmt.Sprintf("### Router file excerpt\nThe file `%s` could not be read: %v\n\n", filePath, err))
		return nil
	}

	content := formatFileExcerpt(excerpt)
	builder.write(fmt.Sprintf("### Router file excerpt (`%s`)\n%s\n", filePath, content))
	return []agentdomain.SourceReference{{
		Repo:      excerpt.RepoName,
		File:      excerpt.FilePath,
		Type:      "file_content",
		Snippet:   content,
		StartLine: excerpt.StartLine,
		EndLine:   excerpt.EndLine,
	}}
}
