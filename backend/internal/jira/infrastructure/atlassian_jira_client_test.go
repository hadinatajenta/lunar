package infrastructure

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	sharedErrors "lunar/backend/internal/shared/errors"
)

func TestAtlassianJiraClient_ValidationErrors(t *testing.T) {
	client := NewAtlassianJiraClient("https://jira.bri.co.id")
	ctx := context.Background()

	t.Run("SearchMyIssues empty PAT", func(t *testing.T) {
		_, err := client.SearchMyIssues(ctx, "")
		if !errors.Is(err, sharedErrors.ErrUnauthorized) {
			t.Errorf("expected ErrUnauthorized, got %v", err)
		}
	})

	t.Run("SearchMyIssues invalid PAT", func(t *testing.T) {
		_, err := client.SearchMyIssues(ctx, "invalid")
		if !errors.Is(err, sharedErrors.ErrUnauthorized) {
			t.Errorf("expected ErrUnauthorized, got %v", err)
		}
	})

	t.Run("GetBacklogSprints empty PAT", func(t *testing.T) {
		_, err := client.GetBacklogSprints(ctx, "")
		if !errors.Is(err, sharedErrors.ErrUnauthorized) {
			t.Errorf("expected ErrUnauthorized, got %v", err)
		}
	})

	t.Run("GetBacklogSprints invalid PAT", func(t *testing.T) {
		_, err := client.GetBacklogSprints(ctx, "invalid")
		if !errors.Is(err, sharedErrors.ErrUnauthorized) {
			t.Errorf("expected ErrUnauthorized, got %v", err)
		}
	})

	t.Run("GetIssueDetail empty issueKey", func(t *testing.T) {
		_, err := client.GetIssueDetail(ctx, "any-pat", "")
		if !errors.Is(err, sharedErrors.ErrBadRequest) {
			t.Errorf("expected ErrBadRequest, got %v", err)
		}
	})

	t.Run("GetIssueDetail empty PAT", func(t *testing.T) {
		_, err := client.GetIssueDetail(ctx, "", "CRMMS-77911")
		if !errors.Is(err, sharedErrors.ErrUnauthorized) {
			t.Errorf("expected ErrUnauthorized, got %v", err)
		}
	})

	t.Run("GetIssueDetail invalid PAT", func(t *testing.T) {
		_, err := client.GetIssueDetail(ctx, "invalid", "CRMMS-77911")
		if !errors.Is(err, sharedErrors.ErrUnauthorized) {
			t.Errorf("expected ErrUnauthorized, got %v", err)
		}
	})
}

func TestAtlassianJiraClient_MockSeedData(t *testing.T) {
	client := NewAtlassianJiraClient("mock")
	ctx := context.Background()

	t.Run("SearchMyIssues returns all assigned seed issues", func(t *testing.T) {
		issues, err := client.SearchMyIssues(ctx, "mock")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(issues) < 20 {
			t.Errorf("expected at least 20 seed issues, got %d", len(issues))
		}
	})

	t.Run("GetBacklogSprints returns seed sprints", func(t *testing.T) {
		sprints, err := client.GetBacklogSprints(ctx, "mock")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(sprints) != 3 {
			t.Fatalf("expected 3 seed sprints, got %d", len(sprints))
		}
		if sprints[0].Name != "Sprint 44 - Fortune Squad" {
			t.Errorf("expected Sprint 44, got %s", sprints[0].Name)
		}
		if sprints[1].Name != "Sprint 45 - Fortune Squad" {
			t.Errorf("expected Sprint 45, got %s", sprints[1].Name)
		}
		if sprints[2].Name != "Sprint Azzuri #5" {
			t.Errorf("expected Sprint Azzuri #5, got %s", sprints[2].Name)
		}
	})

	t.Run("GetIssueDetail finds authentic tickets", func(t *testing.T) {
		ticketKeys := []string{
			"CRMMS-77911", "CRMMS-77913", "CRMMS-77914", "CRMMS-77925", "CRMMS-77926",
			"BUG-201", "BUG-202", "BUG-203",
			"SUB-101", "SUB-102", "SUB-103",
			"UT-101", "UT-102", "UT-103", "UT-104",
			"QR-201", "QR-202", "QR-203",
			"SOP-301", "SOP-302", "SOP-303", "SOP-304", "SOP-305", "SOP-306",
		}

		for _, key := range ticketKeys {
			issue, err := client.GetIssueDetail(ctx, "mock", key)
			if err != nil {
				t.Fatalf("expected to find ticket %s, got error: %v", key, err)
			}
			if issue.Key != key {
				t.Errorf("expected key %s, got %s", key, issue.Key)
			}
			if issue.Kind == "" {
				t.Errorf("expected non-empty kind for %s", key)
			}
		}
	})

	t.Run("GetIssueDetail missing ticket returns ErrNotFound", func(t *testing.T) {
		_, err := client.GetIssueDetail(ctx, "mock", "NONEXISTENT-999")
		if !errors.Is(err, sharedErrors.ErrNotFound) {
			t.Errorf("expected ErrNotFound, got %v", err)
		}
	})
}

