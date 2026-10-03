package application

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	codeindexdomain "lunar/backend/internal/codeindex/domain"
)

func TestIndexerIndexesRepositoryOnFirstRun(t *testing.T) {
	rootPath, _ := createRepositoryFixture(t, "micro-payment", map[string]string{
		"handler/routes.go": routeFixture,
		"kafka/producer.go": producerFixture,
	})
	store := newFakeIndexStore()

	result, err := NewIndexer(store).IndexRepository(context.Background(), rootPath, "micro-payment")
	if err != nil {
		t.Fatalf("IndexRepository: %v", err)
	}

	if result.RepoName != "micro-payment" {
		t.Errorf("unexpected repo name %q", result.RepoName)
	}
	if result.IndexedFiles != 2 || result.UnchangedFiles != 0 {
		t.Errorf("expected 2 indexed files, got indexed=%d unchanged=%d", result.IndexedFiles, result.UnchangedFiles)
	}
	if result.ChunkCount == 0 {
		t.Error("expected chunks to be counted")
	}
	if result.DependencyCount != 1 {
		t.Errorf("expected 1 dependency, got %d", result.DependencyCount)
	}
	if store.saveChunksCallCount != 0 {
		t.Errorf("expected the incremental path to avoid SaveChunks, got %d calls", store.saveChunksCallCount)
	}
	if len(store.states) != 2 {
		t.Fatalf("expected 2 tracked file states, got %d", len(store.states))
	}

	routeState := store.states["handler/routes.go"]
	if routeState.ContentHash == "" || routeState.SizeBytes <= 0 || routeState.ModifiedUnix <= 0 {
		t.Errorf("expected populated file fingerprint, got %+v", routeState)
	}

	for _, chunk := range store.chunksByFile["handler/routes.go"] {
		if chunk.ChunkType != string(codeindexdomain.ChunkTypeRoute) {
			continue
		}
		if chunk.StartLine != 4 || chunk.EndLine != 4 {
			t.Errorf("expected route chunk on line 4, got %d-%d", chunk.StartLine, chunk.EndLine)
		}
		return
	}
	t.Fatal("expected a stored route chunk for handler/routes.go")
}

func TestIndexerSkipsUnchangedFilesOnSecondRun(t *testing.T) {
	rootPath, _ := createRepositoryFixture(t, "micro-payment", map[string]string{
		"handler/routes.go": routeFixture,
		"kafka/producer.go": producerFixture,
	})
	store := newFakeIndexStore()
	indexer := NewIndexer(store)

	if _, err := indexer.IndexRepository(context.Background(), rootPath, "micro-payment"); err != nil {
		t.Fatalf("first IndexRepository: %v", err)
	}
	result, err := indexer.IndexRepository(context.Background(), rootPath, "micro-payment")
	if err != nil {
		t.Fatalf("second IndexRepository: %v", err)
	}

	if result.IndexedFiles != 0 {
		t.Errorf("expected no re-chunked files, got %d", result.IndexedFiles)
	}
	if result.UnchangedFiles != 2 {
		t.Errorf("expected 2 unchanged files, got %d", result.UnchangedFiles)
	}
	if len(store.lastUpdates) != 0 {
		t.Errorf("expected no updates on unchanged run, got %d", len(store.lastUpdates))
	}
	if store.applyCallCount != 2 {
		t.Errorf("expected ApplyIndex on both runs, got %d", store.applyCallCount)
	}
}

