package infrastructure

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const realWorkspaceRootEnvironment = "LUNAR_TEST_WORKSPACE_ROOT"

func realRepositoryPath(t *testing.T, name string) string {
	t.Helper()
	workspaceRoot := strings.TrimSpace(os.Getenv(realWorkspaceRootEnvironment))
	if workspaceRoot == "" {
		t.Skipf("set %s to run the real workspace checks", realWorkspaceRootEnvironment)
	}
	candidate := filepath.Join(workspaceRoot, name)
	if _, err := os.Stat(candidate); err != nil {
		t.Skipf("the workspace root does not contain %s", name)
	}
	return candidate
}

func containsVendorDirectory(relativePath string) bool {
	segments := strings.Split(relativePath, "/")
	for index, segment := range segments[:len(segments)-1] {
		if segment != "vendor" {
			continue
		}
		if index > 0 && segments[index-1] == "src" {
			continue
		}
		return true
	}
	return false
}

func TestRealWorkspaceSelectionKeepsFirstPartySourcesOnly(t *testing.T) {
	for _, repositoryName := range []string{"panama", "atlas", "aurora", "merchant-service-laravel", "micro-user-mms", "mocash_api", "service_qris", "fe-mms", "mms"} {
		repositoryPath := realRepositoryPath(t, repositoryName)
		selection, err := NewSelector().Select(repositoryPath, repositoryName)
		if err != nil {
			t.Fatalf("selecting %s failed: %v", repositoryName, err)
		}
		t.Logf("%s: files=%d bytes=%d fileLimit=%v byteLimit=%v", repositoryName, len(selection.Files), selection.TotalBytes, selection.ExceededFileLimit, selection.ExceededByteLimit)

		for _, file := range selection.Files {
			if strings.Contains(file.RelativePath, "_test") || strings.Contains(file.RelativePath, "/mocks/") || strings.Contains(file.RelativePath, ".spec.") {
				t.Fatalf("%s selected a test or mock file: %s", repositoryName, file.RelativePath)
			}
			if strings.Contains(file.RelativePath, "node_modules/") || containsVendorDirectory(file.RelativePath) {
				t.Fatalf("%s selected a dependency file: %s", repositoryName, file.RelativePath)
			}
		}

		classification := NewClassifier().Classify(selection)
		t.Logf("%s: exportable=%v framework=%q wire=%v reason=%q", repositoryName, classification.IsExportable, classification.Framework, classification.UsesWire, classification.Reason)
	}
}

func TestRealWorkspacePanamaKeepsFirstPartyVendor(t *testing.T) {
	selection, err := NewSelector().Select(realRepositoryPath(t, "panama"), "panama")
	if err != nil {
		t.Fatalf("selecting panama failed: %v", err)
	}

	found := false
	for _, file := range selection.Files {
		if strings.HasPrefix(file.RelativePath, "src/vendor/") {
			found = true
		}
	}
	if !found {
		t.Fatalf("panama must include src/vendor")
	}
}

func TestRealWorkspaceRefusesKnownUnsupportedRepositories(t *testing.T) {
	codeIgniterSelection, err := NewSelector().Select(realRepositoryPath(t, "mms"), "mms")
	if err != nil {
		t.Fatalf("selecting mms failed: %v", err)
	}
	codeIgniter := NewClassifier().Classify(codeIgniterSelection)
	if codeIgniter.IsExportable || !strings.Contains(codeIgniter.Reason, "CodeIgniter") {
		t.Fatalf("mms must be refused as a CodeIgniter monolith, received %q", codeIgniter.Reason)
	}

	frontendSelection, err := NewSelector().Select(realRepositoryPath(t, "fe-mms"), "fe-mms")
	if err != nil {
		t.Fatalf("selecting fe-mms failed: %v", err)
	}
	frontend := NewClassifier().Classify(frontendSelection)
	if frontend.IsExportable || !strings.Contains(frontend.Reason, "vue") {
		t.Fatalf("fe-mms must be refused as a vue front-end, received %q", frontend.Reason)
	}
}
