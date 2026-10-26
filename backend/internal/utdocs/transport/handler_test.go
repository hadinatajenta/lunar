package transport

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"image"
	"image/png"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	confluenceDomain "lunar/backend/internal/confluence/domain"
	sharedAuth "lunar/backend/internal/shared/auth"
	sharedErrors "lunar/backend/internal/shared/errors"
	"lunar/backend/internal/utdocs/application"
	"lunar/backend/internal/utdocs/domain"
)

type stubDocumentLookup struct {
	document *confluenceDomain.ConfluenceDocument
	err      error
}

func (l stubDocumentLookup) GetDocumentDetail(_ context.Context, _ string, _ string) (*confluenceDomain.ConfluenceDocument, error) {
	if l.err != nil {
		return nil, l.err
	}
	return l.document, nil
}

type stubDocumentGenerator struct {
	generated *domain.GeneratedContent
	err       error
}

func (g stubDocumentGenerator) Generate(_ context.Context, _ string, _ string, _ string, _ domain.DocumentRequest) (*domain.GeneratedContent, error) {
	if g.err != nil {
		return nil, g.err
	}
	return g.generated, nil
}

type stubTransportUploader struct {
	calls   int
	err     error
	content []byte
}

func (u *stubTransportUploader) UploadAttachment(_ context.Context, _ string, _ string, _ string, content []byte, _ string) (string, error) {
	u.calls++
	u.content = append([]byte{}, content...)
	if u.err != nil {
		return "", u.err
	}
	return "att-1", nil
}

type stubTransportPages struct {
	storage   string
	getErr    error
	updated   string
	version   int
	pageURL   string
	updateErr error
}

func (p *stubTransportPages) GetPageStorage(_ context.Context, _ string, _ string) (string, error) {
	if p.getErr != nil {
		return "", p.getErr
	}
	return p.storage, nil
}

func (p *stubTransportPages) UpdatePageStorage(_ context.Context, _ string, _ string, storage string) (int, string, error) {
	p.updated = storage
	if p.updateErr != nil {
		return 0, "", p.updateErr
	}
	return p.version, p.pageURL, nil
}

func validDocumentRequestBody(t *testing.T) string {
	t.Helper()

	body := map[string]any{
		"model": "DeepSeek-V4 Flash",
		"repositories": []any{
			map[string]any{
				"repository":    "micro-payment",
				"branch":        "feat/CRMMS-123",
				"base_ref":      "development",
				"test_files":    []any{},
				"blocked_files": []any{},
				"run": map[string]any{
					"status":            "passed",
					"executed_packages": []string{"./internal/repo"},
					"total":             1,
					"passed":            1,
					"failed":            0,
					"duration_ms":       120,
					"output_excerpt":    "ok",
				},
				"tests": []any{
					map[string]any{"name": "TestFind", "package": "lunar/backend/internal/repo", "file": "internal/repo/dao_test.go", "status": "passed", "duration_ms": 120, "output": ""},
				},
			},
		},
	}
	encoded, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("cannot encode the request: %v", err)
	}
	return string(encoded)
}

func validApplyRequestBody(t *testing.T) string {
	t.Helper()

	var pngBuffer bytes.Buffer
	if err := png.Encode(&pngBuffer, image.NewRGBA(image.Rect(0, 0, 2, 2))); err != nil {
		t.Fatalf("cannot encode the sample PNG: %v", err)
	}

	body := map[string]any{
		"repositories": []any{
			map[string]any{
				"repository": "micro-payment",
				"screenshot": map[string]any{
					"filename":       "ut-micro-payment.png",
					"content_base64": base64.StdEncoding.EncodeToString(pngBuffer.Bytes()),
				},
				"rows": []any{
					map[string]any{"function": "Find payment", "scenario": "Mencari data", "expected_result": "Data ditemukan", "input_data": "ID", "result": "Passed"},
				},
			},
		},
	}
	encoded, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("cannot encode the request: %v", err)
	}
	return string(encoded)
}

func newGenerateRequest(t *testing.T, body string, withUser bool) *http.Request {
	t.Helper()

	request := httptest.NewRequest(http.MethodPost, "/api/confluence/documents/123456/ut-docs", strings.NewReader(body))
	request.SetPathValue("id", "123456")
	if withUser {
		request = request.WithContext(sharedAuth.ContextWithUser(request.Context(), &sharedAuth.UserContext{UserID: "user-1"}))
	}
	return request
}

