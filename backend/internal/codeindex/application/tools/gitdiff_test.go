package tools

import (
	"errors"
	"strings"
	"testing"
)

func TestGetGitDiffReturnsMetadataChangedFilesAndSource(t *testing.T) {
	reader := &stubGitReader{
		workspaceRoot: t.TempDir(),
		result: GitDiffResult{
			RepositoryName: "payments-service",
			Branch:         "feat/payments",
			HeadSHA:        "abc123",
			BaseRef:        "HEAD",
			ChangedFiles:   []string{"internal/handler/routes.go"},
			UntrackedFiles: []string{"internal/handler/new.go"},
			Diff:           "diff --git a/internal/handler/routes.go b/internal/handler/routes.go\n+router.GET(\"/api/payments\", getPayments)",
		},
	}
	tools := NewTools(openIndexTestStore(t), reader)

	result, err := executeTool(t, mustFindTool(t, tools, ToolNameGetGitDiff), map[string]any{"repo_name": "payments-service"})
	if err != nil {
		t.Fatalf("get_git_diff failed: %v", err)
	}
	for _, expected := range []string{
		"branch: feat/payments",
		"HEAD: abc123",
		"internal/handler/routes.go",
		"Untracked files (content not shown)",
		"internal/handler/new.go",
		`+router.GET("/api/payments", getPayments)`,
	} {
		if !strings.Contains(result.Content, expected) {
			t.Errorf("expected %q in the git diff result:\n%s", expected, result.Content)
		}
	}
	if !reader.capturedRequest.IncludeUncommitted {
		t.Error("include_uncommitted must default to true")
	}
	if reader.capturedRequest.RepositoryName != "payments-service" {
		t.Errorf("expected the repository name to be forwarded, got %q", reader.capturedRequest.RepositoryName)
	}
	if len(result.Sources) != 1 {
		t.Fatalf("expected one source, got %d", len(result.Sources))
	}
	if result.Sources[0].Type != "git_diff" || result.Sources[0].Repo != "payments-service" {
		t.Errorf("unexpected git diff source %+v", result.Sources[0])
	}
}

func TestGetGitDiffForwardsArgumentsAndBoundsDiff(t *testing.T) {
	reader := &stubGitReader{
		workspaceRoot: t.TempDir(),
		result: GitDiffResult{
			RepositoryName: "payments-service",
			BaseRef:        "main",
			Diff:           strings.Repeat("+line\n", 20000),
			IsTruncated:    true,
		},
	}
	tools := NewTools(openIndexTestStore(t), reader)

	result, err := executeTool(t, mustFindTool(t, tools, ToolNameGetGitDiff), map[string]any{
		"repo_name":           "payments-service",
		"base_ref":            "main",
		"include_uncommitted": false,
		"file_path":           "internal/handler/routes.go",
	})
	if err != nil {
		t.Fatalf("get_git_diff failed: %v", err)
	}
	if reader.capturedRequest.BaseRef != "main" || reader.capturedRequest.IncludeUncommitted {
		t.Errorf("expected the requested diff scope, got %+v", reader.capturedRequest)
	}
	if reader.capturedRequest.FilePath != "internal/handler/routes.go" {
		t.Errorf("expected the requested file path, got %q", reader.capturedRequest.FilePath)
	}
	if !strings.Contains(result.Content, "the diff was truncated") {
		t.Errorf("expected a truncation note:\n%s", result.Content)
	}
	if len(result.Content) > maxGitDiffCharacters {
		t.Errorf("result has %d characters, exceeding the diff bound", len(result.Content))
	}
}

func TestGetGitDiffRejectsUntrustedArguments(t *testing.T) {
	reader := &stubGitReader{workspaceRoot: t.TempDir(), result: GitDiffResult{RepositoryName: "payments-service"}}
	tool := mustFindTool(t, NewTools(openIndexTestStore(t), reader), ToolNameGetGitDiff)

	cases := []map[string]any{
		{"repo_name": "../payments-service"},
		{"repo_name": "payments-service", "base_ref": "--upload-pack=touch /tmp/pwned"},
		{"repo_name": "payments-service", "base_ref": "main..other"},
		{"repo_name": "payments-service", "file_path": "../secret.go"},
		{"repo_name": "payments-service", "file_path": ".env"},
	}
	for _, arguments := range cases {
		if _, err := executeTool(t, tool, arguments); err == nil {
			t.Errorf("expected arguments %v to be rejected", arguments)
		}
	}
}

func TestGetGitDiffReportsStoreAndReaderFailures(t *testing.T) {
	tools := NewTools(openIndexTestStore(t), nil)
	if _, err := executeTool(t, mustFindTool(t, tools, ToolNameGetGitDiff), map[string]any{"repo_name": "payments-service"}); err == nil {
		t.Error("expected an error when the git reader is not configured")
	} else if !strings.Contains(err.Error(), "git reader is not configured") {
		t.Errorf("git reader failure returned an unclear error: %v", err)
	}

	failingReader := &stubGitReader{workspaceRoot: t.TempDir(), diffError: errors.New("git is unavailable")}
	failingTools := NewTools(openIndexTestStore(t), failingReader)
	_, err := executeTool(t, mustFindTool(t, failingTools, ToolNameGetGitDiff), map[string]any{"repo_name": "payments-service"})
	if err == nil {
		t.Fatal("expected the reader failure to be propagated")
	}
	if !strings.Contains(err.Error(), "git is unavailable") {
		t.Errorf("reader failure lost its cause: %v", err)
	}
}
