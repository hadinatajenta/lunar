package infrastructure

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"testing"
	"time"

	"lunar/backend/internal/postman/domain"
)

func writeFixtureFile(t *testing.T, root string, relativePath string, content string) {
	t.Helper()
	fullPath := filepath.Join(root, filepath.FromSlash(relativePath))
	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		t.Fatalf("cannot create the fixture directory for %s: %v", relativePath, err)
	}
	if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
		t.Fatalf("cannot write the fixture file %s: %v", relativePath, err)
	}
}

func selectedPaths(selection domain.Selection) []string {
	paths := make([]string, 0, len(selection.Files))
	for _, file := range selection.Files {
		paths = append(paths, file.RelativePath)
	}
	sort.Strings(paths)
	return paths
}

func containsPath(paths []string, expected string) bool {
	for _, path := range paths {
		if path == expected {
			return true
		}
	}
	return false
}

func TestSelectorIncludesFirstPartyVendorInsideSrc(t *testing.T) {
	root := t.TempDir()
	writeFixtureFile(t, root, "src/vendor/routes.go", "package vendor\n")
	writeFixtureFile(t, root, "src/vendor/delivery.go", "package vendor\n")
	writeFixtureFile(t, root, "src/manage_user/routes.go", "package manage_user\n")

	selection, err := NewSelector().Select(root, "panama")
	if err != nil {
		t.Fatalf("selecting panama-shaped files failed: %v", err)
	}

	paths := selectedPaths(selection)
	for _, expected := range []string{"src/vendor/routes.go", "src/vendor/delivery.go"} {
		if !containsPath(paths, expected) {
			t.Fatalf("expected %s in the selection, received %v", expected, paths)
		}
	}
}

func TestSelectorExcludesFrameworkVendorsAndBuildOutput(t *testing.T) {
	root := t.TempDir()
	writeFixtureFile(t, root, "src/app/routes.go", "package app\n")
	writeFixtureFile(t, root, "index.html", "<html></html>")

	excluded := []string{
		"vendor/github.com/gin-gonic/gin/routes.go",
		"library/vendor/package/routes.go",
		"node_modules/express/routes.js",
		"dist/routes.go",
		"build/routes.go",
		"out/routes.go",
		"target/routes.go",
		"coverage/routes.go",
		"tmp/routes.go",
		"logs/routes.go",
		"storage/routes.go",
		"public/routes.go",
		"assets/routes.go",
		"resources/views/routes.php",
		"bootstrap/cache/routes.php",
		".git/routes.go",
		".idea/routes.go",
		"web/index.html",
	}
	for _, relativePath := range excluded {
		writeFixtureFile(t, root, relativePath, "package excluded\n")
	}

	selection, err := NewSelector().Select(root, "atlas")
	if err != nil {
		t.Fatalf("selecting files failed: %v", err)
	}

	paths := selectedPaths(selection)
	if len(paths) != 1 || paths[0] != "src/app/routes.go" {
		t.Fatalf("expected only src/app/routes.go, received %v", paths)
	}
}

func TestSelectorExcludesTestFilesAndMocks(t *testing.T) {
	root := t.TempDir()
	writeFixtureFile(t, root, "src/manage_user/routes.go", "route.GET(\"/level\", delivery.GetAllLevelUser)\n")
	writeFixtureFile(t, root, "src/manage_user/delivery.go", "func (delivery *Delivery) GetAllLevelUser() {}\n")
	writeFixtureFile(t, root, "src/manage_user/delivery_test.go", "route.GET(\"/level\", delivery.GetAllLevelUser)\n")
	writeFixtureFile(t, root, "src/manage_user/mocks/delivery.go", "route.GET(\"/level\", nil)\n")
	writeFixtureFile(t, root, "src/manage_user/routes.spec.js", "router.get('/level')\n")
	writeFixtureFile(t, root, "tests/routes.go", "route.GET(\"/level\", nil)\n")
	writeFixtureFile(t, root, "test/routes.go", "route.GET(\"/level\", nil)\n")

	selection, err := NewSelector().Select(root, "atlas")
	if err != nil {
		t.Fatalf("selecting files failed: %v", err)
	}

	paths := selectedPaths(selection)
	expected := []string{"src/manage_user/delivery.go", "src/manage_user/routes.go"}
	if strings.Join(paths, ",") != strings.Join(expected, ",") {
		t.Fatalf("expected %v, received %v", expected, paths)
	}
}

