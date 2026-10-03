package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	agentdomain "lunar/backend/internal/agent/domain"
	codeindexdomain "lunar/backend/internal/codeindex/domain"
)

const symbolTruncationNote = "\n[result truncated: lower the limit or narrow the symbol]\n"

var definitionPrefixes = []string{"func ", "func (", "type ", "const ", "var ", "class ", "function ", "def "}

type symbolReferencesArguments struct {
	Symbol     string `json:"symbol"`
	RepoFilter string `json:"repo_filter"`
	Limit      int    `json:"limit"`
}

type symbolMatch struct {
	RepoName     string
	FilePath     string
	Snippet      string
	StartLine    int
	EndLine      int
	IsDefinition bool
	IsTruncated  bool
}

type symbolReferencesTool struct {
	dependencies toolDependencies
}

func (t symbolReferencesTool) Definition() agentdomain.ToolDefinition {
	return toolDefinition(ToolNameFindSymbolReferences, findSymbolReferencesDescription, findSymbolReferencesSchema, true)
}

func (t symbolReferencesTool) Execute(ctx context.Context, args json.RawMessage) (agentdomain.ToolResult, error) {
	var arguments symbolReferencesArguments
	if err := json.Unmarshal(args, &arguments); err != nil {
		return agentdomain.ToolResult{}, fmt.Errorf("find_symbol_references: invalid arguments: %w", err)
	}

	symbol, err := requireArgument(arguments.Symbol, "symbol")
	if err != nil {
		return agentdomain.ToolResult{}, fmt.Errorf("find_symbol_references: %w", err)
	}
	repoFilter, err := normalizeRepoFilter(arguments.RepoFilter)
	if err != nil {
		return agentdomain.ToolResult{}, fmt.Errorf("find_symbol_references: %w", err)
	}
	store, err := t.dependencies.requireStore()
	if err != nil {
		return agentdomain.ToolResult{}, fmt.Errorf("find_symbol_references: %w", err)
	}

	limit := clampLimit(arguments.Limit, defaultSymbolLimit, maxSymbolLimit)
	chunks, err := store.SearchChunksIn(ctx, symbol, repositoryFilterSlice(repoFilter), limit*2)
	if err != nil {
		return agentdomain.ToolResult{}, fmt.Errorf("find_symbol_references: %w", err)
	}

	seen := make(map[string]bool)
	matches := make([]symbolMatch, 0, len(chunks))
	for _, chunk := range chunks {
		appendSymbolMatch(&matches, seen, chunk, symbol)
	}

	fallbackNote := ""
	if len(matches) < 3 {
		if t.dependencies.workspaceRoot == "" {
			fallbackNote = "The disk fallback was skipped because the workspace root is not configured."
		}
		grepMatches, err := t.grepSymbol(ctx, repoFilter, symbol)
		if err != nil {
			return agentdomain.ToolResult{}, fmt.Errorf("find_symbol_references: %w", err)
		}
		for _, match := range grepMatches {
			appendGrepSymbolMatch(&matches, seen, match, symbol)
		}
	}

	if len(matches) == 0 {
		return emptyToolResult(fmt.Sprintf("No reference was found for symbol %q. %s", symbol, fallbackNote)), nil
	}

	definitions := make([]symbolMatch, 0, len(matches))
	references := make([]symbolMatch, 0, len(matches))
	for _, match := range matches {
		if match.IsDefinition {
			definitions = append(definitions, match)
			continue
		}
		references = append(references, match)
	}

	builder := newBoundedResultBuilder(maxResultCharacters)
	builder.write(fmt.Sprintf("## Symbol references for `%s`\n\n", symbol))
	if fallbackNote != "" {
		builder.write(fallbackNote + "\n\n")
	}

	sources := make([]agentdomain.SourceReference, 0, len(matches))
	sources = append(sources, writeSymbolSection(builder, "Definitions", definitions, limit)...)
	sources = append(sources, writeSymbolSection(builder, "Calls and usages", references, limit)...)

	return agentdomain.ToolResult{
		Content: builder.finish(symbolTruncationNote),
		Sources: sources,
	}, nil
}

