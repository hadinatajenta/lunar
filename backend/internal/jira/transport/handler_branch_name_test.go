package transport

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"testing"

	jiraApp "lunar/backend/internal/jira/application"
	jiraDomain "lunar/backend/internal/jira/domain"
	sharedErrors "lunar/backend/internal/shared/errors"
)

const branchNameTestKey = "CRMMS-77911"

type stubBranchNameIssueReader struct {
	issue *jiraDomain.JiraIssue
	err   error
}

func (s stubBranchNameIssueReader) GetIssueDetail(_ context.Context, _ string, _ string) (*jiraDomain.JiraIssue, error) {
	if s.err != nil {
		return nil, s.err
	}

	return s.issue, nil
}

type stubBranchNameCompletion struct {
	answer string
	err    error
}

func (s stubBranchNameCompletion) CompleteJSON(_ context.Context, _ string, _ string, _ string) (string, error) {
	if s.err != nil {
		return "", s.err
	}

	return s.answer, nil
}

func branchNameTestIssue() *jiraDomain.JiraIssue {
	return &jiraDomain.JiraIssue{
		Key: branchNameTestKey,
		Fields: jiraDomain.JiraIssueFields{
			Summary: "Sebagai Administrator, I want to melihat daftar user",
			Description: "h3. Deskripsi Acceptance Criteria (AC)\n\n" +
				"*1. Halaman Utama Manage User (Grouping by PN)*\n" +
				"The page lists every user grouped by personal number.",
		},
	}
}

func newBranchNameTestMux(reader jiraApp.BranchNameIssueSource, client jiraApp.ModelCompletionClient) *http.ServeMux {
	mux := http.NewServeMux()
	RegisterBranchNameRoutes(mux, func(next http.Handler) http.Handler { return next }, jiraApp.NewBranchNameService(reader, client))

	return mux
}

func TestBranchNameHandlerReturnsTheWholeSuggestion(t *testing.T) {
	mux := newBranchNameTestMux(
		stubBranchNameIssueReader{issue: branchNameTestIssue()},
		stubBranchNameCompletion{answer: `{"category":"Enhancement","core":"Manage-User","evidence":"Halaman Utama Manage User (Grouping by PN)"}`},
	)

	recorder := performIssueBranchRequest(t, mux, "user-a", "/api/jira/issues/"+branchNameTestKey+"/branch-name")
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", recorder.Code, recorder.Body.String())
	}

	var fields map[string]json.RawMessage
	if err := json.Unmarshal(recorder.Body.Bytes(), &fields); err != nil {
		t.Fatalf("cannot decode the response: %v", err)
	}
	keys := make([]string, 0, len(fields))
	for key := range fields {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	wantKeys := []string{"branch", "category", "core", "evidence", "key", "source"}
	if strings.Join(keys, ",") != strings.Join(wantKeys, ",") {
		t.Fatalf("response fields = %v, want %v", keys, wantKeys)
	}

	var suggestion jiraDomain.BranchNameSuggestion
	if err := json.Unmarshal(recorder.Body.Bytes(), &suggestion); err != nil {
		t.Fatalf("cannot decode the suggestion: %v", err)
	}
	if suggestion.Key != branchNameTestKey {
		t.Fatalf("key = %q", suggestion.Key)
	}
	if suggestion.Branch != "feature/CRMMS-77911-Enhancement-Manage-User" {
		t.Fatalf("branch = %q", suggestion.Branch)
	}
	if suggestion.Category != "Enhancement" || suggestion.Core != "Manage-User" {
		t.Fatalf("category = %q, core = %q", suggestion.Category, suggestion.Core)
	}
	if suggestion.Source != jiraDomain.BranchNameSourceModel {
		t.Fatalf("source = %q", suggestion.Source)
	}
	if suggestion.Evidence != "Halaman Utama Manage User (Grouping by PN)" {
		t.Fatalf("evidence = %q", suggestion.Evidence)
	}
}

func TestBranchNameHandlerReturns200WithTheFallbackWhenTheModelFails(t *testing.T) {
	mux := newBranchNameTestMux(
		stubBranchNameIssueReader{issue: branchNameTestIssue()},
		stubBranchNameCompletion{err: errors.New("no AI provider is configured")},
	)

	recorder := performIssueBranchRequest(t, mux, "user-a", "/api/jira/issues/"+branchNameTestKey+"/branch-name")
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", recorder.Code, recorder.Body.String())
	}

	var suggestion jiraDomain.BranchNameSuggestion
	if err := json.Unmarshal(recorder.Body.Bytes(), &suggestion); err != nil {
		t.Fatalf("cannot decode the suggestion: %v", err)
	}
	if suggestion.Source != jiraDomain.BranchNameSourceFallback {
		t.Fatalf("source = %q, want the fallback", suggestion.Source)
	}
	if suggestion.Branch != "feature/CRMMS-77911-Melihat-Daftar-User" {
		t.Fatalf("branch = %q", suggestion.Branch)
	}
	if suggestion.Evidence != "melihat daftar user" {
		t.Fatalf("evidence = %q", suggestion.Evidence)
	}
}

func TestBranchNameHandlerRejectsAnInvalidIssueKey(t *testing.T) {
	mux := newBranchNameTestMux(stubBranchNameIssueReader{issue: branchNameTestIssue()}, stubBranchNameCompletion{})

	recorder := performIssueBranchRequest(t, mux, "user-a", "/api/jira/issues/not-a-key/branch-name")
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", recorder.Code, recorder.Body.String())
	}
}

func TestBranchNameHandlerRequiresAnAuthenticatedUser(t *testing.T) {
	mux := newBranchNameTestMux(stubBranchNameIssueReader{issue: branchNameTestIssue()}, stubBranchNameCompletion{})

	recorder := performIssueBranchRequest(t, mux, "", "/api/jira/issues/"+branchNameTestKey+"/branch-name")
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", recorder.Code, recorder.Body.String())
	}
}

func TestBranchNameHandlerReportsAMissingIssue(t *testing.T) {
	mux := newBranchNameTestMux(
		stubBranchNameIssueReader{err: fmt.Errorf("%w: issue %s not found", sharedErrors.ErrNotFound, branchNameTestKey)},
		stubBranchNameCompletion{},
	)

	recorder := performIssueBranchRequest(t, mux, "user-a", "/api/jira/issues/"+branchNameTestKey+"/branch-name")
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", recorder.Code, recorder.Body.String())
	}
}

func TestBranchNameHandlerReportsJiraUnreachable(t *testing.T) {
	mux := newBranchNameTestMux(
		stubBranchNameIssueReader{err: fmt.Errorf("%w: unable to reach Jira", sharedErrors.ErrBadGateway)},
		stubBranchNameCompletion{},
	)

	recorder := performIssueBranchRequest(t, mux, "user-a", "/api/jira/issues/"+branchNameTestKey+"/branch-name")
	if recorder.Code != http.StatusBadGateway {
		t.Fatalf("expected 502, got %d: %s", recorder.Code, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), "unable to reach Jira BRI API") {
		t.Fatalf("error body = %q", recorder.Body.String())
	}
}

func TestBranchNameHandlerServesOnlyGet(t *testing.T) {
	mux := newBranchNameTestMux(stubBranchNameIssueReader{issue: branchNameTestIssue()}, stubBranchNameCompletion{})

	request := httptest.NewRequest(http.MethodPost, "/api/jira/issues/"+branchNameTestKey+"/branch-name", nil)
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d: %s", recorder.Code, recorder.Body.String())
	}
}
