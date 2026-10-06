package domain

import (
	"encoding/json"
	"testing"
)

const fortuneSprintValue = "com.atlassian.greenhopper.service.sprint.Sprint@1f2748[id=10983,rapidViewId=2646,state=ACTIVE,name=Sprint 44 - Fortune Squad,startDate=2026-09-18T16:30:00.000+07:00,endDate=2026-10-02T16:30:00.000+07:00,completeDate=<null>,activatedDate=2026-09-18T16:30:00.000+07:00,sequence=10820,goal=,synced=false,autoStartStop=false,incompleteIssuesDestinationId=<null>]"

const pangestuSprintValue = "com.atlassian.greenhopper.service.sprint.Sprint@2b71[id=10881,rapidViewId=3946,state=ACTIVE,name=Sprint #36 - Pangestu Squad,startDate=2026-08-01T09:00:00.000+07:00,endDate=2026-08-15T09:00:00.000+07:00,completeDate=<null>,activatedDate=2026-08-01T09:00:00.000+07:00,sequence=10700,goal=Ship the onboarding flow,synced=false]"

func TestParseSprintReference(t *testing.T) {
	t.Run("extracts fortune squad name and owning board", func(t *testing.T) {
		reference := ParseSprintReference(fortuneSprintValue)
		if reference.Name != "Sprint 44 - Fortune Squad" {
			t.Errorf("expected name 'Sprint 44 - Fortune Squad', got %q", reference.Name)
		}
		if reference.BoardID != 2646 {
			t.Errorf("expected board 2646, got %d", reference.BoardID)
		}
	})

	t.Run("extracts pangestu squad name and owning board", func(t *testing.T) {
		reference := ParseSprintReference(pangestuSprintValue)
		if reference.Name != "Sprint #36 - Pangestu Squad" {
			t.Errorf("expected name 'Sprint #36 - Pangestu Squad', got %q", reference.Name)
		}
		if reference.BoardID != 3946 {
			t.Errorf("expected board 3946, got %d", reference.BoardID)
		}
	})

	t.Run("keeps the name intact when the goal contains commas", func(t *testing.T) {
		rawValue := "com.atlassian.greenhopper.service.sprint.Sprint@3c19[id=10983,rapidViewId=2646,state=ACTIVE,name=Sprint 44 - Fortune Squad,startDate=2026-09-18T16:30:00.000+07:00,endDate=2026-10-02T16:30:00.000+07:00,completeDate=<null>,activatedDate=2026-09-18T16:30:00.000+07:00,sequence=10820,goal=Reduce latency, raise coverage,name=cleanup,synced=false]"
		reference := ParseSprintReference(rawValue)
		if reference.Name != "Sprint 44 - Fortune Squad" {
			t.Errorf("expected name 'Sprint 44 - Fortune Squad', got %q", reference.Name)
		}
		if reference.BoardID != 2646 {
			t.Errorf("expected board 2646, got %d", reference.BoardID)
		}
	})

	t.Run("tolerates empty null array and malformed values", func(t *testing.T) {
		rawValues := []string{"", "null", "[]", "not a sprint value", "Sprint 44 - Fortune Squad"}
		for _, rawValue := range rawValues {
			reference := ParseSprintReference(rawValue)
			if reference.Name != "" || reference.BoardID != 0 {
				t.Errorf("expected empty reference for %q, got %+v", rawValue, reference)
			}
		}
	})

	t.Run("leaves the board empty when rapidViewId is malformed", func(t *testing.T) {
		rawValue := "com.atlassian.greenhopper.service.sprint.Sprint@9[id=1,rapidViewId=unavailable,state=ACTIVE,name=Sprint 1 - Alpha Squad,startDate=2026-01-01T00:00:00.000+07:00,goal=]"
		reference := ParseSprintReference(rawValue)
		if reference.Name != "Sprint 1 - Alpha Squad" {
			t.Errorf("expected name 'Sprint 1 - Alpha Squad', got %q", reference.Name)
		}
		if reference.BoardID != 0 {
			t.Errorf("expected empty board, got %d", reference.BoardID)
		}
	})
}

func TestJiraSprintValuesUnmarshalJSON(t *testing.T) {
	tests := []struct {
		name          string
		payload       string
		expectedCount int
	}{
		{"missing field", "{\"summary\":\"task\"}", 0},
		{"null value", "{\"customfield_10104\":null}", 0},
		{"empty string", "{\"customfield_10104\":\"\"}", 0},
		{"empty array", "{\"customfield_10104\":[]}", 0},
		{"unsupported number", "{\"customfield_10104\":12345}", 0},
		{"single sprint string", "{\"customfield_10104\":\"" + fortuneSprintValue + "\"}", 1},
		{"array of sprint strings", "{\"customfield_10104\":[\"" + fortuneSprintValue + "\",\"" + pangestuSprintValue + "\"]}", 2},
		{"malformed sprint string", "{\"customfield_10104\":\"total garbage\"}", 1},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			var fields JiraIssueFields
			if err := json.Unmarshal([]byte(testCase.payload), &fields); err != nil {
				t.Fatalf("expected tolerant decoding, got %v", err)
			}
			if len(fields.SprintValues) != testCase.expectedCount {
				t.Fatalf("expected %d sprint values, got %d", testCase.expectedCount, len(fields.SprintValues))
			}
		})
	}
}
