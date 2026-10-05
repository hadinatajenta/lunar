package transport

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"lunar/backend/internal/attachment/application"
	"lunar/backend/internal/attachment/domain"
	"lunar/backend/internal/attachment/testsupport"
	sharedAuth "lunar/backend/internal/shared/auth"
	sharedErrors "lunar/backend/internal/shared/errors"
)

const (
	attachmentTestJWTSecret = "attachment-transport-jwt-secret"
	attachmentTestUserID    = "attachment-user-a"
	attachmentTestOtherUser = "attachment-user-b"
)

type fakeAttachmentStore struct {
	items map[string]*domain.Attachment
}

func newFakeAttachmentStore() *fakeAttachmentStore {
	return &fakeAttachmentStore{items: make(map[string]*domain.Attachment)}
}

func (f *fakeAttachmentStore) Create(ctx context.Context, attachment *domain.Attachment) error {
	stored := *attachment
	f.items[attachment.ID] = &stored
	return nil
}

func (f *fakeAttachmentStore) GetByID(ctx context.Context, userID string, attachmentID string) (*domain.Attachment, error) {
	stored, exists := f.items[attachmentID]
	if !exists || stored.UserID != userID {
		return nil, sharedErrors.ErrNotFound
	}
	return stored, nil
}

func setupAttachmentHandler(t *testing.T) (http.Handler, *fakeAttachmentStore) {
	t.Helper()

	store := newFakeAttachmentStore()
	handler := NewAttachmentHandler(application.NewService(store))

	authMiddleware := sharedAuth.RequireAuth(attachmentTestJWTSecret)
	mux := http.NewServeMux()
	mux.Handle("POST /api/copilot/attachments", authMiddleware(http.HandlerFunc(handler.Upload)))
	mux.Handle("GET /api/copilot/attachments/{id}", authMiddleware(http.HandlerFunc(handler.Get)))

	return mux, store
}

func multipartUploadBody(t *testing.T, fieldName string, fileName string, content []byte) (*bytes.Buffer, string) {
	t.Helper()

	var buffer bytes.Buffer
	writer := multipart.NewWriter(&buffer)
	part, err := writer.CreateFormFile(fieldName, fileName)
	if err != nil {
		t.Fatalf("cannot create the multipart file part: %v", err)
	}
	if _, err := part.Write(content); err != nil {
		t.Fatalf("cannot write the multipart file part: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("cannot close the multipart writer: %v", err)
	}
	return &buffer, writer.FormDataContentType()
}

func authorizedAttachmentRequest(t *testing.T, method string, path string, body *bytes.Buffer, contentType string, userID string) *http.Request {
	t.Helper()

	var requestBody *bytes.Buffer
	if body == nil {
		requestBody = bytes.NewBuffer(nil)
	} else {
		requestBody = body
	}

	request := httptest.NewRequest(method, path, requestBody)
	if contentType != "" {
		request.Header.Set("Content-Type", contentType)
	}

	token, err := sharedAuth.GenerateToken(userID, userID+"@example.com", attachmentTestJWTSecret, time.Hour)
	if err != nil {
		t.Fatalf("cannot generate a session token: %v", err)
	}
	request.Header.Set("Authorization", "Bearer "+token)
	return request
}

func performAttachmentRequest(handler http.Handler, request *http.Request) *httptest.ResponseRecorder {
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	return recorder
}

func decodeUploadResponse(t *testing.T, recorder *httptest.ResponseRecorder) uploadResponse {
	t.Helper()

	var response uploadResponse
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("cannot decode the upload response: %v", err)
	}
	return response
}

func TestAttachmentHandler_UploadStoresAnImageAndReturnsTheContractFields(t *testing.T) {
	handler, store := setupAttachmentHandler(t)
	imageContent := testsupport.PNG(t)
	body, contentType := multipartUploadBody(t, uploadFieldName, "chart.png", imageContent)

	recorder := performAttachmentRequest(handler, authorizedAttachmentRequest(t, http.MethodPost, "/api/copilot/attachments", body, contentType, attachmentTestUserID))
	if recorder.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d: %s", recorder.Code, recorder.Body.String())
	}

	response := decodeUploadResponse(t, recorder)
	if response.ID == "" {
		t.Error("expected an attachment id")
	}
	if response.Name != "chart.png" {
		t.Errorf("expected the file name, got %q", response.Name)
	}
	if response.Kind != string(domain.KindImage) {
		t.Errorf("expected kind image, got %q", response.Kind)
	}
	if response.Size != int64(len(imageContent)) {
		t.Errorf("expected size %d, got %d", len(imageContent), response.Size)
	}
	if response.Preview != "PNG image, 3x2" {
		t.Errorf("expected an image preview, got %q", response.Preview)
	}

	stored := store.items[response.ID]
	if stored == nil {
		t.Fatal("expected the attachment to be stored")
	}
	if stored.ContentType != "image/png" || stored.UserID != attachmentTestUserID {
		t.Errorf("unexpected stored attachment: %+v", stored)
	}
}

