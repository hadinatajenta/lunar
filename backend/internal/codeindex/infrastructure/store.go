package infrastructure

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	codeindexdomain "lunar/backend/internal/codeindex/domain"
)

type Store struct {
	database   *sql.DB
	writeMutex sync.Mutex
}

var _ codeindexdomain.IndexStore = (*Store)(nil)

func NewStore(database *sql.DB) *Store {
	return &Store{database: database}
}

func (s *Store) Close() error {
	if err := s.database.Close(); err != nil {
		return fmt.Errorf("closeStore: %w", err)
	}
	return nil
}

func (s *Store) SaveChunks(ctx context.Context, repoName string, chunks []codeindexdomain.CodeChunk) error {
	s.writeMutex.Lock()
	defer s.writeMutex.Unlock()

	transaction, err := s.database.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("saveChunks: begin transaction: %w", err)
	}
	defer rollbackQuietly(transaction)

	if err := deleteChunks(ctx, transaction, repoName, ""); err != nil {
		return err
	}
	if err := insertChunks(ctx, transaction, repoName, chunks); err != nil {
		return err
	}
	if err := refreshServiceNode(ctx, transaction, repoName); err != nil {
		return err
	}
	if err := transaction.Commit(); err != nil {
		return fmt.Errorf("saveChunks: commit: %w", err)
	}
	return nil
}

func (s *Store) ApplyIndex(ctx context.Context, repoName string, updates []FileUpdate, removedPaths []string) error {
	s.writeMutex.Lock()
	defer s.writeMutex.Unlock()

	transaction, err := s.database.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("applyIndex: begin transaction: %w", err)
	}
	defer rollbackQuietly(transaction)

	for _, removedPath := range removedPaths {
		if err := deleteIndexedFile(ctx, transaction, repoName, removedPath); err != nil {
			return err
		}
	}

	for _, update := range updates {
		if err := deleteIndexedFile(ctx, transaction, repoName, update.State.FilePath); err != nil {
			return err
		}
		if err := insertChunks(ctx, transaction, repoName, update.Chunks); err != nil {
			return err
		}
		if err := insertDependencies(ctx, transaction, repoName, update.Dependencies); err != nil {
			return err
		}
		if err := upsertFileState(ctx, transaction, update.State); err != nil {
			return err
		}
	}

	if err := refreshServiceNode(ctx, transaction, repoName); err != nil {
		return err
	}
	if err := transaction.Commit(); err != nil {
		return fmt.Errorf("applyIndex: commit: %w", err)
	}
	return nil
}

func (s *Store) CountIndexedFiles(ctx context.Context) (int, error) {
	var indexedFileCount int
	if err := s.database.QueryRowContext(ctx, "SELECT COUNT(*) FROM code_files").Scan(&indexedFileCount); err != nil {
		return 0, fmt.Errorf("countIndexedFiles: %w", err)
	}
	return indexedFileCount, nil
}

func (s *Store) LoadFileStates(ctx context.Context, repoName string) (map[string]FileState, error) {
	rows, err := s.database.QueryContext(
		ctx,
		"SELECT file_path, content_hash, size_bytes, modified_unix FROM code_files WHERE repo_name = ?",
		repoName,
	)
	if err != nil {
		return nil, fmt.Errorf("loadFileStates: %w", err)
	}
	defer rows.Close()

	fileStates := make(map[string]FileState)
	for rows.Next() {
		fileState := FileState{RepoName: repoName}
		if err := rows.Scan(&fileState.FilePath, &fileState.ContentHash, &fileState.SizeBytes, &fileState.ModifiedUnix); err != nil {
			return nil, fmt.Errorf("loadFileStates: scan: %w", err)
		}
		fileStates[fileState.FilePath] = fileState
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("loadFileStates: %w", err)
	}
	return fileStates, nil
}

