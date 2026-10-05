package transport

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"lunar/backend/internal/postman/application"
	"lunar/backend/internal/postman/domain"
	sharedAuth "lunar/backend/internal/shared/auth"
	sharedErrors "lunar/backend/internal/shared/errors"
	workspacedomain "lunar/backend/internal/workspace/domain"
)

type stubExporter struct {
	result     application.Result
	err        error
	emitEvents []domain.StageEvent
	received   application.ExportRequest
	receivedID string
}

func (s *stubExporter) Export(ctx context.Context, userID string, request application.ExportRequest, emit application.Emitter) (application.Result, error) {
	s.received = request
	s.receivedID = userID
	for _, event := range s.emitEvents {
		if err := emit(event); err != nil {
			return application.Result{}, err
		}
	}
	if s.err != nil {
		return application.Result{}, s.err
	}
	return s.result, nil
}

func authenticatedRequest(t *testing.T, body string) *http.Request {
	t.Helper()
	request := httptest.NewRequest(http.MethodPost, "/api/workspace/export", strings.NewReader(body))
	return request.WithContext(sharedAuth.ContextWithUser(request.Context(), &sharedAuth.UserContext{UserID: "user-1"}))
}

func TestExportHandlerRequiresAuthentication(t *testing.T) {
	handler := NewExportHandler(&stubExporter{})
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/workspace/export", strings.NewReader(`{"name":"atlas","model":"deepseek-chat"}`))

	handler.Export(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, received %d", recorder.Code)
	}
}

func TestExportHandlerStreamsStagesAndTheResult(t *testing.T) {
	exporter := &stubExporter{
		result: application.Result{Service: "atlas", RequestCount: 3, FolderCount: 2, Collection: &domain.Collection{}},
		emitEvents: []domain.StageEvent{
			{Stage: domain.StageSelecting, Detail: "selected 3 files"},
			{Stage: domain.StageModel, Detail: "awaiting the model"},
		},
	}
	handler := NewExportHandler(exporter)
	recorder := httptest.NewRecorder()

	handler.Export(recorder, authenticatedRequest(t, `{"root":"/Users/erendt/BRI","name":"atlas","model":"deepseek-chat"}`))

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, received %d", recorder.Code)
	}
	if contentType := recorder.Header().Get("Content-Type"); contentType != "text/event-stream" {
		t.Fatalf("expected text/event-stream, received %q", contentType)
	}
	body := recorder.Body.String()
	for _, marker := range []string{
		"event: stage",
		`"stage":"selecting"`,
		`"detail":"selected 3 files"`,
		"event: result",
		`"service":"atlas"`,
		`"request_count":3`,
		`"folder_count":2`,
	} {
		if !strings.Contains(body, marker) {
			t.Fatalf("expected the stream to contain %s, received %s", marker, body)
		}
	}
	if exporter.received.RepositoryName != "atlas" || exporter.received.Model != "deepseek-chat" {
		t.Fatalf("unexpected export request: %+v", exporter.received)
	}
	if exporter.receivedID != "user-1" {
		t.Fatalf("expected the authenticated user, received %q", exporter.receivedID)
	}
}

func TestExportHandlerReportsAFailureBeforeStreamingWithAStatusCode(t *testing.T) {
	exporter := &stubExporter{err: domain.ErrProviderKeyMissing}
	handler := NewExportHandler(exporter)
	recorder := httptest.NewRecorder()

	handler.Export(recorder, authenticatedRequest(t, `{"name":"atlas","model":"deepseek-chat"}`))

	if recorder.Code != http.StatusConflict {
		t.Fatalf("expected 409, received %d with body %s", recorder.Code, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), "no provider key") {
		t.Fatalf("expected a provider key reason, received %s", recorder.Body.String())
	}
}

