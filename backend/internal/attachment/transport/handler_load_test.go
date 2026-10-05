package transport

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"lunar/backend/internal/attachment/domain"
	"lunar/backend/internal/attachment/testsupport"
)

func TestAttachmentHandler_GetReturnsAnImageDataURL(t *testing.T) {
	handler, store := setupAttachmentHandler(t)
	imageContent := testsupport.PNG(t)
	store.items["image-1"] = &domain.Attachment{
		ID:          "image-1",
		UserID:      attachmentTestUserID,
		Name:        "chart.png",
		Kind:        domain.KindImage,
		ContentType: "image/png",
		Size:        int64(len(imageContent)),
		Preview:     "PNG image, 3x2",
		Content:     imageContent,
	}

	recorder := performAttachmentRequest(handler, authorizedAttachmentRequest(t, http.MethodGet, "/api/copilot/attachments/image-1", nil, "", attachmentTestUserID))
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", recorder.Code, recorder.Body.String())
	}

	var response loadResponse
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("cannot decode the load response: %v", err)
	}
	if !strings.HasPrefix(response.DataURL, "data:image/png;base64,") {
		t.Errorf("expected an inline image data url, got %q", response.DataURL)
	}
	if response.Text != "" {
		t.Errorf("expected no extracted text for an image, got %q", response.Text)
	}
}

func TestAttachmentHandler_GetReturnsDocumentTextAndHidesItFromOtherUsers(t *testing.T) {
	handler, store := setupAttachmentHandler(t)
	store.items["doc-1"] = &domain.Attachment{
		ID:          "doc-1",
		UserID:      attachmentTestUserID,
		Name:        "plan.docx",
		Kind:        domain.KindDocument,
		ContentType: "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		Size:        2048,
		Preview:     "Release plan",
		Text:        "Release plan\nShip it",
	}

	recorder := performAttachmentRequest(handler, authorizedAttachmentRequest(t, http.MethodGet, "/api/copilot/attachments/doc-1", nil, "", attachmentTestUserID))
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", recorder.Code, recorder.Body.String())
	}

	var response loadResponse
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("cannot decode the load response: %v", err)
	}
	if response.Text != "Release plan\nShip it" {
		t.Errorf("expected the extracted text, got %q", response.Text)
	}
	if response.DataURL != "" {
		t.Errorf("expected no image data for a document, got %q", response.DataURL)
	}

	deniedRecorder := performAttachmentRequest(handler, authorizedAttachmentRequest(t, http.MethodGet, "/api/copilot/attachments/doc-1", nil, "", attachmentTestOtherUser))
	if deniedRecorder.Code != http.StatusNotFound {
		t.Fatalf("expected status 404 for another user, got %d: %s", deniedRecorder.Code, deniedRecorder.Body.String())
	}
}

func TestAttachmentHandler_GetRejectsAMalformedID(t *testing.T) {
	handler, _ := setupAttachmentHandler(t)

	recorder := performAttachmentRequest(handler, authorizedAttachmentRequest(t, http.MethodGet, "/api/copilot/attachments/not%20an%20id", nil, "", attachmentTestUserID))
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400 for a malformed id, got %d: %s", recorder.Code, recorder.Body.String())
	}
}
