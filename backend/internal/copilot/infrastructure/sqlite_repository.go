package infrastructure

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"lunar/backend/internal/copilot/domain"
	sharedErrors "lunar/backend/internal/shared/errors"
)

type SQLiteCopilotRepository struct {
	db *sql.DB
}

func NewSQLiteCopilotRepository(db *sql.DB) *SQLiteCopilotRepository {
	return &SQLiteCopilotRepository{db: db}
}

func (r *SQLiteCopilotRepository) GetSessionsByUserID(ctx context.Context, userID string) ([]domain.ChatSession, error) {
	query := `
		SELECT id, user_id, title, model, created_at, updated_at
		FROM copilot_chats
		WHERE user_id = ?
		ORDER BY updated_at DESC
	`
	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []domain.ChatSession
	for rows.Next() {
		var s domain.ChatSession
		var createdAt, updatedAt string
		if err := rows.Scan(&s.ID, &s.UserID, &s.Title, &s.Model, &createdAt, &updatedAt); err != nil {
			return nil, err
		}
		s.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		s.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
		s.ThinkingMode = true
		s.ReasoningEffort = domain.ReasoningEffortHigh
		sessions = append(sessions, s)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return sessions, nil
}

func (r *SQLiteCopilotRepository) GetSessionByID(ctx context.Context, sessionID string) (*domain.ChatSession, error) {
	query := `
		SELECT id, user_id, title, model, created_at, updated_at
		FROM copilot_chats
		WHERE id = ?
	`
	row := r.db.QueryRowContext(ctx, query, sessionID)

	var s domain.ChatSession
	var createdAt, updatedAt string
	if err := row.Scan(&s.ID, &s.UserID, &s.Title, &s.Model, &createdAt, &updatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, sharedErrors.ErrNotFound
		}
		return nil, err
	}

	s.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	s.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
	s.ThinkingMode = true
	s.ReasoningEffort = domain.ReasoningEffortHigh
	return &s, nil
}

func (r *SQLiteCopilotRepository) CreateSession(ctx context.Context, session *domain.ChatSession) error {
	query := `
		INSERT INTO copilot_chats (id, user_id, title, model, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`
	_, err := r.db.ExecContext(
		ctx,
		query,
		session.ID,
		session.UserID,
		session.Title,
		session.Model,
		session.CreatedAt.Format(time.RFC3339),
		session.UpdatedAt.Format(time.RFC3339),
	)
	return err
}

func (r *SQLiteCopilotRepository) UpdateSession(ctx context.Context, session *domain.ChatSession) error {
	query := `
		UPDATE copilot_chats
		SET title = ?, model = ?, updated_at = ?
		WHERE id = ? AND user_id = ?
	`
	res, err := r.db.ExecContext(
		ctx,
		query,
		session.Title,
		session.Model,
		session.UpdatedAt.Format(time.RFC3339),
		session.ID,
		session.UserID,
	)
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return sharedErrors.ErrNotFound
	}
	return nil
}

func (r *SQLiteCopilotRepository) DeleteSession(ctx context.Context, userID string, sessionID string) error {
	query := `DELETE FROM copilot_chats WHERE id = ? AND user_id = ?`
	res, err := r.db.ExecContext(ctx, query, sessionID, userID)
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return sharedErrors.ErrNotFound
	}
	return nil
}

func (r *SQLiteCopilotRepository) GetMessagesByChatID(ctx context.Context, chatID string) ([]domain.ChatMessage, error) {
	query := `
		SELECT id, chat_id, role, content, reasoning, created_at
		FROM copilot_messages
		WHERE chat_id = ?
		ORDER BY created_at ASC
	`
	rows, err := r.db.QueryContext(ctx, query, chatID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []domain.ChatMessage
	for rows.Next() {
		var m domain.ChatMessage
		var createdAt string
		if err := rows.Scan(&m.ID, &m.ChatID, &m.Role, &m.Content, &m.Reasoning, &createdAt); err != nil {
			return nil, err
		}
		m.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		messages = append(messages, m)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return messages, nil
}

func (r *SQLiteCopilotRepository) SaveMessage(ctx context.Context, msg *domain.ChatMessage) error {
	query := `
		INSERT INTO copilot_messages (id, chat_id, role, content, reasoning, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`
	_, err := r.db.ExecContext(
		ctx,
		query,
		msg.ID,
		msg.ChatID,
		msg.Role,
		msg.Content,
		msg.Reasoning,
		msg.CreatedAt.Format(time.RFC3339),
	)
	return err
}
