package domain

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestValidateRepoNameRejectsUnsafeInput(t *testing.T) {
	cases := []struct {
		name  string
		input string
	}{
		{name: "empty", input: ""},
		{name: "whitespace only", input: "   "},
		{name: "dot", input: "."},
		{name: "dot dot", input: ".."},
		{name: "forward slash prefix", input: "/etc"},
		{name: "forward slash inside", input: "nested/repo"},
		{name: "traversal with separator", input: "../secret"},
		{name: "backslash", input: `nested\repo`},
		{name: "null byte", input: "repo\x00name"},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := ValidateRepoName(testCase.input)
			if !errors.Is(err, ErrInvalidRepoName) {
				t.Fatalf("expected ErrInvalidRepoName for %q, got %v", testCase.input, err)
			}
		})
	}
}

func TestValidateRepoNameAcceptsPlainNames(t *testing.T) {
	cases := []string{"everest", "micro-service-go", "ganges-mms", "repo_2", "Repo.With.Dots"}

	for _, input := range cases {
		t.Run(input, func(t *testing.T) {
			got, err := ValidateRepoName(input)
			if err != nil {
				t.Fatalf("expected %q to be accepted, got %v", input, err)
			}
			if got != input {
				t.Fatalf("expected %q, got %q", input, got)
			}
		})
	}
}

func TestExpandHomeResolvesTilde(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skipf("home directory unavailable: %v", err)
	}

	t.Run("bare tilde", func(t *testing.T) {
		got, err := ExpandHome("~")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != home {
			t.Fatalf("expected %q, got %q", home, got)
		}
	})

	t.Run("tilde with path", func(t *testing.T) {
		got, err := ExpandHome("~/BRI")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		expected := filepath.Join(home, "BRI")
		if got != expected {
			t.Fatalf("expected %q, got %q", expected, got)
		}
	})

	t.Run("absolute path untouched", func(t *testing.T) {
		got, err := ExpandHome("/tmp/workspace")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != "/tmp/workspace" {
			t.Fatalf("expected /tmp/workspace, got %q", got)
		}
	})

	t.Run("empty path rejected", func(t *testing.T) {
		if _, err := ExpandHome("  "); !errors.Is(err, ErrInvalidRoot) {
			t.Fatalf("expected ErrInvalidRoot, got %v", err)
		}
	})
}

func TestIsSensitiveRoot(t *testing.T) {
	sensitive := []string{"/", "/etc", "/etc/", "/System", "/private/var", "/usr", "/bin", "/sbin"}
	for _, path := range sensitive {
		t.Run("sensitive "+path, func(t *testing.T) {
			if !IsSensitiveRoot(path) {
				t.Fatalf("expected %q to be sensitive", path)
			}
		})
	}

	allowed := []string{"/Users/erendt/BRI", "/home/user/work", "/tmp/workspace", "/opt/lunar/repos"}
	for _, path := range allowed {
		t.Run("allowed "+path, func(t *testing.T) {
			if IsSensitiveRoot(path) {
				t.Fatalf("expected %q to be allowed", path)
			}
		})
	}
}

func TestResolveWorkspaceRootRejects(t *testing.T) {
	workspace := t.TempDir()

	t.Run("sensitive root", func(t *testing.T) {
		if _, err := ResolveWorkspaceRoot("/etc", nil); !errors.Is(err, ErrRootNotAllowed) {
			t.Fatalf("expected ErrRootNotAllowed, got %v", err)
		}
	})

	t.Run("missing path", func(t *testing.T) {
		missing := filepath.Join(workspace, "does-not-exist")
		if _, err := ResolveWorkspaceRoot(missing, nil); !errors.Is(err, ErrRootNotFound) {
			t.Fatalf("expected ErrRootNotFound, got %v", err)
		}
	})

	t.Run("file instead of directory", func(t *testing.T) {
		filePath := filepath.Join(workspace, "plain.txt")
		if err := os.WriteFile(filePath, []byte("data"), 0o600); err != nil {
			t.Fatalf("cannot create fixture: %v", err)
		}
		if _, err := ResolveWorkspaceRoot(filePath, nil); !errors.Is(err, ErrRootNotDirectory) {
			t.Fatalf("expected ErrRootNotDirectory, got %v", err)
		}
	})

	t.Run("relative path", func(t *testing.T) {
		if _, err := ResolveWorkspaceRoot("relative/path", nil); !errors.Is(err, ErrInvalidRoot) {
			t.Fatalf("expected ErrInvalidRoot, got %v", err)
		}
	})

	t.Run("outside allowlist", func(t *testing.T) {
		allowedOnly := "/opt/lunar/approved"
		if _, err := ResolveWorkspaceRoot(workspace, []string{allowedOnly}); !errors.Is(err, ErrRootOutsideScope) {
			t.Fatalf("expected ErrRootOutsideScope, got %v", err)
		}
	})

	t.Run("inside allowlist", func(t *testing.T) {
		resolved, err := ResolveWorkspaceRoot(workspace, []string{workspace})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		resolvedWorkspace, err := filepath.EvalSymlinks(workspace)
		if err != nil {
			t.Fatalf("cannot resolve fixture: %v", err)
		}
		if resolved != resolvedWorkspace {
			t.Fatalf("expected %q, got %q", resolvedWorkspace, resolved)
		}
	})
}

func TestResolveWorkspaceRootAcceptsRealDirectory(t *testing.T) {
	workspace := t.TempDir()
	resolved, err := ResolveWorkspaceRoot(workspace, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected, err := filepath.EvalSymlinks(workspace)
	if err != nil {
		t.Fatalf("cannot resolve fixture: %v", err)
	}
	if resolved != expected {
		t.Fatalf("expected %q, got %q", expected, resolved)
	}
}

func TestValidateRepoNameRejectsEncodedAndUnusualCharacters(t *testing.T) {
	cases := []struct {
		name  string
		input string
	}{
		{name: "encoded separator", input: "..%2Fetc"},
		{name: "encoded backslash", input: "..%5Cetc"},
		{name: "percent sign", input: "repo%20name"},
		{name: "space inside", input: "repo name"},
		{name: "leading dot", input: ".hidden"},
		{name: "leading hyphen", input: "-service"},
		{name: "colon", input: "repo:name"},
		{name: "glob star", input: "repo*"},
		{name: "question mark", input: "repo?"},
		{name: "tab", input: "repo\tname"},
		{name: "unicode", input: "reponäme"},
		{name: "newline", input: "repo\nname"},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := ValidateRepoName(testCase.input)
			if !errors.Is(err, ErrInvalidRepoName) {
				t.Fatalf("expected ErrInvalidRepoName for %q, got %v", testCase.input, err)
			}
		})
	}
}

func TestValidateRepoNameAcceptsRealRepositoryNames(t *testing.T) {
	cases := []string{
		"everest", "mms-api", "ganges-mms", "micro_user", "service.qris",
		"fe-mms", "mocash_api", "helm-micro-qc", "Repo.With.Dots", "repo2",
	}

	for _, input := range cases {
		t.Run(input, func(t *testing.T) {
			got, err := ValidateRepoName(input)
			if err != nil {
				t.Fatalf("expected %q to be accepted, got %v", input, err)
			}
			if got != input {
				t.Fatalf("expected %q, got %q", input, got)
			}
		})
	}
}
