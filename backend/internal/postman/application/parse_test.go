package application

import (
	"errors"
	"strings"
	"testing"

	"lunar/backend/internal/postman/domain"
)

func TestStripCodeFenceUnwrapsFencedJSON(t *testing.T) {
	cases := []struct {
		name     string
		raw      string
		expected string
	}{
		{name: "json fence", raw: "```json\n{\"a\":1}\n```", expected: "{\"a\":1}"},
		{name: "language fence", raw: "```JSON\n{\"a\":1}\n```", expected: "{\"a\":1}"},
		{name: "plain fence", raw: "```\n{\"a\":1}\n```", expected: "{\"a\":1}"},
		{name: "no fence", raw: "  {\"a\":1}  ", expected: "{\"a\":1}"},
		{name: "prefix fence only", raw: "```json\n{\"a\":1}", expected: "{\"a\":1}"},
		{name: "empty", raw: "   ", expected: ""},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			result := StripCodeFence(testCase.raw)
			if result != testCase.expected {
				t.Fatalf("expected %q, received %q", testCase.expected, result)
			}
		})
	}
}

func TestParseCollectionAcceptsAFencedCollection(t *testing.T) {
	raw := "```json\n" + validCollectionFixture + "\n```"

	collection, err := ParseCollection(raw, "atlas")
	if err != nil {
		t.Fatalf("a fenced collection must be accepted, received %v", err)
	}
	if collection.Info.Name != "atlas API" || collection.Info.Schema != domain.CollectionSchemaURL {
		t.Fatalf("expected a normalised collection, received %q and %q", collection.Info.Name, collection.Info.Schema)
	}
	if CountRequests(collection) != 1 {
		t.Fatalf("expected 1 request, received %d", CountRequests(collection))
	}
}

func TestParseCollectionRefusesAResponseThatIsNotJSON(t *testing.T) {
	_, err := ParseCollection("I could not find any routes in this repository.", "atlas")
	if !errors.Is(err, domain.ErrCollectionInvalid) {
		t.Fatalf("expected ErrCollectionInvalid, received %v", err)
	}
	if !strings.Contains(err.Error(), "not a Postman collection") {
		t.Fatalf("expected a decode reason, received %q", err.Error())
	}
}

func TestParseCollectionRefusesAHollowCollection(t *testing.T) {
	_, err := ParseCollection("{\"info\":{\"name\":\"atlas API\"},\"item\":[]}", "atlas")
	if !errors.Is(err, domain.ErrCollectionInvalid) || !strings.Contains(err.Error(), "no endpoints found") {
		t.Fatalf("expected a no-endpoints refusal, received %v", err)
	}
}

func TestParseCollectionRefusesAnEmptyResponse(t *testing.T) {
	_, err := ParseCollection("   ", "atlas")
	if !errors.Is(err, domain.ErrCollectionInvalid) || !strings.Contains(err.Error(), "empty response") {
		t.Fatalf("expected an empty-response refusal, received %v", err)
	}
}
