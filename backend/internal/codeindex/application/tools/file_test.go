package tools

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeNumberedFile(t *testing.T, workspaceRoot string, relativePath string, lineCount int) {
	t.Helper()

	var content strings.Builder
	for index := 1; index <= lineCount; index++ {
		fmt.Fprintf(&content, "line %03d\n", index)
	}
	writeWorkspaceFile(t, workspaceRoot, relativePath, content.String())
}

func TestGetFileContentClampsRequestedLineRange(t *testing.T) {
	workspaceRoot := t.TempDir()
	writeNumberedFile(t, workspaceRoot, "payments-service/handler.go", 200)
	tools := newTestTools(openIndexTestStore(t), workspaceRoot)

	result, err := executeTool(t, mustFindTool(t, tools, ToolNameGetFileContent), map[string]any{
		"repo_name":  "payments-service",
		"file_path":  "handler.go",
		"start_line": 10,
		"end_line":   1000,
	})
	if err != nil {
		t.Fatalf("get_file_content failed: %v", err)
	}
	if len(result.Sources) != 1 {
		t.Fatalf("expected one source, got %d", len(result.Sources))
	}

	source := result.Sources[0]
	if source.StartLine != 10 || source.EndLine != 130 {
		t.Errorf("expected the clamped range 10-130, got %d-%d", source.StartLine, source.EndLine)
	}
	if !strings.Contains(result.Content, "  10 | line 010") {
		t.Errorf("expected the first requested line in the result:\n%s", result.Content)
	}
	if !strings.Contains(result.Content, " 130 | line 130") {
		t.Errorf("expected the last windowed line in the result:\n%s", result.Content)
	}
	if strings.Contains(result.Content, "line 131") {
		t.Errorf("the result must not read beyond the window:\n%s", result.Content)
	}
	if !strings.Contains(result.Content, "of 200") {
		t.Errorf("expected the total line count in the result:\n%s", result.Content)
	}
}

func TestGetFileContentUsesDefaultWindowWhenEndLineIsMissing(t *testing.T) {
	workspaceRoot := t.TempDir()
	writeNumberedFile(t, workspaceRoot, "payments-service/handler.go", 200)
	tools := newTestTools(openIndexTestStore(t), workspaceRoot)

	result, err := executeTool(t, mustFindTool(t, tools, ToolNameGetFileContent), map[string]any{
		"repo_name":  "payments-service",
		"file_path":  "handler.go",
		"start_line": 1,
	})
	if err != nil {
		t.Fatalf("get_file_content failed: %v", err)
	}
	if len(result.Sources) != 1 {
		t.Fatalf("expected one source, got %d", len(result.Sources))
	}
	if result.Sources[0].StartLine != 1 || result.Sources[0].EndLine != 81 {
		t.Errorf("expected the default window 1-81, got %d-%d", result.Sources[0].StartLine, result.Sources[0].EndLine)
	}
}

func TestGetFileContentRejectsPathsOutsideRepository(t *testing.T) {
	workspaceRoot := t.TempDir()
	writeNumberedFile(t, workspaceRoot, "payments-service/handler.go", 5)
	tool := mustFindTool(t, newTestTools(openIndexTestStore(t), workspaceRoot), ToolNameGetFileContent)

	for _, filePath := range []string{"../outside.go", "../../etc/passwd", "/etc/passwd", "..", "."} {
		_, err := executeTool(t, tool, map[string]any{"repo_name": "payments-service", "file_path": filePath})
		if err == nil {
			t.Errorf("expected path %q to be rejected", filePath)
			continue
		}
		if !strings.Contains(err.Error(), "escapes the repository root") {
			t.Errorf("path %q returned an unclear error: %v", filePath, err)
		}
	}
}

func TestGetFileContentRejectsSymlinkEscape(t *testing.T) {
	workspaceRoot := t.TempDir()
	writeNumberedFile(t, workspaceRoot, "payments-service/handler.go", 5)

	outsideFile := filepath.Join(t.TempDir(), "outside.txt")
	if err := os.WriteFile(outsideFile, []byte("outside content\n"), 0o600); err != nil {
		t.Fatalf("cannot write the outside file: %v", err)
	}
	linkPath := filepath.Join(workspaceRoot, "payments-service", "linked.txt")
	if err := os.Symlink(outsideFile, linkPath); err != nil {
		t.Skipf("cannot create a symlink in this environment: %v", err)
	}

	tool := mustFindTool(t, newTestTools(openIndexTestStore(t), workspaceRoot), ToolNameGetFileContent)
	_, err := executeTool(t, tool, map[string]any{"repo_name": "payments-service", "file_path": "linked.txt"})
	if err == nil {
		t.Fatal("expected a symlink target outside the repository to be rejected")
	}
	if !strings.Contains(err.Error(), "escapes the repository root") {
		t.Errorf("symlink escape returned an unclear error: %v", err)
	}
}