func newApplyRequest(t *testing.T, body string, withUser bool) *http.Request {
	t.Helper()

	request := httptest.NewRequest(http.MethodPost, "/api/confluence/documents/123456/ut-docs/apply", strings.NewReader(body))
	request.SetPathValue("id", "123456")
	if withUser {
		request = request.WithContext(sharedAuth.ContextWithUser(request.Context(), &sharedAuth.UserContext{UserID: "user-1"}))
	}
	return request
}

func TestGenerateReturnsTheGeneratedDocument(t *testing.T) {
	lookup := stubDocumentLookup{document: &confluenceDomain.ConfluenceDocument{ID: "123456", Title: "Payment API"}}
	generator := stubDocumentGenerator{generated: &domain.GeneratedContent{
		Title:   "Unit tests for Payment API",
		Summary: "Mock based coverage.",
		Rows:    []domain.GeneratedRow{{TestFunctionName: "TestFind", Function: "Find payment", Scenario: "Mencari data", ExpectedResult: "Data ditemukan", InputData: "ID"}},
		Caption: "UT run for micro-payment",
	}}
	handler := NewHandler(application.NewDocumentService(lookup, generator))

	recorder := httptest.NewRecorder()
	handler.Generate(recorder, newGenerateRequest(t, validDocumentRequestBody(t), true))

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d with body %s", recorder.Code, recorder.Body.String())
	}

	var result domain.DocumentResult
	if err := json.Unmarshal(recorder.Body.Bytes(), &result); err != nil {
		t.Fatalf("cannot decode the response: %v", err)
	}
	if result.DocumentID != "123456" || result.PromptVersion != domain.PromptVersion {
		t.Fatalf("expected the document identity, got %+v", result)
	}
	if result.Title != "Unit tests for Payment API" || result.ScreenshotCaption != "UT run for micro-payment" {
		t.Fatalf("expected the generated fields, got %+v", result)
	}
	if len(result.Rows) != 1 || result.Rows[0].Result != domain.ResultPassed {
		t.Fatalf("expected the merged row, got %+v", result.Rows)
	}
	if !strings.Contains(result.Table, "Normal Test Scenario") {
		t.Fatalf("expected the storage table, got %q", result.Table)
	}
}

func TestGenerateRequiresAnAuthenticatedUser(t *testing.T) {
	lookup := stubDocumentLookup{document: &confluenceDomain.ConfluenceDocument{ID: "123456", Title: "Payment API"}}
	handler := NewHandler(application.NewDocumentService(lookup, stubDocumentGenerator{generated: &domain.GeneratedContent{Title: "T", Summary: "S"}}))

	recorder := httptest.NewRecorder()
	handler.Generate(recorder, newGenerateRequest(t, validDocumentRequestBody(t), false))

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d with body %s", recorder.Code, recorder.Body.String())
	}
}

func TestGenerateRejectsMalformedBodies(t *testing.T) {
	lookup := stubDocumentLookup{document: &confluenceDomain.ConfluenceDocument{ID: "123456", Title: "Payment API"}}
	handler := NewHandler(application.NewDocumentService(lookup, stubDocumentGenerator{generated: &domain.GeneratedContent{Title: "T", Summary: "S"}}))

	for _, body := range []string{"", "{", `{"unknown_field":true}`, `{"model":12}`} {
		recorder := httptest.NewRecorder()
		handler.Generate(recorder, newGenerateRequest(t, body, true))
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 for %q, got %d with body %s", body, recorder.Code, recorder.Body.String())
		}
		if !strings.Contains(recorder.Body.String(), messageInvalidRequestBody) {
			t.Fatalf("expected the invalid body message for %q, got %s", body, recorder.Body.String())
		}
	}
}

func TestGenerateMapsNotFound(t *testing.T) {
	handler := NewHandler(application.NewDocumentService(stubDocumentLookup{err: sharedErrors.ErrNotFound}, stubDocumentGenerator{}))

	recorder := httptest.NewRecorder()
	handler.Generate(recorder, newGenerateRequest(t, validDocumentRequestBody(t), true))

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d with body %s", recorder.Code, recorder.Body.String())
	}
}

