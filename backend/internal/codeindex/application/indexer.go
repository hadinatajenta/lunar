package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	codeindexdomain "lunar/backend/internal/codeindex/domain"
	codeindexinfrastructure "lunar/backend/internal/codeindex/infrastructure"
	workspacedomain "lunar/backend/internal/workspace/domain"
)

type fileIndexWriter interface {
	LoadFileStates(ctx context.Context, repoName string) (map[string]codeindexinfrastructure.FileState, error)
	ApplyIndex(ctx context.Context, repoName string, updates []codeindexinfrastructure.FileUpdate, removedPaths []string) error
}

type extractedSourceFile struct {
	chunks       []codeindexdomain.CodeChunk
	dependencies []codeindexdomain.ServiceDependency
}

type Indexer struct {
	store codeindexdomain.IndexStore
}

type IndexResult struct {
	RepoName        string
	ChunkCount      int
	DependencyCount int
	IndexedFiles    int
	UnchangedFiles  int
	SkippedFiles    int
	RemovedFiles    int
}

func NewIndexer(store codeindexdomain.IndexStore) *Indexer {
	return &Indexer{store: store}
}

func (i *Indexer) IndexRepository(ctx context.Context, rootPath string, repoName string) (IndexResult, error) {
	repoRoot, err := workspacedomain.ResolveRepoPath(rootPath, repoName)
	if err != nil {
		return IndexResult{}, fmt.Errorf("indexRepository: %w", err)
	}
	return i.indexRepoRoot(ctx, repoRoot, repoName)
}

func (i *Indexer) IndexWorkspace(ctx context.Context, rootPath string, allowedRoots []string) ([]IndexResult, error) {
	resolvedRoot, err := workspacedomain.ResolveWorkspaceRoot(rootPath, allowedRoots)
	if err != nil {
		return nil, fmt.Errorf("indexWorkspace: %w", err)
	}

	entries, err := os.ReadDir(resolvedRoot)
	if err != nil {
		return nil, fmt.Errorf("indexWorkspace: read root: %w", err)
	}

	var results []IndexResult
	for _, entry := range entries {
		if !entry.IsDir() || !shouldIndexRepository(entry.Name()) {
			continue
		}
		result, err := i.IndexRepository(ctx, resolvedRoot, entry.Name())
		if err != nil {
			return nil, fmt.Errorf("indexWorkspace: %w", err)
		}
		results = append(results, result)
	}
	return results, nil
}

func (i *Indexer) indexRepoRoot(ctx context.Context, repoRoot string, repoName string) (IndexResult, error) {
	repoInfo, err := os.Stat(repoRoot)
	if err != nil {
		return IndexResult{}, fmt.Errorf("indexRepoRoot: %w", err)
	}
	if !repoInfo.IsDir() {
		return IndexResult{}, fmt.Errorf("indexRepoRoot: %s is not a directory", repoRoot)
	}

	sourceFiles, err := codeindexinfrastructure.WalkRepository(repoRoot, repoName)
	if err != nil {
		return IndexResult{}, fmt.Errorf("indexRepoRoot: %w", err)
	}

	if writer, supportsIncremental := i.store.(fileIndexWriter); supportsIncremental {
		return i.indexIncremental(ctx, writer, repoName, sourceFiles)
	}
	return i.indexFull(ctx, repoName, sourceFiles)
}

func (i *Indexer) indexIncremental(ctx context.Context, writer fileIndexWriter, repoName string, sourceFiles []codeindexinfrastructure.SourceFile) (IndexResult, error) {
	previousStates, err := writer.LoadFileStates(ctx, repoName)
	if err != nil {
		return IndexResult{}, fmt.Errorf("indexIncremental: %w", err)
	}

	result := IndexResult{RepoName: repoName}
	presentPaths := make(map[string]bool, len(sourceFiles))
	var updates []codeindexinfrastructure.FileUpdate

	for _, sourceFile := range sourceFiles {
		if err := ctx.Err(); err != nil {
			return IndexResult{}, fmt.Errorf("indexIncremental: %w", err)
		}

		fileUpdate, isIndexable, err := buildFileUpdate(sourceFile, previousStates)
		if err != nil {
			return IndexResult{}, fmt.Errorf("indexIncremental: %w", err)
		}
		if !isIndexable {
			result.SkippedFiles++
			continue
		}

		presentPaths[sourceFile.RelativePath] = true
		if fileUpdate == nil {
			result.UnchangedFiles++
			continue
		}
		updates = append(updates, *fileUpdate)
		result.IndexedFiles++
		result.ChunkCount += len(fileUpdate.Chunks)
		result.DependencyCount += len(fileUpdate.Dependencies)
	}

	removedPaths := collectRemovedPaths(previousStates, presentPaths)
	result.RemovedFiles = len(removedPaths)

	if err := writer.ApplyIndex(ctx, repoName, updates, removedPaths); err != nil {
		return IndexResult{}, fmt.Errorf("indexIncremental: %w", err)
	}
	return result, nil
}

