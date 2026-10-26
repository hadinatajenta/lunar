package transport

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	sharedAuth "lunar/backend/internal/shared/auth"
	sharedErrors "lunar/backend/internal/shared/errors"
)

const applyRequestBody = `{"repositories":[{"repository":"micro-payment","queries":[{"no":1,"feature":"[micro-payment] Get user by PN","query_statement":"explain select id from user_data where pn = ?","index":"pn"}]}]}`

func performQueryReviewApply(t *testing.T, handler *QueryReviewHandler, userID string, body string) *httptest.ResponseRecorder {
	t.Helper()

	request := httptest.NewRequest(http.MethodPost, "/api/confluence/documents/123/query-review/apply", strings.NewReader(body))
	request.SetPathValue("id", "123")
	if userID != "" {
		ctx := sharedAuth.ContextWithUser(request.Context(), &sharedAuth.UserContext{UserID: userID, Email: "developer@example.com"})
		request = request.WithContext(ctx)
	}

	recorder := httptest.NewRecorder()
	handler.Apply(recorder, request)
	return recorder
}

func TestQueryReviewApplyHandlerReturns200WithTheWriteResult(t *testing.T) {
	fixture := newHandlerFixture()
	fixture.documents.updateVersion = 12
	fixture.documents.updateURL = "https://confluence.example.com/display/DEV/Page"

	recorder := performQueryReviewApply(t, fixture.handler, "usr-1", applyRequestBody)
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", recorder.Code, recorder.Body.String())
	}

	var payload struct {
		PageID  string `json:"page_id"`
		Version int    `json:"version"`
		URL     string `json:"url"`
	}
	if err := json.NewDecoder(recorder.Body).Decode(&payload); err != nil {
		t.Fatalf("cannot decode the response: %v", err)
	}
	if payload.PageID != "123" || payload.Version != 12 || payload.URL != "https://confluence.example.com/display/DEV/Page" {
		t.Fatalf("unexpected apply result %+v", payload)
	}
	if !strings.Contains(fixture.documents.updatedValue, "Capture explain (mysql)") {
		t.Fatalf("expected the capture table to be written, got %q", fixture.documents.updatedValue)
	}
	if !strings.Contains(fixture.documents.updatedValue, "explain select id from user_data where pn = ?") {
		t.Fatalf("expected the query statement in the written page, got %q", fixture.documents.updatedValue)
	}
}

func TestQueryReviewApplyHandlerRejectsUnauthenticatedRequests(t *testing.T) {
	fixture := newHandlerFixture()

	recorder := performQueryReviewApply(t, fixture.handler, "", applyRequestBody)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", recorder.Code)
	}
}

func TestQueryReviewApplyHandlerRejectsAnInvalidBody(t *testing.T) {
	fixture := newHandlerFixture()

	recorder := performQueryReviewApply(t, fixture.handler, "usr-1", `{"model":"model-a"}`)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", recorder.Code, recorder.Body.String())
	}
}

func TestQueryReviewApplyHandlerRejectsAnEmptyRequest(t *testing.T) {
	fixture := newHandlerFixture()

	recorder := performQueryReviewApply(t, fixture.handler, "usr-1", `{}`)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", recorder.Code, recorder.Body.String())
	}
}

func TestQueryReviewApplyHandlerReturns404WhenThePageIsMissing(t *testing.T) {
	fixture := newHandlerFixture()
	fixture.documents.storageErr = sharedErrors.ErrNotFound

	recorder := performQueryReviewApply(t, fixture.handler, "usr-1", applyRequestBody)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", recorder.Code, recorder.Body.String())
	}
}

func TestQueryReviewApplyHandlerReturns502WhenTheWriteFails(t *testing.T) {
	fixture := newHandlerFixture()
	fixture.documents.updateErr = errors.New("confluence is unreachable")

	recorder := performQueryReviewApply(t, fixture.handler, "usr-1", applyRequestBody)

	if recorder.Code != http.StatusBadGateway {
		t.Fatalf("expected 502, got %d: %s", recorder.Code, recorder.Body.String())
	}
}
