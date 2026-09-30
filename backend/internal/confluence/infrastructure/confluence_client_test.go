package infrastructure

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	sharedErrors "lunar/backend/internal/shared/errors"
)

const testPAT = "test-confluence-pat"

func TestConfluenceClient_ListDocumentsEmptyPATReturnsUnauthorized(t *testing.T) {
	client := NewConfluenceClient("https://confluence.example.com")

	_, err := client.ListDocuments(context.Background(), "")
	if !errors.Is(err, sharedErrors.ErrUnauthorized) {
		t.Fatalf("expected ErrUnauthorized, got %v", err)
	}
	if !strings.Contains(err.Error(), "Confluence PAT is not configured") {
		t.Errorf("expected missing PAT message, got %q", err.Error())
	}
}

func TestConfluenceClient_GetDocumentDetailValidationErrors(t *testing.T) {
	client := NewConfluenceClient("https://confluence.example.com")
	ctx := context.Background()

	t.Run("empty document id returns bad request", func(t *testing.T) {
		_, err := client.GetDocumentDetail(ctx, testPAT, "   ")
		if !errors.Is(err, sharedErrors.ErrBadRequest) {
			t.Errorf("expected ErrBadRequest, got %v", err)
		}
	})

	t.Run("empty PAT returns unauthorized", func(t *testing.T) {
		_, err := client.GetDocumentDetail(ctx, "", "101")
		if !errors.Is(err, sharedErrors.ErrUnauthorized) {
			t.Errorf("expected ErrUnauthorized, got %v", err)
		}
	})
}

func TestConfluenceClient_UpstreamUnauthorizedReturnsErrUnauthorized(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer upstream.Close()

	client := NewConfluenceClient(upstream.URL)

	_, err := client.ListDocuments(context.Background(), "invalid-pat")
	if !errors.Is(err, sharedErrors.ErrUnauthorized) {
		t.Fatalf("expected ErrUnauthorized for upstream 401, got %v", err)
	}
	if !strings.Contains(err.Error(), "invalid or expired Confluence PAT") {
		t.Errorf("expected invalid PAT message, got %q", err.Error())
	}
	if strings.Contains(err.Error(), "invalid-pat") {
		t.Error("expected PAT value to never appear in the error")
	}
}

func TestConfluenceClient_UpstreamUnreachableReturnsNonAuthError(t *testing.T) {
	closedUpstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	baseURL := closedUpstream.URL
	closedUpstream.Close()

	client := NewConfluenceClient(baseURL)

	_, err := client.ListDocuments(context.Background(), testPAT)
	if err == nil {
		t.Fatal("expected error for unreachable upstream, got nil")
	}
	if errors.Is(err, sharedErrors.ErrUnauthorized) {
		t.Error("expected unreachable upstream to not map to ErrUnauthorized")
	}
	if !strings.Contains(err.Error(), "unable to reach Confluence API") {
		t.Errorf("expected unreachable message, got %q", err.Error())
	}
}

func TestConfluenceClient_UpstreamServerErrorReturnsStatusError(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer upstream.Close()

	client := NewConfluenceClient(upstream.URL)

	_, err := client.ListDocuments(context.Background(), testPAT)
	if err == nil {
		t.Fatal("expected error for upstream 502, got nil")
	}
	if errors.Is(err, sharedErrors.ErrUnauthorized) || errors.Is(err, sharedErrors.ErrNotFound) {
		t.Errorf("expected generic status error, got %v", err)
	}
	if !strings.Contains(err.Error(), "confluence returned status 502") {
		t.Errorf("expected upstream status to be wrapped, got %q", err.Error())
	}
}

func TestConfluenceClient_GetDocumentDetailNotFoundReturnsErrNotFound(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer upstream.Close()

	client := NewConfluenceClient(upstream.URL)

	_, err := client.GetDocumentDetail(context.Background(), testPAT, "999")
	if !errors.Is(err, sharedErrors.ErrNotFound) {
		t.Fatalf("expected ErrNotFound for upstream 404, got %v", err)
	}
	if !strings.Contains(err.Error(), "999") {
		t.Errorf("expected missing document id in message, got %q", err.Error())
	}
}

