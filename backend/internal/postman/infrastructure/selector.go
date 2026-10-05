package infrastructure

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	codeindexinfrastructure "lunar/backend/internal/codeindex/infrastructure"
	"lunar/backend/internal/postman/domain"
)

const srcPrefix = "src/"

var excludedDirectories = map[string]bool{
	"node_modules": true,
	"dist":         true,
	"build":        true,
	"out":          true,
	"target":       true,
	"coverage":     true,
	"tmp":          true,
	"logs":         true,
	"storage":      true,
	"mocks":        true,
	"tests":        true,
	"test":         true,
	"public":       true,
	"assets":       true,
}

var includedFileNames = map[string]bool{
	"routes.go":     true,
	"delivery.go":   true,
	"service.go":    true,
	"repository.go": true,
	"dto.go":        true,
	"dao.go":        true,
	"interface.go":  true,
	"const.go":      true,
	"main.go":       true,
	"wire.go":       true,
	"wire_gen.go":   true,
	"app.js":        true,
	"server.js":     true,
}

type Selector struct {
	MaxFiles      int
	MaxTotalBytes int64
	MaxFileBytes  int64
}

func NewSelector() *Selector {
	return &Selector{
		MaxFiles:      domain.MaxSelectedFiles,
		MaxTotalBytes: domain.MaxSelectedBytes,
		MaxFileBytes:  domain.MaxSelectedFileSize,
	}
}

func (s *Selector) Select(repoPath string, service string) (domain.Selection, error) {
	selection := domain.Selection{Service: service}
	selection.HasApplicationDirectory = isDirectory(filepath.Join(repoPath, "application"))
	selection.HasSystemDirectory = isDirectory(filepath.Join(repoPath, "system"))

	packageJSON, found, err := readRootPackageJSON(repoPath)
	if err != nil {
		return domain.Selection{}, fmt.Errorf("select export files: %w", err)
	}
	selection.HasPackageJSON = found
	selection.PackageJSONContent = packageJSON

	walkErr := filepath.WalkDir(repoPath, func(currentPath string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return fmt.Errorf("select export files: %s: %w", currentPath, walkErr)
		}
		if selection.ExceededFileLimit || selection.ExceededByteLimit {
			return filepath.SkipAll
		}

		relativePath, relErr := filepath.Rel(repoPath, currentPath)
		if relErr != nil {
			return fmt.Errorf("select export files: relative path for %s: %w", currentPath, relErr)
		}
		relativePath = filepath.ToSlash(relativePath)

		if entry.IsDir() {
			if relativePath == "." {
				return nil
			}
			if !isIncludedDirectory(relativePath, entry.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		if !isSelectableFile(relativePath, entry) {
			return nil
		}

		info, infoErr := entry.Info()
		if infoErr != nil {
			return fmt.Errorf("select export files: stat %s: %w", currentPath, infoErr)
		}
		if info.Size() > s.MaxFileBytes {
			selection.SkippedFileCount++
			return nil
		}
		if len(selection.Files) >= s.MaxFiles {
			selection.ExceededFileLimit = true
			return filepath.SkipAll
		}
		if selection.TotalBytes+info.Size() > s.MaxTotalBytes {
			selection.ExceededByteLimit = true
			return filepath.SkipAll
		}

		content, readErr := os.ReadFile(currentPath)
		if readErr != nil {
			return fmt.Errorf("select export files: read %s: %w", currentPath, readErr)
		}
		if codeindexinfrastructure.IsBinaryContent(content) {
			selection.SkippedFileCount++
			return nil
		}

		selection.Files = append(selection.Files, domain.SelectedFile{
			RelativePath: relativePath,
			Content:      string(content),
			SizeBytes:    info.Size(),
		})
		selection.TotalBytes += info.Size()
		return nil
	})
	if walkErr != nil {
		return domain.Selection{}, walkErr
	}

	sort.Slice(selection.Files, func(first int, second int) bool {
		return selection.Files[first].RelativePath < selection.Files[second].RelativePath
	})
	return selection, nil
}

func isIncludedDirectory(relativePath string, name string) bool {
	if strings.HasPrefix(name, ".") {
		return false
	}
	lowerName := strings.ToLower(name)
	lowerPath := strings.ToLower(relativePath)
	if excludedDirectories[lowerName] {
		return false
	}
	if lowerName == "vendor" {
		return lowerPath == "src/vendor"
	}
	if lowerName == "views" {
		return path.Dir(lowerPath) != "resources"
	}
	if lowerName == "cache" {
		return path.Dir(lowerPath) != "bootstrap"
	}
	return true
}

func isSelectableFile(relativePath string, entry fs.DirEntry) bool {
	if !entry.Type().IsRegular() {
		return false
	}
	name := entry.Name()
	if strings.HasPrefix(name, ".") {
		return false
	}
	if !strings.Contains(relativePath, "/") && strings.HasSuffix(strings.ToLower(name), ".html") {
		return false
	}
	if isTestFileName(name) {
		return false
	}
	if strings.HasPrefix(relativePath, srcPrefix) {
		return true
	}
	if includedFileNames[name] {
		return true
	}
	return isPHPEquivalent(relativePath) || isNodeEquivalent(relativePath)
}

func isTestFileName(name string) bool {
	if strings.Contains(name, ".spec.") {
		return true
	}
	stem := strings.TrimSuffix(name, path.Ext(name))
	return strings.HasSuffix(stem, "_test")
}

func isPHPEquivalent(relativePath string) bool {
	if path.Ext(relativePath) != ".php" {
		return false
	}
	if path.Base(path.Dir(relativePath)) == "routes" {
		return true
	}
	if strings.HasPrefix(relativePath, "app/Http/Controllers/") || strings.Contains(relativePath, "/app/Http/Controllers/") {
		return true
	}
	return strings.HasPrefix(relativePath, "app/Models/") || strings.Contains(relativePath, "/app/Models/")
}

func isNodeEquivalent(relativePath string) bool {
	switch path.Ext(relativePath) {
	case ".js", ".ts", ".mjs", ".cjs", ".jsx", ".tsx":
	default:
		return false
	}
	for _, segment := range strings.Split(path.Dir(relativePath), "/") {
		if segment == "routes" || segment == "controllers" {
			return true
		}
	}
	return false
}

func isDirectory(candidatePath string) bool {
	info, err := os.Lstat(candidatePath)
	if err != nil {
		return false
	}
	return info.IsDir()
}

func readRootPackageJSON(repoPath string) (string, bool, error) {
	packageJSONPath := filepath.Join(repoPath, "package.json")
	info, err := os.Lstat(packageJSONPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", false, nil
		}
		return "", false, fmt.Errorf("stat package.json: %w", err)
	}
	if !info.Mode().IsRegular() {
		return "", false, nil
	}

	content, err := os.ReadFile(packageJSONPath)
	if err != nil {
		return "", false, fmt.Errorf("read package.json: %w", err)
	}
	return string(content), true, nil
}