func TestAtlassianJiraClient_NetworkErrorAndFallback(t *testing.T) {
	ctx := context.Background()

	t.Run("Fallback enabled on network error", func(t *testing.T) {
		client := NewAtlassianJiraClient("http://127.0.0.1:59999")
		client.SetFallbackEnabled(true)

		issues, err := client.SearchMyIssues(ctx, "valid-pat")
		if err != nil {
			t.Fatalf("expected fallback to succeed, got %v", err)
		}
		if len(issues) == 0 {
			t.Fatal("expected fallback seed issues")
		}

		sprints, err := client.GetBacklogSprints(ctx, "valid-pat")
		if err != nil {
			t.Fatalf("expected fallback to succeed, got %v", err)
		}
		if len(sprints) == 0 {
			t.Fatal("expected fallback seed sprints")
		}

		issue, err := client.GetIssueDetail(ctx, "valid-pat", "CRMMS-77911")
		if err != nil {
			t.Fatalf("expected fallback to succeed, got %v", err)
		}
		if issue.Key != "CRMMS-77911" {
			t.Errorf("expected CRMMS-77911, got %s", issue.Key)
		}
	})

	t.Run("Fallback disabled on network error returns VPN message", func(t *testing.T) {
		client := NewAtlassianJiraClient("http://127.0.0.1:59999")
		client.SetFallbackEnabled(false)

		_, err := client.SearchMyIssues(ctx, "valid-pat")
		if err == nil {
			t.Fatal("expected network error, got nil")
		}
		if !errors.Is(err, sharedErrors.ErrUnauthorized) && err.Error() == "" {
			t.Fatal("expected descriptive error")
		}

		_, err = client.GetBacklogSprints(ctx, "valid-pat")
		if err == nil {
			t.Fatal("expected network error, got nil")
		}

		_, err = client.GetIssueDetail(ctx, "valid-pat", "CRMMS-77911")
		if err == nil {
			t.Fatal("expected network error, got nil")
		}
	})
}

func TestAtlassianJiraClient_LiveHTTPHandling(t *testing.T) {
	ctx := context.Background()

	t.Run("401 Unauthorized from live Jira", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
		}))
		defer server.Close()

		client := NewAtlassianJiraClient(server.URL)
		_, err := client.SearchMyIssues(ctx, "expired-pat")
		if !errors.Is(err, sharedErrors.ErrUnauthorized) {
			t.Errorf("expected ErrUnauthorized, got %v", err)
		}

		_, err = client.GetBacklogSprints(ctx, "expired-pat")
		if !errors.Is(err, sharedErrors.ErrUnauthorized) {
			t.Errorf("expected ErrUnauthorized, got %v", err)
		}

		_, err = client.GetIssueDetail(ctx, "expired-pat", "CRMMS-77911")
		if !errors.Is(err, sharedErrors.ErrUnauthorized) {
			t.Errorf("expected ErrUnauthorized, got %v", err)
		}
	})

	t.Run("404 Not Found from live Jira", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer server.Close()

		client := NewAtlassianJiraClient(server.URL)
		_, err := client.GetIssueDetail(ctx, "valid-pat", "CRMMS-UNKNOWN")
		if !errors.Is(err, sharedErrors.ErrNotFound) {
			t.Errorf("expected ErrNotFound, got %v", err)
		}
	})
}