func TestSelectorRefusesWhenTheFileLimitIsExceeded(t *testing.T) {
	root := t.TempDir()
	for index := 0; index < 3; index++ {
		writeFixtureFile(t, root, "src/file"+string(rune('a'+index))+".go", "package src\n")
	}

	selector := &Selector{MaxFiles: 2, MaxTotalBytes: domain.MaxSelectedBytes, MaxFileBytes: domain.MaxSelectedFileSize}
	selection, err := selector.Select(root, "atlas")
	if err != nil {
		t.Fatalf("selecting files failed: %v", err)
	}
	if !selection.ExceededFileLimit {
		t.Fatalf("expected the file limit to be reported as exceeded")
	}
	if len(selection.Files) != 2 {
		t.Fatalf("expected the selector to stop at 2 files, received %d", len(selection.Files))
	}
}

func TestSelectorRefusesWhenTheByteLimitIsExceeded(t *testing.T) {
	root := t.TempDir()
	writeFixtureFile(t, root, "src/large.go", strings.Repeat("a", 200))

	selector := &Selector{MaxFiles: domain.MaxSelectedFiles, MaxTotalBytes: 10, MaxFileBytes: domain.MaxSelectedFileSize}
	selection, err := selector.Select(root, "atlas")
	if err != nil {
		t.Fatalf("selecting files failed: %v", err)
	}
	if !selection.ExceededByteLimit {
		t.Fatalf("expected the byte limit to be reported as exceeded")
	}
	if len(selection.Files) != 0 {
		t.Fatalf("expected no files to be selected, received %d", len(selection.Files))
	}
}

func TestSelectorSkipsBinariesSymlinksAndOversizedFiles(t *testing.T) {
	root := t.TempDir()
	writeFixtureFile(t, root, "src/text.go", "package src\n")
	writeFixtureFile(t, root, "src/binary.go", "package src"+string(rune(0))+"\n")
	writeFixtureFile(t, root, "src/oversized.go", strings.Repeat("a", 4096))
	if err := os.Symlink(filepath.Join(root, "src", "text.go"), filepath.Join(root, "src", "link.go")); err != nil {
		t.Fatalf("cannot create the symlink fixture: %v", err)
	}

	selector := &Selector{MaxFiles: domain.MaxSelectedFiles, MaxTotalBytes: domain.MaxSelectedBytes, MaxFileBytes: 1024}
	selection, err := selector.Select(root, "atlas")
	if err != nil {
		t.Fatalf("selecting files failed: %v", err)
	}

	paths := selectedPaths(selection)
	if len(paths) != 1 || paths[0] != "src/text.go" {
		t.Fatalf("expected only src/text.go, received %v", paths)
	}
	if selection.SkippedFileCount != 2 {
		t.Fatalf("expected 2 skipped files, received %d", selection.SkippedFileCount)
	}
}

func TestSelectorExcludesDeniedDirectoriesRegardlessOfCase(t *testing.T) {
	root := t.TempDir()
	writeFixtureFile(t, root, "src/routes/api.ts", "export const routes = []\n")

	for _, denied := range []string{"Node_Modules", "DIST", "TARGET", "PUBLIC", "Assets", "Tests", "Mocks", "Logs", "Storage", "Coverage", "Vendor", "Dist"} {
		writeFixtureFile(t, root, denied+"/pkg/routes/thing.js", "module.exports = {}\n")
	}
	writeFixtureFile(t, root, "src/Vendor/routes/kept.go", "package kept\n")

	selection, err := NewSelector().Select(root, "case")
	if err != nil {
		t.Fatalf("select: %v", err)
	}

	paths := selectedPaths(selection)
	for _, denied := range []string{"Node_Modules", "DIST", "TARGET", "PUBLIC", "Assets", "Tests", "Mocks", "Logs", "Storage", "Coverage", "Vendor", "Dist"} {
		for _, selected := range paths {
			if strings.HasPrefix(selected, denied+"/") {
				t.Fatalf("expected %s to be excluded, got %v", denied, paths)
			}
		}
	}

	if !containsPath(paths, "src/Vendor/routes/kept.go") {
		t.Fatalf("expected src/Vendor to be treated as first party code, got %v", paths)
	}
}