func TestExportHandlerReportsAFailureAfterStreamingAsAnErrorEvent(t *testing.T) {
	exporter := &stubExporter{
		emitEvents: []domain.StageEvent{{Stage: domain.StageSelecting, Detail: "selected 1 file"}},
		err:        fmt.Errorf("%w: request \"hollow\" has no url path", domain.ErrCollectionInvalid),
	}
	handler := NewExportHandler(exporter)
	recorder := httptest.NewRecorder()

	handler.Export(recorder, authenticatedRequest(t, `{"name":"atlas","model":"deepseek-chat"}`))

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200 once the stream started, received %d", recorder.Code)
	}
	body := recorder.Body.String()
	if !strings.Contains(body, "event: error") || !strings.Contains(body, `"error":"collection invalid: request \"hollow\" has no url path"`) {
		t.Fatalf("expected an error event with the refusal reason, received %s", body)
	}
}

func TestExportHandlerRejectsAMalformedRequest(t *testing.T) {
	handler := NewExportHandler(&stubExporter{})

	malformed := httptest.NewRecorder()
	handler.Export(malformed, authenticatedRequest(t, `{"name":"atlas","unknown":"field"}`))
	if malformed.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for an unknown field, received %d", malformed.Code)
	}

	missingName := httptest.NewRecorder()
	handler.Export(missingName, authenticatedRequest(t, `{"model":"deepseek-chat"}`))
	if missingName.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for a missing name, received %d", missingName.Code)
	}

	missingModel := httptest.NewRecorder()
	handler.Export(missingModel, authenticatedRequest(t, `{"name":"atlas"}`))
	if missingModel.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for a missing model, received %d", missingModel.Code)
	}

	relativeRoot := httptest.NewRecorder()
	handler.Export(relativeRoot, authenticatedRequest(t, `{"root":"BRI","name":"atlas","model":"deepseek-chat"}`))
	if relativeRoot.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for a relative root, received %d", relativeRoot.Code)
	}
}

func TestFailureResponseMapsExportErrors(t *testing.T) {
	cases := []struct {
		name           string
		err            error
		expectedStatus int
		expectedText   string
	}{
		{name: "missing key", err: domain.ErrProviderKeyMissing, expectedStatus: http.StatusConflict, expectedText: "no provider key"},
		{name: "not exportable", err: fmt.Errorf("%w: the repository is a vue front-end", domain.ErrServiceNotExportable), expectedStatus: http.StatusUnprocessableEntity, expectedText: "vue front-end"},
		{name: "invalid collection", err: fmt.Errorf("%w: no endpoints found", domain.ErrCollectionInvalid), expectedStatus: http.StatusUnprocessableEntity, expectedText: "no endpoints found"},
		{name: "provider timeout", err: domain.ErrProviderTimeout, expectedStatus: http.StatusGatewayTimeout, expectedText: "did not respond in time"},
		{name: "invalid repository name", err: workspacedomain.ErrInvalidRepoName, expectedStatus: http.StatusBadRequest, expectedText: "repository name is invalid"},
		{name: "missing repository", err: workspacedomain.ErrRootNotFound, expectedStatus: http.StatusNotFound, expectedText: "does not exist"},
		{name: "outside root", err: workspacedomain.ErrRepoOutsideRoot, expectedStatus: http.StatusBadRequest, expectedText: "outside the workspace root"},
		{name: "client side workspace", err: fmt.Errorf("%w: repository data is read client-side by the local helper", sharedErrors.ErrBadRequest), expectedStatus: http.StatusBadRequest, expectedText: "client-side"},
		{name: "unknown", err: errors.New("boom"), expectedStatus: http.StatusBadGateway, expectedText: "export failed"},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			status, message := FailureResponse(testCase.err)
			if status != testCase.expectedStatus {
				t.Fatalf("expected status %d, received %d", testCase.expectedStatus, status)
			}
			if !strings.Contains(message, testCase.expectedText) {
				t.Fatalf("expected %q in %q", testCase.expectedText, message)
			}
			if strings.Contains(strings.ToLower(message), "effort") {
				t.Fatalf("an error message must not mention an effort setting: %q", message)
			}
		})
	}
}
