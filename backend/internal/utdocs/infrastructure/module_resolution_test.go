package infrastructure

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"lunar/backend/internal/utdocs/domain"
)

func TestRunnerResolvesModuleRequirementsWhenGoTestAsksForIt(t *testing.T) {
	repositoryPath := t.TempDir()
	originalModule := []byte("module example\n\ngo 1.23\n")
	if err := os.WriteFile(filepath.Join(repositoryPath, "go.mod"), originalModule, 0o644); err != nil {
		t.Fatalf("cannot write the fixture go.mod: %v", err)
	}

	calls := make([][]string, 0, 2)
	runner := NewRunnerWithExecutor(func(_ context.Context, dir string, args []string, _ []string) ([]byte, error) {
		calls = append(calls, append([]string{}, args...))
		if len(calls) == 1 {
			return []byte("go: updates to go.mod needed; to update it:\n\tgo mod tidy\n"), errTestCommandFailed
		}
		if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example\n\ngo 1.23\n\nrequire example.com/x v1.0.0\n"), 0o644); err != nil {
			t.Fatalf("cannot update the fixture go.mod: %v", err)
		}
		return []byte(`{"Action":"pass","Package":"example/a","Test":"TestA","Elapsed":0.01}` + "\n"), nil
	})

	result, _, _, err := runner.Run(context.Background(), repositoryPath, []string{"./a"}, []string{"TestA"})
	if err != nil {
		t.Fatalf("expected the run to succeed on the retry, got %v", err)
	}
	if len(calls) != 2 {
		t.Fatalf("expected two executions, got %d: %+v", len(calls), calls)
	}
	if slices.Contains(calls[0], moduleResolutionFlag) {
		t.Fatalf("expected the first attempt without %s, got %v", moduleResolutionFlag, calls[0])
	}
	if !slices.Contains(calls[1], moduleResolutionFlag) {
		t.Fatalf("expected the retry with %s, got %v", moduleResolutionFlag, calls[1])
	}
	if result.Status != domain.RunStatusPassed {
		t.Fatalf("expected a passed result, got %+v", result)
	}

	restored, err := os.ReadFile(filepath.Join(repositoryPath, "go.mod"))
	if err != nil {
		t.Fatalf("cannot read the restored go.mod: %v", err)
	}
	if !bytes.Equal(restored, originalModule) {
		t.Fatalf("expected go.mod to be restored, got %q", restored)
	}
}

func TestNeedsModuleResolutionReadsTheGoTestMessage(t *testing.T) {
	if !needsModuleResolution("go: updates to go.mod needed; to update it: go mod tidy") {
		t.Fatal("expected the go.mod message to be detected")
	}
	if !needsModuleResolution("missing go.sum entry for module") {
		t.Fatal("expected the go.sum message to be detected")
	}
	if needsModuleResolution("--- FAIL: TestA (0.00s)") {
		t.Fatal("expected a normal failure not to be treated as a module problem")
	}
}
