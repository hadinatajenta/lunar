package application

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	codeindexdomain "lunar/backend/internal/codeindex/domain"
	codeindexinfrastructure "lunar/backend/internal/codeindex/infrastructure"
)

const routeFixture = `package handler

func RegisterRoutes(router *gin.Engine) {
	router.GET("/api/payments", getPayments)
}
`

const extendedRouteFixture = `package handler

func RegisterRoutes(router *gin.Engine) {
	router.GET("/api/payments", getPayments)
	router.POST("/api/payments", createPayment)
}
`

const producerFixture = `package kafka

func publishPayment(writer *kafka.Writer) error {
	return writer.WriteMessages(ctx, kafka.Message{Topic: "payment-events"})
}
`

const secretFixture = `package config

func loadAPIKey() string {
	var apiKey = "sk-supersecretvalue"
	return apiKey
}
`

type fakeIndexStore struct {
	states              map[string]codeindexinfrastructure.FileState
	chunksByFile        map[string][]codeindexdomain.CodeChunk
	lastUpdates         []codeindexinfrastructure.FileUpdate
	lastRemovals        []string
	applyCallCount      int
	saveChunksCallCount int
}

var _ codeindexdomain.IndexStore = (*fakeIndexStore)(nil)

func newFakeIndexStore() *fakeIndexStore {
	return &fakeIndexStore{
		states:       map[string]codeindexinfrastructure.FileState{},
		chunksByFile: map[string][]codeindexdomain.CodeChunk{},
	}
}

func (f *fakeIndexStore) SaveChunks(ctx context.Context, repoName string, chunks []codeindexdomain.CodeChunk) error {
	f.saveChunksCallCount++
	for _, chunk := range chunks {
		f.chunksByFile[chunk.FilePath] = append(f.chunksByFile[chunk.FilePath], chunk)
	}
	return nil
}

func (f *fakeIndexStore) SearchChunks(ctx context.Context, query string, repoFilter string, limit int) ([]codeindexdomain.CodeChunk, error) {
	return nil, nil
}

func (f *fakeIndexStore) SearchChunksIn(ctx context.Context, query string, repoFilter []string, limit int) ([]codeindexdomain.CodeChunk, error) {
	return nil, nil
}

func (f *fakeIndexStore) GetServiceDependencies(ctx context.Context, serviceName string) ([]codeindexdomain.ServiceDependency, error) {
	return nil, nil
}

func (f *fakeIndexStore) SaveDependencies(ctx context.Context, repoName string, dependencies []codeindexdomain.ServiceDependency) error {
	return nil
}

func (f *fakeIndexStore) GetIndexStatus(ctx context.Context) ([]codeindexdomain.IndexStatus, error) {
	return nil, nil
}

func (f *fakeIndexStore) GetAllServices(ctx context.Context) ([]codeindexdomain.ServiceNode, error) {
	return nil, nil
}

func (f *fakeIndexStore) ClearRepo(ctx context.Context, repoName string) error {
	return nil
}

func (f *fakeIndexStore) SearchKnowledge(ctx context.Context, query string, limit int) ([]codeindexdomain.DomainKnowledgeItem, error) {
	return nil, nil
}

func (f *fakeIndexStore) SaveKnowledge(ctx context.Context, items []codeindexdomain.DomainKnowledgeItem) error {
	return nil
}

func (f *fakeIndexStore) ClearKnowledge(ctx context.Context) error {
	return nil
}

func (f *fakeIndexStore) Close() error {
	return nil
}

func (f *fakeIndexStore) LoadFileStates(ctx context.Context, repoName string) (map[string]codeindexinfrastructure.FileState, error) {
	loadedStates := make(map[string]codeindexinfrastructure.FileState, len(f.states))
	for filePath, fileState := range f.states {
		if fileState.RepoName == repoName {
			loadedStates[filePath] = fileState
		}
	}
	return loadedStates, nil
}

func (f *fakeIndexStore) ApplyIndex(ctx context.Context, repoName string, updates []codeindexinfrastructure.FileUpdate, removedPaths []string) error {
	f.applyCallCount++
	f.lastUpdates = updates
	f.lastRemovals = removedPaths

	for _, removedPath := range removedPaths {
		delete(f.states, removedPath)
		delete(f.chunksByFile, removedPath)
	}
	for _, update := range updates {
		f.states[update.State.FilePath] = update.State
		f.chunksByFile[update.State.FilePath] = update.Chunks
	}
	return nil
}

func createRepositoryFixture(t *testing.T, repoName string, files map[string]string) (string, string) {
	t.Helper()

	rootPath := t.TempDir()
	repoRoot := filepath.Join(rootPath, repoName)
	for relativePath, content := range files {
		absolutePath := filepath.Join(repoRoot, filepath.FromSlash(relativePath))
		if err := os.MkdirAll(filepath.Dir(absolutePath), 0o700); err != nil {
			t.Fatalf("create directory for %s: %v", relativePath, err)
		}
		if err := os.WriteFile(absolutePath, []byte(content), 0o600); err != nil {
			t.Fatalf("write %s: %v", relativePath, err)
		}
	}
	return rootPath, repoRoot
}