func TestAtlassianJiraClient_FailFastTransport(t *testing.T) {
	ctx := context.Background()

	t.Run("Unreachable host falls back to seed sprints in under 6s", func(t *testing.T) {
		client := NewAtlassianJiraClient("http://10.255.255.1")
		client.SetFallbackEnabled(true)

		startedAt := time.Now()
		sprints, err := client.GetBacklogSprints(ctx, "valid-pat")
		elapsed := time.Since(startedAt)

		if err != nil {
			t.Fatalf("expected seed fallback, got %v", err)
		}
		if len(sprints) == 0 {
			t.Fatal("expected seed sprints")
		}
		if elapsed >= 6*time.Second {
			t.Errorf("expected fail-fast response under 6s, took %s", elapsed)
		}
	})

	t.Run("Unreachable host with fallback disabled returns error", func(t *testing.T) {
		client := NewAtlassianJiraClient("http://10.255.255.1")
		client.SetFallbackEnabled(false)

		sprints, err := client.GetBacklogSprints(ctx, "valid-pat")
		if err == nil {
			t.Fatal("expected network error, got nil")
		}
		if sprints != nil {
			t.Errorf("expected no sprints on error, got %d", len(sprints))
		}
	})
}

func TestAtlassianJiraClient_BacklogSprintsFetchConcurrently(t *testing.T) {
	ctx := context.Background()
	var stateMu sync.Mutex
	inFlight := 0
	peakInFlight := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/rest/agile/1.0/board/2646/sprint":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"values":[
				{"id":101,"name":"Sprint One","state":"active"},
				{"id":102,"name":"Sprint Two","state":"active"},
				{"id":103,"name":"Sprint Three","state":"future"},
				{"id":104,"name":"Sprint Four","state":"future"}
			]}`)
		case "/rest/agile/1.0/sprint/101/issue", "/rest/agile/1.0/sprint/102/issue",
			"/rest/agile/1.0/sprint/103/issue", "/rest/agile/1.0/sprint/104/issue":
			recordConcurrentArrival(&stateMu, &inFlight, &peakInFlight)
			waitForConcurrentPeer(&stateMu, &peakInFlight, 3*time.Second)
			releaseConcurrentArrival(&stateMu, &inFlight)
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"issues":[{"id":"1","key":"CRMMS-1","fields":{"summary":"concurrent task"}}]}`)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := NewAtlassianJiraClient(server.URL)
	sprints, err := client.GetBacklogSprints(ctx, "valid-pat")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	stateMu.Lock()
	observedPeak := peakInFlight
	stateMu.Unlock()
	if observedPeak < 2 {
		t.Errorf("expected at least 2 sprint issue requests in flight, peak was %d", observedPeak)
	}

	if len(sprints) != 4 {
		t.Fatalf("expected 4 sprints, got %d", len(sprints))
	}
	expectedNames := []string{"Sprint One", "Sprint Two", "Sprint Three", "Sprint Four"}
	for index, sprint := range sprints {
		if sprint.Name != expectedNames[index] {
			t.Errorf("expected sprint %s, got %s", expectedNames[index], sprint.Name)
		}
		if len(sprint.Issues) != 1 {
			t.Fatalf("expected 1 issue for %s, got %d", sprint.Name, len(sprint.Issues))
		}
		if sprint.Issues[0].SprintName != sprint.Name {
			t.Errorf("expected SprintName %s, got %s", sprint.Name, sprint.Issues[0].SprintName)
		}
	}
}

func recordConcurrentArrival(stateMu *sync.Mutex, inFlight *int, peakInFlight *int) {
	stateMu.Lock()
	defer stateMu.Unlock()
	*inFlight++
	if *inFlight > *peakInFlight {
		*peakInFlight = *inFlight
	}
}

func waitForConcurrentPeer(stateMu *sync.Mutex, peakInFlight *int, timeout time.Duration) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		stateMu.Lock()
		observedPeak := *peakInFlight
		stateMu.Unlock()
		if observedPeak >= 2 {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func releaseConcurrentArrival(stateMu *sync.Mutex, inFlight *int) {
	stateMu.Lock()
	defer stateMu.Unlock()
	*inFlight--
}

func TestAtlassianJiraClient_SprintListUnauthorizedPropagates(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	client := NewAtlassianJiraClient(server.URL)
	client.SetFallbackEnabled(true)

	_, err := client.GetBacklogSprints(context.Background(), "expired-pat")
	if !errors.Is(err, sharedErrors.ErrUnauthorized) {
		t.Errorf("expected ErrUnauthorized, got %v", err)
	}
}