func TestGenerateMapsModelFailuresToBadGateway(t *testing.T) {
	lookup := stubDocumentLookup{document: &confluenceDomain.ConfluenceDocument{ID: "123456", Title: "Payment API"}}
	handler := NewHandler(application.NewDocumentService(lookup, stubDocumentGenerator{err: errors.New("the provider is down")}))

	recorder := httptest.NewRecorder()
	handler.Generate(recorder, newGenerateRequest(t, validDocumentRequestBody(t), true))

	if recorder.Code != http.StatusBadGateway {
		t.Fatalf("expected 502, got %d with body %s", recorder.Code, recorder.Body.String())
	}
}

func TestApplyUploadsAndUpdatesThePage(t *testing.T) {
	uploader := &stubTransportUploader{}
	pages := &stubTransportPages{storage: normalTableMarkerStorage(), version: 5, pageURL: "https://confluence.example.com/display/DEV/UT"}
	handler := NewHandler(nil, application.NewApplyService(uploader, pages))

	recorder := httptest.NewRecorder()
	handler.Apply(recorder, newApplyRequest(t, validApplyRequestBody(t), true))

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d with body %s", recorder.Code, recorder.Body.String())
	}

	var result domain.ApplyResult
	if err := json.Unmarshal(recorder.Body.Bytes(), &result); err != nil {
		t.Fatalf("cannot decode the response: %v", err)
	}
	if result.PageID != "123456" || result.Version != 5 || result.URL != "https://confluence.example.com/display/DEV/UT" {
		t.Fatalf("unexpected apply result %+v", result)
	}
	if uploader.calls != 1 || len(uploader.content) == 0 {
		t.Fatalf("expected the screenshot upload, got %+v", uploader)
	}
	if !strings.Contains(pages.updated, "Normal Test Scenario") || !strings.Contains(pages.updated, "Find payment") {
		t.Fatalf("expected the replaced table, got %q", pages.updated)
	}
}

func TestApplyRequiresAnAuthenticatedUser(t *testing.T) {
	handler := NewHandler(nil, application.NewApplyService(&stubTransportUploader{}, &stubTransportPages{}))

	recorder := httptest.NewRecorder()
	handler.Apply(recorder, newApplyRequest(t, validApplyRequestBody(t), false))

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d with body %s", recorder.Code, recorder.Body.String())
	}
}

func TestApplyRejectsAnInvalidBody(t *testing.T) {
	handler := NewHandler(nil, application.NewApplyService(&stubTransportUploader{}, &stubTransportPages{}))

	for _, body := range []string{"", "{", `{"unknown_field":true}`, `{"repositories":[]}`} {
		recorder := httptest.NewRecorder()
		handler.Apply(recorder, newApplyRequest(t, body, true))
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 for %q, got %d with body %s", body, recorder.Code, recorder.Body.String())
		}
	}
}

func TestApplyMapsAPageNotFound(t *testing.T) {
	handler := NewHandler(nil, application.NewApplyService(&stubTransportUploader{}, &stubTransportPages{getErr: sharedErrors.ErrNotFound}))

	recorder := httptest.NewRecorder()
	handler.Apply(recorder, newApplyRequest(t, validApplyRequestBody(t), true))

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d with body %s", recorder.Code, recorder.Body.String())
	}
}

func TestApplyMapsAConfluenceFailureToBadGateway(t *testing.T) {
	handler := NewHandler(nil, application.NewApplyService(&stubTransportUploader{err: errors.New("confluence is unreachable")}, &stubTransportPages{}))

	recorder := httptest.NewRecorder()
	handler.Apply(recorder, newApplyRequest(t, validApplyRequestBody(t), true))

	if recorder.Code != http.StatusBadGateway {
		t.Fatalf("expected 502, got %d with body %s", recorder.Code, recorder.Body.String())
	}
}

func TestApplyWithoutAnApplyServiceReturnsBadGateway(t *testing.T) {
	handler := NewHandler(nil)

	recorder := httptest.NewRecorder()
	handler.Apply(recorder, newApplyRequest(t, validApplyRequestBody(t), true))

	if recorder.Code != http.StatusBadGateway {
		t.Fatalf("expected 502, got %d with body %s", recorder.Code, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), messageApplyUnavailable) {
		t.Fatalf("expected the unavailable message, got %s", recorder.Body.String())
	}
}

func normalTableMarkerStorage() string {
	return "<p><em>Normal Test Scenario</em></p>\n<table><tbody><tr><td>old</td></tr></tbody></table>"
}
