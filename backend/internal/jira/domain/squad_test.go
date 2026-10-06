package domain

import "testing"

func TestExtractSquadName(t *testing.T) {
	tests := []struct {
		sprintName    string
		expectedSquad string
	}{
		{"Sprint 44 - Fortune Squad", "Fortune Squad"},
		{"Sprint #36 - Pangestu Squad", "Pangestu Squad"},
		{"Sprint Azzuri #5", "Azzuri"},
	}

	for _, testCase := range tests {
		t.Run(testCase.sprintName, func(t *testing.T) {
			if squad := ExtractSquadName(testCase.sprintName); squad != testCase.expectedSquad {
				t.Errorf("expected squad %q, got %q", testCase.expectedSquad, squad)
			}
		})
	}
}
