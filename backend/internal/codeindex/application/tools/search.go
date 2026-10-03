package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	agentdomain "lunar/backend/internal/agent/domain"
	codeindexdomain "lunar/backend/internal/codeindex/domain"
	codeindexinfrastructure "lunar/backend/internal/codeindex/infrastructure"
)

const (
	maxGrepMatches       = 10
	maxGrepPatternRunes  = 200
	maxGrepReadBytes     = 2 << 20
	grepContextLines     = 1
	maxGrepCharacters    = 12000
	searchTruncationNote = "\n[result truncated: narrow the query or the repository filter]\n"
	grepTruncationNote   = "\n[result truncated: narrow the pattern or the repository filter]\n"
)

type searchCodeArguments struct {
	Query      string `json:"query"`
	RepoFilter string `json:"repo_filter"`
	Limit      int    `json:"limit"`
}

type grepRawFallbackArguments struct {
	Pattern    string `json:"pattern"`
	RepoFilter string `json:"repo_filter"`
}

type grepMatch struct {
	RepoName    string
	FilePath    string
	StartLine   int
	EndLine     int
	Snippet     string
	IsTruncated bool
}

type searchCodeTool struct {
	dependencies toolDependencies
}

func (t searchCodeTool) Definition() agentdomain.ToolDefinition {
	return toolDefinition(ToolNameSearchCode, searchCodeDescription, searchCodeSchema, true)
}

func (t searchCodeTool) Execute(ctx context.Context, args json.RawMessage) (agentdomain.ToolResult, error) {
	var arguments searchCodeArguments
	if err := json.Unmarshal(args, &arguments); err != nil {
		return agentdomain.ToolResult{}, fmt.Errorf("search_code: invalid arguments: %w", err)
	}

	query, err := requireArgument(arguments.Query, "query")
	if err != nil {
		return agentdomain.ToolResult{}, fmt.Errorf("search_code: %w", err)
	}
	store, err := t.dependencies.requireStore()
	if err != nil {
		return agentdomain.ToolResult{}, fmt.Errorf("search_code: %w", err)
	}
	repoFilter, err := normalizeRepoFilter(arguments.RepoFilter)
	if err != nil {
		return agentdomain.ToolResult{}, fmt.Errorf("search_code: %w", err)
	}

	limit := clampLimit(arguments.Limit, defaultSearchLimit, maxSearchLimit)
	chunks, err := store.SearchChunksIn(ctx, query, repositoryFilterSlice(repoFilter), limit)
	if err != nil {
		return agentdomain.ToolResult{}, fmt.Errorf("search_code: %w", err)
	}
	if len(chunks) == 0 {
		return emptyToolResult(fmt.Sprintf("No indexed code chunk matched the query %q.", query)), nil
	}

	builder := newBoundedResultBuilder(maxResultCharacters)
	builder.write(fmt.Sprintf("Found %d code chunks for query %q:\n\n", len(chunks), query))

	sources := make([]agentdomain.SourceReference, 0, len(chunks))
	for index, chunk := range chunks {
		text, isTruncated := truncateText(chunk.RawContent, maxChunkCharacters)
		entry := fmt.Sprintf("[%d] repo=%s file=%s type=%s%s%s\n```\n%s\n```\n\n", index+1, chunk.RepoName,
			chunk.FilePath, chunk.ChunkType, lineRangeLabel(chunk.StartLine, chunk.EndLine), truncationLabel(isTruncated), text)
		if !builder.write(entry) {
			break
		}
		sources = append(sources, sourceReferenceForChunk(chunk, text))
	}

	return agentdomain.ToolResult{
		Content: builder.finish(searchTruncationNote),
		Sources: sources,
	}, nil
}

type grepRawFallbackTool struct {
	dependencies toolDependencies
}

func (t grepRawFallbackTool) Definition() agentdomain.ToolDefinition {
	return toolDefinition(ToolNameGrepRawFallback, grepRawFallbackDescription, grepRawFallbackSchema, true)
}