func TestSelectorIgnoresASymlinkedPackageJSON(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	writeFixtureFile(t, root, "src/routes/api.ts", "export const routes = []\n")

	secretPath := filepath.Join(outside, "outside-secret.json")
	writeFixtureFile(t, outside, "outside-secret.json", "{\"name\":\"outside-secret\"}\n")
	if err := os.Symlink(secretPath, filepath.Join(root, "package.json")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	selection, err := NewSelector().Select(root, "symlink")
	if err != nil {
		t.Fatalf("select: %v", err)
	}
	if selection.HasPackageJSON {
		t.Fatal("expected a symlinked package.json to be ignored")
	}
	if strings.Contains(selection.PackageJSONContent, "outside-secret") {
		t.Fatal("expected no content to be read through the symlink")
	}
}

func TestSelectorIgnoresASymlinkedDirectoryMarker(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	writeFixtureFile(t, root, "src/routes/api.ts", "export const routes = []\n")
	if err := os.MkdirAll(filepath.Join(outside, "application"), 0o755); err != nil {
		t.Fatalf("prepare outside: %v", err)
	}
	if err := os.Symlink(filepath.Join(outside, "application"), filepath.Join(root, "application")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	selection, err := NewSelector().Select(root, "symlink-dir")
	if err != nil {
		t.Fatalf("select: %v", err)
	}
	if selection.HasApplicationDirectory {
		t.Fatal("expected a symlinked application directory not to count as the marker")
	}
}

func TestSelectorSkipsNonRegularFilesWithoutBlocking(t *testing.T) {
	root := t.TempDir()
	writeFixtureFile(t, root, "src/routes/api.ts", "export const routes = []\n")

	fifoPath := filepath.Join(root, "src", "routes", "hang.go")
	if err := syscall.Mkfifo(fifoPath, 0o600); err != nil {
		t.Skipf("fifos unavailable: %v", err)
	}

	done := make(chan domain.Selection, 1)
	go func() {
		selection, err := NewSelector().Select(root, "fifo")
		if err != nil {
			t.Errorf("select: %v", err)
		}
		done <- selection
	}()

	select {
	case selection := <-done:
		if containsPath(selectedPaths(selection), "src/routes/hang.go") {
			t.Fatal("expected a fifo to be skipped")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("expected the selector to skip a fifo instead of blocking on it")
	}
}

func TestSelectorCollectsClassificationSignals(t *testing.T) {
	root := t.TempDir()
	writeFixtureFile(t, root, "application/config/routes.php", "<?php\n")
	writeFixtureFile(t, root, "system/core/Router.php", "<?php\n")
	writeFixtureFile(t, root, "package.json", "{\"dependencies\":{\"vue\":\"^3.0.0\"}}")

	selection, err := NewSelector().Select(root, "mms")
	if err != nil {
		t.Fatalf("selecting files failed: %v", err)
	}
	if !selection.HasApplicationDirectory || !selection.HasSystemDirectory {
		t.Fatalf("expected the CodeIgniter directory signals to be collected")
	}
	if !selection.HasPackageJSON || !strings.Contains(selection.PackageJSONContent, "vue") {
		t.Fatalf("expected the package.json signal to be collected")
	}
}

func TestSelectorIncludesPHPAndNodeEquivalents(t *testing.T) {
	root := t.TempDir()
	writeFixtureFile(t, root, "routes/api.php", "<?php Route::get('/x', 'XController@index');\n")
	writeFixtureFile(t, root, "app/Http/Controllers/UserController.php", "<?php\n")
	writeFixtureFile(t, root, "app/Models/User.php", "<?php\n")
	writeFixtureFile(t, root, "app/views/welcome.php", "<?php\n")
	writeFixtureFile(t, root, "routes/web.js", "router.get('/x')\n")
	writeFixtureFile(t, root, "controllers/user.js", "module.exports = {}\n")
	writeFixtureFile(t, root, "app.js", "const express = require('express')\n")
	writeFixtureFile(t, root, "server.js", "const express = require('express')\n")

	selection, err := NewSelector().Select(root, "merchant-service-laravel")
	if err != nil {
		t.Fatalf("selecting files failed: %v", err)
	}

	paths := selectedPaths(selection)
	expected := []string{
		"app.js",
		"app/Http/Controllers/UserController.php",
		"app/Models/User.php",
		"controllers/user.js",
		"routes/api.php",
		"routes/web.js",
		"server.js",
	}
	if strings.Join(paths, ",") != strings.Join(expected, ",") {
		t.Fatalf("expected %v, received %v", expected, paths)
	}
}
