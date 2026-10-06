package infrastructure

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	sharedErrors "lunar/backend/internal/shared/errors"
)

func writeBranchListPage(t *testing.T, w http.ResponseWriter, values []map[string]string, isLastPage bool, nextPageStart int) {
	t.Helper()

	payload, err := json.Marshal(map[string]any{
		"values":        values,
		"isLastPage":    isLastPage,
		"nextPageStart": nextPageStart,
	})
	if err != nil {
		t.Fatalf("cannot encode the branch list page: %v", err)
	}

	w.Header().Set("Content-Type", "application/json")
	if _, err := w.Write(payload); err != nil {
		t.Errorf("cannot write the branch list page: %v", err)
	}
}

func TestListBranchesAddressesTheRepositoryAndReturnsEveryBranch(t *testing.T) {
	requestedPath := ""
	requestedQuery := ""
	authorizationHeader := ""

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestedPath = r.URL.Path
		requestedQuery = r.URL.RawQuery
		authorizationHeader = r.Header.Get("Authorization")
		writeBranchListPage(t, w, []map[string]string{
			{"id": "refs/heads/release/production", "displayId": "release/production"},
			{"id": "refs/heads/main"},
		}, true, 0)
	}))
	defer server.Close()

	client := NewAtlassianBitbucketClient(server.URL)
	branches, err := client.ListBranches(context.Background(), "pat-token", "NME/mms-api")
	if err != nil {
		t.Fatalf("cannot list branches: %v", err)
	}

	if requestedPath != "/rest/api/1.0/projects/NME/repos/mms-api/branches" {
		t.Fatalf("unexpected request path %q", requestedPath)
	}
	if !strings.Contains(requestedQuery, fmt.Sprintf("limit=%d", branchListPageLimit)) {
		t.Fatalf("expected the page limit in the query, got %q", requestedQuery)
	}
	if !strings.Contains(requestedQuery, "start=0") {
		t.Fatalf("expected the first page to start at zero, got %q", requestedQuery)
	}
	if authorizationHeader != "Bearer pat-token" {
		t.Fatalf("unexpected authorization header %q", authorizationHeader)
	}

	expectedNames := []string{"release/production", "main"}
	if len(branches) != len(expectedNames) {
		t.Fatalf("expected %d branches, got %+v", len(expectedNames), branches)
	}
	for index, expectedName := range expectedNames {
		if branches[index] != expectedName {
			t.Fatalf("expected branch %d to be %q, got %q", index, expectedName, branches[index])
		}
	}
}

func TestListBranchesFollowsPagingUntilTheLastPage(t *testing.T) {
	requestedStarts := make([]string, 0, 2)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := r.URL.Query().Get("start")
		requestedStarts = append(requestedStarts, start)

		if start == "0" {
			writeBranchListPage(t, w, []map[string]string{{"displayId": "release/production"}}, false, 2)
			return
		}
		writeBranchListPage(t, w, []map[string]string{{"displayId": "main"}, {"id": "refs/heads/develop"}}, true, 0)
	}))
	defer server.Close()

	client := NewAtlassianBitbucketClient(server.URL)
	branches, err := client.ListBranches(context.Background(), "pat-token", "NME/mms-api")
	if err != nil {
		t.Fatalf("cannot list branches: %v", err)
	}

	if len(requestedStarts) != 2 || requestedStarts[0] != "0" || requestedStarts[1] != "2" {
		t.Fatalf("expected pages at start 0 and 2, got %+v", requestedStarts)
	}

	expectedNames := []string{"release/production", "main", "develop"}
	if len(branches) != len(expectedNames) {
		t.Fatalf("expected %d branches, got %+v", len(expectedNames), branches)
	}
	for index, expectedName := range expectedNames {
		if branches[index] != expectedName {
			t.Fatalf("expected branch %d to be %q, got %q", index, expectedName, branches[index])
		}
	}
}

