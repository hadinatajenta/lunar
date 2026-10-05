package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	agentinfrastructure "lunar/backend/internal/agent/infrastructure"
)

type attachmentBackendStub struct {
	statusCode int
	payload    attachmentPayload
	requests   []string
}

func newAttachmentBackend(t *testing.T, stub *attachmentBackendStub) *httptest.Server {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		stub.requests = append(stub.requests, r.URL.Path)
		if r.Header.Get("Authorization") != "Bearer session-token" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(stub.statusCode)
		if stub.statusCode == http.StatusOK {
			if err := json.NewEncoder(w).Encode(stub.payload); err != nil {
				t.Errorf("cannot encode the stub payload: %v", err)
			}
		}
	}))
	t.Cleanup(server.Close)
	return server
}

func TestLoadAttachmentContext_WrapsADocumentAsUntrustedData(t *testing.T) {
	stub := &attachmentBackendStub{
		statusCode: http.StatusOK,
		payload: attachmentPayload{
			ID:   "doc-1",
			Name: "plan.docx",
			Kind: attachmentKindDocument,
			Text: "Release plan",
		},
	}
	server := newAttachmentBackend(t, stub)

	block, images, err := loadAttachmentContext(context.Background(), server.URL, "session-token", []string{"doc-1"})
	if err != nil {
		t.Fatalf("cannot load the attachment: %v", err)
	}
	if len(images) != 0 {
		t.Errorf("expected no image data for a document, got %d", len(images))
	}
	for _, expected := range []string{"untrusted data", "<<<ATTACHMENT START: plan.docx>>>", "Release plan", "<<<ATTACHMENT END: plan.docx>>>"} {
		if !strings.Contains(block, expected) {
			t.Errorf("expected %q in the attachment block, got %q", expected, block)
		}
	}
	if len(stub.requests) != 1 || stub.requests[0] != "/api/copilot/attachments/doc-1" {
		t.Errorf("unexpected backend requests: %v", stub.requests)
	}
}

func TestLoadAttachmentContext_CollectsImageDataURLs(t *testing.T) {
	stub := &attachmentBackendStub{
		statusCode: http.StatusOK,
		payload: attachmentPayload{
			ID:      "image-1",
			Name:    "chart.png",
			Kind:    attachmentKindImage,
			DataURL: "data:image/png;base64,AAAA",
		},
	}
	server := newAttachmentBackend(t, stub)

	block, images, err := loadAttachmentContext(context.Background(), server.URL, "session-token", []string{"image-1"})
	if err != nil {
		t.Fatalf("cannot load the attachment: %v", err)
	}
	if block != "" {
		t.Errorf("expected no document block for an image, got %q", block)
	}
	if len(images) != 1 || images[0] != "data:image/png;base64,AAAA" {
		t.Errorf("expected the image data url, got %v", images)
	}
}

func TestLoadAttachmentContext_ReportsAMissingAttachment(t *testing.T) {
	stub := &attachmentBackendStub{statusCode: http.StatusNotFound}
	server := newAttachmentBackend(t, stub)

	_, _, err := loadAttachmentContext(context.Background(), server.URL, "session-token", []string{"missing-1"})
	if err == nil {
		t.Fatal("expected an error for a missing attachment")
	}
	if !strings.Contains(err.Error(), "missing-1") {
		t.Errorf("expected the error to name the attachment, got %q", err.Error())
	}
}

func TestLoadAttachmentContext_RejectsAMalformedIDBeforeRequesting(t *testing.T) {
	stub := &attachmentBackendStub{statusCode: http.StatusOK}
	server := newAttachmentBackend(t, stub)

	_, _, err := loadAttachmentContext(context.Background(), server.URL, "session-token", []string{"../etc/passwd"})
	if err == nil {
		t.Fatal("expected a malformed id to be rejected")
	}
	if len(stub.requests) != 0 {
		t.Errorf("expected no backend request for a malformed id, got %v", stub.requests)
	}
}

func TestLoadAttachmentContext_RejectsTooManyAttachments(t *testing.T) {
	stub := &attachmentBackendStub{statusCode: http.StatusOK}
	server := newAttachmentBackend(t, stub)

	identifiers := make([]string, 0, maxAgentAttachments+1)
	for index := 0; index <= maxAgentAttachments; index++ {
		identifiers = append(identifiers, fmt.Sprintf("attachment-%d", index))
	}

	_, _, err := loadAttachmentContext(context.Background(), server.URL, "session-token", identifiers)
	if err == nil {
		t.Fatal("expected too many attachments to be rejected")
	}
	if !strings.Contains(err.Error(), fmt.Sprint(maxAgentAttachments)) {
		t.Errorf("expected the error to state the limit, got %q", err.Error())
	}
	if len(stub.requests) != 0 {
		t.Errorf("expected no backend request, got %v", stub.requests)
	}
}

func TestValidateImageAttachments_NamesTheModelThatCannotReadImages(t *testing.T) {
	client := agentinfrastructure.NewChatClient(agentinfrastructure.ProviderConfig{})

	if err := validateImageAttachments("DeepSeek Flash", "deepseek-chat", 1, client); err != nil {
		t.Errorf("expected the vision model to accept an image, got %v", err)
	}

	reasoningError := validateImageAttachments("DeepSeek-V4 Pro (Thinking)", "deepseek-reasoner", 1, client)
	if reasoningError == nil {
		t.Fatal("expected the reasoning model to refuse an image")
	}
	if !strings.Contains(reasoningError.Error(), "DeepSeek-V4 Pro (Thinking)") {
		t.Errorf("expected the error to name the selected model, got %q", reasoningError.Error())
	}
	if !strings.Contains(reasoningError.Error(), "cannot read images") {
		t.Errorf("expected the error to state the limitation, got %q", reasoningError.Error())
	}

	if err := validateImageAttachments("DeepSeek-V4 Pro (Thinking)", "deepseek-reasoner", 0, client); err != nil {
		t.Errorf("expected no error when no image is attached, got %v", err)
	}
}
