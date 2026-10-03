package tools

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"unicode/utf8"

	codeindexdomain "lunar/backend/internal/codeindex/domain"
)

const paymentsHandlerFixture = `package handler

import "github.com/gin-gonic/gin"

func RegisterRoutes(router *gin.Engine) {
	router.GET("/api/payments", getPayments)
}

func getPayments() string {
	return "payments"
}
`

const lateFeeFixture = `package fee

func CalculateLateFee() int {
	return lateFeeRate
}
`

func TestSearchCodeReturnsExpectedChunkWithLineRange(t *testing.T) {
	store := openIndexTestStore(t)
	chunks := saveExtractedChunks(t, store, "payments-service", "internal/handler/routes.go", paymentsHandlerFixture, ".go")
	tools := newTestTools(store, t.TempDir())

	result, err := executeTool(t, mustFindTool(t, tools, ToolNameSearchCode), map[string]any{"query": "payments"})
	if err != nil {
		t.Fatalf("search_code failed: %v", err)
	}
	if !strings.Contains(result.Content, `router.GET("/api/payments", getPayments)`) {
		t.Errorf("search result does not contain the indexed route:\n%s", result.Content)
	}
	if !strings.Contains(result.Content, "repo=payments-service") {
		t.Errorf("search result does not identify the repository:\n%s", result.Content)
	}
	if !strings.Contains(result.Content, "file=internal/handler/routes.go") {
		t.Errorf("search result does not identify the file:\n%s", result.Content)
	}

	source := findSource(result.Sources, "internal/handler/routes.go", "route")
	if source == nil {
		t.Fatalf("expected a route source, got %+v", result.Sources)
	}
	if source.StartLine != 6 || source.EndLine != 6 {
		t.Errorf("expected the route line range 6-6, got %d-%d", source.StartLine, source.EndLine)
	}
	if !strings.Contains(result.Content, "lines=6") {
		t.Errorf("search result does not show the line range:\n%s", result.Content)
	}

	var indexedRoute *codeindexdomain.CodeChunk
	for index := range chunks {
		if chunks[index].ChunkType == "route" {
			indexedRoute = &chunks[index]
			break
		}
	}
	if indexedRoute == nil {
		t.Fatal("the fixture did not produce a route chunk")
	}
	if source.StartLine != indexedRoute.StartLine || source.EndLine != indexedRoute.EndLine {
		t.Errorf("source range %d-%d does not match the indexed chunk range %d-%d",
			source.StartLine, source.EndLine, indexedRoute.StartLine, indexedRoute.EndLine)
	}

	stored, err := store.SearchChunksIn(context.Background(), "payments", []string{"payments-service"}, 5)
	if err != nil {
		t.Fatalf("cannot re-read the index: %v", err)
	}
	if len(stored) == 0 {
		t.Fatal("the store returned no chunk for the indexed route")
	}
	if stored[0].StartLine != 6 || stored[0].EndLine != 6 {
		t.Errorf("the store did not persist the indexed line range, got %d-%d", stored[0].StartLine, stored[0].EndLine)
	}
}

func TestSearchCodeClampsRequestedLimit(t *testing.T) {
	store := openIndexTestStore(t)
	chunks := make([]codeindexdomain.CodeChunk, 0, 25)
	for index := 0; index < 25; index++ {
		chunks = append(chunks, codeindexdomain.CodeChunk{
			RepoName:   "catalog-service",
			FilePath:   fmt.Sprintf("internal/item_%02d.go", index),
			ChunkType:  "code_block",
			RawContent: fmt.Sprintf("func item%02d() {}", index),
			Content:    fmt.Sprintf("func item%02d paymenttoken", index),
			StartLine:  index + 1,
			EndLine:    index + 1,
		})
	}
	if err := store.SaveChunks(context.Background(), "catalog-service", chunks); err != nil {
		t.Fatalf("cannot seed chunks: %v", err)
	}

	tools := newTestTools(store, t.TempDir())
	result, err := executeTool(t, mustFindTool(t, tools, ToolNameSearchCode), map[string]any{
		"query": "paymenttoken",
		"limit": 1000,
	})
	if err != nil {
		t.Fatalf("search_code failed: %v", err)
	}
	if len(result.Sources) != maxSearchLimit {
		t.Fatalf("expected the limit to be clamped to %d sources, got %d", maxSearchLimit, len(result.Sources))
	}
	for _, source := range result.Sources {
		if source.StartLine < 1 || source.StartLine > 25 {
			t.Errorf("source %s has an unexpected line range %d-%d", source.File, source.StartLine, source.EndLine)
		}
	}
}

