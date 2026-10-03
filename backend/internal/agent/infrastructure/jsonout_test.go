package infrastructure

import (
	"errors"
	"strings"
	"testing"
)

type parsedReview struct {
	Summary  string   `json:"summary"`
	Findings []string `json:"findings"`
}

func TestCleanJSONResponse(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want string
	}{
		{name: "plain json", raw: `{"a":1}`, want: `{"a":1}`},
		{name: "json fence", raw: "```json\n{\"a\":1}\n```", want: `{"a":1}`},
		{name: "uppercase json fence", raw: "```JSON\n{\"a\":1}\n```", want: `{"a":1}`},
		{name: "plain fence", raw: "```\n{\"a\":1}\n```", want: `{"a":1}`},
		{name: "surrounding whitespace", raw: "\n\n  {\"a\":1}  \n", want: `{"a":1}`},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			if got := cleanJSONResponse(testCase.raw); got != testCase.want {
				t.Errorf("cleanJSONResponse(%q) = %q, want %q", testCase.raw, got, testCase.want)
			}
		})
	}
}

func TestParseJSON_DecodesFencedAndPlainPayloads(t *testing.T) {
	plain, err := ParseJSON[parsedReview](`{"summary":"ok","findings":["a","b"]}`, nil, nil)
	if err != nil {
		t.Fatalf("plain payload: unexpected error: %v", err)
	}
	if plain.Summary != "ok" || len(plain.Findings) != 2 {
		t.Errorf("plain payload = %+v, want the decoded review", plain)
	}

	fenced, err := ParseJSON[parsedReview]("```json\n{\"summary\":\"fenced\"}\n```", nil, nil)
	if err != nil {
		t.Fatalf("fenced payload: unexpected error: %v", err)
	}
	if fenced.Summary != "fenced" {
		t.Errorf("fenced payload = %+v, want the decoded summary", fenced)
	}
}

func TestParseJSON_RejectsEmptyPayload(t *testing.T) {
	_, err := ParseJSON[parsedReview]("   ", nil, nil)
	if err == nil {
		t.Fatal("expected an error for an empty payload")
	}
	if !strings.Contains(err.Error(), "empty response") {
		t.Errorf("error = %v, want the empty response message", err)
	}
}

func TestParseJSON_ReportsSyntaxErrors(t *testing.T) {
	_, err := ParseJSON[parsedReview](`{"summary":`, nil, nil)
	if err == nil {
		t.Fatal("expected an error for malformed JSON")
	}
	if !strings.Contains(err.Error(), "parse json") {
		t.Errorf("error = %v, want the parse context", err)
	}
}

func TestParseJSON_AppliesRepairOnlyWhenDecodingFails(t *testing.T) {
	repairCalls := 0
	repair := func(raw string) string {
		repairCalls++
		return strings.Replace(raw, `"summary":ok`, `"summary":"ok"`, 1)
	}

	repaired, err := ParseJSON[parsedReview](`{"summary":ok}`, repair, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repaired.Summary != "ok" {
		t.Errorf("summary = %q, want the repaired value", repaired.Summary)
	}
	if repairCalls != 1 {
		t.Errorf("repair calls = %d, want 1", repairCalls)
	}

	repairCalls = 0
	if _, err := ParseJSON[parsedReview](`{"summary":"valid"}`, repair, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repairCalls != 0 {
		t.Errorf("repair calls = %d, want 0 for a valid payload", repairCalls)
	}
}

func TestParseJSON_RepairThatDoesNotChangeInputStillFails(t *testing.T) {
	_, err := ParseJSON[parsedReview](`{"summary":`, func(raw string) string { return raw }, nil)
	if err == nil {
		t.Fatal("expected an error when the repair leaves the payload broken")
	}
}

func TestParseJSON_ValidationError(t *testing.T) {
	validationErr := errors.New("summary is required")
	_, err := ParseJSON[parsedReview](`{"summary":""}`, nil, func(review parsedReview) error {
		if strings.TrimSpace(review.Summary) == "" {
			return validationErr
		}
		return nil
	})
	if !errors.Is(err, validationErr) {
		t.Fatalf("error = %v, want it to wrap the validation error", err)
	}
	if !strings.Contains(err.Error(), "validation failed") {
		t.Errorf("error = %v, want the validation context", err)
	}
}

func TestParseJSON_ValidationReceivesDecodedValue(t *testing.T) {
	decoded, err := ParseJSON[parsedReview](`{"summary":"checked","findings":["x"]}`, nil, func(review parsedReview) error {
		if len(review.Findings) != 1 || review.Findings[0] != "x" {
			t.Errorf("validation received %+v, want the decoded findings", review)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if decoded.Summary != "checked" {
		t.Errorf("summary = %q, want the decoded value", decoded.Summary)
	}
}

func TestParseJSON_MapPayload(t *testing.T) {
	collection, err := ParseJSON[map[string]interface{}]("```json\n{\"info\":{\"name\":\"aurora\"},\"item\":[]}\n```", nil, func(value map[string]interface{}) error {
		if _, exists := value["info"]; !exists {
			return errors.New("info is required")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if collection["info"] == nil {
		t.Errorf("collection = %+v, want the info object", collection)
	}
}