func TestAttachmentHandler_UploadStoresADocumentWithExtractedText(t *testing.T) {
	handler, store := setupAttachmentHandler(t)
	documentContent := testsupport.DOCX(t, []string{"Release plan"})
	body, contentType := multipartUploadBody(t, uploadFieldName, "plan.docx", documentContent)

	recorder := performAttachmentRequest(handler, authorizedAttachmentRequest(t, http.MethodPost, "/api/copilot/attachments", body, contentType, attachmentTestUserID))
	if recorder.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d: %s", recorder.Code, recorder.Body.String())
	}

	response := decodeUploadResponse(t, recorder)
	if response.Kind != string(domain.KindDocument) {
		t.Errorf("expected kind document, got %q", response.Kind)
	}
	if response.Preview != "Release plan" {
		t.Errorf("expected the document text as the preview, got %q", response.Preview)
	}
	if stored := store.items[response.ID]; stored == nil || !strings.Contains(stored.Text, "Release plan") {
		t.Errorf("expected the extracted text to be stored, got %+v", stored)
	}
}

func TestAttachmentHandler_UploadRejectsAnUnsupportedTypeAndNamesTheAcceptedOnes(t *testing.T) {
	handler, _ := setupAttachmentHandler(t)
	body, contentType := multipartUploadBody(t, uploadFieldName, "archive.zip", []byte("PK\x03\x04"))

	recorder := performAttachmentRequest(handler, authorizedAttachmentRequest(t, http.MethodPost, "/api/copilot/attachments", body, contentType, attachmentTestUserID))
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d: %s", recorder.Code, recorder.Body.String())
	}

	var errorResponse struct {
		Error string `json:"error"`
	}
	if err := json.NewDecoder(recorder.Body).Decode(&errorResponse); err != nil {
		t.Fatalf("cannot decode the error response: %v", err)
	}
	for _, expectedType := range []string{"png", "jpeg", "webp", "pdf", "xlsx", "xls", "docx"} {
		if !strings.Contains(errorResponse.Error, expectedType) {
			t.Errorf("expected the error to name %q, got %q", expectedType, errorResponse.Error)
		}
	}
}

func TestAttachmentHandler_UploadRejectsAnOversizedFileWithoutReadingItWhole(t *testing.T) {
	handler, store := setupAttachmentHandler(t)
	oversized := make([]byte, application.MaxFileBytes+1)
	body, contentType := multipartUploadBody(t, uploadFieldName, "huge.png", oversized)

	recorder := performAttachmentRequest(handler, authorizedAttachmentRequest(t, http.MethodPost, "/api/copilot/attachments", body, contentType, attachmentTestUserID))
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400 for an oversized file, got %d: %s", recorder.Code, recorder.Body.String())
	}

	var errorResponse struct {
		Error string `json:"error"`
	}
	if err := json.NewDecoder(recorder.Body).Decode(&errorResponse); err != nil {
		t.Fatalf("cannot decode the error response: %v", err)
	}
	if !strings.Contains(errorResponse.Error, "10 MB") {
		t.Errorf("expected the error to state the size limit, got %q", errorResponse.Error)
	}
	if len(store.items) != 0 {
		t.Errorf("expected nothing to be stored, got %d attachments", len(store.items))
	}
}

func TestAttachmentHandler_UploadRejectsARequestWithoutAFileField(t *testing.T) {
	handler, _ := setupAttachmentHandler(t)
	body, contentType := multipartUploadBody(t, "document", "chart.png", testsupport.PNG(t))

	recorder := performAttachmentRequest(handler, authorizedAttachmentRequest(t, http.MethodPost, "/api/copilot/attachments", body, contentType, attachmentTestUserID))
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400 without a file field, got %d: %s", recorder.Code, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), uploadFieldName) {
		t.Errorf("expected the error to name the file field, got %s", recorder.Body.String())
	}
}

func TestAttachmentHandler_UploadRequiresAuthentication(t *testing.T) {
	handler, _ := setupAttachmentHandler(t)
	body, contentType := multipartUploadBody(t, uploadFieldName, "chart.png", testsupport.PNG(t))

	request := httptest.NewRequest(http.MethodPost, "/api/copilot/attachments", body)
	request.Header.Set("Content-Type", contentType)

	recorder := performAttachmentRequest(handler, request)
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d: %s", recorder.Code, recorder.Body.String())
	}
}
