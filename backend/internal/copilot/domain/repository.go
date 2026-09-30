package domain

import "context"

type CopilotRepository interface {
	GetSessionsByUserID(ctx context.Context, userID string) ([]ChatSession, error)
	GetSessionByID(ctx context.Context, sessionID string) (*ChatSession, error)
	CreateSession(ctx context.Context, session *ChatSession) error
	UpdateSession(ctx context.Context, session *ChatSession) error
	DeleteSession(ctx context.Context, userID string, sessionID string) error
	GetMessagesByChatID(ctx context.Context, chatID string) ([]ChatMessage, error)
	SaveMessage(ctx context.Context, msg *ChatMessage) error
}
