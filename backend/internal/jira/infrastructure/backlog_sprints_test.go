package infrastructure

import (
	"testing"

	"lunar/backend/internal/jira/domain"
)

func TestFilterBacklogSprintsKeepsSprintsFromOtherBoards(t *testing.T) {
	sprintList := []domain.JiraSprint{
		{ID: 10103, Name: "Sprint Azzuri #5", State: "active", OriginBoardID: 3946},
		{ID: 10983, Name: "Sprint 44 - Fortune Squad", State: "active", OriginBoardID: 2646},
		{ID: 8827, Name: "Other Data - Fortune Squad", State: "future", OriginBoardID: 2646},
	}

	filteredSprints := filterBacklogSprints(sprintList)
	if len(filteredSprints) != len(sprintList) {
		t.Fatalf("expected %d sprints to survive filtering, got %d", len(sprintList), len(filteredSprints))
	}
	for index, sprint := range filteredSprints {
		if sprint.ID != sprintList[index].ID {
			t.Errorf("expected sprint %d at position %d, got %d", sprintList[index].ID, index, sprint.ID)
		}
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
