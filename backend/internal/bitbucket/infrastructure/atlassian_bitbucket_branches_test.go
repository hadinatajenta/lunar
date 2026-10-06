package infrastructure

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	sharedErrors "lunar/backend/internal/shared/errors"
)

func writeBranchPage(t *testing.T, w http.ResponseWriter, values []map[string]string) {
	t.Helper()

	payload, err := json.Marshal(map[string]any{"values": values})
	if err != nil {
		t.Fatalf("cannot encode the branch page: %v", err)
	}

	w.Header().Set("Content-Type", "application/json")
	if _, err := w.Write(payload); err != nil {
		t.Errorf("cannot write the branch page: %v", err)
	}
}

func TestSearchBranchesAddressesTheNamedRepositoryAndReturnsEveryMatch(t *testing.T) {
	requestedPath := ""
	requestedQuery := ""
	authorizationHeader := ""

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestedPath = r.URL.Path
		requestedQuery = r.URL.RawQuery
		authorizationHeader = r.Header.Get("Authorization")
		writeBranchPage(t, w, []map[string]string{
			{"id": "refs/heads/feature/CRMMS-77911-Enhancement-Manage-User", "displayId": "feature/CRMMS-77911-Enhancement-Manage-User"},
			{"id": "refs/heads/feature/CRMMS-77911-Other-Tail"},
		})
	}))
	defer server.Close()

	client := NewAtlassianBitbucketClient(server.URL)
	branches, err := client.SearchBranches(context.Background(), "pat-token", "NME/mms-api", "CRMMS-77911")
	if err != nil {
		t.Fatalf("cannot search branches: %v", err)
	}

	if requestedPath != "/rest/api/1.0/projects/NME/repos/mms-api/branches" {
		t.Fatalf("unexpected request path %q", requestedPath)
	}
	if !strings.Contains(requestedQuery, "filterText=CRMMS-77911") {
		t.Fatalf("expected the filter text in the query, got %q", requestedQuery)
	}
	if authorizationHeader != "Bearer pat-token" {
		t.Fatalf("unexpected authorization header %q", authorizationHeader)
	}

	expectedNames := []string{"feature/CRMMS-77911-Enhancement-Manage-User", "feature/CRMMS-77911-Other-Tail"}
	if len(branches) != len(expectedNames) {
		t.Fatalf("expected %d branches, got %+v", len(expectedNames), branches)
	}
	for index, expectedName := range expectedNames {
		if branches[index] != expectedName {
			t.Fatalf("expected branch %d to be %q, got %q", index, expectedName, branches[index])
		}
	}
}

func TestSearchBranchesFallsBackToTheReferenceId(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeBranchPage(t, w, []map[string]string{{"id": "refs/heads/release/production"}})
	}))
	defer server.Close()

	client := NewAtlassianBitbucketClient(server.URL)
	branches, err := client.SearchBranches(context.Background(), "pat-token", "NME/mms-api", "CRMMS-77911")
	if err != nil {
		t.Fatalf("cannot search branches: %v", err)
	}
	if len(branches) != 1 || branches[0] != "release/production" {
		t.Fatalf("expected the reference id to be used, got %+v", branches)
	}
}

func TestSearchBranchesRequiresAPAT(t *testing.T) {
	client := NewAtlassianBitbucketClient("https://bitbucket.example")

	_, err := client.SearchBranches(context.Background(), "  ", "NME/mms-api", "CRMMS-77911")
	if !errors.Is(err, sharedErrors.ErrUnauthorized) {
		t.Fatalf("expected an unauthorized error, got %v", err)
	}
}

func TestSearchBranchesReportsAnExpiredPAT(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	client := NewAtlassianBitbucketClient(server.URL)
	_, err := client.SearchBranches(context.Background(), "pat-token", "NME/mms-api", "CRMMS-77911")
	if !errors.Is(err, sharedErrors.ErrUnauthorized) {
		t.Fatalf("expected an unauthorized error, got %v", err)
	}
}

func TestSearchBranchesReportsAnUnexpectedStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewAtlassianBitbucketClient(server.URL)
	_, err := client.SearchBranches(context.Background(), "pat-token", "NME/mms-api", "CRMMS-77911")
	if err == nil {
		t.Fatal("expected an error for an unexpected status")
	}
	if !strings.Contains(err.Error(), "500") {
		t.Fatalf("expected the status to be reported, got %q", err.Error())
	}
}
