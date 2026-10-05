package infrastructure

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"testing"
	"time"

	"lunar/backend/internal/attachment/domain"
	sharedDatabase "lunar/backend/internal/shared/database"
	sharedErrors "lunar/backend/internal/shared/errors"
)

func openAttachmentStore(t *testing.T) (*SQLiteStore, *sql.DB) {
	t.Helper()

	db, err := sharedDatabase.OpenDB(t.TempDir() + "/attachments.db")
	if err != nil {
		t.Fatalf("cannot open the test database: %v", err)
	}
	if err := sharedDatabase.AutoMigrate(db); err != nil {
		t.Fatalf("cannot migrate the test database: %v", err)
	}
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Errorf("cannot close the test database: %v", err)
		}
	})

	seedAttachmentUser(t, db, "user-a")
	seedAttachmentUser(t, db, "user-b")

	return NewSQLiteStore(db), db
}

func seedAttachmentUser(t *testing.T, db *sql.DB, userID string) {
	t.Helper()

	now := time.Now().UTC().Format(time.RFC3339Nano)
	if _, err := db.Exec(
		`INSERT INTO users (id, email, password_hash, salt, full_name, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		userID,
		userID+"@example.com",
		"hash",
		"salt",
		userID,
		now,
		now,
	); err != nil {
		t.Fatalf("cannot seed user %s: %v", userID, err)
	}
}

func TestSQLiteStore_CreateAndLoadAnAttachment(t *testing.T) {
	store, _ := openAttachmentStore(t)
	content := []byte("PK\x03\x04 document bytes")
	createdAt := time.Now().UTC().Truncate(time.Millisecond)

	attachment := &domain.Attachment{
		ID:          "attachment-1",
		UserID:      "user-a",
		Name:        "plan.docx",
		Kind:        domain.KindDocument,
		ContentType: "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		Size:        int64(len(content)),
		Preview:     "Release plan",
		Text:        "Release plan",
		Content:     content,
		CreatedAt:   createdAt,
	}
	if err := store.Create(context.Background(), attachment); err != nil {
		t.Fatalf("cannot store the attachment: %v", err)
	}

	loaded, err := store.GetByID(context.Background(), "user-a", "attachment-1")
	if err != nil {
		t.Fatalf("cannot load the attachment: %v", err)
	}

	if loaded.ID != "attachment-1" || loaded.Kind != domain.KindDocument || loaded.Name != "plan.docx" {
		t.Errorf("unexpected loaded attachment: %+v", loaded)
	}
	if loaded.Text != "Release plan" || loaded.Preview != "Release plan" {
		t.Errorf("expected the extracted text and preview to survive, got %+v", loaded)
	}
	if string(loaded.Content) != string(content) {
		t.Errorf("expected the stored content, got %q", loaded.Content)
	}
	if loaded.Size != int64(len(content)) {
		t.Errorf("expected size %d, got %d", len(content), loaded.Size)
	}
	if !loaded.CreatedAt.Equal(createdAt) {
		t.Errorf("expected created at %s, got %s", createdAt, loaded.CreatedAt)
	}
}

func TestSQLiteStore_HidesAnAttachmentFromAnotherUser(t *testing.T) {
	store, _ := openAttachmentStore(t)
	if err := store.Create(context.Background(), &domain.Attachment{
		ID:          "attachment-1",
		UserID:      "user-a",
		Name:        "chart.png",
		Kind:        domain.KindImage,
		ContentType: "image/png",
		Size:        4,
		Content:     []byte("data"),
		CreatedAt:   time.Now().UTC(),
	}); err != nil {
		t.Fatalf("cannot store the attachment: %v", err)
	}

	if _, err := store.GetByID(context.Background(), "user-b", "attachment-1"); !errors.Is(err, sharedErrors.ErrNotFound) {
		t.Errorf("expected another user to receive ErrNotFound, got %v", err)
	}
	if _, err := store.GetByID(context.Background(), "user-a", "attachment-missing"); !errors.Is(err, sharedErrors.ErrNotFound) {
		t.Errorf("expected a missing id to receive ErrNotFound, got %v", err)
	}
}

func TestSQLiteStore_PrunesTheOldestAttachmentsBeyondThePerUserCap(t *testing.T) {
	store, db := openAttachmentStore(t)
	baseTime := time.Now().UTC()

	const insertedCount = 55
	for index := 0; index < insertedCount; index++ {
		if err := store.Create(context.Background(), &domain.Attachment{
			ID:          fmt.Sprintf("attachment-%02d", index),
			UserID:      "user-a",
			Name:        "chart.png",
			Kind:        domain.KindImage,
			ContentType: "image/png",
			Size:        4,
			Content:     []byte("data"),
			CreatedAt:   baseTime.Add(time.Duration(index) * time.Millisecond),
		}); err != nil {
			t.Fatalf("cannot store attachment %d: %v", index, err)
		}
	}

	var remaining int
	if err := db.QueryRow(
		"SELECT COUNT(*) FROM copilot_attachments WHERE user_id = ?",
		"user-a",
	).Scan(&remaining); err != nil {
		t.Fatalf("cannot count the stored attachments: %v", err)
	}
	if remaining != maxAttachmentsPerUser {
		t.Errorf("expected %d attachments to remain, got %d", maxAttachmentsPerUser, remaining)
	}

	if _, err := store.GetByID(context.Background(), "user-a", "attachment-00"); !errors.Is(err, sharedErrors.ErrNotFound) {
		t.Errorf("expected the oldest attachment to be evicted, got %v", err)
	}

	newest, err := store.GetByID(context.Background(), "user-a", fmt.Sprintf("attachment-%02d", insertedCount-1))
	if err != nil {
		t.Fatalf("expected the newest attachment to survive, got %v", err)
	}
	if newest.Name != "chart.png" {
		t.Errorf("unexpected newest attachment: %+v", newest)
	}
}