func (i *Indexer) indexFull(ctx context.Context, repoName string, sourceFiles []codeindexinfrastructure.SourceFile) (IndexResult, error) {
	result := IndexResult{RepoName: repoName}
	var allChunks []codeindexdomain.CodeChunk
	var allDependencies []codeindexdomain.ServiceDependency

	for _, sourceFile := range sourceFiles {
		if err := ctx.Err(); err != nil {
			return IndexResult{}, fmt.Errorf("indexFull: %w", err)
		}

		extracted, isIndexable, err := extractSourceFile(sourceFile)
		if err != nil {
			return IndexResult{}, fmt.Errorf("indexFull: %w", err)
		}
		if !isIndexable {
			result.SkippedFiles++
			continue
		}

		allChunks = append(allChunks, extracted.chunks...)
		allDependencies = append(allDependencies, extracted.dependencies...)
		result.IndexedFiles++
		result.ChunkCount += len(extracted.chunks)
		result.DependencyCount += len(extracted.dependencies)
	}

	if err := i.store.ClearRepo(ctx, repoName); err != nil {
		return IndexResult{}, fmt.Errorf("indexFull: %w", err)
	}
	if err := i.store.SaveChunks(ctx, repoName, allChunks); err != nil {
		return IndexResult{}, fmt.Errorf("indexFull: %w", err)
	}
	if err := i.store.SaveDependencies(ctx, repoName, allDependencies); err != nil {
		return IndexResult{}, fmt.Errorf("indexFull: %w", err)
	}
	return result, nil
}

func buildFileUpdate(sourceFile codeindexinfrastructure.SourceFile, previousStates map[string]codeindexinfrastructure.FileState) (*codeindexinfrastructure.FileUpdate, bool, error) {
	content, err := os.ReadFile(sourceFile.AbsolutePath)
	if err != nil {
		return nil, false, fmt.Errorf("read %s: %w", sourceFile.RelativePath, err)
	}
	if !isIndexableContent(content) {
		return nil, false, nil
	}

	fileState := codeindexinfrastructure.FileState{
		RepoName:     sourceFile.RepoName,
		FilePath:     sourceFile.RelativePath,
		ContentHash:  hashContent(content),
		SizeBytes:    int64(len(content)),
		ModifiedUnix: sourceFile.ModifiedUnix,
	}
	if previousState, isKnown := previousStates[sourceFile.RelativePath]; isKnown && previousState.ContentHash == fileState.ContentHash {
		return nil, true, nil
	}

	extracted := extractRedactedContent(string(content), sourceFile)
	fileUpdate := codeindexinfrastructure.FileUpdate{
		State:        fileState,
		Chunks:       extracted.chunks,
		Dependencies: extracted.dependencies,
	}
	return &fileUpdate, true, nil
}

func extractSourceFile(sourceFile codeindexinfrastructure.SourceFile) (extractedSourceFile, bool, error) {
	content, err := os.ReadFile(sourceFile.AbsolutePath)
	if err != nil {
		return extractedSourceFile{}, false, fmt.Errorf("read %s: %w", sourceFile.RelativePath, err)
	}
	if !isIndexableContent(content) {
		return extractedSourceFile{}, false, nil
	}
	return extractRedactedContent(string(content), sourceFile), true, nil
}

func extractRedactedContent(content string, sourceFile codeindexinfrastructure.SourceFile) extractedSourceFile {
	redactedContent, _ := codeindexinfrastructure.RedactSecretsPerLine(content)
	chunks, dependencies := codeindexinfrastructure.ExtractFile(
		redactedContent,
		sourceFile.RelativePath,
		sourceFile.RepoName,
		filepath.Ext(sourceFile.RelativePath),
	)
	return extractedSourceFile{chunks: chunks, dependencies: dependencies}
}

func isIndexableContent(content []byte) bool {
	return len(content) <= codeindexinfrastructure.MaxIndexableFileSize && !codeindexinfrastructure.IsBinaryContent(content)
}

func collectRemovedPaths(previousStates map[string]codeindexinfrastructure.FileState, presentPaths map[string]bool) []string {
	removedPaths := make([]string, 0, len(previousStates))
	for filePath := range previousStates {
		if !presentPaths[filePath] {
			removedPaths = append(removedPaths, filePath)
		}
	}
	sort.Strings(removedPaths)
	return removedPaths
}

func shouldIndexRepository(name string) bool {
	if codeindexinfrastructure.IsSkippedDirectory(name) {
		return false
	}
	if _, err := workspacedomain.ValidateRepoName(name); err != nil {
		return false
	}
	return true
}

func hashContent(content []byte) string {
	contentSum := sha256.Sum256(content)
	return hex.EncodeToString(contentSum[:])
}
