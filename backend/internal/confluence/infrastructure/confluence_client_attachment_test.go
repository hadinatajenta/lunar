package infrastructure

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	sharedErrors "lunar/backend/internal/shared/errors"
)

func TestConfluenceClient_UploadAttachmentPostsMultipartWithTheNoCheckHeader(t *testing.T) {
	var method string
	var path string
	var authorization string
	var noCheckHeader string
	var contentType string
	var partName string
	var partFilename string
	var partContentType string
	var partContent []byte

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method = r.Method
		path = r.URL.Path
		authorization = r.Header.Get("Authorization")
		noCheckHeader = r.Header.Get("X-Atlassian-Token")
		contentType = r.Header.Get("Content-Type")

		file, header, err := r.FormFile("file")
		if err != nil {
			t.Errorf("expected the file part: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		defer file.Close()
		partName = "file"
		partFilename = header.Filename
		partContentType = header.Header.Get("Content-Type")
		partContent, _ = io.ReadAll(file)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"results":[{"id":"att-9"}]}`))
	}))
	defer upstream.Close()

	client := NewConfluenceClient(upstream.URL)
	attachmentID, err := client.UploadAttachment(context.Background(), testPAT, "10", "ut-micro-payment.png", []byte("png-bytes"), "image/png")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if method != http.MethodPost || path != "/rest/api/content/10/child/attachment" {
		t.Errorf("unexpected request %s %s", method, path)
	}
	if authorization != "Bearer "+testPAT {
		t.Errorf("expected the bearer token, got %q", authorization)
	}
	if noCheckHeader != "no-check" {
		t.Errorf("expected the X-Atlassian-Token header, got %q", noCheckHeader)
	}
	if !strings.HasPrefix(contentType, "multipart/form-data") {
		t.Errorf("expected a multipart body, got %q", contentType)
	}
	if partName != "file" || partFilename != "ut-micro-payment.png" || partContentType != "image/png" {
		t.Errorf("unexpected file part %q %q %q", partName, partFilename, partContentType)
	}
	if string(partContent) != "png-bytes" {
		t.Errorf("unexpected file content %q", string(partContent))
	}
	if attachmentID != "att-9" {
		t.Errorf("expected the attachment id, got %q", attachmentID)
	}
}

func TestConfluenceClient_UploadAttachmentReplacesAnExistingAttachment(t *testing.T) {
	var paths []string

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.Method+" "+r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/child/attachment"):
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"statusCode":400,"message":"Cannot add a new attachment with same file name"}`))
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/child/attachment"):
			_, _ = w.Write([]byte(`{"results":[{"id":"att-5"}]}`))
		default:
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"results":[{"id":"att-5"}]}`))
		}
	}))
	defer upstream.Close()

	client := NewConfluenceClient(upstream.URL)
	attachmentID, err := client.UploadAttachment(context.Background(), testPAT, "10", "ut-micro-payment.png", []byte("png-bytes"), "image/png")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if attachmentID != "att-5" {
		t.Fatalf("expected the replaced attachment id, got %q", attachmentID)
	}

	expected := []string{
		"POST /rest/api/content/10/child/attachment",
		"GET /rest/api/content/10/child/attachment",
		"POST /rest/api/content/10/child/attachment/att-5/data",
	}
	if strings.Join(paths, ",") != strings.Join(expected, ",") {
		t.Fatalf("expected the create, lookup, replace sequence, got %+v", paths)
	}
}

func TestConfluenceClient_UploadAttachmentMapsUnauthorized(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer upstream.Close()

	client := NewConfluenceClient(upstream.URL)
	_, err := client.UploadAttachment(context.Background(), testPAT, "10", "ut-micro-payment.png", []byte("png-bytes"), "image/png")
	if !errors.Is(err, sharedErrors.ErrUnauthorized) {
		t.Fatalf("expected ErrUnauthorized, got %v", err)
	}
}

func TestConfluenceClient_UploadAttachmentInputValidation(t *testing.T) {
	client := NewConfluenceClient("https://confluence.example.com")
	ctx := context.Background()

	if _, err := client.UploadAttachment(ctx, testPAT, "  ", "ut.png", []byte("x"), "image/png"); !errors.Is(err, sharedErrors.ErrBadRequest) {
		t.Errorf("expected ErrBadRequest for a blank page id, got %v", err)
	}
	if _, err := client.UploadAttachment(ctx, testPAT, "10", "  ", []byte("x"), "image/png"); !errors.Is(err, sharedErrors.ErrBadRequest) {
		t.Errorf("expected ErrBadRequest for a blank filename, got %v", err)
	}
	if _, err := client.UploadAttachment(ctx, testPAT, "10", "ut.png", nil, "image/png"); !errors.Is(err, sharedErrors.ErrBadRequest) {
		t.Errorf("expected ErrBadRequest for empty content, got %v", err)
	}
	if _, err := client.UploadAttachment(ctx, testPAT, "10", "ut.png", []byte("x"), "  "); !errors.Is(err, sharedErrors.ErrBadRequest) {
		t.Errorf("expected ErrBadRequest for a blank media type, got %v", err)
	}
	if _, err := client.UploadAttachment(ctx, "  ", "10", "ut.png", []byte("x"), "image/png"); !errors.Is(err, sharedErrors.ErrUnauthorized) {
		t.Errorf("expected ErrUnauthorized for a blank PAT, got %v", err)
	}
}