func TestConfluenceClient_ListDocumentsRequestShape(t *testing.T) {
	var receivedQuery string
	var receivedAuth string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedQuery = r.URL.Query().Get("cql")
		receivedAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"results":[]}`))
	}))
	defer upstream.Close()

	client := NewConfluenceClient(upstream.URL)
	documents, err := client.ListDocuments(context.Background(), testPAT)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if receivedQuery != "type=page AND status=current" {
		t.Errorf("expected cql query, got %q", receivedQuery)
	}
	if receivedAuth != "Bearer "+testPAT {
		t.Errorf("expected bearer PAT header, got %q", receivedAuth)
	}
	if documents == nil {
		t.Fatal("expected non-nil documents slice for empty results")
	}
	if len(documents) != 0 {
		t.Errorf("expected 0 documents, got %d", len(documents))
	}
}

func TestConfluenceClient_ListDocumentsParsesContentFields(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/api/content/search" {
			t.Errorf("expected search path, got %q", r.URL.Path)
		}
		if r.URL.Query().Get("limit") != "100" {
			t.Errorf("expected limit=100, got %q", r.URL.Query().Get("limit"))
		}
		if r.URL.Query().Get("expand") != "version,space" {
			t.Errorf("expected expand=version,space, got %q", r.URL.Query().Get("expand"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"results":[{"id":"101","title":"UT Coverage Report","space":{"key":"DEV","name":"DEV Team"},"version":{"when":"2026-01-02T03:04:05.000+07:00","by":{"displayName":"Jane Doe"}},"_links":{"base":"https://confluence.example.com","webui":"/display/DEV/UT-Coverage-Report"}}]}`))
	}))
	defer upstream.Close()

	client := NewConfluenceClient(upstream.URL)
	documents, err := client.ListDocuments(context.Background(), testPAT)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(documents) != 1 {
		t.Fatalf("expected 1 document, got %d", len(documents))
	}

	document := documents[0]
	if document.ID != "101" {
		t.Errorf("expected id 101, got %q", document.ID)
	}
	if document.Owner != "Jane Doe" {
		t.Errorf("expected owner Jane Doe, got %q", document.Owner)
	}
	if document.Updated != "2026-01-02T03:04:05.000+07:00" {
		t.Errorf("expected ISO8601 updated timestamp, got %q", document.Updated)
	}
	if document.Space != "DEV Team" {
		t.Errorf("expected space DEV Team, got %q", document.Space)
	}
	if document.URL != "https://confluence.example.com/display/DEV/UT-Coverage-Report" {
		t.Errorf("expected absolute page URL, got %q", document.URL)
	}
	if document.Title != "UT Coverage Report" {
		t.Errorf("expected title, got %q", document.Title)
	}
}

func TestConfluenceClient_DocumentTypeMapping(t *testing.T) {
	testCases := []struct {
		name          string
		labelNames    []string
		title         string
		expectedType  string
		expectedLabel string
	}{
		{"label beats title", []string{"sop"}, "Query Review Checklist", "sop", "SOP"},
		{"unit test label beats title", []string{"unit-test"}, "SOP Onboarding Guide", "ut", "UT"},
		{"label is lowercased", []string{"Unit-Test"}, "Plain title", "ut", "UT"},
		{"query label", []string{"query-review"}, "Plain title", "query", "QR"},
		{"qr label", []string{"qr"}, "Plain title", "query", "QR"},
		{"sop label", []string{"sop"}, "Plain title", "sop", "SOP"},
		{"unmatched label falls back to title", []string{"draft"}, "Query Review Notes", "query", "QR"},
		{"query review in title", nil, "Q1 Query Review Notes", "query", "QR"},
		{"standalone sop in title", nil, "SOP for Settlement Reconciliation", "sop", "SOP"},
		{"standalone sop suffix in title", nil, "Daily Checklist SOP", "sop", "SOP"},
		{"standalone ut in title", nil, "UT Coverage Report", "ut", "UT"},
		{"unit test in title", nil, "Unit Test Strategy 2026", "ut", "UT"},
		{"doc fallback", nil, "Architecture Overview", "doc", "DOC"},
		{"ut inside word is not a match", nil, "Butcher Utilities Overview", "doc", "DOC"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			content := confluenceContent{ID: "101", Title: testCase.title}
			for _, labelName := range testCase.labelNames {
				content.Metadata.Labels.Results = append(content.Metadata.Labels.Results, confluenceLabel{Name: labelName})
			}

			document := buildDocumentFromContent(content, "https://confluence.example.com")

			if document.Type != testCase.expectedType {
				t.Errorf("expected type %q, got %q", testCase.expectedType, document.Type)
			}
			if document.TypeLabel != testCase.expectedLabel {
				t.Errorf("expected type_label %q, got %q", testCase.expectedLabel, document.TypeLabel)
			}
		})
	}
}