func (t grepRawFallbackTool) Execute(ctx context.Context, args json.RawMessage) (agentdomain.ToolResult, error) {
	var arguments grepRawFallbackArguments
	if err := json.Unmarshal(args, &arguments); err != nil {
		return agentdomain.ToolResult{}, fmt.Errorf("grep_raw_fallback: invalid arguments: %w", err)
	}

	pattern, err := requireArgument(arguments.Pattern, "pattern")
	if err != nil {
		return agentdomain.ToolResult{}, fmt.Errorf("grep_raw_fallback: %w", err)
	}
	if utf8.RuneCountInString(pattern) > maxGrepPatternRunes {
		return agentdomain.ToolResult{}, fmt.Errorf("grep_raw_fallback: pattern exceeds %d characters", maxGrepPatternRunes)
	}

	workspaceRoot, err := t.dependencies.requireWorkspaceRoot()
	if err != nil {
		return agentdomain.ToolResult{}, fmt.Errorf("grep_raw_fallback: %w", err)
	}
	repoFilter, err := normalizeRepoFilter(arguments.RepoFilter)
	if err != nil {
		return agentdomain.ToolResult{}, fmt.Errorf("grep_raw_fallback: %w", err)
	}
	searchRoot, relativeRoot, err := resolveSearchRoot(workspaceRoot, repoFilter)
	if err != nil {
		return agentdomain.ToolResult{}, fmt.Errorf("grep_raw_fallback: %w", err)
	}

	matches, err := collectGrepMatches(ctx, searchRoot, relativeRoot, pattern, maxGrepMatches)
	if err != nil {
		return agentdomain.ToolResult{}, fmt.Errorf("grep_raw_fallback: %w", err)
	}
	if len(matches) == 0 {
		return emptyToolResult(fmt.Sprintf("No literal match was found for pattern %q.", pattern)), nil
	}

	builder := newBoundedResultBuilder(maxGrepCharacters)
	builder.write(fmt.Sprintf("Found %d literal matches for pattern %q (grep_raw_fallback):\n\n", len(matches), pattern))

	sources := make([]agentdomain.SourceReference, 0, len(matches))
	for index, match := range matches {
		entry := fmt.Sprintf("[%d] repo=%s file=%s lines=%d-%d%s\n```\n%s\n```\n\n", index+1, match.RepoName,
			match.FilePath, match.StartLine, match.EndLine, truncationLabel(match.IsTruncated), match.Snippet)
		if !builder.write(entry) {
			break
		}
		sources = append(sources, agentdomain.SourceReference{
			Repo:      match.RepoName,
			File:      match.FilePath,
			Type:      "grep_match",
			Snippet:   match.Snippet,
			StartLine: match.StartLine,
			EndLine:   match.EndLine,
		})
	}

	return agentdomain.ToolResult{
		Content: builder.finish(grepTruncationNote),
		Sources: sources,
	}, nil
}

func collectGrepMatches(ctx context.Context, searchRoot string, relativeRoot string, pattern string, maximum int) ([]grepMatch, error) {
	loweredPattern := strings.ToLower(pattern)
	var matches []grepMatch

	walkErr := filepath.WalkDir(searchRoot, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if len(matches) >= maximum {
			return fs.SkipAll
		}
		if entry.IsDir() {
			if path == searchRoot {
				return nil
			}
			if codeindexinfrastructure.IsSkippedDirectory(entry.Name()) {
				return fs.SkipDir
			}
			return nil
		}
		if entry.Type()&fs.ModeSymlink != 0 || !codeindexinfrastructure.IsIndexableFile(entry.Name()) {
			return nil
		}

		info, err := entry.Info()
		if err != nil {
			return err
		}
		if info.Size() > maxGrepReadBytes {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if codeindexinfrastructure.IsBinaryContent(content) {
			return nil
		}

		matches = append(matches, matchLines(path, relativeRoot, string(content), loweredPattern, maximum-len(matches))...)
		return nil
	})
	if walkErr != nil {
		return nil, fmt.Errorf("collectGrepMatches: %w", walkErr)
	}
	return matches, nil
}

func matchLines(path string, relativeRoot string, content string, loweredPattern string, maximum int) []grepMatch {
	if maximum <= 0 {
		return nil
	}

	relativePath, err := filepath.Rel(relativeRoot, path)
	if err != nil {
		return nil
	}
	repoName, filePath := splitRepositoryPath(filepath.ToSlash(relativePath))

	lines := strings.Split(content, "\n")
	var matches []grepMatch

	for index, line := range lines {
		if len(matches) >= maximum {
			break
		}
		if !strings.Contains(strings.ToLower(line), loweredPattern) {
			continue
		}

		startIndex := index - grepContextLines
		if startIndex < 0 {
			startIndex = 0
		}
		endIndex := index + grepContextLines + 1
		if endIndex > len(lines) {
			endIndex = len(lines)
		}

		redacted, _ := codeindexinfrastructure.RedactSecretsPerLine(strings.Join(lines[startIndex:endIndex], "\n"))
		snippet, isTruncated := truncateText(redacted, maxSnippetCharacters)
		matches = append(matches, grepMatch{
			RepoName:    repoName,
			FilePath:    filePath,
			StartLine:   startIndex + 1,
			EndLine:     endIndex,
			Snippet:     snippet,
			IsTruncated: isTruncated,
		})
	}
	return matches
}

func filterRouteChunks(chunks []codeindexdomain.CodeChunk) []codeindexdomain.CodeChunk {
	routes := make([]codeindexdomain.CodeChunk, 0, len(chunks))
	for _, chunk := range chunks {
		if chunk.ChunkType == serviceRouteChunkType {
			routes = append(routes, chunk)
		}
	}
	return routes
}

func findRoute(routes []codeindexdomain.CodeChunk, endpointPath string) *codeindexdomain.CodeChunk {
	for index := range routes {
		if strings.Contains(routes[index].RawContent, endpointPath) {
			return &routes[index]
		}
	}

	normalizedPath := strings.ToLower(strings.Trim(endpointPath, "/"))
	for index := range routes {
		if strings.Contains(strings.ToLower(routes[index].RawContent), normalizedPath) {
			return &routes[index]
		}
	}
	return nil
}
