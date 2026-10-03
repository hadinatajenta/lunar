package infrastructure

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	codeindexdomain "lunar/backend/internal/codeindex/domain"
)

const (
	defaultSearchLimit     = 10
	repositoryListingLimit = 50
	chunkSelectColumns     = "repo_name, file_path, chunk_type, raw_content, start_line, end_line"
)

var indexedLanguages = []string{"Go", "Java", "Node.js", "PHP"}

func (s *Store) SearchChunks(ctx context.Context, query string, repoFilter string, limit int) ([]codeindexdomain.CodeChunk, error) {
	var repositories []string
	if strings.TrimSpace(repoFilter) != "" {
		repositories = []string{repoFilter}
	}
	return s.searchChunks(ctx, query, repositories, limit)
}

func (s *Store) SearchChunksIn(ctx context.Context, query string, repoFilter []string, limit int) ([]codeindexdomain.CodeChunk, error) {
	return s.searchChunks(ctx, query, repoFilter, limit)
}

func (s *Store) GetIndexStatus(ctx context.Context) ([]codeindexdomain.IndexStatus, error) {
	rows, err := s.database.QueryContext(ctx, "SELECT repo_name, chunk_count, indexed_at FROM service_nodes ORDER BY repo_name")
	if err != nil {
		return nil, fmt.Errorf("getIndexStatus: %w", err)
	}
	defer rows.Close()

	var statuses []codeindexdomain.IndexStatus
	for rows.Next() {
		var status codeindexdomain.IndexStatus
		if err := rows.Scan(&status.RepoName, &status.ChunkCount, &status.IndexedAt); err != nil {
			return nil, fmt.Errorf("getIndexStatus: scan: %w", err)
		}
		status.Status = "empty"
		if status.ChunkCount > 0 {
			status.Status = "indexed"
		}
		statuses = append(statuses, status)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("getIndexStatus: %w", err)
	}
	return statuses, nil
}

func (s *Store) GetAllServices(ctx context.Context) ([]codeindexdomain.ServiceNode, error) {
	rows, err := s.database.QueryContext(ctx, "SELECT repo_name, language, chunk_count, indexed_at FROM service_nodes ORDER BY repo_name")
	if err != nil {
		return nil, fmt.Errorf("getAllServices: %w", err)
	}
	defer rows.Close()

	var services []codeindexdomain.ServiceNode
	for rows.Next() {
		var service codeindexdomain.ServiceNode
		if err := rows.Scan(&service.RepoName, &service.Language, &service.ChunkCount, &service.IndexedAt); err != nil {
			return nil, fmt.Errorf("getAllServices: scan: %w", err)
		}
		services = append(services, service)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("getAllServices: %w", err)
	}
	return services, nil
}

func (s *Store) searchChunks(ctx context.Context, query string, repositories []string, limit int) ([]codeindexdomain.CodeChunk, error) {
	hasQuery := strings.TrimSpace(query) != ""
	if limit <= 0 {
		limit = repositoryListingLimit
		if hasQuery {
			limit = defaultSearchLimit
		}
	}

	filterClause, filterArguments := repositoryFilterClause(repositories)
	if !hasQuery {
		return s.listChunks(ctx, filterClause, filterArguments, limit)
	}

	statement := "SELECT " + chunkSelectColumns + " FROM code_chunks_fts WHERE code_chunks_fts MATCH ?"
	arguments := []any{sanitizeFTSQuery(query)}
	if filterClause != "" {
		statement += " AND " + filterClause
		arguments = append(arguments, filterArguments...)
	}
	statement += " ORDER BY rank LIMIT ?"
	arguments = append(arguments, limit)

	rows, err := s.database.QueryContext(ctx, statement, arguments...)
	if err != nil {
		return s.searchChunksWithLike(ctx, query, filterClause, filterArguments, limit)
	}
	defer rows.Close()
	return scanChunks(rows)
}

func (s *Store) listChunks(ctx context.Context, filterClause string, filterArguments []any, limit int) ([]codeindexdomain.CodeChunk, error) {
	statement := "SELECT " + chunkSelectColumns + " FROM code_chunks_fts"
	arguments := make([]any, 0, len(filterArguments)+1)
	if filterClause != "" {
		statement += " WHERE " + filterClause
		arguments = append(arguments, filterArguments...)
	}
	statement += " LIMIT ?"
	arguments = append(arguments, limit)

	rows, err := s.database.QueryContext(ctx, statement, arguments...)
	if err != nil {
		return nil, fmt.Errorf("listChunks: %w", err)
	}
	defer rows.Close()
	return scanChunks(rows)
}