func TestConfluenceClient_StatusMapping(t *testing.T) {
	testCases := []struct {
		name          string
		labelNames    []string
		expectedState string
	}{
		{"done label", []string{"done"}, "done"},
		{"closed label", []string{"closed"}, "done"},
		{"resolved label", []string{"resolved"}, "done"},
		{"in-progress label", []string{"in-progress"}, "progress"},
		{"in progress label", []string{"in progress"}, "progress"},
		{"progress label", []string{"progress"}, "progress"},
		{"unrelated label stays open", []string{"draft"}, "open"},
		{"no labels stays open", nil, "open"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			content := confluenceContent{ID: "101", Title: "Plain title"}
			for _, labelName := range testCase.labelNames {
				content.Metadata.Labels.Results = append(content.Metadata.Labels.Results, confluenceLabel{Name: labelName})
			}

			document := buildDocumentFromContent(content, "https://confluence.example.com")

			if document.Status != testCase.expectedState {
				t.Errorf("expected status %q, got %q", testCase.expectedState, document.Status)
			}
		})
	}
}

func TestConfluenceClient_DetailDescriptionStripsHTML(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/api/content/101" {
			t.Errorf("expected detail path, got %q", r.URL.Path)
		}
		if r.URL.Query().Get("expand") != "body.storage,version,space,metadata.labels" {
			t.Errorf("expected detail expand, got %q", r.URL.Query().Get("expand"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"101","title":"UT Coverage Report","space":{"key":"DEV","name":"DEV Team"},"version":{"when":"2026-01-02T03:04:05.000+07:00","by":{"displayName":"Jane Doe"}},"metadata":{"labels":{"results":[{"name":"unit-test"}]}},"body":{"storage":{"value":"<h2>Summary</h2><p>Coverage   &amp; regression</p><div>next   line</div>"}},"_links":{"base":"https://confluence.example.com","webui":""}}`))
	}))
	defer upstream.Close()

	client := NewConfluenceClient(upstream.URL)
	document, err := client.GetDocumentDetail(context.Background(), testPAT, "101")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if document.Description != "Summary Coverage & regression next line" {
		t.Errorf("expected plain collapsed description, got %q", document.Description)
	}
	if document.Type != "ut" || document.TypeLabel != "UT" {
		t.Errorf("expected label-driven type ut/UT, got %q/%q", document.Type, document.TypeLabel)
	}
	if document.Status != "open" {
		t.Errorf("expected status open, got %q", document.Status)
	}
	if document.URL != "https://confluence.example.com/pages/viewpage.action?pageId=101" {
		t.Errorf("expected fallback page URL, got %q", document.URL)
	}
}

func TestConfluenceClient_DescriptionIsCappedAtLimit(t *testing.T) {
	longDescription := "<p>" + strings.Repeat("a", 2500) + "</p>"
	plainDescription := plainTextFromStorage(longDescription)

	if len([]rune(plainDescription)) != descriptionMaxLength {
		t.Errorf("expected description capped at %d runes, got %d", descriptionMaxLength, len([]rune(plainDescription)))
	}
}

func TestConfluenceClient_FallbackBaseUsedWhenLinksMissing(t *testing.T) {
	content := confluenceContent{ID: "42", Title: "Plain title"}

	document := buildDocumentFromContent(content, "https://confluence.example.com/")

	if document.URL != "https://confluence.example.com/pages/viewpage.action?pageId=42" {
		t.Errorf("expected URL built from client base, got %q", document.URL)
	}
}
