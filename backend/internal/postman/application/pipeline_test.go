package application

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"lunar/backend/internal/postman/domain"
)

const validCollectionFixture = `{"info":{"name":"atlas API"},"item":[{"name":"manage_user","item":[{"name":"Get level user","request":{"method":"GET","header":[{"key":"Accept","value":"application/json"}],"url":{"raw":"{{baseUrl}}/v1/user/level","host":["{{baseUrl}}"],"path":["v1","user","level"]}}}]}],"variable":[{"key":"baseUrl","value":"http://localhost:8080"}]}`

type stubSelector struct {
	selection domain.Selection
	err       error
}

func (s stubSelector) Select(repoPath string, service string) (domain.Selection, error) {
	return s.selection, s.err
}

type stubClassifier struct {
	classification domain.Classification
}

func (c stubClassifier) Classify(selection domain.Selection) domain.Classification {
	return c.classification
}

type stubGenerator struct {
	content  string
	err      error
	received domain.GenerateRequest
}

func (g *stubGenerator) Generate(ctx context.Context, request domain.GenerateRequest) (string, error) {
	g.received = request
	return g.content, g.err
}

func exportableSelection() domain.Selection {
	return domain.Selection{
		Service: "atlas",
		Files: []domain.SelectedFile{
			{RelativePath: "src/manage_user/routes.go", Content: "package manage_user\n", SizeBytes: 22},
			{RelativePath: "src/wire_gen.go", Content: "package main\n", SizeBytes: 13},
		},
		TotalBytes: 35,
	}
}

func exportableClassification() domain.Classification {
	return domain.Classification{IsExportable: true, Framework: domain.FrameworkGin, UsesWire: true}
}

func exportCredentials() domain.ProviderCredentials {
	return domain.ProviderCredentials{
		Provider: "deepseek",
		APIKey:   "secret-key",
		BaseURL:  "https://api.deepseek.com",
		Model:    "deepseek-chat",
	}
}

