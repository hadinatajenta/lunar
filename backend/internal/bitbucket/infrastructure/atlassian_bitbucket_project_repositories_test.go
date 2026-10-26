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

func writeProjectRepositoryPage(t *testing.T, w http.ResponseWriter, values []map[string]any, isLastPage bool, nextPageStart int) {
	t.Helper()

	payload, err := json.Marshal(map[string]any{
		"values":        values,
		"isLastPage":    isLastPage,
		"nextPageStart": nextPageStart,
	})
	if err != nil {
		t.Fatalf("cannot encode the project repository page: %v", err)
	}

	w.Header().Set("Content-Type", "application/json")
	if _, err := w.Write(payload); err != nil {
		t.Errorf("cannot write the project repository page: %v", err)
	}
}

func TestListProjectRepositoriesAddressesTheProjectAndReturnsEveryRepository(t *testing.T) {
	requestedPath := ""
	requestedQuery := ""
	authorizationHeader := ""

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestedPath = r.URL.Path
		requestedQuery = r.URL.RawQuery
		authorizationHeader = r.Header.Get("Authorization")
		writeProjectRepositoryPage(t, w, []map[string]any{
			{"slug": "mms-api", "project": map[string]string{"key": "NME", "name": "NME"}},
			{"slug": "atlas", "project": map[string]string{"key": "NME"}},
			{"slug": "elbrus"},
			{"slug": "   "},
		}, true, 0)
	}))
	defer server.Close()

	client := NewAtlassianBitbucketClient(server.URL)
	repositories, err := client.ListProjectRepositories(context.Background(), "pat-token", "NME")
	if err != nil {
		t.Fatalf("cannot list project repositories: %v", err)
	}

	if requestedPath != "/rest/api/1.0/projects/NME/repos" {
		t.Fatalf("unexpected request path %q", requestedPath)
	}
	if !strings.Contains(requestedQuery, fmt.Sprintf("limit=%d", projectRepositoryPageLimit)) {
		t.Fatalf("expected the page limit in the query, got %q", requestedQuery)
	}
	if !strings.Contains(requestedQuery, "start=0") {
		t.Fatalf("expected the first page to start at zero, got %q", requestedQuery)
	}
	if authorizationHeader != "Bearer pat-token" {
		t.Fatalf("unexpected authorization header %q", authorizationHeader)
	}

	expectedSlugs := []string{"mms-api", "atlas", "elbrus"}
	if len(repositories) != len(expectedSlugs) {
		t.Fatalf("expected %d repositories, got %+v", len(expectedSlugs), repositories)
	}
	for index, expectedSlug := range expectedSlugs {
		if repositories[index].Slug != expectedSlug {
			t.Fatalf("expected repository %d to be %q, got %q", index, expectedSlug, repositories[index].Slug)
		}
		if repositories[index].ProjectKey != "NME" {
			t.Fatalf("expected repository %d to fall back to the requested project, got %q", index, repositories[index].ProjectKey)
		}
	}
}

func TestListProjectRepositoriesFollowsPagingUntilTheLastPage(t *testing.T) {
	requestedStarts := make([]string, 0, 2)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := r.URL.Query().Get("start")
		requestedStarts = append(requestedStarts, start)

		if start == "0" {
			writeProjectRepositoryPage(t, w, []map[string]any{{"slug": "mms-api"}}, false, 100)
			return
		}
		writeProjectRepositoryPage(t, w, []map[string]any{{"slug": "atlas"}}, true, 0)
	}))
	defer server.Close()

	client := NewAtlassianBitbucketClient(server.URL)
	repositories, err := client.ListProjectRepositories(context.Background(), "pat-token", "NME")
	if err != nil {
		t.Fatalf("cannot list project repositories: %v", err)
	}

	if len(requestedStarts) != 2 || requestedStarts[0] != "0" || requestedStarts[1] != "100" {
		t.Fatalf("expected pages at start 0 and 100, got %+v", requestedStarts)
	}
	if len(repositories) != 2 || repositories[0].Slug != "mms-api" || repositories[1].Slug != "atlas" {
		t.Fatalf("unexpected repositories %+v", repositories)
	}
}