func TestListBranchesRefusesAPageThatDoesNotAdvance(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeBranchListPage(t, w, []map[string]string{{"displayId": "main"}}, false, 0)
	}))
	defer server.Close()

	client := NewAtlassianBitbucketClient(server.URL)
	_, err := client.ListBranches(context.Background(), "pat-token", "NME/mms-api")
	if err == nil {
		t.Fatal("expected an error when the listing does not advance")
	}
	if !strings.Contains(err.Error(), "did not advance") {
		t.Fatalf("expected the stalled listing to be named, got %q", err.Error())
	}
}

func TestListBranchesStopsAtThePageLimit(t *testing.T) {
	requestedPages := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestedPages++
		start := requestedPages
		writeBranchListPage(t, w, []map[string]string{{"displayId": "main"}}, false, start)
	}))
	defer server.Close()

	client := NewAtlassianBitbucketClient(server.URL)
	_, err := client.ListBranches(context.Background(), "pat-token", "NME/mms-api")
	if err == nil {
		t.Fatal("expected an error for a listing that never ends")
	}
	if !strings.Contains(err.Error(), "exceeded") {
		t.Fatalf("expected the page limit to be named, got %q", err.Error())
	}
	if requestedPages != maxBranchListPages {
		t.Fatalf("expected %d pages, got %d", maxBranchListPages, requestedPages)
	}
}

func TestListBranchesRequiresAPAT(t *testing.T) {
	client := NewAtlassianBitbucketClient("https://bitbucket.example")

	_, err := client.ListBranches(context.Background(), "   ", "NME/mms-api")
	if !errors.Is(err, sharedErrors.ErrUnauthorized) {
		t.Fatalf("expected an unauthorized error, got %v", err)
	}
}

func TestListBranchesReportsAnExpiredPAT(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	client := NewAtlassianBitbucketClient(server.URL)
	_, err := client.ListBranches(context.Background(), "pat-token", "NME/mms-api")
	if !errors.Is(err, sharedErrors.ErrUnauthorized) {
		t.Fatalf("expected an unauthorized error, got %v", err)
	}
}

func TestListBranchesReportsAForbiddenRepository(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer server.Close()

	client := NewAtlassianBitbucketClient(server.URL)
	_, err := client.ListBranches(context.Background(), "pat-token", "NME/mms-api")
	if !errors.Is(err, sharedErrors.ErrForbidden) {
		t.Fatalf("expected a forbidden error, got %v", err)
	}
}

func TestListBranchesReportsAMissingRepository(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := NewAtlassianBitbucketClient(server.URL)
	_, err := client.ListBranches(context.Background(), "pat-token", "NME/missing")
	if !errors.Is(err, sharedErrors.ErrNotFound) {
		t.Fatalf("expected a not found error, got %v", err)
	}
}

func TestListBranchesReportsAnUnexpectedStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewAtlassianBitbucketClient(server.URL)
	_, err := client.ListBranches(context.Background(), "pat-token", "NME/mms-api")
	if err == nil {
		t.Fatal("expected an error for an unexpected status")
	}
	if !strings.Contains(err.Error(), "500") {
		t.Fatalf("expected the status to be reported, got %q", err.Error())
	}
}

func TestListBranchesReportsAnUndecodablePage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := w.Write([]byte("{not-json")); err != nil {
			t.Errorf("cannot write the malformed page: %v", err)
		}
	}))
	defer server.Close()

	client := NewAtlassianBitbucketClient(server.URL)
	_, err := client.ListBranches(context.Background(), "pat-token", "NME/mms-api")
	if err == nil {
		t.Fatal("expected an error for an undecodable page")
	}
	if !strings.Contains(err.Error(), "decode") {
		t.Fatalf("expected the decoding failure to be named, got %q", err.Error())
	}
}

func TestListBranchesHonoursTheCallerContext(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeBranchListPage(t, w, []map[string]string{{"displayId": "main"}}, true, 0)
	}))
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	client := NewAtlassianBitbucketClient(server.URL)
	_, err := client.ListBranches(ctx, "pat-token", "NME/mms-api")
	if err == nil {
		t.Fatal("expected an error for a cancelled context")
	}
}