func TestPipelineEmitsEveryStageInOrder(t *testing.T) {
	generator := &stubGenerator{content: validCollectionFixture}
	pipeline := NewPipeline(stubSelector{selection: exportableSelection()}, stubClassifier{classification: exportableClassification()}, generator)

	stages := make([]domain.Stage, 0, 4)
	result, err := pipeline.Run(context.Background(), RunRequest{Service: "atlas", RepoPath: "/workspace/atlas", Credentials: exportCredentials()}, func(event domain.StageEvent) error {
		stages = append(stages, event.Stage)
		if strings.TrimSpace(event.Detail) == "" {
			t.Fatalf("every stage event must carry a detail")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("running the pipeline failed: %v", err)
	}

	expected := []domain.Stage{domain.StageSelecting, domain.StagePrompt, domain.StageModel, domain.StageValidating}
	if !reflect.DeepEqual(stages, expected) {
		t.Fatalf("expected the stages %v, received %v", expected, stages)
	}
	if result.Service != "atlas" || result.RequestCount != 1 || result.FolderCount != 1 {
		t.Fatalf("unexpected result counts: %+v", result)
	}
	if result.Collection == nil || result.Collection.Info.Name != "atlas API" {
		t.Fatalf("expected the normalised collection in the result")
	}
	if generator.received.Model != "deepseek-chat" || generator.received.APIKey != "secret-key" {
		t.Fatalf("the resolved credentials must reach the generator, received %+v", generator.received)
	}
	if generator.received.System == "" || generator.received.User == "" {
		t.Fatalf("the prompt must reach the generator")
	}
}

func TestPipelineRefusesAnUnexportableServiceBeforeStreaming(t *testing.T) {
	generator := &stubGenerator{content: validCollectionFixture}
	classification := domain.Classification{Reason: "the repository is a vue front-end application and declares no server framework"}
	pipeline := NewPipeline(stubSelector{selection: exportableSelection()}, stubClassifier{classification: classification}, generator)

	emitted := false
	_, err := pipeline.Run(context.Background(), RunRequest{Service: "fe-mms", RepoPath: "/workspace/fe-mms", Credentials: exportCredentials()}, func(event domain.StageEvent) error {
		emitted = true
		return nil
	})
	if !errors.Is(err, domain.ErrServiceNotExportable) {
		t.Fatalf("expected ErrServiceNotExportable, received %v", err)
	}
	if !strings.Contains(err.Error(), "vue front-end") {
		t.Fatalf("expected the refusal reason, received %q", err.Error())
	}
	if emitted {
		t.Fatalf("no stage may be emitted before the service is accepted")
	}
	if generator.received.Model != "" {
		t.Fatalf("the model must not be called for a refused service")
	}
}

func TestPipelineRefusesWhenASelectionLimitIsExceeded(t *testing.T) {
	selection := exportableSelection()
	selection.ExceededFileLimit = true
	pipeline := NewPipeline(stubSelector{selection: selection}, stubClassifier{classification: exportableClassification()}, &stubGenerator{content: validCollectionFixture})

	_, err := pipeline.Run(context.Background(), RunRequest{Service: "mms", RepoPath: "/workspace/mms"}, func(event domain.StageEvent) error { return nil })
	if !errors.Is(err, domain.ErrServiceNotExportable) || !strings.Contains(err.Error(), "candidate files") {
		t.Fatalf("expected a file limit refusal, received %v", err)
	}
}

func TestPipelineMapsAProviderTimeout(t *testing.T) {
	generator := &stubGenerator{err: fmt.Errorf("%w: the provider did not respond in time", domain.ErrProviderTimeout)}
	pipeline := NewPipeline(stubSelector{selection: exportableSelection()}, stubClassifier{classification: exportableClassification()}, generator)

	_, err := pipeline.Run(context.Background(), RunRequest{Service: "atlas", RepoPath: "/workspace/atlas", Credentials: exportCredentials()}, func(event domain.StageEvent) error { return nil })
	if !errors.Is(err, domain.ErrProviderTimeout) {
		t.Fatalf("expected ErrProviderTimeout, received %v", err)
	}
}

func TestPipelineRefusesAHollowCollectionFromTheModel(t *testing.T) {
	generator := &stubGenerator{content: `{"item":[]}`}
	pipeline := NewPipeline(stubSelector{selection: exportableSelection()}, stubClassifier{classification: exportableClassification()}, generator)

	stages := make([]domain.Stage, 0, 4)
	_, err := pipeline.Run(context.Background(), RunRequest{Service: "atlas", RepoPath: "/workspace/atlas", Credentials: exportCredentials()}, func(event domain.StageEvent) error {
		stages = append(stages, event.Stage)
		return nil
	})
	if !errors.Is(err, domain.ErrCollectionInvalid) {
		t.Fatalf("expected ErrCollectionInvalid, received %v", err)
	}
	if len(stages) != 4 || stages[3] != domain.StageValidating {
		t.Fatalf("expected the validating stage before the refusal, received %v", stages)
	}
}

func snapshotTree(t *testing.T, root string) []string {
	t.Helper()
	entries := make([]string, 0, 8)
	walkErr := filepath.WalkDir(root, func(currentPath string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		relativePath, relErr := filepath.Rel(root, currentPath)
		if relErr != nil {
			return relErr
		}
		entries = append(entries, relativePath)
		return nil
	})
	if walkErr != nil {
		t.Fatalf("cannot snapshot %s: %v", root, walkErr)
	}
	sort.Strings(entries)
	return entries
}

func TestPipelineWritesNothingToTheRepository(t *testing.T) {
	repositoryPath := t.TempDir()
	repositoryFile := filepath.Join(repositoryPath, "routes.go")
	if err := os.WriteFile(repositoryFile, []byte("package main\n"), 0o644); err != nil {
		t.Fatalf("cannot create the repository fixture: %v", err)
	}
	before := snapshotTree(t, repositoryPath)

	selection := exportableSelection()
	selection.Files = []domain.SelectedFile{{RelativePath: "routes.go", Content: "package main\n", SizeBytes: 13}}
	generator := &stubGenerator{content: validCollectionFixture}
	pipeline := NewPipeline(stubSelector{selection: selection}, stubClassifier{classification: exportableClassification()}, generator)

	if _, err := pipeline.Run(context.Background(), RunRequest{Service: "atlas", RepoPath: repositoryPath, Credentials: exportCredentials()}, func(event domain.StageEvent) error { return nil }); err != nil {
		t.Fatalf("running the pipeline failed: %v", err)
	}

	after := snapshotTree(t, repositoryPath)
	if !reflect.DeepEqual(before, after) {
		t.Fatalf("the export must not write to the repository, before %v, after %v", before, after)
	}
}
