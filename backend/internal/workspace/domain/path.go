package domain

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	ErrInvalidRepoName  = errors.New("invalid repository name")
	ErrInvalidRoot      = errors.New("invalid workspace root")
	ErrRootNotFound     = errors.New("workspace root does not exist")
	ErrRootNotDirectory = errors.New("workspace root is not a directory")
	ErrRootOutsideScope = errors.New("workspace root is outside the allowed roots")
	ErrRootNotAllowed   = errors.New("workspace root is not permitted")
	ErrRepoOutsideRoot  = errors.New("repository path escapes the workspace root")
)

var sensitiveRootPatterns = []string{
	`^/$`,
	`^/etc$`,
	`^/private/etc$`,
	`^/System$`,
	`^/private/var$`,
	`^/var$`,
	`^/usr$`,
	`^/bin$`,
	`^/sbin$`,
	`^/dev$`,
	`^/proc$`,
	`^/sys$`,
	`^/Users/[^/]+$`,
	`^/home/[^/]+$`,
	`^/Volumes$`,
}

var sensitiveRootRegexps = func() []*regexp.Regexp {
	compiled := make([]*regexp.Regexp, 0, len(sensitiveRootPatterns))
	for _, pattern := range sensitiveRootPatterns {
		compiled = append(compiled, regexp.MustCompile(pattern))
	}
	return compiled
}()

var repositoryNamePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)

func ValidateRepoName(name string) (string, error) {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return "", fmt.Errorf("%w: name is empty", ErrInvalidRepoName)
	}
	if trimmed == "." || trimmed == ".." {
		return "", fmt.Errorf("%w: relative segments are not allowed", ErrInvalidRepoName)
	}
	if strings.ContainsAny(trimmed, `/\`) {
		return "", fmt.Errorf("%w: path separators are not allowed", ErrInvalidRepoName)
	}
	if strings.ContainsRune(trimmed, 0) {
		return "", fmt.Errorf("%w: null bytes are not allowed", ErrInvalidRepoName)
	}
	if filepath.Base(trimmed) != trimmed {
		return "", fmt.Errorf("%w: name must be a bare basename", ErrInvalidRepoName)
	}
	if !repositoryNamePattern.MatchString(trimmed) {
		return "", fmt.Errorf("%w: name contains characters outside letters, digits, dot, hyphen, and underscore", ErrInvalidRepoName)
	}
	return trimmed, nil
}

func ExpandHome(path string) (string, error) {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return "", fmt.Errorf("%w: path is empty", ErrInvalidRoot)
	}

	if trimmed == "~" || strings.HasPrefix(trimmed, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("%w: cannot resolve home directory", ErrInvalidRoot)
		}
		if trimmed == "~" {
			return home, nil
		}
		return filepath.Join(home, trimmed[2:]), nil
	}

	return trimmed, nil
}

func IsSensitiveRoot(path string) bool {
	cleaned := filepath.Clean(path)
	if resolved, err := filepath.EvalSymlinks(cleaned); err == nil {
		cleaned = resolved
	}

	for _, pattern := range sensitiveRootRegexps {
		if pattern.MatchString(cleaned) {
			return true
		}
	}
	return false
}

func WithinAllowedRoots(path string, allowedRoots []string) bool {
	if len(allowedRoots) == 0 {
		return true
	}

	resolvedPath, err := filepath.EvalSymlinks(path)
	if err != nil {
		resolvedPath = filepath.Clean(path)
	}

	for _, allowed := range allowedRoots {
		resolvedAllowed, err := filepath.EvalSymlinks(allowed)
		if err != nil {
			resolvedAllowed = filepath.Clean(allowed)
		}
		if resolvedPath == resolvedAllowed ||
			strings.HasPrefix(resolvedPath, resolvedAllowed+string(os.PathSeparator)) {
			return true
		}
	}
	return false
}

func ResolveWorkspaceRoot(rawPath string, allowedRoots []string) (string, error) {
	expanded, err := ExpandHome(rawPath)
	if err != nil {
		return "", err
	}

	if !filepath.IsAbs(expanded) {
		return "", fmt.Errorf("%w: path must be absolute", ErrInvalidRoot)
	}

	resolved, err := filepath.EvalSymlinks(filepath.Clean(expanded))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", fmt.Errorf("%w: %s", ErrRootNotFound, filepath.Clean(expanded))
		}
		return "", fmt.Errorf("%w: cannot resolve path", ErrInvalidRoot)
	}

	info, err := os.Stat(resolved)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", fmt.Errorf("%w: %s", ErrRootNotFound, resolved)
		}
		return "", fmt.Errorf("%w: cannot read path", ErrInvalidRoot)
	}

	if !info.IsDir() {
		return "", fmt.Errorf("%w: %s", ErrRootNotDirectory, resolved)
	}

	if IsSensitiveRoot(resolved) {
		return "", fmt.Errorf("%w: %s", ErrRootNotAllowed, resolved)
	}

	if !WithinAllowedRoots(resolved, allowedRoots) {
		return "", fmt.Errorf("%w: %s", ErrRootOutsideScope, resolved)
	}

	return resolved, nil
}

func ResolveRepoPath(rootPath string, repoName string) (string, error) {
	validName, err := ValidateRepoName(repoName)
	if err != nil {
		return "", err
	}

	resolvedRoot, err := filepath.EvalSymlinks(filepath.Clean(rootPath))
	if err != nil {
		return "", fmt.Errorf("%w: cannot resolve root", ErrInvalidRoot)
	}

	candidate := filepath.Join(resolvedRoot, validName)
	resolvedCandidate, err := filepath.EvalSymlinks(candidate)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", fmt.Errorf("%w: %s", ErrRootNotFound, validName)
		}
		return "", fmt.Errorf("%w: cannot resolve repository path", ErrInvalidRoot)
	}

	if resolvedCandidate != resolvedRoot && !strings.HasPrefix(resolvedCandidate, resolvedRoot+string(os.PathSeparator)) {
		return "", fmt.Errorf("%w: %s", ErrRepoOutsideRoot, validName)
	}

	return resolvedCandidate, nil
}
