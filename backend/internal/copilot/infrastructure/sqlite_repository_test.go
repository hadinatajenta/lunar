package infrastructure

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"lunar/backend/internal/copilot/domain"
	sharedDatabase "lunar/backend/internal/shared/database"
)

func openCopilotTestDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sharedDatabase.OpenDB(filepath.Join(t.TempDir(), "copilot.db"))
	if err != nil {
		t.Fatalf("cannot open test database: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})

	if err := sharedDatabase.AutoMigrate(db); err != nil {
		t.Fatalf("cannot migrate test database: %v", err)
	}

	return db
}

func seedCopilotChat(t *testing.T, db *sql.DB, chatID string, userID string) {
	t.Helper()

	now := time.Now().UTC()
	if _, err := db.Exec(
		"INSERT INTO users (id, email, password_hash, salt, full_name, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)",
		userID,
		userID+"@example.com",
		"password-hash",
		"salt-value",
		userID,
		now,
		now,
	); err != nil {
		t.Fatalf("cannot seed user %q: %v", userID, err)
	}

	if _, err := db.Exec(
		"INSERT INTO copilot_chats (id, user_id, title, model, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)",
		chatID,
		userID,
		"Code question",
		"deepseek-v4-pro",
		now.Format(time.RFC3339),
		now.Format(time.RFC3339),
	); err != nil {
		t.Fatalf("cannot seed chat %q: %v", chatID, err)
	}
}

func TestSaveMessagePersistsSourcesAndReasoningDuration(t *testing.T) {
	db := openCopilotTestDB(t)
	seedCopilotChat(t, db, "chat-1", "user-1")

	repository := NewSQLiteCopilotRepository(db)
	ctx := context.Background()

	message := &domain.ChatMessage{
		ID:                 "msg-ast-1",
		ChatID:             "chat-1",
		Role:               "assistant",
		Content:            "The late fee is calculated in the settlement calculator.",
		Reasoning:          "Searched the settlement repository.",
		ThinkingDurationMs: 4210,
		Sources: []domain.ChatSource{
			{
				Type:      "code",
				Label:     "settlement-service/internal/fee/calculator.go",
				Repo:      "settlement-service",
				Path:      "internal/fee/calculator.go",
				StartLine: 40,
				EndLine:   96,
			},
			{
				Type:      "code",
				Label:     "aurora/src/qcQris/handler.go",
				Repo:      "aurora",
				Path:      "src/qcQris/handler.go",
				StartLine: 12,
				EndLine:   38,
			},
		},
		CreatedAt: time.Now().UTC(),
	}

	if err := repository.SaveMessage(ctx, message); err != nil {
		t.Fatalf("SaveMessage returned error: %v", err)
	}

	loaded, err := repository.GetMessagesByChatID(ctx, "chat-1")
	if err != nil {
		t.Fatalf("GetMessagesByChatID returned error: %v", err)
	}
	if len(loaded) != 1 {
		t.Fatalf("expected 1 message, got %d", len(loaded))
	}

	restored := loaded[0]
	if restored.ThinkingDurationMs != 4210 {
		t.Fatalf("expected thinking duration 4210, got %d", restored.ThinkingDurationMs)
	}
	if len(restored.Sources) != 2 {
		t.Fatalf("expected 2 sources, got %d", len(restored.Sources))
	}

	first := restored.Sources[0]
	if first.Repo != "settlement-service" {
		t.Fatalf("expected settlement-service, got %q", first.Repo)
	}
	if first.Path != "internal/fee/calculator.go" {
		t.Fatalf("expected internal/fee/calculator.go, got %q", first.Path)
	}
	if first.StartLine != 40 || first.EndLine != 96 {
		t.Fatalf("expected line range 40-96, got %d-%d", first.StartLine, first.EndLine)
	}
	if first.Label != "settlement-service/internal/fee/calculator.go" {
		t.Fatalf("expected the label to survive, got %q", first.Label)
	}

	second := restored.Sources[1]
	if second.Repo != "aurora" || second.StartLine != 12 || second.EndLine != 38 {
		t.Fatalf("second source did not round-trip: %+v", second)
	}
}

func TestSaveMessageWithoutSourcesReadsBackEmpty(t *testing.T) {
	db := openCopilotTestDB(t)
	seedCopilotChat(t, db, "chat-2", "user-2")

	repository := NewSQLiteCopilotRepository(db)
	ctx := context.Background()

	message := &domain.ChatMessage{
		ID:        "msg-user-1",
		ChatID:    "chat-2",
		Role:      "user",
		Content:   "Where is the late fee calculated?",
		CreatedAt: time.Now().UTC(),
	}

	if err := repository.SaveMessage(ctx, message); err != nil {
		t.Fatalf("SaveMessage returned error: %v", err)
	}

	loaded, err := repository.GetMessagesByChatID(ctx, "chat-2")
	if err != nil {
		t.Fatalf("GetMessagesByChatID returned error: %v", err)
	}
	if len(loaded) != 1 {
		t.Fatalf("expected 1 message, got %d", len(loaded))
	}
	if len(loaded[0].Sources) != 0 {
		t.Fatalf("expected no sources, got %d", len(loaded[0].Sources))
	}
	if loaded[0].ThinkingDurationMs != 0 {
		t.Fatalf("expected zero thinking duration, got %d", loaded[0].ThinkingDurationMs)
	}
	if loaded[0].Content != "Where is the late fee calculated?" {
		t.Fatalf("content did not round-trip: %q", loaded[0].Content)
	}
}

func TestAutoMigrateAddsMessageSourcesColumnToLegacySchema(t *testing.T) {
	db, err := sharedDatabase.OpenDB(filepath.Join(t.TempDir(), "legacy.db"))
	if err != nil {
		t.Fatalf("cannot open database: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})

	legacyChats := `CREATE TABLE copilot_chats (
		id TEXT PRIMARY KEY, user_id TEXT NOT NULL, title TEXT NOT NULL,
		model TEXT NOT NULL, created_at DATETIME NOT NULL, updated_at DATETIME NOT NULL)`
	if _, err := db.Exec(legacyChats); err != nil {
		t.Fatalf("cannot create legacy chats table: %v", err)
	}

	legacyMessages := `CREATE TABLE copilot_messages (
		id TEXT PRIMARY KEY, chat_id TEXT NOT NULL, role TEXT NOT NULL,
		content TEXT NOT NULL, reasoning TEXT NOT NULL DEFAULT '', created_at DATETIME NOT NULL)`
	if _, err := db.Exec(legacyMessages); err != nil {
		t.Fatalf("cannot create legacy messages table: %v", err)
	}

	if err := sharedDatabase.AutoMigrate(db); err != nil {
		t.Fatalf("AutoMigrate on a legacy schema returned error: %v", err)
	}

	if !columnExists(t, db, "copilot_messages", "sources") {
		t.Fatalf("expected the sources column to be added to a legacy schema")
	}
	if !columnExists(t, db, "copilot_messages", "thinking_duration_ms") {
		t.Fatalf("expected the thinking_duration_ms column to be added to a legacy schema")
	}
}

func columnExists(t *testing.T, db *sql.DB, table string, column string) bool {
	t.Helper()

	var count int
	query := "SELECT COUNT(*) FROM pragma_table_info('" + table + "') WHERE name = ?"
	if err := db.QueryRow(query, column).Scan(&count); err != nil {
		t.Fatalf("cannot inspect %s.%s: %v", table, column, err)
	}
	return count > 0
}
