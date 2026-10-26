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

const compareDiffFixture = `{
  "diffs": [
    {
      "source": {"toString": "internal/repo/dao.go"},
      "destination": {"toString": "internal/repo/dao.go"},
      "hunks": [
        {
          "sourceLine": 10,
          "sourceSpan": 2,
          "destinationLine": 10,
          "destinationSpan": 2,
          "segments": [
            {"type": "CONTEXT", "lines": [{"source": 10, "destination": 10, "line": "package repo"}]},
            {"type": "REMOVED", "lines": [{"source": 11, "destination": 0, "line": "old := 1"}]},
            {"type": "ADDED", "lines": [{"source": 0, "destination": 11, "line": "rows := db.Raw(\"select 1\")"}]}
          ]
        }
      ]
    },
    {
      "source": null,
      "destination": {"toString": "internal/repo/new_query.sql"},
      "hunks": [
        {
          "sourceLine": 0,
          "sourceSpan": 0,
          "destinationLine": 1,
          "destinationSpan": 1,
          "segments": [
            {"type": "ADDED", "lines": [{"source": 0, "destination": 1, "line": "select id from user_data"}]}
          ]
        }
      ]
    }
  ]
}`

const expectedCompareUnifiedDiff = `diff --git a/internal/repo/dao.go b/internal/repo/dao.go
--- a/internal/repo/dao.go
+++ b/internal/repo/dao.go
@@ -10,2 +10,2 @@
 package repo
-old := 1
+rows := db.Raw("select 1")
diff --git a/internal/repo/new_query.sql b/internal/repo/new_query.sql
--- /dev/null
+++ b/internal/repo/new_query.sql
@@ -0,0 +1,1 @@
+select id from user_data
`

func TestGetCompareDiffRequestsTheEndpointWithTheSourceAsFromAndTheTargetAsTo(t *testing.T) {
	requestedPath := ""
	requestedFrom := ""
	requestedTo := ""
	authorizationHeader := ""

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestedPath = r.URL.Path
		requestedFrom = r.URL.Query().Get("from")
		requestedTo = r.URL.Query().Get("to")
		authorizationHeader = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(compareDiffFixture))
	}))
	defer server.Close()

	client := NewAtlassianBitbucketClient(server.URL)
	diff, err := client.GetCompareDiff(context.Background(), "pat-token", "NME", "micro-payment", "feat/query-review", "development")
	if err != nil {
		t.Fatalf("cannot fetch the compare diff: %v", err)
	}

	if requestedPath != "/rest/api/1.0/projects/NME/repos/micro-payment/compare/diff" {
		t.Fatalf("unexpected request path %q", requestedPath)
	}
	if requestedFrom != "feat/query-review" {
		t.Fatalf("expected from=feat/query-review, got %q", requestedFrom)
	}
	if requestedTo != "development" {
		t.Fatalf("expected to=development, got %q", requestedTo)
	}
	if authorizationHeader != "Bearer pat-token" {
		t.Fatalf("unexpected authorization header %q", authorizationHeader)
	}
	if diff != expectedCompareUnifiedDiff {
		t.Fatalf("unexpected unified diff\nwant:\n%s\ngot:\n%s", expectedCompareUnifiedDiff, diff)
	}
}

func TestGetCompareDiffEscapesTheRepositoryAndBranchNames(t *testing.T) {
	requestedRawQuery := ""

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestedRawQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"diffs":[]}`))
	}))
	defer server.Close()

	client := NewAtlassianBitbucketClient(server.URL)
	if _, err := client.GetCompareDiff(context.Background(), "pat-token", "NME", "micro-payment", "feat/x y", "release/1.0"); err != nil {
		t.Fatalf("cannot fetch the compare diff: %v", err)
	}

	if !strings.Contains(requestedRawQuery, "from=feat%2Fx+y") {
		t.Fatalf("expected the source branch to be escaped, got %q", requestedRawQuery)
	}
	if !strings.Contains(requestedRawQuery, "to=release%2F1.0") {
		t.Fatalf("expected the target branch to be escaped, got %q", requestedRawQuery)
	}
}

func TestGetCompareDiffRequiresAPAT(t *testing.T) {
	client := NewAtlassianBitbucketClient("https://bitbucket.example")

	_, err := client.GetCompareDiff(context.Background(), "  ", "NME", "micro-payment", "feat/x", "development")
	if !errors.Is(err, sharedErrors.ErrUnauthorized) {
		t.Fatalf("expected an unauthorized error, got %v", err)
	}
}

func TestGetCompareDiffRequiresEveryBranchArgument(t *testing.T) {
	client := NewAtlassianBitbucketClient("https://bitbucket.example")

	_, err := client.GetCompareDiff(context.Background(), "pat-token", "NME", "micro-payment", "  ", "development")
	if !errors.Is(err, sharedErrors.ErrBadRequest) {
		t.Fatalf("expected a bad request error, got %v", err)
	}
}

func TestGetCompareDiffReportsAnUnexpectedStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	client := NewAtlassianBitbucketClient(server.URL)
	_, err := client.GetCompareDiff(context.Background(), "pat-token", "NME", "micro-payment", "feat/x", "development")
	if err == nil {
		t.Fatal("expected an error for a non 200 status")
	}
	if !strings.Contains(err.Error(), "503") {
		t.Fatalf("expected the status to be reported, got %q", err.Error())
	}
}

func TestGetCompareDiffReportsAnExpiredPAT(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	client := NewAtlassianBitbucketClient(server.URL)
	_, err := client.GetCompareDiff(context.Background(), "pat-token", "NME", "micro-payment", "feat/x", "development")
	if !errors.Is(err, sharedErrors.ErrUnauthorized) {
		t.Fatalf("expected an unauthorized error, got %v", err)
	}
}

func TestGetCompareDiffReportsAMissingCompare(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := NewAtlassianBitbucketClient(server.URL)
	_, err := client.GetCompareDiff(context.Background(), "pat-token", "NME", "micro-payment", "feat/x", "development")
	if !errors.Is(err, sharedErrors.ErrNotFound) {
		t.Fatalf("expected a not found error, got %v", err)
	}
}

func TestGetCompareDiffReportsAnUndecodableBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("this is not json"))
	}))
	defer server.Close()

	client := NewAtlassianBitbucketClient(server.URL)
	_, err := client.GetCompareDiff(context.Background(), "pat-token", "NME", "micro-payment", "feat/x", "development")
	if err == nil {
		t.Fatal("expected an error for an undecodable body")
	}
	if !strings.Contains(err.Error(), "decode") {
		t.Fatalf("expected a decode error, got %q", err.Error())
	}
}

func TestRenderCompareDiffSkipsDiffsWithoutHunks(t *testing.T) {
	rendered := renderCompareDiff(rawCompareDiffResponse{Diffs: []rawCompareDiff{
		{
			Source:      &rawCompareDiffPath{ToString: "internal/repo/dao.go"},
			Destination: &rawCompareDiffPath{ToString: "internal/repo/dao.go"},
			Hunks:       []rawCompareDiffHunk{},
		},
	}})

	if rendered != "" {
		t.Fatalf("expected no unified diff for a hunkless change, got %q", rendered)
	}
}
