package transport

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	confluenceDomain "lunar/backend/internal/confluence/domain"
	queryReviewDomain "lunar/backend/internal/queryreview/domain"
	sharedAuth "lunar/backend/internal/shared/auth"
	sharedErrors "lunar/backend/internal/shared/errors"
	sopApplication "lunar/backend/internal/sop/application"
)

type stubGateway struct {
	documents      map[string]*confluenceDomain.ConfluenceDocument
	ancestors      map[string][]confluenceDomain.ConfluencePageRef
	children       map[string][]confluenceDomain.ConfluencePageRef
	storage        map[string]string
	versions       map[string]int
	updateFailures int
	nextPage       int
}

func newStubGateway() *stubGateway {
	gateway := &stubGateway{
		documents: map[string]*confluenceDomain.ConfluenceDocument{},
		ancestors: map[string][]confluenceDomain.ConfluencePageRef{},
		children:  map[string][]confluenceDomain.ConfluencePageRef{},
		storage:   map[string]string{},
		versions:  map[string]int{},
	}
	gateway.documents["999"] = &confluenceDomain.ConfluenceDocument{ID: "999", Title: "CRMMS-1 Query Review"}
	gateway.ancestors["999"] = []confluenceDomain.ConfluencePageRef{{ID: "100", Title: "CRMMS-1-Development"}}
	return gateway
}

func (g *stubGateway) GetDocumentDetail(_ context.Context, _ string, documentID string) (*confluenceDomain.ConfluenceDocument, error) {
	return g.documents[documentID], nil
}

func (g *stubGateway) GetPageAncestors(_ context.Context, _ string, pageID string) ([]confluenceDomain.ConfluencePageRef, error) {
	return g.ancestors[pageID], nil
}

func (g *stubGateway) GetChildPages(_ context.Context, _ string, pageID string) ([]confluenceDomain.ConfluencePageRef, error) {
	return g.children[pageID], nil
}

func (g *stubGateway) GetPageStorage(_ context.Context, _ string, pageID string) (string, error) {
	return g.storage[pageID], nil
}

func (g *stubGateway) UpdatePageStorage(_ context.Context, _ string, pageID string, storage string) (int, string, error) {
	if g.updateFailures > 0 {
		g.updateFailures--
		return 0, "", fmt.Errorf("%w: page changed", sharedErrors.ErrConflict)
	}
	g.storage[pageID] = storage
	g.versions[pageID]++
	return g.versions[pageID], "https://wiki.example.com/pages/" + pageID, nil
}

func (g *stubGateway) UploadAttachment(_ context.Context, _ string, _ string, _ string, _ []byte, _ string) (string, error) {
	return "attachment", nil
}

func (g *stubGateway) CreatePage(_ context.Context, _ string, parentPageID string, title string, storage string) (string, int, string, error) {
	g.nextPage++
	pageID := fmt.Sprintf("page-%d", g.nextPage)
	g.documents[pageID] = &confluenceDomain.ConfluenceDocument{ID: pageID, Title: title}
	g.storage[pageID] = storage
	g.versions[pageID] = 1
	g.children[parentPageID] = append(g.children[parentPageID], confluenceDomain.ConfluencePageRef{ID: pageID, Title: title})
	return pageID, 1, "https://wiki.example.com/pages/" + pageID, nil
}

func (g *stubGateway) GetPageVersion(_ context.Context, _ string, pageID string) (int, error) {
	return g.versions[pageID], nil
}

type stubMigReader struct{}

func (stubMigReader) ReadMigRepositories(_ context.Context, _ string, _ string) ([]queryReviewDomain.MigRepository, error) {
	return []queryReviewDomain.MigRepository{{Repository: "everest", Branch: "feature/CRMMS-1", TargetBranch: "development"}}, nil
}

type stubHashStore struct {
	values map[string]string
}

func (s *stubHashStore) Get(_ context.Context, pageID string, region string) (string, bool, error) {
	value, exists := s.values[pageID+"\x1f"+region]
	return value, exists, nil
}

func (s *stubHashStore) Set(_ context.Context, pageID string, region string, hash string) error {
	s.values[pageID+"\x1f"+region] = hash
	return nil
}