func (t symbolReferencesTool) grepSymbol(ctx context.Context, repoFilter string, symbol string) ([]grepMatch, error) {
	workspaceRoot := t.dependencies.workspaceRoot
	if workspaceRoot == "" {
		return nil, nil
	}

	searchRoot, relativeRoot, err := resolveSearchRoot(workspaceRoot, repoFilter)
	if err != nil {
		return nil, err
	}
	return collectGrepMatches(ctx, searchRoot, relativeRoot, symbol, maxGrepMatches)
}

func appendSymbolMatch(matches *[]symbolMatch, seen map[string]bool, chunk codeindexdomain.CodeChunk, symbol string) {
	if !containsSymbol(chunk.RawContent, symbol) {
		return
	}
	key := chunk.RepoName + "|" + chunk.FilePath + "|" + chunk.RawContent
	if seen[key] {
		return
	}
	seen[key] = true

	text, isTruncated := truncateText(chunk.RawContent, maxChunkCharacters)
	*matches = append(*matches, symbolMatch{
		RepoName:     chunk.RepoName,
		FilePath:     chunk.FilePath,
		Snippet:      text,
		StartLine:    chunk.StartLine,
		EndLine:      chunk.EndLine,
		IsDefinition: isDefinitionSnippet(symbol, chunk.RawContent),
		IsTruncated:  isTruncated,
	})
}

func appendGrepSymbolMatch(matches *[]symbolMatch, seen map[string]bool, match grepMatch, symbol string) {
	if !containsSymbol(match.Snippet, symbol) {
		return
	}
	key := match.RepoName + "|" + match.FilePath + "|" + match.Snippet
	if seen[key] {
		return
	}
	seen[key] = true

	*matches = append(*matches, symbolMatch{
		RepoName:     match.RepoName,
		FilePath:     match.FilePath,
		Snippet:      match.Snippet,
		StartLine:    match.StartLine,
		EndLine:      match.EndLine,
		IsDefinition: isDefinitionSnippet(symbol, match.Snippet),
		IsTruncated:  match.IsTruncated,
	})
}

func containsSymbol(content string, symbol string) bool {
	return strings.Contains(strings.ToLower(content), strings.ToLower(symbol))
}

func isDefinitionSnippet(symbol string, content string) bool {
	loweredSymbol := strings.ToLower(symbol)
	loweredContent := strings.ToLower(content)
	for _, prefix := range definitionPrefixes {
		if strings.Contains(loweredContent, prefix+loweredSymbol) {
			return true
		}
	}
	return false
}

func writeSymbolSection(builder *boundedResultBuilder, title string, matches []symbolMatch, limit int) []agentdomain.SourceReference {
	builder.write(fmt.Sprintf("### %s (%d found)\n", title, len(matches)))
	if len(matches) == 0 {
		builder.write("None found in the indexed chunks.\n\n")
		return nil
	}

	written := make([]agentdomain.SourceReference, 0, len(matches))
	for index, match := range matches {
		if index >= limit {
			break
		}
		entry := fmt.Sprintf(
			"- repo=%s file=%s%s%s\n```\n%s\n```\n\n",
			match.RepoName,
			match.FilePath,
			lineRangeLabel(match.StartLine, match.EndLine),
			truncationLabel(match.IsTruncated),
			match.Snippet,
		)
		if !builder.write(entry) {
			break
		}
		written = append(written, agentdomain.SourceReference{
			Repo:      match.RepoName,
			File:      match.FilePath,
			Type:      "symbol_reference",
			Snippet:   match.Snippet,
			StartLine: match.StartLine,
			EndLine:   match.EndLine,
		})
	}
	return written
}