func TestSearchCodeTruncatesOversizedChunk(t *testing.T) {
	store := openIndexTestStore(t)
	rawContent := strings.Repeat("abcdefghij", 900)
	chunks := []codeindexdomain.CodeChunk{{
		RepoName:   "catalog-service",
		FilePath:   "internal/big.go",
		ChunkType:  "code_block",
		RawContent: rawContent,
		Content:    "oversizedchunk",
		StartLine:  2,
		EndLine:    40,
	}}
	if err := store.SaveChunks(context.Background(), "catalog-service", chunks); err != nil {
		t.Fatalf("cannot seed the oversized chunk: %v", err)
	}

	tools := newTestTools(store, t.TempDir())
	result, err := executeTool(t, mustFindTool(t, tools, ToolNameSearchCode), map[string]any{"query": "oversizedchunk"})
	if err != nil {
		t.Fatalf("search_code failed: %v", err)
	}
	if strings.Contains(result.Content, rawContent) {
		t.Error("expected the oversized chunk text to be truncated")
	}
	if !strings.Contains(result.Content, "truncated=true") {
		t.Errorf("expected a truncation marker in the result:\n%s", result.Content)
	}
	if len(result.Sources) != 1 {
		t.Fatalf("expected one source, got %d", len(result.Sources))
	}
	if characters := utf8.RuneCountInString(result.Sources[0].Snippet); characters != maxChunkCharacters {
		t.Errorf("expected a snippet of %d characters, got %d", maxChunkCharacters, characters)
	}
}

func TestSearchCodeRejectsInvalidArguments(t *testing.T) {
	store := openIndexTestStore(t)
	tool := mustFindTool(t, newTestTools(store, t.TempDir()), ToolNameSearchCode)

	if _, err := executeTool(t, tool, map[string]any{"query": "   "}); err == nil {
		t.Error("expected an error for a blank query")
	}
	if _, err := executeTool(t, tool, map[string]any{"query": "payment", "repo_filter": "../outside"}); err == nil {
		t.Error("expected an error for a repository filter that escapes the workspace root")
	}
}

func TestGrepRawFallbackReturnsLiteralMatchWithContextLines(t *testing.T) {
	workspaceRoot := t.TempDir()
	writeWorkspaceFile(t, workspaceRoot, "settlement-service/internal/fee.go", lateFeeFixture)
	tools := newTestTools(openIndexTestStore(t), workspaceRoot)

	result, err := executeTool(t, mustFindTool(t, tools, ToolNameGrepRawFallback), map[string]any{"pattern": "lateFeeRate"})
	if err != nil {
		t.Fatalf("grep_raw_fallback failed: %v", err)
	}
	if len(result.Sources) != 1 {
		t.Fatalf("expected one literal match, got %d", len(result.Sources))
	}

	source := result.Sources[0]
	if source.Repo != "settlement-service" || source.File != "internal/fee.go" {
		t.Errorf("unexpected match location %s/%s", source.Repo, source.File)
	}
	if source.StartLine != 3 || source.EndLine != 5 {
		t.Errorf("expected context lines 3-5, got %d-%d", source.StartLine, source.EndLine)
	}
	if !strings.Contains(result.Content, "lateFeeRate") {
		t.Errorf("expected the matched line in the result:\n%s", result.Content)
	}
}

func TestGrepRawFallbackHonoursRepositoryFilter(t *testing.T) {
	workspaceRoot := t.TempDir()
	writeWorkspaceFile(t, workspaceRoot, "settlement-service/fee.go", lateFeeFixture)
	writeWorkspaceFile(t, workspaceRoot, "payments-service/fee.go", lateFeeFixture)
	tools := newTestTools(openIndexTestStore(t), workspaceRoot)

	result, err := executeTool(t, mustFindTool(t, tools, ToolNameGrepRawFallback), map[string]any{
		"pattern":     "lateFeeRate",
		"repo_filter": "payments-service",
	})
	if err != nil {
		t.Fatalf("grep_raw_fallback failed: %v", err)
	}
	if len(result.Sources) != 1 {
		t.Fatalf("expected one filtered match, got %d", len(result.Sources))
	}
	if result.Sources[0].Repo != "payments-service" {
		t.Errorf("expected the payments-service repository, got %q", result.Sources[0].Repo)
	}
}

func TestGrepRawFallbackSkipsExcludedFiles(t *testing.T) {
	workspaceRoot := t.TempDir()
	writeWorkspaceFile(t, workspaceRoot, "settlement-service/.env", "LATE_FEE_TOKEN=lateFeeRate\n")
	tools := newTestTools(openIndexTestStore(t), workspaceRoot)

	result, err := executeTool(t, mustFindTool(t, tools, ToolNameGrepRawFallback), map[string]any{"pattern": "lateFeeRate"})
	if err != nil {
		t.Fatalf("grep_raw_fallback failed: %v", err)
	}
	if len(result.Sources) != 0 {
		t.Errorf("an excluded file must not be searched, got %+v", result.Sources)
	}
	if !strings.Contains(result.Content, "No literal match") {
		t.Errorf("expected an empty-result message, got:\n%s", result.Content)
	}
}

func TestGrepRawFallbackRejectsInvalidArguments(t *testing.T) {
	workspaceRoot := t.TempDir()
	writeWorkspaceFile(t, workspaceRoot, "settlement-service/fee.go", lateFeeFixture)
	tool := mustFindTool(t, newTestTools(openIndexTestStore(t), workspaceRoot), ToolNameGrepRawFallback)

	if _, err := executeTool(t, tool, map[string]any{"pattern": "   "}); err == nil {
		t.Error("expected an error for a blank pattern")
	}
	if _, err := executeTool(t, tool, map[string]any{"pattern": "lateFeeRate", "repo_filter": "../outside"}); err == nil {
		t.Error("expected an error for a repository filter that escapes the workspace root")
	}
	if _, err := executeTool(t, tool, map[string]any{"pattern": strings.Repeat("a", maxGrepPatternRunes+1)}); err == nil {
		t.Error("expected an error for an oversized pattern")
	}
}