func newTestHandler(gateway *stubGateway) *Handler {
	service := sopApplication.NewService(gateway, stubMigReader{}, &stubHashStore{values: map[string]string{}}, nil, nil)
	return NewHandler(service)
}

func serveRequest(t *testing.T, handler *Handler, route string, target string, body string, withUser bool) *httptest.ResponseRecorder {
	t.Helper()

	mux := http.NewServeMux()
	mux.Handle("POST "+route, http.HandlerFunc(handler.Generate))
	mux.Handle("POST "+route+"/apply", http.HandlerFunc(handler.Apply))

	request := httptest.NewRequest(http.MethodPost, target, strings.NewReader(body))
	if withUser {
		request = request.WithContext(sharedAuth.ContextWithUser(request.Context(), &sharedAuth.UserContext{UserID: "usr-1"}))
	}
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, request)
	return recorder
}

func TestGenerateRequiresAnAuthenticatedUser(t *testing.T) {
	handler := newTestHandler(newStubGateway())
	recorder := serveRequest(t, handler, "/documents/{id}/sop/generate", "/documents/999/sop/generate", `{}`, false)
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d with body %s", recorder.Code, recorder.Body.String())
	}
}

func TestGenerateRejectsAnInvalidBody(t *testing.T) {
	handler := newTestHandler(newStubGateway())
	recorder := serveRequest(t, handler, "/documents/{id}/sop/generate", "/documents/999/sop/generate", `{"unknown":true}`, true)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d with body %s", recorder.Code, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), messageInvalidGenerateBody) {
		t.Fatalf("expected the invalid body message, got %s", recorder.Body.String())
	}
}

func TestGenerateReturnsThePlan(t *testing.T) {
	handler := newTestHandler(newStubGateway())
	recorder := serveRequest(t, handler, "/documents/{id}/sop/generate", "/documents/999/sop/generate", `{"model":"DeepSeek-V4 Flash"}`, true)
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d with body %s", recorder.Code, recorder.Body.String())
	}
	for _, expected := range []string{`"issue_key":"CRMMS-1"`, `"title":"CRMMS-1-SOP Deploy"`, `"title":"CRMMS-1-SOP Operasional"`} {
		if !strings.Contains(recorder.Body.String(), expected) {
			t.Fatalf("expected the response to contain %s, got %s", expected, recorder.Body.String())
		}
	}
}

func TestApplyRejectsAnInvalidMode(t *testing.T) {
	handler := newTestHandler(newStubGateway())
	recorder := serveRequest(t, handler, "/documents/{id}/sop/generate", "/documents/999/sop/generate/apply", `{"deploy":{"mode":"nonsense"}}`, true)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d with body %s", recorder.Code, recorder.Body.String())
	}
}

func TestApplyMapsAPersistentConflictToABadGateway(t *testing.T) {
	gateway := newStubGateway()
	gateway.children["100"] = []confluenceDomain.ConfluencePageRef{{ID: "200", Title: "CRMMS-1-SOP Deploy"}}
	gateway.storage["200"] = "<p>Manual</p>"
	gateway.versions["200"] = 2
	gateway.updateFailures = 2

	handler := newTestHandler(gateway)
	recorder := serveRequest(t, handler, "/documents/{id}/sop/generate", "/documents/999/sop/generate/apply", `{"deploy":{"mode":"update"}}`, true)
	if recorder.Code != http.StatusBadGateway {
		t.Fatalf("expected 502, got %d with body %s", recorder.Code, recorder.Body.String())
	}
}

func TestApplyReturnsTheCreatedPages(t *testing.T) {
	handler := newTestHandler(newStubGateway())
	body := `{"deploy":{"mode":"create"},"operasional":{"create_children":true}}`
	recorder := serveRequest(t, handler, "/documents/{id}/sop/generate", "/documents/999/sop/generate/apply", body, true)
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d with body %s", recorder.Code, recorder.Body.String())
	}
	for _, expected := range []string{`"action":"created"`, `"created_children":["Maintenance","Monitoring","Troubleshooting"]`} {
		if !strings.Contains(recorder.Body.String(), expected) {
			t.Fatalf("expected the response to contain %s, got %s", expected, recorder.Body.String())
		}
	}
}
