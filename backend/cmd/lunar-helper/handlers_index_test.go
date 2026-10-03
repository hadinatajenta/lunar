package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	codeindexapplication "lunar/backend/internal/codeindex/application"
	codeindexdomain "lunar/backend/internal/codeindex/domain"
	codeindexinfrastructure "lunar/backend/internal/codeindex/infrastructure"
)

func newIndexedHelperServer(t *testing.T, chunks []codeindexdomain.CodeChunk) *httptest.Server {
	t.Helper()

	database, err := openIndexDatabase(t.TempDir())
	if err != nil {
		t.Fatalf("cannot open the index database: %v", err)
	}

	store := codeindexinfrastructure.NewStore(database)
	t.Cleanup(func() {
		_ = store.Close()
	})

	if len(chunks) > 0 {
		if err := store.SaveChunks(context.Background(), "settlement-service", chunks); err != nil {
			t.Fatalf("cannot seed chunks: %v", err)
		}
	}

	handler, stop := newHelperHandler(helperDependencies{
		token:      testHelperToken,
		indexStore: store,
		indexer:    codeindexapplication.NewIndexer(store),
		version:    "test-version",
		logger:     discardLogger(),
	})
	t.Cleanup(stop)

	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	return server
}

func decodeIndexStatus(t *testing.T, response *http.Response) indexStatusResponse {
	t.Helper()

	var status indexStatusResponse
	if err := json.NewDecoder(response.Body).Decode(&status); err != nil {
		t.Fatalf("cannot decode index status: %v", err)
	}
	return status
}

func TestIndexStatusRequiresToken(t *testing.T) {
	server := newIndexedHelperServer(t, nil)

	response := doRequest(t, http.MethodGet, server.URL+indexStatusPath, "", "", "")
	if response.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 without a token, got %d", response.StatusCode)
	}
}

func TestIndexStatusUnavailableWithoutIndexStore(t *testing.T) {
	server := newStubHelperServer(t, stubGitReader{}, nil)

	response := doRequest(t, http.MethodGet, server.URL+indexStatusPath, "", bearerScheme+" "+testHelperToken, "")
	if response.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 when the index is not available, got %d", response.StatusCode)
	}
}

func TestIndexStatusReportsIndexedCounts(t *testing.T) {
	chunks := []codeindexdomain.CodeChunk{
		{
			RepoName:   "settlement-service",
			FilePath:   "internal/fee/calculator.go",
			ChunkType:  "code_block",
			RawContent: "func CalculateLateFee() float64 { return 0 }",
			Content:    "func calculate late fee calculator",
			StartLine:  40,
			EndLine:    42,
		},
		{
			RepoName:   "settlement-service",
			FilePath:   "internal/fee/ledger.go",
			ChunkType:  "code_block",
			RawContent: "func WriteLedger() {}",
			Content:    "func write ledger settlement",
			StartLine:  10,
			EndLine:    12,
		},
	}
	server := newIndexedHelperServer(t, chunks)

	response := doRequest(t, http.MethodGet, server.URL+indexStatusPath, "", bearerScheme+" "+testHelperToken, "")
	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", response.StatusCode)
	}

	status := decodeIndexStatus(t, response)
	if !status.Indexed {
		t.Fatalf("expected the index to report as indexed")
	}
	if status.RepoCount != 1 {
		t.Fatalf("expected 1 repository, got %d", status.RepoCount)
	}
	if status.ChunkCount != 2 {
		t.Fatalf("expected 2 chunks, got %d", status.ChunkCount)
	}
	if status.IsBuilding {
		t.Fatalf("expected no build to be running")
	}
}

func TestSearchRequiresQuery(t *testing.T) {
	server := newIndexedHelperServer(t, nil)

	response := doRequest(t, http.MethodGet, server.URL+searchPath+"?q=", "", bearerScheme+" "+testHelperToken, "")
	if response.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for an empty query, got %d", response.StatusCode)
	}
}

func TestSearchRejectsMalformedLimit(t *testing.T) {
	server := newIndexedHelperServer(t, nil)

	response := doRequest(t, http.MethodGet, server.URL+searchPath+"?q=ledger&limit=abc", "", bearerScheme+" "+testHelperToken, "")
	if response.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for a malformed limit, got %d", response.StatusCode)
	}
}

func TestSearchClampsLimitAboveMaximum(t *testing.T) {
	chunks := []codeindexdomain.CodeChunk{
		{
			RepoName:   "settlement-service",
			FilePath:   "internal/fee/calculator.go",
			ChunkType:  "code_block",
			RawContent: "late fee settlement computation",
			Content:    "late fee settlement computation",
			StartLine:  1,
			EndLine:    3,
		},
	}
	server := newIndexedHelperServer(t, chunks)

	response := doRequest(t, http.MethodGet, server.URL+searchPath+"?q=settlement&limit=9999", "", bearerScheme+" "+testHelperToken, "")
	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for an oversized limit, got %d", response.StatusCode)
	}

	var payload searchResponse
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatalf("cannot decode the search response: %v", err)
	}
	if len(payload.Results) > maxSearchLimit {
		t.Fatalf("expected at most %d results, got %d", maxSearchLimit, len(payload.Results))
	}
	if len(payload.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(payload.Results))
	}
	if payload.Results[0].StartLine != 1 || payload.Results[0].EndLine != 3 {
		t.Fatalf("expected the line range to survive, got %d-%d", payload.Results[0].StartLine, payload.Results[0].EndLine)
	}
}

func TestSearchDropsUnsafeRepositoryFilters(t *testing.T) {
	chunks := []codeindexdomain.CodeChunk{
		{
			RepoName:   "settlement-service",
			FilePath:   "internal/fee/calculator.go",
			ChunkType:  "code_block",
			RawContent: "late fee settlement computation",
			Content:    "late fee settlement computation",
		},
	}
	server := newIndexedHelperServer(t, chunks)

	response := doRequest(
		t,
		http.MethodGet,
		server.URL+searchPath+"?q=settlement&repos=..%2Fetc,..%2F..%2Froot,settlement-service",
		"",
		bearerScheme+" "+testHelperToken,
		"",
	)
	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", response.StatusCode)
	}

	var payload searchResponse
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatalf("cannot decode the search response: %v", err)
	}
	if len(payload.Results) != 1 {
		t.Fatalf("expected the valid repository to still be searched, got %d results", len(payload.Results))
	}
}

func TestIndexBuildRequiresRepositoryRoot(t *testing.T) {
	server := newIndexedHelperServer(t, nil)

	response := doRequest(t, http.MethodPost, server.URL+indexBuildPath, `{"repos":[],"rebuild":false}`, bearerScheme+" "+testHelperToken, "")
	if response.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 without a root, got %d", response.StatusCode)
	}
}

func TestIndexBuildRejectsTooManyRepositories(t *testing.T) {
	server := newIndexedHelperServer(t, nil)

	repositories := make([]string, maxBuildRepos+1)
	for index := range repositories {
		repositories[index] = "repo"
	}
	body, err := json.Marshal(indexBuildRequest{Root: "/tmp", Repos: repositories})
	if err != nil {
		t.Fatalf("cannot build the request body: %v", err)
	}

	response := doRequest(t, http.MethodPost, server.URL+indexBuildPath, string(body), bearerScheme+" "+testHelperToken, "")
	if response.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for too many repositories, got %d", response.StatusCode)
	}
}