func TestIndexerRechunksOnlyChangedFile(t *testing.T) {
	rootPath, repoRoot := createRepositoryFixture(t, "micro-payment", map[string]string{
		"handler/routes.go": routeFixture,
		"kafka/producer.go": producerFixture,
	})
	store := newFakeIndexStore()
	indexer := NewIndexer(store)

	if _, err := indexer.IndexRepository(context.Background(), rootPath, "micro-payment"); err != nil {
		t.Fatalf("first IndexRepository: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoRoot, "handler", "routes.go"), []byte(extendedRouteFixture), 0o600); err != nil {
		t.Fatalf("modify routes fixture: %v", err)
	}

	result, err := indexer.IndexRepository(context.Background(), rootPath, "micro-payment")
	if err != nil {
		t.Fatalf("second IndexRepository: %v", err)
	}

	if result.IndexedFiles != 1 || result.UnchangedFiles != 1 {
		t.Errorf("expected 1 changed and 1 unchanged file, got indexed=%d unchanged=%d", result.IndexedFiles, result.UnchangedFiles)
	}
	if len(store.lastUpdates) != 1 || store.lastUpdates[0].State.FilePath != "handler/routes.go" {
		t.Fatalf("expected only handler/routes.go to be re-chunked, got %+v", store.lastUpdates)
	}

	var hasPostRoute bool
	for _, chunk := range store.chunksByFile["handler/routes.go"] {
		if strings.Contains(chunk.RawContent, `router.POST("/api/payments"`) {
			hasPostRoute = true
		}
	}
	if !hasPostRoute {
		t.Error("expected re-chunked content to contain the new POST route")
	}
}

func TestIndexerRemovesDeletedFiles(t *testing.T) {
	rootPath, repoRoot := createRepositoryFixture(t, "micro-payment", map[string]string{
		"handler/routes.go": routeFixture,
		"kafka/producer.go": producerFixture,
	})
	store := newFakeIndexStore()
	indexer := NewIndexer(store)

	if _, err := indexer.IndexRepository(context.Background(), rootPath, "micro-payment"); err != nil {
		t.Fatalf("first IndexRepository: %v", err)
	}
	if err := os.Remove(filepath.Join(repoRoot, "handler", "routes.go")); err != nil {
		t.Fatalf("remove routes fixture: %v", err)
	}

	result, err := indexer.IndexRepository(context.Background(), rootPath, "micro-payment")
	if err != nil {
		t.Fatalf("second IndexRepository: %v", err)
	}

	if result.RemovedFiles != 1 {
		t.Fatalf("expected 1 removed file, got %d", result.RemovedFiles)
	}
	if len(store.lastRemovals) != 1 || store.lastRemovals[0] != "handler/routes.go" {
		t.Errorf("expected handler/routes.go removal, got %v", store.lastRemovals)
	}
	if _, isTracked := store.states["handler/routes.go"]; isTracked {
		t.Error("expected removed file state to be dropped")
	}
}

func TestIndexerRedactsSecretsBeforeExtraction(t *testing.T) {
	rootPath, _ := createRepositoryFixture(t, "micro-config", map[string]string{
		"config/config.go": secretFixture,
	})
	store := newFakeIndexStore()

	if _, err := NewIndexer(store).IndexRepository(context.Background(), rootPath, "micro-config"); err != nil {
		t.Fatalf("IndexRepository: %v", err)
	}

	storedChunks := store.chunksByFile["config/config.go"]
	if len(storedChunks) == 0 {
		t.Fatal("expected stored chunks for config/config.go")
	}

	var hasRedactedAssignment bool
	for _, chunk := range storedChunks {
		if strings.Contains(chunk.RawContent, "sk-supersecretvalue") {
			t.Errorf("stored chunk leaked the secret value: %q", chunk.RawContent)
		}
		if strings.Contains(chunk.Content, "sk-supersecretvalue") {
			t.Errorf("indexed content leaked the secret value: %q", chunk.Content)
		}
		if strings.Contains(chunk.RawContent, `var apiKey = "***"`) {
			hasRedactedAssignment = true
		}
	}
	if !hasRedactedAssignment {
		t.Errorf("expected redacted assignment in stored chunks, got %+v", storedChunks)
	}
}

func TestIndexerSkipsExcludedAndBinaryFiles(t *testing.T) {
	rootPath, _ := createRepositoryFixture(t, "micro-payment", map[string]string{
		"src/main.go": routeFixture,
		".env":        "API_KEY=abcdefghijklmnop\n",
		"src/blob.go": "package main\x00binary payload\n",
	})
	store := newFakeIndexStore()

	result, err := NewIndexer(store).IndexRepository(context.Background(), rootPath, "micro-payment")
	if err != nil {
		t.Fatalf("IndexRepository: %v", err)
	}

	if result.IndexedFiles != 1 {
		t.Errorf("expected only src/main.go to be indexed, got %d", result.IndexedFiles)
	}
	if result.SkippedFiles != 1 {
		t.Errorf("expected 1 skipped binary file, got %d", result.SkippedFiles)
	}
	if _, isTracked := store.states[".env"]; isTracked {
		t.Error("expected .env to never be tracked")
	}
	if _, isTracked := store.states["src/blob.go"]; isTracked {
		t.Error("expected binary file to never be tracked")
	}
}

func TestIndexerRejectsEscapingRepositoryName(t *testing.T) {
	rootPath, _ := createRepositoryFixture(t, "micro-payment", map[string]string{
		"src/main.go": routeFixture,
	})

	if _, err := NewIndexer(newFakeIndexStore()).IndexRepository(context.Background(), rootPath, "../micro-payment"); err == nil {
		t.Fatal("expected an error for a path-traversing repository name")
	}
}

func TestIndexerIndexesWorkspaceRepositories(t *testing.T) {
	rootPath := t.TempDir()
	for _, repoName := range []string{"alpha-service", "beta-service", "node_modules", "invalid name"} {
		if err := os.MkdirAll(filepath.Join(rootPath, repoName, "src"), 0o700); err != nil {
			t.Fatalf("create %s: %v", repoName, err)
		}
	}
	for _, repoName := range []string{"alpha-service", "beta-service"} {
		if err := os.WriteFile(filepath.Join(rootPath, repoName, "src", "main.go"), []byte(routeFixture), 0o600); err != nil {
			t.Fatalf("write %s fixture: %v", repoName, err)
		}
	}

	results, err := NewIndexer(newFakeIndexStore()).IndexWorkspace(context.Background(), rootPath, nil)
	if err != nil {
		t.Fatalf("IndexWorkspace: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 indexed repositories, got %d", len(results))
	}
	if results[0].RepoName != "alpha-service" || results[1].RepoName != "beta-service" {
		t.Errorf("unexpected repository order %v", results)
	}
}