func TestListProjectRepositoriesRefusesAPageThatDoesNotAdvance(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeProjectRepositoryPage(t, w, []map[string]any{{"slug": "mms-api"}}, false, 0)
	}))
	defer server.Close()

	client := NewAtlassianBitbucketClient(server.URL)
	_, err := client.ListProjectRepositories(context.Background(), "pat-token", "NME")
	if err == nil {
		t.Fatal("expected an error when the listing does not advance")
	}
	if !strings.Contains(err.Error(), "did not advance") {
		t.Fatalf("expected the stalled listing to be named, got %q", err.Error())
	}
}

func TestListProjectRepositoriesStopsAtThePageBound(t *testing.T) {
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		start := requestCount
		writeProjectRepositoryPage(t, w, []map[string]any{{"slug": fmt.Sprintf("repository-%d", start)}}, false, start)
	}))
	defer server.Close()

	client := NewAtlassianBitbucketClient(server.URL)
	_, err := client.ListProjectRepositories(context.Background(), "pat-token", "NME")
	if err == nil {
		t.Fatal("expected an error when the listing exceeds the page bound")
	}
	if !strings.Contains(err.Error(), "exceeded") {
		t.Fatalf("expected the page bound to be named, got %q", err.Error())
	}
	if requestCount != maxProjectRepositoryPages {
		t.Fatalf("expected at most %d pages, got %d", maxProjectRepositoryPages, requestCount)
	}
}

func TestListProjectRepositoriesRequiresAPATBeforeAnyRequest(t *testing.T) {
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		writeProjectRepositoryPage(t, w, nil, true, 0)
	}))
	defer server.Close()

	client := NewAtlassianBitbucketClient(server.URL)
	_, err := client.ListProjectRepositories(context.Background(), "   ", "NME")
	if !errors.Is(err, sharedErrors.ErrUnauthorized) {
		t.Fatalf("expected an unauthorized error, got %v", err)
	}
	if requestCount != 0 {
		t.Fatalf("a request without a PAT must not reach Bitbucket, got %d requests", requestCount)
	}
}

func TestListProjectRepositoriesRequiresAProjectKey(t *testing.T) {
	client := NewAtlassianBitbucketClient("http://127.0.0.1:1")
	_, err := client.ListProjectRepositories(context.Background(), "pat-token", "  ")
	if !errors.Is(err, sharedErrors.ErrBadRequest) {
		t.Fatalf("expected a bad request error, got %v", err)
	}
}

func TestListProjectRepositoriesMapsBitbucketFailures(t *testing.T) {
	cases := []struct {
		name     string
		status   int
		expected error
	}{
		{"unauthorized", http.StatusUnauthorized, sharedErrors.ErrUnauthorized},
		{"forbidden", http.StatusForbidden, sharedErrors.ErrForbidden},
		{"missing project", http.StatusNotFound, sharedErrors.ErrNotFound},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(testCase.status)
			}))
			defer server.Close()

			client := NewAtlassianBitbucketClient(server.URL)
			_, err := client.ListProjectRepositories(context.Background(), "pat-token", "NME")
			if !errors.Is(err, testCase.expected) {
				t.Fatalf("expected %v, got %v", testCase.expected, err)
			}
		})
	}
}

func TestListProjectRepositoriesReportsAnUnexpectedStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewAtlassianBitbucketClient(server.URL)
	_, err := client.ListProjectRepositories(context.Background(), "pat-token", "NME")
	if err == nil {
		t.Fatal("expected an error for an unexpected status")
	}
	if !strings.Contains(err.Error(), "500") {
		t.Fatalf("expected the status to be named, got %q", err.Error())
	}
}
