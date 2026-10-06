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

func TestDefaultBranchAddressesTheRepositoryAndReturnsTheDeclaredBranch(t *testing.T) {
	requestedPath := ""
	authorizationHeader := ""

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestedPath = r.URL.Path
		authorizationHeader = r.Header.Get("Authorization")
		if _, err := w.Write([]byte(`{"id":"refs/heads/master","displayId":"master"}`)); err != nil {
			t.Errorf("cannot write the default branch: %v", err)
		}
	}))
	defer server.Close()

	client := NewAtlassianBitbucketClient(server.URL)
	branch, err := client.DefaultBranch(context.Background(), "pat-token", "NME/aurora")
	if err != nil {
		t.Fatalf("cannot read the default branch: %v", err)
	}

	if requestedPath != "/rest/api/1.0/projects/NME/repos/aurora/default-branch" {
		t.Fatalf("unexpected request path %q", requestedPath)
	}
	if authorizationHeader != "Bearer pat-token" {
		t.Fatalf("unexpected authorization header %q", authorizationHeader)
	}
	if branch != "master" {
		t.Fatalf("expected master, got %q", branch)
	}
}

func TestDefaultBranchFallsBackToTheReferenceId(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := w.Write([]byte(`{"id":"refs/heads/development"}`)); err != nil {
			t.Errorf("cannot write the default branch: %v", err)
		}
	}))
	defer server.Close()

	client := NewAtlassianBitbucketClient(server.URL)
	branch, err := client.DefaultBranch(context.Background(), "pat-token", "NME/aurora")
	if err != nil {
		t.Fatalf("cannot read the default branch: %v", err)
	}
	if branch != "development" {
		t.Fatalf("expected development, got %q", branch)
	}
}

func TestDefaultBranchRequiresAPAT(t *testing.T) {
	client := NewAtlassianBitbucketClient("https://bitbucket.example")

	_, err := client.DefaultBranch(context.Background(), "  ", "NME/aurora")
	if !errors.Is(err, sharedErrors.ErrUnauthorized) {
		t.Fatalf("expected an unauthorized error, got %v", err)
	}
}

func TestDefaultBranchReportsAnExpiredPAT(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	client := NewAtlassianBitbucketClient(server.URL)
	_, err := client.DefaultBranch(context.Background(), "pat-token", "NME/aurora")
	if !errors.Is(err, sharedErrors.ErrUnauthorized) {
		t.Fatalf("expected an unauthorized error, got %v", err)
	}
}

func TestDefaultBranchReportsARepositoryWithoutADefaultBranch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := NewAtlassianBitbucketClient(server.URL)
	_, err := client.DefaultBranch(context.Background(), "pat-token", "NME/empty-repository")
	if !errors.Is(err, sharedErrors.ErrNotFound) {
		t.Fatalf("expected a not found error, got %v", err)
	}
}

func TestDefaultBranchReportsAnEmptyAnswer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := w.Write([]byte("{}")); err != nil {
			t.Errorf("cannot write the empty answer: %v", err)
		}
	}))
	defer server.Close()

	client := NewAtlassianBitbucketClient(server.URL)
	_, err := client.DefaultBranch(context.Background(), "pat-token", "NME/aurora")
	if err == nil {
		t.Fatal("expected an error for an empty default branch answer")
	}
	if !strings.Contains(err.Error(), "default branch") {
		t.Fatalf("expected the missing branch to be named, got %q", err.Error())
	}
}

func TestDefaultBranchReportsAnUnexpectedStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer server.Close()

	client := NewAtlassianBitbucketClient(server.URL)
	_, err := client.DefaultBranch(context.Background(), "pat-token", "NME/aurora")
	if err == nil {
		t.Fatal("expected an error for an unexpected status")
	}
	if !strings.Contains(err.Error(), "502") {
		t.Fatalf("expected the status to be reported, got %q", err.Error())
	}
}

func TestDefaultBranchReportsAnUndecodableAnswer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := w.Write([]byte("{not-json")); err != nil {
			t.Errorf("cannot write the malformed answer: %v", err)
		}
	}))
	defer server.Close()

	client := NewAtlassianBitbucketClient(server.URL)
	_, err := client.DefaultBranch(context.Background(), "pat-token", "NME/aurora")
	if err == nil {
		t.Fatal("expected an error for an undecodable answer")
	}
	if !strings.Contains(err.Error(), "decode") {
		t.Fatalf("expected the decoding failure to be named, got %q", err.Error())
	}
}
