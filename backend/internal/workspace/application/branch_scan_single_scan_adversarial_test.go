package application

import (
	"context"
	"errors"
	"testing"
	"time"

	sharedErrors "lunar/backend/internal/shared/errors"
	"lunar/backend/internal/workspace/domain"
)

const adversarialMicroscopicCacheTTL = time.Nanosecond

func TestAdversarialSprintBranchesScansOnceEvenWithTheCacheDefeated(t *testing.T) {
	scanner := keyedBranchScanner()
	service := newIssueBranchService(scanner, adversarialMicroscopicCacheTTL)
	issueKeys := []string{"CRMMS-6397", "CRMMS-77745", "CRMMS-77740", "CRMMS-77911", "CRMMS-77741", "CRMMS-77742", "CRMMS-77743"}

	if _, err := service.SprintBranchesAtRoot(context.Background(), "/workspace", issueKeys); err != nil {
		t.Fatalf("sprint branches failed: %v", err)
	}
	if scanner.calls != 1 {
		t.Fatalf("a microscopic cache TTL cannot hide a per-key scan: seven keys must still scan once, got %d", scanner.calls)
	}

	time.Sleep(time.Millisecond)
	if _, err := service.SprintBranchesAtRoot(context.Background(), "/workspace", issueKeys); err != nil {
		t.Fatalf("second sprint branches failed: %v", err)
	}
	if scanner.calls != 2 {
		t.Fatalf("the recorder must count the second request too, got %d", scanner.calls)
	}
}

func TestAdversarialSprintBranchesScansOnceForRepeatedAndEmptyKeyLists(t *testing.T) {
	scanner := keyedBranchScanner()
	service := newIssueBranchService(scanner, adversarialMicroscopicCacheTTL)

	if _, err := service.SprintBranchesAtRoot(context.Background(), "/workspace", []string{"CRMMS-6397", "crmms-6397", "CRMMS-6397"}); err != nil {
		t.Fatalf("repeated keys failed: %v", err)
	}
	if scanner.calls != 1 {
		t.Fatalf("a repeated key must not add a scan, got %d", scanner.calls)
	}

	time.Sleep(time.Millisecond)
	branches, err := service.SprintBranchesAtRoot(context.Background(), "/workspace", []string{})
	if err != nil {
		t.Fatalf("an empty key list failed: %v", err)
	}
	if scanner.calls != 2 {
		t.Fatalf("an empty key list must still scan once, got %d", scanner.calls)
	}
	if branches.Matches == nil {
		t.Fatal("an empty key list must still encode an empty object, not null")
	}
	if len(branches.Matches) != 0 {
		t.Fatalf("expected no matches for an empty key list, got %+v", branches.Matches)
	}
}

func TestAdversarialSprintBranchesKeepsALongerDigitRunOutOfAShorterKey(t *testing.T) {
	scanner := &stubBranchScanner{
		branchesByRepository: map[string][]domain.Branch{
			"alpha": {
				{Name: "feature/CRMMS-6397-exact"},
				{Name: "feature/CRMMS-63970-one-extra-digit"},
				{Name: "feature/CRMMS-639712-six-digits"},
				{Name: "feature/CRMMS-6397123456-ten-digits"},
			},
		},
	}
	service := newIssueBranchService(scanner, adversarialMicroscopicCacheTTL)

	branches, err := service.SprintBranchesAtRoot(context.Background(), "/workspace", []string{"CRMMS-6397", "CRMMS-63970", "CRMMS-639712"})
	if err != nil {
		t.Fatalf("sprint branches failed: %v", err)
	}

	if len(branches.Matches["CRMMS-6397"]) != 1 || branches.Matches["CRMMS-6397"][0].Branch != "feature/CRMMS-6397-exact" {
		t.Fatalf("CRMMS-6397 must match only its exact branch, got %+v", branches.Matches["CRMMS-6397"])
	}
	for _, longerKey := range []string{"CRMMS-63970", "CRMMS-639712"} {
		if len(branches.Matches[longerKey]) != 1 {
			t.Fatalf("%s must match its own branch only, got %+v", longerKey, branches.Matches[longerKey])
		}
	}

	tenDigitKey := []string{"CRMMS-6397", "CRMMS-6397123456"}
	if _, err := service.SprintBranchesAtRoot(context.Background(), "/workspace", tenDigitKey); !errors.Is(err, sharedErrors.ErrBadRequest) {
		t.Fatalf("a seven digit issue number cannot be requested and must be refused, got %v", err)
	}
}
