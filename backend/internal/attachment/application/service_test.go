package application

import (
	"context"
	"errors"
	"strings"
	"testing"

	"lunar/backend/internal/attachment/domain"
	"lunar/backend/internal/attachment/testsupport"
	sharedErrors "lunar/backend/internal/shared/errors"
)

type fakeStore struct {
	items       map[string]*domain.Attachment
	createCalls int
	lastUserID  string
	createError error
}

func newFakeStore() *fakeStore {
	return &fakeStore{items: make(map[string]*domain.Attachment)}
}

func (f *fakeStore) Create(ctx context.Context, attachment *domain.Attachment) error {
	f.createCalls++
	f.lastUserID = attachment.UserID
	if f.createError != nil {
		return f.createError
	}
	stored := *attachment
	f.items[attachment.ID] = &stored
	return nil
}

func (f *fakeStore) GetByID(ctx context.Context, userID string, attachmentID string) (*domain.Attachment, error) {
	stored, exists := f.items[attachmentID]
	if !exists || stored.UserID != userID {
		return nil, sharedErrors.ErrNotFound
	}
	return stored, nil
}

func TestServiceUpload_StoresADocumentWithExtractedTextAndPreview(t *testing.T) {
	store := newFakeStore()
	service := NewService(store)

	attachment, err := service.Upload(context.Background(), "user-a", "plan.docx", testsupport.DOCX(t, []string{"Release plan"}))
	if err != nil {
		t.Fatalf("cannot upload the docx: %v", err)
	}

	if attachment.Kind != domain.KindDocument {
		t.Errorf("expected a document, got %q", attachment.Kind)
	}
	if attachment.UserID != "user-a" {
		t.Errorf("expected the attachment to belong to user-a, got %q", attachment.UserID)
	}
	if !strings.Contains(attachment.Text, "Release plan") {
		t.Errorf("expected the extracted text to be stored, got %q", attachment.Text)
	}
	if attachment.Preview != "Release plan" {
		t.Errorf("expected the preview to summarize the document, got %q", attachment.Preview)
	}
	if attachment.ID == "" {
		t.Error("expected a generated attachment id")
	}
	if store.createCalls != 1 {
		t.Errorf("expected exactly one store write, got %d", store.createCalls)
	}
	if stored := store.items[attachment.ID]; stored == nil || len(stored.Content) == 0 {
		t.Error("expected the raw document content to be stored")
	}
}

func TestServiceUpload_StoresAnImageWithADescriptivePreview(t *testing.T) {
	store := newFakeStore()
	service := NewService(store)

	attachment, err := service.Upload(context.Background(), "user-a", "chart.png", testsupport.PNG(t))
	if err != nil {
		t.Fatalf("cannot upload the png: %v", err)
	}

	if attachment.Kind != domain.KindImage {
		t.Errorf("expected an image, got %q", attachment.Kind)
	}
	if attachment.ContentType != "image/png" {
		t.Errorf("expected the detected content type, got %q", attachment.ContentType)
	}
	if attachment.Preview != "PNG image, 3x2" {
		t.Errorf("expected the preview to describe the image, got %q", attachment.Preview)
	}
	if attachment.Text != "" {
		t.Errorf("expected no extracted text for an image, got %q", attachment.Text)
	}
}

func TestServiceUpload_RejectsAnUnsupportedTypeWithoutWriting(t *testing.T) {
	store := newFakeStore()
	service := NewService(store)

	_, err := service.Upload(context.Background(), "user-a", "payload.zip", []byte("PK\x03\x04"))
	if !errors.Is(err, ErrUnsupportedType) {
		t.Fatalf("expected ErrUnsupportedType, got %v", err)
	}
	if store.createCalls != 0 {
		t.Errorf("expected no store write for a rejected upload, got %d", store.createCalls)
	}
}

func TestServiceUpload_RejectsAnUnreadableDocument(t *testing.T) {
	store := newFakeStore()
	service := NewService(store)

	_, err := service.Upload(context.Background(), "user-a", "broken.pdf", []byte("not a pdf at all"))
	if !errors.Is(err, ErrUnreadableDocument) {
		t.Fatalf("expected ErrUnreadableDocument, got %v", err)
	}
	if store.createCalls != 0 {
		t.Errorf("expected no store write for an unreadable document, got %d", store.createCalls)
	}
}

func TestServiceUpload_RequiresAnAuthenticatedUser(t *testing.T) {
	service := NewService(newFakeStore())

	_, err := service.Upload(context.Background(), "  ", "chart.png", testsupport.PNG(t))
	if !errors.Is(err, sharedErrors.ErrUnauthorized) {
		t.Fatalf("expected ErrUnauthorized, got %v", err)
	}
}

func TestServiceLoad_RejectsAMalformedIDBeforeTouchingTheStore(t *testing.T) {
	store := newFakeStore()
	service := NewService(store)

	for _, malformedID := range []string{"../secret", "a/b", "", strings.Repeat("a", 65)} {
		_, err := service.Load(context.Background(), "user-a", malformedID)
		if !errors.Is(err, sharedErrors.ErrBadRequest) {
			t.Fatalf("expected ErrBadRequest for %q, got %v", malformedID, err)
		}
	}
}

func TestServiceLoad_ScopesTheLookupToTheRequestingUser(t *testing.T) {
	store := newFakeStore()
	service := NewService(store)

	attachment, err := service.Upload(context.Background(), "user-a", "chart.png", testsupport.PNG(t))
	if err != nil {
		t.Fatalf("cannot upload the png: %v", err)
	}

	loaded, err := service.Load(context.Background(), "user-a", attachment.ID)
	if err != nil {
		t.Fatalf("the owner must be able to load the attachment: %v", err)
	}
	if loaded.ID != attachment.ID {
		t.Errorf("expected attachment %q, got %q", attachment.ID, loaded.ID)
	}

	if _, err := service.Load(context.Background(), "user-b", attachment.ID); !errors.Is(err, sharedErrors.ErrNotFound) {
		t.Fatalf("expected another user to be denied with ErrNotFound, got %v", err)
	}
}
