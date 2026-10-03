package domain

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestResolveRepoPathRejectsEscapes(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "everest"), 0o755); err != nil {
		t.Fatalf("cannot create fixture: %v", err)
	}

	t.Run("traversal name", func(t *testing.T) {
		if _, err := ResolveRepoPath(root, "../outside"); !errors.Is(err, ErrInvalidRepoName) {
			t.Fatalf("expected ErrInvalidRepoName, got %v", err)
		}
	})

	t.Run("symlink escaping root", func(t *testing.T) {
		outside := t.TempDir()
		linkPath := filepath.Join(root, "escaped")
		if err := os.Symlink(outside, linkPath); err != nil {
			t.Skipf("symlink unsupported here: %v", err)
		}

		_, err := ResolveRepoPath(root, "escaped")
		if !errors.Is(err, ErrRepoOutsideRoot) {
			t.Fatalf("expected ErrRepoOutsideRoot, got %v", err)
		}
	})

	t.Run("missing repository", func(t *testing.T) {
		if _, err := ResolveRepoPath(root, "absent"); !errors.Is(err, ErrRootNotFound) {
			t.Fatalf("expected ErrRootNotFound, got %v", err)
		}
	})

	t.Run("valid repository", func(t *testing.T) {
		got, err := ResolveRepoPath(root, "everest")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		resolvedRoot, err := filepath.EvalSymlinks(root)
		if err != nil {
			t.Fatalf("cannot resolve fixture: %v", err)
		}
		expected := filepath.Join(resolvedRoot, "everest")
		if got != expected {
			t.Fatalf("expected %q, got %q", expected, got)
		}
	})
}
