package infrastructure

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAdversarialSearchBranchesReturnsWhateverTheServerMatched(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeBranchPage(t, w, []map[string]string{
			{"displayId": "feature/CRMMS-63970-other-issue"},
			{"displayId": "feature/CRMMS-639712-six-digits"},
			{"displayId": "feature/CRMMS-6397-exact"},
		})
	}))
	defer server.Close()

	client := NewAtlassianBitbucketClient(server.URL)
	branches, err := client.SearchBranches(context.Background(), "pat-token", "NME/alpha", "CRMMS-6397")
	if err != nil {
		t.Fatalf("cannot search branches: %v", err)
	}

	if len(branches) != 3 {
		t.Fatalf("the client must not filter what Bitbucket returned, got %+v", branches)
	}
	if branches[0] != "feature/CRMMS-63970-other-issue" {
		t.Fatalf("expected the longer digit run to survive the client unchanged, got %+v", branches)
	}
	if branches[1] != "feature/CRMMS-639712-six-digits" {
		t.Fatalf("expected the six digit run to survive the client unchanged, got %+v", branches)
	}
}
