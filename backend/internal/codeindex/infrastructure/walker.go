package infrastructure

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"
	"unicode/utf8"

	codeindexdomain "lunar/backend/internal/codeindex/domain"
)

const (
	MaxIndexableFileSize = 2 << 20
	binarySniffLength    = 8000
)

var skippedDirectories = map[string]bool{
	"node_modules": true,
	"vendor":       true,
	"dist":         true,
	"build":        true,
	".git":         true,
	"target":       true,
	"out":          true,
	"coverage":     true,
	"__pycache__":  true,
	".idea":        true,
	".vscode":      true,
	".mvn":         true,
}

var indexableExtensions = map[string]bool{
	".go":    true,
	".java":  true,
	".js":    true,
	".ts":    true,
	".php":   true,
	".py":    true,
	".kt":    true,
	".scala": true,
	".rb":    true,
	".cs":    true,
	".rs":    true,
}

type FileState struct {
	RepoName     string
	FilePath     string
	ContentHash  string
	SizeBytes    int64
	ModifiedUnix int64
}

type FileUpdate struct {
	State        FileState
	Chunks       []codeindexdomain.CodeChunk
	Dependencies []codeindexdomain.ServiceDependency
}

type SourceFile struct {
	RepoName     string
	AbsolutePath string
	RelativePath string
	SizeBytes    int64
	ModifiedUnix int64
}

func IsSkippedDirectory(name string) bool {
	if skippedDirectories[name] {
		return true
	}
	return strings.HasPrefix(name, ".")
}

func IsIndexableExtension(extension string) bool {
	return indexableExtensions[strings.ToLower(extension)]
}

func IsIndexableFile(name string) bool {
	if IsExcludedFileName(name) {
		return false
	}
	return IsIndexableExtension(filepath.Ext(name))
}

func IsBinaryContent(content []byte) bool {
	sample := content
	if len(sample) > binarySniffLength {
		sample = sample[:binarySniffLength]
	}
	if !utf8.Valid(sample) {
		return true
	}
	return strings.ContainsRune(string(sample), rune(0))
}

func WalkRepository(repoRoot string, repoName string) ([]SourceFile, error) {
	var sourceFiles []SourceFile

	walkErr := filepath.WalkDir(repoRoot, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("walkRepository: %s: %w", path, err)
		}

		if entry.IsDir() {
			if path == repoRoot {
				return nil
			}
			if IsSkippedDirectory(entry.Name()) {
				return filepath.SkipDir
			}
			return nil
		}

		if entry.Type()&fs.ModeSymlink != 0 {
			return nil
		}
		if !IsIndexableFile(entry.Name()) {
			return nil
		}

		info, err := entry.Info()
		if err != nil {
			return fmt.Errorf("walkRepository: stat %s: %w", path, err)
		}
		if info.Size() > MaxIndexableFileSize {
			return nil
		}

		relativePath, err := filepath.Rel(repoRoot, path)
		if err != nil {
			return fmt.Errorf("walkRepository: relative path for %s: %w", path, err)
		}

		sourceFiles = append(sourceFiles, SourceFile{
			RepoName:     repoName,
			AbsolutePath: path,
			RelativePath: filepath.ToSlash(relativePath),
			SizeBytes:    info.Size(),
			ModifiedUnix: info.ModTime().Unix(),
		})
		return nil
	})
	if walkErr != nil {
		return nil, fmt.Errorf("walkRepository: %w", walkErr)
	}

	return sourceFiles, nil
}
