package tools

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	agentdomain "lunar/backend/internal/agent/domain"
	codeindexdomain "lunar/backend/internal/codeindex/domain"
	codeindexinfrastructure "lunar/backend/internal/codeindex/infrastructure"
	sharedDatabase "lunar/backend/internal/shared/database"
)

const indexTestSchema = `
CREATE TABLE IF NOT EXISTS service_nodes (
    repo_name    TEXT PRIMARY KEY,
    language     TEXT NOT NULL DEFAULT '',
    chunk_count  INTEGER NOT NULL DEFAULT 0,
    indexed_at   TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);

CREATE TABLE IF NOT EXISTS service_deps (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    from_service TEXT NOT NULL,
    to_service   TEXT NOT NULL,
    call_type    TEXT NOT NULL DEFAULT '',
    endpoint     TEXT NOT NULL DEFAULT '',
    topic        TEXT NOT NULL DEFAULT '',
    source_file  TEXT NOT NULL DEFAULT '',
    UNIQUE(from_service, to_service, call_type, endpoint, topic)
);

CREATE VIRTUAL TABLE IF NOT EXISTS code_chunks_fts USING fts5(
    repo_name    UNINDEXED,
    file_path    UNINDEXED,
    chunk_type   UNINDEXED,
    raw_content  UNINDEXED,
    start_line   UNINDEXED,
    end_line     UNINDEXED,
    content,
    tokenize = 'porter unicode61'
);

CREATE VIRTUAL TABLE IF NOT EXISTS domain_knowledge_fts USING fts5(
    term,
    explanation,
    tokenize = 'porter unicode61'
);

CREATE TABLE IF NOT EXISTS code_files (
    repo_name      TEXT NOT NULL,
    file_path      TEXT NOT NULL,
    content_hash   TEXT NOT NULL,
    size_bytes     INTEGER NOT NULL,
    modified_unix  INTEGER NOT NULL,
    indexed_at     TEXT NOT NULL,
    PRIMARY KEY (repo_name, file_path)
);
`

type stubGitReader struct {
	workspaceRoot   string
	result          GitDiffResult
	diffError       error
	capturedRequest GitDiffRequest
}

func (s *stubGitReader) WorkspaceRoot() string {
	return s.workspaceRoot
}

func (s *stubGitReader) GetDiff(_ context.Context, request GitDiffRequest) (GitDiffResult, error) {
	s.capturedRequest = request
	if s.diffError != nil {
		return GitDiffResult{}, s.diffError
	}
	return s.result, nil
}

type diffOnlyGitReader struct {
	result GitDiffResult
}

func (d diffOnlyGitReader) GetDiff(_ context.Context, _ GitDiffRequest) (GitDiffResult, error) {
	return d.result, nil
}

func openIndexTestStore(t *testing.T) codeindexdomain.IndexStore {
	t.Helper()

	database, err := sharedDatabase.OpenDB(filepath.Join(t.TempDir(), "index.db"))
	if err != nil {
		t.Fatalf("cannot open the index test database: %v", err)
	}
	t.Cleanup(func() {
		_ = database.Close()
	})

	if _, err := database.Exec(indexTestSchema); err != nil {
		t.Fatalf("cannot apply the index test schema: %v", err)
	}
	return codeindexinfrastructure.NewStore(database)
}

func newTestTools(store codeindexdomain.IndexStore, workspaceRoot string) []agentdomain.Tool {
	return NewTools(store, &stubGitReader{workspaceRoot: workspaceRoot})
}

func mustFindTool(t *testing.T, tools []agentdomain.Tool, name string) agentdomain.Tool {
	t.Helper()

	for _, tool := range tools {
		if tool.Definition().Function.Name == name {
			return tool
		}
	}
	t.Fatalf("tool %q is not registered", name)
	return nil
}

func executeTool(t *testing.T, tool agentdomain.Tool, arguments map[string]any) (agentdomain.ToolResult, error) {
	t.Helper()

	encodedArguments, err := json.Marshal(arguments)
	if err != nil {
		t.Fatalf("cannot encode arguments for %q: %v", tool.Definition().Function.Name, err)
	}
	return tool.Execute(context.Background(), encodedArguments)
}

func findSource(sources []agentdomain.SourceReference, filePath string, sourceType string) *agentdomain.SourceReference {
	for index := range sources {
		if sources[index].File == filePath && sources[index].Type == sourceType {
			return &sources[index]
		}
	}
	return nil
}

func saveExtractedChunks(
	t *testing.T,
	store codeindexdomain.IndexStore,
	repoName string,
	filePath string,
	content string,
	extension string,
) []codeindexdomain.CodeChunk {
	t.Helper()

	chunks, _ := codeindexinfrastructure.ExtractFile(content, filePath, repoName, extension)
	if len(chunks) == 0 {
		t.Fatalf("fixture %s did not produce any chunk", filePath)
	}
	if err := store.SaveChunks(context.Background(), repoName, chunks); err != nil {
		t.Fatalf("cannot save chunks for %s: %v", repoName, err)
	}
	return chunks
}

func writeWorkspaceFile(t *testing.T, workspaceRoot string, relativePath string, content string) {
	t.Helper()

	fullPath := filepath.Join(workspaceRoot, relativePath)
	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		t.Fatalf("cannot create the directory for %s: %v", relativePath, err)
	}
	if err := os.WriteFile(fullPath, []byte(content), 0o600); err != nil {
		t.Fatalf("cannot write %s: %v", relativePath, err)
	}
}
