package infrastructure

import (
	"testing"

	"lunar/backend/internal/jira/domain"
)

func TestFilterBacklogSprintsDropsSprintsFromAnotherBoard(t *testing.T) {
	sprintList := []domain.JiraSprint{
		{ID: 10103, Name: "Sprint Azzuri #5", State: "active", OriginBoardID: 3946},
		{ID: 10983, Name: "Sprint 44 - Fortune Squad", State: "active", OriginBoardID: 2646},
		{ID: 10984, Name: "Sprint 45 - Fortune Squad", State: "future", OriginBoardID: 2646},
	}

	filteredSprints := filterBacklogSprints(sprintList, 2646)
	if len(filteredSprints) != 2 {
		t.Fatalf("expected 2 sprints for board 2646, got %d: %+v", len(filteredSprints), filteredSprints)
	}
	for _, sprint := range filteredSprints {
		if sprint.OriginBoardID != 2646 {
			t.Errorf("expected only board 2646 sprints, got %+v", sprint)
		}
	}
}

func TestFilterBacklogSprintsKeepsASprintWhoseBoardIsUnknown(t *testing.T) {
	sprintList := []domain.JiraSprint{
		{ID: 10983, Name: "Sprint 44 - Fortune Squad", State: "active", OriginBoardID: 2646},
		{ID: 10990, Name: "Sprint Without Origin", State: "active"},
	}

	filteredSprints := filterBacklogSprints(sprintList, 2646)
	if len(filteredSprints) != 2 {
		t.Fatalf("expected an unknown origin board to survive, got %d: %+v", len(filteredSprints), filteredSprints)
	}
}

func TestResolveBacklogBoardID(t *testing.T) {
	tests := []struct {
		name     string
		boardID  int
		expected int
	}{
		{"detected board is used", 3946, 3946},
		{"zero board falls back to default", 0, 2646},
		{"negative board falls back to default", -1, 2646},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			if resolved := resolveBacklogBoardID(testCase.boardID); resolved != testCase.expected {
				t.Errorf("expected board %d, got %d", testCase.expected, resolved)
			}
		})
	}
}
