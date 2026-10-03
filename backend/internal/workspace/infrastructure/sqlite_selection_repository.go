package infrastructure

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

type SQLiteSelectionRepository struct {
	db *sql.DB
}

func NewSQLiteSelectionRepository(db *sql.DB) *SQLiteSelectionRepository {
	return &SQLiteSelectionRepository{db: db}
}

func (r *SQLiteSelectionRepository) ListSelected(ctx context.Context, userID string) ([]string, error) {
	query := `
		SELECT repo_name
		FROM repo_selections
		WHERE user_id = ? AND is_selected = 1
		ORDER BY repo_name ASC
	`
	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("list selections: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	repoNames := make([]string, 0)
	for rows.Next() {
		var repoName string
		if err := rows.Scan(&repoName); err != nil {
			return nil, fmt.Errorf("list selections: %w", err)
		}
		repoNames = append(repoNames, repoName)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list selections: %w", err)
	}

	return repoNames, nil
}

func (r *SQLiteSelectionRepository) ReplaceSelections(ctx context.Context, userID string, repoNames []string) error {
	transaction, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin selection transaction: %w", err)
	}
	defer func() {
		_ = transaction.Rollback()
	}()

	if _, err := transaction.ExecContext(ctx, "DELETE FROM repo_selections WHERE user_id = ?", userID); err != nil {
		return fmt.Errorf("clear selections: %w", err)
	}

	insertQuery := `
		INSERT INTO repo_selections (user_id, repo_name, is_selected, updated_at)
		VALUES (?, ?, 1, ?)
	`
	updatedAt := time.Now().UTC()
	for _, repoName := range repoNames {
		if _, err := transaction.ExecContext(ctx, insertQuery, userID, repoName, updatedAt); err != nil {
			return fmt.Errorf("insert selection: %w", err)
		}
	}

	if err := transaction.Commit(); err != nil {
		return fmt.Errorf("commit selections: %w", err)
	}

	return nil
}