func (s *Store) searchChunksWithLike(ctx context.Context, query string, filterClause string, filterArguments []any, limit int) ([]codeindexdomain.CodeChunk, error) {
	statement := "SELECT " + chunkSelectColumns + " FROM code_chunks_fts WHERE lower(content) LIKE ?"
	arguments := []any{"%" + strings.ToLower(query) + "%"}
	if filterClause != "" {
		statement += " AND " + filterClause
		arguments = append(arguments, filterArguments...)
	}
	statement += " LIMIT ?"
	arguments = append(arguments, limit)

	rows, err := s.database.QueryContext(ctx, statement, arguments...)
	if err != nil {
		return nil, fmt.Errorf("searchChunksWithLike: %w", err)
	}
	defer rows.Close()
	return scanChunks(rows)
}

func refreshServiceNode(ctx context.Context, transaction *sql.Tx, repoName string) error {
	var chunkCount int
	if err := transaction.QueryRowContext(
		ctx,
		"SELECT COUNT(*) FROM code_chunks_fts WHERE repo_name = ?",
		repoName,
	).Scan(&chunkCount); err != nil {
		return fmt.Errorf("refreshServiceNode: count chunks: %w", err)
	}

	language, err := detectLanguage(ctx, transaction, repoName)
	if err != nil {
		return err
	}

	if _, err := transaction.ExecContext(ctx, `
		INSERT INTO service_nodes(repo_name, language, chunk_count, indexed_at)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(repo_name) DO UPDATE SET
			language    = excluded.language,
			chunk_count = excluded.chunk_count,
			indexed_at  = excluded.indexed_at
	`,
		repoName,
		language,
		chunkCount,
		time.Now().UTC().Format(time.RFC3339),
	); err != nil {
		return fmt.Errorf("refreshServiceNode: upsert: %w", err)
	}
	return nil
}

func detectLanguage(ctx context.Context, transaction *sql.Tx, repoName string) (string, error) {
	rows, err := transaction.QueryContext(ctx, "SELECT DISTINCT file_path FROM code_chunks_fts WHERE repo_name = ?", repoName)
	if err != nil {
		return "", fmt.Errorf("detectLanguage: %w", err)
	}
	defer rows.Close()

	languageCounts := make(map[string]int)
	for rows.Next() {
		var filePath string
		if err := rows.Scan(&filePath); err != nil {
			return "", fmt.Errorf("detectLanguage: scan: %w", err)
		}
		if language := languageForFile(filePath); language != "" {
			languageCounts[language]++
		}
	}
	if err := rows.Err(); err != nil {
		return "", fmt.Errorf("detectLanguage: %w", err)
	}

	bestLanguage, bestCount := "Unknown", 0
	for _, language := range indexedLanguages {
		if languageCounts[language] > bestCount {
			bestLanguage = language
			bestCount = languageCounts[language]
		}
	}
	return bestLanguage, nil
}

func languageForFile(filePath string) string {
	switch strings.ToLower(filepath.Ext(filePath)) {
	case ".go":
		return "Go"
	case ".java":
		return "Java"
	case ".js", ".ts":
		return "Node.js"
	case ".php":
		return "PHP"
	default:
		return ""
	}
}

func repositoryFilterClause(repositories []string) (string, []any) {
	if len(repositories) == 0 {
		return "", nil
	}

	placeholders := strings.TrimSuffix(strings.Repeat("?, ", len(repositories)), ", ")
	arguments := make([]any, 0, len(repositories))
	for _, repository := range repositories {
		arguments = append(arguments, repository)
	}
	return "repo_name IN (" + placeholders + ")", arguments
}

func scanChunks(rows *sql.Rows) ([]codeindexdomain.CodeChunk, error) {
	var chunks []codeindexdomain.CodeChunk
	for rows.Next() {
		var chunk codeindexdomain.CodeChunk
		if err := rows.Scan(
			&chunk.RepoName,
			&chunk.FilePath,
			&chunk.ChunkType,
			&chunk.RawContent,
			&chunk.StartLine,
			&chunk.EndLine,
		); err != nil {
			return nil, fmt.Errorf("scanChunks: %w", err)
		}
		chunks = append(chunks, chunk)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("scanChunks: %w", err)
	}
	return chunks, nil
}
