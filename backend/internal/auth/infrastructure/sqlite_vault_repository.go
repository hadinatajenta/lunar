package infrastructure

import (
	"context"
	"database/sql"
	"errors"

	"lunar/backend/internal/auth/domain"
	sharedErrors "lunar/backend/internal/shared/errors"
)

type SQLiteVaultRepository struct {
	db *sql.DB
}

func NewSQLiteVaultRepository(db *sql.DB) *SQLiteVaultRepository {
	return &SQLiteVaultRepository{db: db}
}

func (r *SQLiteVaultRepository) SaveSecrets(ctx context.Context, secrets *domain.UserSecrets) error {
	query := `
		INSERT INTO user_secrets (
			user_id,
			jira_pat_enc,
			jira_username,
			bitbucket_pat_enc,
			bitbucket_username,
			confluence_pat_enc,
			ai_keys_json_enc,
			updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(user_id) DO UPDATE SET
			jira_pat_enc = excluded.jira_pat_enc,
			jira_username = excluded.jira_username,
			bitbucket_pat_enc = excluded.bitbucket_pat_enc,
			bitbucket_username = excluded.bitbucket_username,
			confluence_pat_enc = excluded.confluence_pat_enc,
			ai_keys_json_enc = excluded.ai_keys_json_enc,
			updated_at = excluded.updated_at
	`
	_, err := r.db.ExecContext(
		ctx,
		query,
		secrets.UserID,
		secrets.JiraPATEnc,
		secrets.JiraUsername,
		secrets.BitbucketPATEnc,
		secrets.BitbucketUsername,
		secrets.ConfluencePATEnc,
		secrets.AIKeysJSONEnc,
		secrets.UpdatedAt,
	)
	return err
}

func (r *SQLiteVaultRepository) GetSecretsByUserID(ctx context.Context, userID string) (*domain.UserSecrets, error) {
	query := `
		SELECT
			user_id,
			jira_pat_enc,
			jira_username,
			bitbucket_pat_enc,
			bitbucket_username,
			confluence_pat_enc,
			ai_keys_json_enc,
			updated_at
		FROM user_secrets
		WHERE user_id = ?
	`
	row := r.db.QueryRowContext(ctx, query, userID)
	var secrets domain.UserSecrets
	err := row.Scan(
		&secrets.UserID,
		&secrets.JiraPATEnc,
		&secrets.JiraUsername,
		&secrets.BitbucketPATEnc,
		&secrets.BitbucketUsername,
		&secrets.ConfluencePATEnc,
		&secrets.AIKeysJSONEnc,
		&secrets.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, sharedErrors.ErrNotFound
		}
		return nil, err
	}
	return &secrets, nil
}
