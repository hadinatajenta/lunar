package infrastructure

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	codeindexdomain "lunar/backend/internal/codeindex/domain"
)

const defaultKnowledgeLimit = 5

func (s *Store) SearchKnowledge(ctx context.Context, query string, limit int) ([]codeindexdomain.DomainKnowledgeItem, error) {
	if strings.TrimSpace(query) == "" {
		return nil, nil
	}
	if limit <= 0 {
		limit = defaultKnowledgeLimit
	}

	rows, err := s.database.QueryContext(ctx, `
		SELECT term, explanation
		FROM domain_knowledge_fts
		WHERE domain_knowledge_fts MATCH ?
		ORDER BY rank
		LIMIT ?
	`, sanitizeFTSQuery(query), limit)
	if err != nil {
		return s.searchKnowledgeWithLike(ctx, query, limit)
	}
	defer rows.Close()
	return scanKnowledgeItems(rows)
}

func (s *Store) searchKnowledgeWithLike(ctx context.Context, query string, limit int) ([]codeindexdomain.DomainKnowledgeItem, error) {
	pattern := "%" + strings.ToLower(query) + "%"
	rows, err := s.database.QueryContext(ctx, `
		SELECT term, explanation
		FROM domain_knowledge_fts
		WHERE lower(term) LIKE ? OR lower(explanation) LIKE ?
		LIMIT ?
	`, pattern, pattern, limit)
	if err != nil {
		return nil, fmt.Errorf("searchKnowledgeWithLike: %w", err)
	}
	defer rows.Close()
	return scanKnowledgeItems(rows)
}

func (s *Store) SaveKnowledge(ctx context.Context, items []codeindexdomain.DomainKnowledgeItem) error {
	if len(items) == 0 {
		return nil
	}

	s.writeMutex.Lock()
	defer s.writeMutex.Unlock()

	transaction, err := s.database.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("saveKnowledge: begin transaction: %w", err)
	}
	defer rollbackQuietly(transaction)

	statement, err := transaction.PrepareContext(ctx, "INSERT INTO domain_knowledge_fts(term, explanation) VALUES (?, ?)")
	if err != nil {
		return fmt.Errorf("saveKnowledge: prepare: %w", err)
	}
	defer func() { _ = statement.Close() }()

	for _, item := range items {
		if strings.TrimSpace(item.Term) == "" && strings.TrimSpace(item.Explanation) == "" {
			continue
		}
		if _, err := statement.ExecContext(ctx, item.Term, item.Explanation); err != nil {
			return fmt.Errorf("saveKnowledge: insert %s: %w", item.Term, err)
		}
	}
	if err := transaction.Commit(); err != nil {
		return fmt.Errorf("saveKnowledge: commit: %w", err)
	}
	return nil
}

func (s *Store) ClearKnowledge(ctx context.Context) error {
	s.writeMutex.Lock()
	defer s.writeMutex.Unlock()

	if _, err := s.database.ExecContext(ctx, "DELETE FROM domain_knowledge_fts"); err != nil {
		return fmt.Errorf("clearKnowledge: %w", err)
	}
	return nil
}

func scanKnowledgeItems(rows *sql.Rows) ([]codeindexdomain.DomainKnowledgeItem, error) {
	var items []codeindexdomain.DomainKnowledgeItem
	for rows.Next() {
		var item codeindexdomain.DomainKnowledgeItem
		if err := rows.Scan(&item.Term, &item.Explanation); err != nil {
			return nil, fmt.Errorf("scanKnowledgeItems: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("scanKnowledgeItems: %w", err)
	}
	return items, nil
}
