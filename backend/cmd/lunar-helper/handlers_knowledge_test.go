package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	codeindexapplication "lunar/backend/internal/codeindex/application"
	codeindexdomain "lunar/backend/internal/codeindex/domain"
)

type recordingKnowledgeLoader struct {
	callCount int
	loadErr   error
	glossary  *codeindexdomain.Glossary
}

func (l *recordingKnowledgeLoader) LoadAndIndex(_ context.Context, _ codeindexdomain.IndexStore) (*codeindexdomain.Glossary, error) {
	l.callCount++
	if l.loadErr != nil {
		return nil, l.loadErr
	}
	return l.glossary, nil
}

type stubKnowledgeIndexStore struct {
	codeindexdomain.IndexStore
}

func newKnowledgeHelperServer(loader codeindexapplication.KnowledgeLoader, glossaryPath string) *helperServer {
	return &helperServer{
		indexStore:            stubKnowledgeIndexStore{},
		knowledgeLoader:       loader,
		knowledgeGlossaryPath: glossaryPath,
		logger:                discardLogger(),
		buildContext:          context.Background(),
	}
}

func TestIndexKnowledgeSkipsMissingGlossary(t *testing.T) {
	loader := &recordingKnowledgeLoader{glossary: &codeindexdomain.Glossary{}}
	server := newKnowledgeHelperServer(loader, filepath.Join(t.TempDir(), "missing-glossary.yaml"))

	server.indexKnowledge()

	if loader.callCount != 0 {
		t.Fatalf("expected the loader not to be called for a missing glossary, got %d calls", loader.callCount)
	}
}

func TestIndexKnowledgeLoadsExistingGlossary(t *testing.T) {
	glossaryPath := filepath.Join(t.TempDir(), "glossary.yaml")
	if err := os.WriteFile(glossaryPath, []byte("enums: {}\nnotes: []\n"), 0o600); err != nil {
		t.Fatalf("cannot write the test glossary: %v", err)
	}

	loader := &recordingKnowledgeLoader{glossary: &codeindexdomain.Glossary{
		Enums: map[string]codeindexdomain.GlossaryEnum{"OrderStatus": {}},
		Notes: []codeindexdomain.GlossaryNote{{Term: "Settlement", Explanation: "Funds transfer"}},
	}}
	server := newKnowledgeHelperServer(loader, glossaryPath)

	server.indexKnowledge()

	if loader.callCount != 1 {
		t.Fatalf("expected the loader to be called once for an existing glossary, got %d calls", loader.callCount)
	}
}

func TestIndexKnowledgeSwallowsLoaderError(t *testing.T) {
	glossaryPath := filepath.Join(t.TempDir(), "glossary.yaml")
	if err := os.WriteFile(glossaryPath, []byte("enums: {}\nnotes: []\n"), 0o600); err != nil {
		t.Fatalf("cannot write the test glossary: %v", err)
	}

	loader := &recordingKnowledgeLoader{loadErr: errors.New("corrupt glossary")}
	server := newKnowledgeHelperServer(loader, glossaryPath)

	server.indexKnowledge()

	if loader.callCount != 1 {
		t.Fatalf("expected the loader to be called once before failing, got %d calls", loader.callCount)
	}
}
