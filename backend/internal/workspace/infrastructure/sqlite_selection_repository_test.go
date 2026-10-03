package infrastructure

import (
	"context"
	"testing"
)

func TestSQLiteSelectionRepository_ListSelectedIsEmptyForUnknownUser(t *testing.T) {
	db := openWorkspaceTestDB(t)
	seedWorkspaceUsers(t, db, testUserA)

	repository := NewSQLiteSelectionRepository(db)
	selectedRepoNames, err := repository.ListSelected(context.Background(), testUserA)
	if err != nil {
		t.Fatalf("cannot list selections: %v", err)
	}

	if len(selectedRepoNames) != 0 {
		t.Fatalf("expected no selections, got %v", selectedRepoNames)
	}
}

func TestSQLiteSelectionRepository_ReplaceSelectionsRemovesEntriesNoLongerPresent(t *testing.T) {
	db := openWorkspaceTestDB(t)
	seedWorkspaceUsers(t, db, testUserA)

	repository := NewSQLiteSelectionRepository(db)
	ctx := context.Background()

	if err := repository.ReplaceSelections(ctx, testUserA, []string{"everest", "aurora", "way4"}); err != nil {
		t.Fatalf("cannot store initial selections: %v", err)
	}

	selectedRepoNames, err := repository.ListSelected(ctx, testUserA)
	if err != nil {
		t.Fatalf("cannot list selections: %v", err)
	}
	if len(selectedRepoNames) != 3 {
		t.Fatalf("expected 3 selections, got %v", selectedRepoNames)
	}

	if err := repository.ReplaceSelections(ctx, testUserA, []string{"everest"}); err != nil {
		t.Fatalf("cannot replace selections: %v", err)
	}

	selectedRepoNames, err = repository.ListSelected(ctx, testUserA)
	if err != nil {
		t.Fatalf("cannot list selections after replace: %v", err)
	}
	if len(selectedRepoNames) != 1 || selectedRepoNames[0] != "everest" {
		t.Fatalf("expected only everest to remain, got %v", selectedRepoNames)
	}

	if err := repository.ReplaceSelections(ctx, testUserA, nil); err != nil {
		t.Fatalf("cannot clear selections: %v", err)
	}

	selectedRepoNames, err = repository.ListSelected(ctx, testUserA)
	if err != nil {
		t.Fatalf("cannot list selections after clearing: %v", err)
	}
	if len(selectedRepoNames) != 0 {
		t.Fatalf("expected clearing to remove every selection, got %v", selectedRepoNames)
	}
}

func TestSQLiteSelectionRepository_ReplaceSelectionsDoesNotAffectAnotherUser(t *testing.T) {
	db := openWorkspaceTestDB(t)
	seedWorkspaceUsers(t, db, testUserA, testUserB)

	repository := NewSQLiteSelectionRepository(db)
	ctx := context.Background()

	if err := repository.ReplaceSelections(ctx, testUserA, []string{"everest", "aurora"}); err != nil {
		t.Fatalf("cannot store selections for user A: %v", err)
	}
	if err := repository.ReplaceSelections(ctx, testUserB, []string{"way4"}); err != nil {
		t.Fatalf("cannot store selections for user B: %v", err)
	}

	if err := repository.ReplaceSelections(ctx, testUserB, nil); err != nil {
		t.Fatalf("cannot clear selections for user B: %v", err)
	}

	selectionsForA, err := repository.ListSelected(ctx, testUserA)
	if err != nil {
		t.Fatalf("cannot list selections for user A: %v", err)
	}
	if len(selectionsForA) != 2 || selectionsForA[0] != "aurora" || selectionsForA[1] != "everest" {
		t.Fatalf("expected user A selections to survive, got %v", selectionsForA)
	}

	selectionsForB, err := repository.ListSelected(ctx, testUserB)
	if err != nil {
		t.Fatalf("cannot list selections for user B: %v", err)
	}
	if len(selectionsForB) != 0 {
		t.Fatalf("expected user B to be empty, got %v", selectionsForB)
	}
}
