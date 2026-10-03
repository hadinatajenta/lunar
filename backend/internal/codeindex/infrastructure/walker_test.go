package infrastructure

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func createFixtureTree(t *testing.T, files map[string]string) string {
	t.Helper()

	repoRoot := filepath.Join(t.TempDir(), "sample-service")
	for relativePath, content := range files {
		absolutePath := filepath.Join(repoRoot, filepath.FromSlash(relativePath))
		if err := os.MkdirAll(filepath.Dir(absolutePath), 0o700); err != nil {
			t.Fatalf("create directory for %s: %v", relativePath, err)
		}
		if err := os.WriteFile(absolutePath, []byte(content), 0o600); err != nil {
			t.Fatalf("write %s: %v", relativePath, err)
		}
	}
	return repoRoot
}

func walkedPaths(t *testing.T, repoRoot string) []string {
	t.Helper()

	sourceFiles, err := WalkRepository(repoRoot, "sample-service")
	if err != nil {
		t.Fatalf("WalkRepository: %v", err)
	}

	paths := make([]string, 0, len(sourceFiles))
	for _, sourceFile := range sourceFiles {
		paths = append(paths, sourceFile.RelativePath)
	}
	return paths
}

func TestWalkRepositorySkipsNoiseDirectories(t *testing.T) {
	repoRoot := createFixtureTree(t, map[string]string{
		"src/main.go":               "package main\n",
		"node_modules/lib/index.js": "module.exports = {};\n",
		"vendor/legacy/helper.go":   "package legacy\n",
		"dist/bundle.js":            "console.log(1);\n",
		"build/output.go":           "package build\n",
		"target/App.java":           "class App {}\n",
		".git/hooks/hook.go":        "package hooks\n",
	})

	paths := walkedPaths(t, repoRoot)
	if strings.Join(paths, ",") != "src/main.go" {
		t.Errorf("expected only src/main.go, got %v", paths)
	}
}

func TestWalkRepositorySkipsNonCodeAndSecretFiles(t *testing.T) {
	repoRoot := createFixtureTree(t, map[string]string{
		"src/main.go":           "package main\n",
		"README.md":             "documentation\n",
		"config/.env":           "API_KEY=abcdefghijkl\n",
		"config/.env.local":     "API_KEY=abcdefghijkl\n",
		"certs/server.pem":      "certificate\n",
		"certs/app.key":         "private material\n",
		"certs/client.p12":      "bundle\n",
		"certs/app.keystore":    "keystore\n",
		"certs/signing.jks":     "jks\n",
		"ssh/id_rsa":            "private material\n",
		"ssh/id_ed25519":        "private material\n",
		"auth/credentials.go":   "package auth\n",
		"auth/client_secret.go": "package auth\n",
		"registry/.npmrc":       "registry-token=abcdefghijkl\n",
		"registry/.netrc":       "machine example\n",
		"registry/.htpasswd":    "user:hash\n",
	})

	paths := walkedPaths(t, repoRoot)
	if strings.Join(paths, ",") != "src/main.go" {
		t.Errorf("expected only src/main.go, got %v", paths)
	}
}

func TestWalkRepositorySkipsOversizedAndSymlinkedFiles(t *testing.T) {
	repoRoot := createFixtureTree(t, map[string]string{
		"src/main.go": "package main\n",
	})
	if err := os.WriteFile(
		filepath.Join(repoRoot, "src", "huge.go"),
		[]byte(strings.Repeat("a", MaxIndexableFileSize+1)),
		0o600,
	); err != nil {
		t.Fatalf("write oversized file: %v", err)
	}

	outsideRoot := t.TempDir()
	outsidePath := filepath.Join(outsideRoot, "outside.go")
	if err := os.WriteFile(outsidePath, []byte("package outside\n"), 0o600); err != nil {
		t.Fatalf("write outside file: %v", err)
	}
	if err := os.Symlink(outsidePath, filepath.Join(repoRoot, "src", "linked.go")); err != nil {
		t.Skipf("symlink not supported: %v", err)
	}

	paths := walkedPaths(t, repoRoot)
	if strings.Join(paths, ",") != "src/main.go" {
		t.Errorf("expected only src/main.go, got %v", paths)
	}
}

func TestWalkRepositoryReportsMetadata(t *testing.T) {
	repoRoot := createFixtureTree(t, map[string]string{
		"src/main.go": "package main\n",
	})

	sourceFiles, err := WalkRepository(repoRoot, "sample-service")
	if err != nil {
		t.Fatalf("WalkRepository: %v", err)
	}
	if len(sourceFiles) != 1 {
		t.Fatalf("expected 1 source file, got %d", len(sourceFiles))
	}
	if sourceFiles[0].RepoName != "sample-service" {
		t.Errorf("unexpected repo name %q", sourceFiles[0].RepoName)
	}
	if sourceFiles[0].SizeBytes != int64(len("package main\n")) {
		t.Errorf("unexpected size %d", sourceFiles[0].SizeBytes)
	}
	if sourceFiles[0].ModifiedUnix <= 0 {
		t.Errorf("expected positive modification time, got %d", sourceFiles[0].ModifiedUnix)
	}
}

func TestIsBinaryContentDetectsBinaryPayloads(t *testing.T) {
	if !IsBinaryContent([]byte("package main\x00\x01\x02")) {
		t.Error("expected NUL bytes to be detected as binary")
	}
	if IsBinaryContent([]byte("package main\n\nfunc main() {}\n")) {
		t.Error("expected UTF-8 source text to be detected as text")
	}
}

func TestIsIndexableExtension(t *testing.T) {
	for _, extension := range []string{".go", ".php", ".java", ".js", ".ts"} {
		if !IsIndexableExtension(extension) {
			t.Errorf("expected %s to be indexable", extension)
		}
	}
	for _, extension := range []string{".exe", ".png", ".md", ""} {
		if IsIndexableExtension(extension) {
			t.Errorf("expected %q not to be indexable", extension)
		}
	}
	if !IsIndexableExtension(".GO") {
		t.Error("expected uppercase extension to be normalized")
	}
}

func TestIsSkippedDirectory(t *testing.T) {
	for _, name := range []string{"node_modules", "vendor", "dist", "build", ".git", ".cache"} {
		if !IsSkippedDirectory(name) {
			t.Errorf("expected %s to be skipped", name)
		}
	}
	if IsSkippedDirectory("src") {
		t.Error("expected src not to be skipped")
	}
}
