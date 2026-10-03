package application

import (
	"context"
	"testing"

	codeindexdomain "lunar/backend/internal/codeindex/domain"
)

type plainIndexStore struct {
	savedChunks       []codeindexdomain.CodeChunk
	savedDependencies []codeindexdomain.ServiceDependency
	clearedRepos      []string
}

var _ codeindexdomain.IndexStore = (*plainIndexStore)(nil)

func (p *plainIndexStore) SaveChunks(ctx context.Context, repoName string, chunks []codeindexdomain.CodeChunk) error {
	p.savedChunks = append(p.savedChunks, chunks...)
	return nil
}

func (p *plainIndexStore) SearchChunks(ctx context.Context, query string, repoFilter string, limit int) ([]codeindexdomain.CodeChunk, error) {
	return nil, nil
}

func (p *plainIndexStore) SearchChunksIn(ctx context.Context, query string, repoFilter []string, limit int) ([]codeindexdomain.CodeChunk, error) {
	return nil, nil
}

func (p *plainIndexStore) GetServiceDependencies(ctx context.Context, serviceName string) ([]codeindexdomain.ServiceDependency, error) {
	return nil, nil
}

func (p *plainIndexStore) SaveDependencies(ctx context.Context, repoName string, dependencies []codeindexdomain.ServiceDependency) error {
	p.savedDependencies = append(p.savedDependencies, dependencies...)
	return nil
}

func (p *plainIndexStore) GetIndexStatus(ctx context.Context) ([]codeindexdomain.IndexStatus, error) {
	return nil, nil
}

func (p *plainIndexStore) GetAllServices(ctx context.Context) ([]codeindexdomain.ServiceNode, error) {
	return nil, nil
}

func (p *plainIndexStore) ClearRepo(ctx context.Context, repoName string) error {
	p.clearedRepos = append(p.clearedRepos, repoName)
	return nil
}

func (p *plainIndexStore) SearchKnowledge(ctx context.Context, query string, limit int) ([]codeindexdomain.DomainKnowledgeItem, error) {
	return nil, nil
}

func (p *plainIndexStore) SaveKnowledge(ctx context.Context, items []codeindexdomain.DomainKnowledgeItem) error {
	return nil
}

func (p *plainIndexStore) ClearKnowledge(ctx context.Context) error {
	return nil
}

func (p *plainIndexStore) Close() error {
	return nil
}

func TestIndexerFallsBackToFullRebuildWithoutIncrementalStore(t *testing.T) {
	rootPath, _ := createRepositoryFixture(t, "micro-payment", map[string]string{
		"handler/routes.go": routeFixture,
		"kafka/producer.go": producerFixture,
	})
	store := &plainIndexStore{}
	indexer := NewIndexer(store)

	result, err := indexer.IndexRepository(context.Background(), rootPath, "micro-payment")
	if err != nil {
		t.Fatalf("IndexRepository: %v", err)
	}

	if result.IndexedFiles != 2 {
		t.Errorf("expected 2 indexed files, got %d", result.IndexedFiles)
	}
	if len(store.clearedRepos) != 1 || store.clearedRepos[0] != "micro-payment" {
		t.Errorf("expected the repository to be cleared before the full rebuild, got %v", store.clearedRepos)
	}
	if len(store.savedChunks) == 0 {
		t.Fatal("expected chunks to be saved through the frozen interface")
	}
	for _, chunk := range store.savedChunks {
		if chunk.FilePath == "" {
			t.Errorf("expected every saved chunk to carry a file path, got %+v", chunk)
		}
	}
	if len(store.savedDependencies) != 1 {
		t.Errorf("expected 1 saved dependency, got %d", len(store.savedDependencies))
	}

	secondResult, err := indexer.IndexRepository(context.Background(), rootPath, "micro-payment")
	if err != nil {
		t.Fatalf("second IndexRepository: %v", err)
	}
	if secondResult.IndexedFiles != 2 {
		t.Errorf("expected the fallback path to rebuild every run, got %d indexed files", secondResult.IndexedFiles)
	}
	if len(store.clearedRepos) != 2 {
		t.Errorf("expected a second clear before rebuild, got %v", store.clearedRepos)
	}
}
