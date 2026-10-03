package infrastructure

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	sharedErrors "lunar/backend/internal/shared/errors"
	"lunar/backend/internal/workspace/domain"
)

type SQLiteWorkspaceRepository struct {
	db *sql.DB
}

func NewSQLiteWorkspaceRepository(db *sql.DB) *SQLiteWorkspaceRepository {
	return &SQLiteWorkspaceRepository{db: db}
}

func (r *SQLiteWorkspaceRepository) GetWorkspace(ctx context.Context, userID string) (*domain.Workspace, error) {
	query := `
		SELECT user_id, source, root_path, helper_url, helper_token_enc, created_at, updated_at
		FROM workspaces
		WHERE user_id = ?
	`
	row := r.db.QueryRowContext(ctx, query, userID)

	var workspace domain.Workspace
	err := row.Scan(
		&workspace.UserID,
		&workspace.Source,
		&workspace.RootPath,
		&workspace.HelperURL,
		&workspace.HelperTokenEnc,
		&workspace.CreatedAt,
		&workspace.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, sharedErrors.ErrNotFound
		}
		return nil, fmt.Errorf("get workspace: %w", err)
	}

	return &workspace, nil
}

func (r *SQLiteWorkspaceRepository) UpsertWorkspace(ctx context.Context, workspace *domain.Workspace) error {
	query := `
		INSERT INTO workspaces (user_id, source, root_path, helper_url, helper_token_enc, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(user_id) DO UPDATE SET
			source = excluded.source,
			root_path = excluded.root_path,
			helper_url = excluded.helper_url,
			helper_token_enc = excluded.helper_token_enc,
			updated_at = excluded.updated_at
	`
	_, err := r.db.ExecContext(
		ctx,
		query,
		workspace.UserID,
		string(workspace.Source),
		workspace.RootPath,
		workspace.HelperURL,
		workspace.HelperTokenEnc,
		workspace.CreatedAt,
		workspace.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("upsert workspace: %w", err)
	}

	return nil
}
