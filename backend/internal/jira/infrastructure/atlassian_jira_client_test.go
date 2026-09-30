package infrastructure

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"lunar/backend/internal/jira/domain"
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
	pathCounter := &sprintPathCounter{counts: make(map[string]int)}
	server, peakInFlight := startConcurrentSprintServer(t, pathCounter)

	client := NewAtlassianJiraClient(server.URL)
	sprints, err := client.GetBacklogSprints(ctx, "valid-pat")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if observedPeak := peakInFlight(); observedPeak < 2 {
		t.Errorf("expected at least 2 sprint issue requests in flight, peak was %d", observedPeak)
	}

	if len(sprints) != 2 {
		t.Fatalf("expected 2 sprints, got %d", len(sprints))
	}
	expectedNames := []string{"Sprint One - Squad Q", "Sprint Two - Squad Q"}
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
	assertSprintIDsAbsent(t, sprints, []int{103, 104})
	assertIssueFetchCount(t, pathCounter, 103, 0)
	assertIssueFetchCount(t, pathCounter, 104, 0)
}

func startConcurrentSprintServer(t *testing.T, pathCounter *sprintPathCounter) (*httptest.Server, func() int) {
	t.Helper()
	var stateMu sync.Mutex
	inFlight := 0
	peakInFlight := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		pathCounter.record(r.URL.Path)
		switch r.URL.Path {
		case "/rest/agile/1.0/board/2646/sprint":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"values":[
				{"id":101,"name":"Sprint One - Squad Q","state":"active"},
				{"id":102,"name":"Sprint Two - Squad Q","state":"active"},
				{"id":103,"name":"Sprint Three - Squad Q","state":"future"},
				{"id":104,"name":"Sprint Four - Squad Q","state":"future"}
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
	t.Cleanup(server.Close)

	readPeak := func() int {
		stateMu.Lock()
		defer stateMu.Unlock()
		return peakInFlight
	}
	return server, readPeak
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

type sprintPathCounter struct {
	mu     sync.Mutex
	counts map[string]int
}

func (c *sprintPathCounter) record(requestPath string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.counts[requestPath]++
}

func (c *sprintPathCounter) countFor(requestPath string) int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.counts[requestPath]
}

func newSprintFixtureServer(t *testing.T, sprintListJSON string) (*httptest.Server, *sprintPathCounter) {
	t.Helper()
	pathCounter := &sprintPathCounter{counts: make(map[string]int)}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		pathCounter.record(r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == "/rest/agile/1.0/board/2646/sprint":
			fmt.Fprint(w, sprintListJSON)
		case strings.HasPrefix(r.URL.Path, "/rest/agile/1.0/sprint/") && strings.HasSuffix(r.URL.Path, "/issue"):
			fmt.Fprint(w, `{"issues":[{"id":"1","key":"CRMMS-1","fields":{"summary":"sample task"}}]}`)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(server.Close)
	return server, pathCounter
}

func assertSprintIDsAbsent(t *testing.T, sprints []domain.JiraSprint, absentIDs []int) {
	t.Helper()
	for _, sprint := range sprints {
		for _, absentID := range absentIDs {
			if sprint.ID == absentID {
				t.Errorf("expected sprint %d to be absent from response", absentID)
			}
		}
	}
}

func assertIssueFetchCount(t *testing.T, pathCounter *sprintPathCounter, sprintID int, expectedHits int) {
	t.Helper()
	requestPath := fmt.Sprintf("/rest/agile/1.0/sprint/%d/issue", sprintID)
	if hits := pathCounter.countFor(requestPath); hits != expectedHits {
		t.Errorf("expected %d issue fetches for sprint %d, got %d", expectedHits, sprintID, hits)
	}
}

func TestAtlassianJiraClient_BacklogSprintsSkipsFutureFetchWhenActiveExists(t *testing.T) {
	server, pathCounter := newSprintFixtureServer(t, `{"values":[
		{"id":301,"name":"Sprint 10 - Zephyr","state":"active"},
		{"id":302,"name":"Sprint 11 - Zephyr","state":"future"},
		{"id":303,"name":"Sprint 12 - Zephyr","state":"future"}
	]}`)

	client := NewAtlassianJiraClient(server.URL)
	sprints, err := client.GetBacklogSprints(context.Background(), "valid-pat")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(sprints) != 1 {
		t.Fatalf("expected 1 sprint, got %d", len(sprints))
	}
	if sprints[0].ID != 301 {
		t.Errorf("expected sprint 301, got %d", sprints[0].ID)
	}
	assertSprintIDsAbsent(t, sprints, []int{302, 303})
	assertIssueFetchCount(t, pathCounter, 301, 1)
	assertIssueFetchCount(t, pathCounter, 302, 0)
	assertIssueFetchCount(t, pathCounter, 303, 0)
}

func TestAtlassianJiraClient_BacklogSprintsFetchesOnlyFirstFutureWithoutActive(t *testing.T) {
	server, pathCounter := newSprintFixtureServer(t, `{"values":[
		{"id":311,"name":"Sprint 20 - Zephyr","state":"future"},
		{"id":312,"name":"Sprint 21 - Zephyr","state":"future"},
		{"id":313,"name":"Sprint 22 - Zephyr","state":"future"}
	]}`)

	client := NewAtlassianJiraClient(server.URL)
	sprints, err := client.GetBacklogSprints(context.Background(), "valid-pat")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(sprints) != 1 {
		t.Fatalf("expected 1 sprint, got %d", len(sprints))
	}
	if sprints[0].ID != 311 {
		t.Errorf("expected first future sprint 311, got %d", sprints[0].ID)
	}
	assertSprintIDsAbsent(t, sprints, []int{312, 313})
	assertIssueFetchCount(t, pathCounter, 311, 1)
	assertIssueFetchCount(t, pathCounter, 312, 0)
	assertIssueFetchCount(t, pathCounter, 313, 0)
}

func TestAtlassianJiraClient_BacklogSprintsKeepPrioritySprintPerSquad(t *testing.T) {
	server, pathCounter := newSprintFixtureServer(t, `{"values":[
		{"id":401,"name":"Sprint 1 - Alpha","state":"active"},
		{"id":402,"name":"Sprint 2 - Alpha","state":"future"},
		{"id":403,"name":"Sprint 3 - Beta","state":"future"},
		{"id":404,"name":"Sprint 4 - Beta","state":"future"},
		{"id":405,"name":"Sprint 5 - Gamma","state":"active"}
	]}`)

	client := NewAtlassianJiraClient(server.URL)
	sprints, err := client.GetBacklogSprints(context.Background(), "valid-pat")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedIDs := []int{401, 403, 405}
	if len(sprints) != len(expectedIDs) {
		t.Fatalf("expected %d sprints, got %d", len(expectedIDs), len(sprints))
	}
	for index, sprint := range sprints {
		if sprint.ID != expectedIDs[index] {
			t.Errorf("expected sprint %d at position %d, got %d", expectedIDs[index], index, sprint.ID)
		}
	}
	assertSprintIDsAbsent(t, sprints, []int{402, 404})

	expectedSquads := []string{"Alpha", "Beta", "Gamma"}
	for index, sprint := range sprints {
		squad := domain.ExtractSquadName(sprint.Name)
		if squad != expectedSquads[index] {
			t.Errorf("expected squad %s at position %d, got %s", expectedSquads[index], index, squad)
		}
	}

	assertIssueFetchCount(t, pathCounter, 401, 1)
	assertIssueFetchCount(t, pathCounter, 403, 1)
	assertIssueFetchCount(t, pathCounter, 405, 1)
	assertIssueFetchCount(t, pathCounter, 402, 0)
	assertIssueFetchCount(t, pathCounter, 404, 0)
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