func (s *Store) ClearRepo(ctx context.Context, repoName string) error {
	s.writeMutex.Lock()
	defer s.writeMutex.Unlock()

	transaction, err := s.database.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("clearRepo: begin transaction: %w", err)
	}
	defer rollbackQuietly(transaction)

	statements := []struct {
		query     string
		arguments []any
	}{
		{"DELETE FROM code_chunks_fts WHERE repo_name = ?", []any{repoName}},
		{"DELETE FROM service_deps WHERE from_service = ? OR to_service = ?", []any{repoName, repoName}},
		{"DELETE FROM service_nodes WHERE repo_name = ?", []any{repoName}},
		{"DELETE FROM code_files WHERE repo_name = ?", []any{repoName}},
	}
	for _, statement := range statements {
		if _, err := transaction.ExecContext(ctx, statement.query, statement.arguments...); err != nil {
			return fmt.Errorf("clearRepo: %w", err)
		}
	}
	if err := transaction.Commit(); err != nil {
		return fmt.Errorf("clearRepo: commit: %w", err)
	}
	return nil
}

func deleteIndexedFile(ctx context.Context, transaction *sql.Tx, repoName string, filePath string) error {
	if err := deleteChunks(ctx, transaction, repoName, filePath); err != nil {
		return err
	}
	if err := deleteFileDependencies(ctx, transaction, repoName, filePath); err != nil {
		return err
	}
	if _, err := transaction.ExecContext(
		ctx,
		"DELETE FROM code_files WHERE repo_name = ? AND file_path = ?",
		repoName,
		filePath,
	); err != nil {
		return fmt.Errorf("deleteIndexedFile: %w", err)
	}
	return nil
}

func deleteChunks(ctx context.Context, transaction *sql.Tx, repoName string, filePath string) error {
	if filePath == "" {
		if _, err := transaction.ExecContext(ctx, "DELETE FROM code_chunks_fts WHERE repo_name = ?", repoName); err != nil {
			return fmt.Errorf("deleteChunks: %w", err)
		}
		return nil
	}
	if _, err := transaction.ExecContext(
		ctx,
		"DELETE FROM code_chunks_fts WHERE repo_name = ? AND file_path = ?",
		repoName,
		filePath,
	); err != nil {
		return fmt.Errorf("deleteChunks: %w", err)
	}
	return nil
}

func insertChunks(ctx context.Context, transaction *sql.Tx, repoName string, chunks []codeindexdomain.CodeChunk) error {
	if len(chunks) == 0 {
		return nil
	}

	statement, err := transaction.PrepareContext(
		ctx,
		"INSERT INTO code_chunks_fts(repo_name, file_path, chunk_type, raw_content, start_line, end_line, content) VALUES (?, ?, ?, ?, ?, ?, ?)",
	)
	if err != nil {
		return fmt.Errorf("insertChunks: prepare: %w", err)
	}
	defer func() { _ = statement.Close() }()

	for _, chunk := range chunks {
		if _, err := statement.ExecContext(
			ctx,
			repoName,
			chunk.FilePath,
			chunk.ChunkType,
			chunk.RawContent,
			chunk.StartLine,
			chunk.EndLine,
			chunk.Content,
		); err != nil {
			return fmt.Errorf("insertChunks: insert %s: %w", chunk.FilePath, err)
		}
	}
	return nil
}

func upsertFileState(ctx context.Context, transaction *sql.Tx, fileState FileState) error {
	if _, err := transaction.ExecContext(ctx, `
		INSERT INTO code_files(repo_name, file_path, content_hash, size_bytes, modified_unix, indexed_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(repo_name, file_path) DO UPDATE SET
			content_hash  = excluded.content_hash,
			size_bytes    = excluded.size_bytes,
			modified_unix = excluded.modified_unix,
			indexed_at    = excluded.indexed_at
	`,
		fileState.RepoName,
		fileState.FilePath,
		fileState.ContentHash,
		fileState.SizeBytes,
		fileState.ModifiedUnix,
		time.Now().UTC().Format(time.RFC3339),
	); err != nil {
		return fmt.Errorf("upsertFileState: %w", err)
	}
	return nil
}

func rollbackQuietly(transaction *sql.Tx) {
	_ = transaction.Rollback()
}