func TestGetFileContentRejectsExcludedFileNames(t *testing.T) {
	workspaceRoot := t.TempDir()
	writeWorkspaceFile(t, workspaceRoot, "payments-service/.env", "PAYMENT_TOKEN=abc\n")
	tool := mustFindTool(t, newTestTools(openIndexTestStore(t), workspaceRoot), ToolNameGetFileContent)

	_, err := executeTool(t, tool, map[string]any{"repo_name": "payments-service", "file_path": ".env"})
	if err == nil {
		t.Fatal("expected an excluded file name to be rejected")
	}
	if !strings.Contains(err.Error(), "excluded from reads") {
		t.Errorf("excluded file returned an unclear error: %v", err)
	}
}

func TestGetFileContentRejectsInvalidRepositoryName(t *testing.T) {
	workspaceRoot := t.TempDir()
	tool := mustFindTool(t, newTestTools(openIndexTestStore(t), workspaceRoot), ToolNameGetFileContent)

	_, err := executeTool(t, tool, map[string]any{"repo_name": "../payments-service", "file_path": "handler.go"})
	if err == nil {
		t.Fatal("expected an invalid repository name to be rejected")
	}
	if !strings.Contains(err.Error(), "invalid repository name") {
		t.Errorf("invalid repository name returned an unclear error: %v", err)
	}
}

func TestGetFileContentRejectsBinaryFiles(t *testing.T) {
	workspaceRoot := t.TempDir()
	writeWorkspaceFile(t, workspaceRoot, "payments-service/data.go", "package data\x00\x01binary")
	tool := mustFindTool(t, newTestTools(openIndexTestStore(t), workspaceRoot), ToolNameGetFileContent)

	_, err := executeTool(t, tool, map[string]any{"repo_name": "payments-service", "file_path": "data.go"})
	if err == nil {
		t.Fatal("expected a binary file to be rejected")
	}
	if !strings.Contains(err.Error(), "not a text file") {
		t.Errorf("binary file returned an unclear error: %v", err)
	}
}

func TestGetFileContentReportsEmptyFile(t *testing.T) {
	workspaceRoot := t.TempDir()
	writeWorkspaceFile(t, workspaceRoot, "payments-service/empty.go", "")
	tools := newTestTools(openIndexTestStore(t), workspaceRoot)

	result, err := executeTool(t, mustFindTool(t, tools, ToolNameGetFileContent), map[string]any{
		"repo_name": "payments-service",
		"file_path": "empty.go",
	})
	if err != nil {
		t.Fatalf("get_file_content failed: %v", err)
	}
	if !strings.Contains(result.Content, "is empty") {
		t.Errorf("expected an empty-file message, got:\n%s", result.Content)
	}
}

func TestGetFileContentTruncatesOversizedFile(t *testing.T) {
	workspaceRoot := t.TempDir()
	line := strings.Repeat("x", 100) + "\n"
	content := strings.Repeat(line, (maxFileReadBytes/len(line))+200)
	writeWorkspaceFile(t, workspaceRoot, "payments-service/big.txt", content)

	tools := newTestTools(openIndexTestStore(t), workspaceRoot)
	result, err := executeTool(t, mustFindTool(t, tools, ToolNameGetFileContent), map[string]any{
		"repo_name":  "payments-service",
		"file_path":  "big.txt",
		"start_line": 1,
		"end_line":   1,
	})
	if err != nil {
		t.Fatalf("get_file_content failed: %v", err)
	}
	if !strings.Contains(result.Content, "truncated at") {
		t.Errorf("expected a read truncation note in the result:\n%s", result.Content)
	}
	if len(result.Content) > maxFileContentChars+len(fileTruncationNote) {
		t.Errorf("result has %d characters, exceeding the bounded output", len(result.Content))
	}
}
