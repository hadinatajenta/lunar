package infrastructure

import (
	"context"
	"errors"
	"strings"
	"testing"

	sharedErrors "lunar/backend/internal/shared/errors"
	"lunar/backend/internal/utdocs/domain"
)

func documentRequestFixture() domain.DocumentRequest {
	return domain.DocumentRequest{
		Model: "DeepSeek-V4 Flash",
		Repositories: []domain.DocumentRepository{
			{
				Repository: "micro-payment",
				Branch:     "feat/CRMMS-123",
				BaseRef:    "development",
				Run:        domain.RunResult{Status: domain.RunStatusPassed, Total: 1, Passed: 1},
				Tests:      []domain.TestResult{{Name: "TestFind", Status: domain.TestStatusPassed, DurationMs: 120}},
			},
		},
	}
}

func validGeneratedJSON() string {
	return `{"title":"Unit tests for Payment API","summary":"Mock based coverage.","caption":"UT run for micro-payment","rows":[{"testFunctionName":"TestFind","function":"Find payment","scenario":"Mencari data pembayaran","expectedResult":"Data pembayaran ditemukan","inputData":"ID pembayaran"}]}`
}

func TestGenerateReturnsTheParsedDocument(t *testing.T) {
	calls := 0
	client := &LLMClient{completion: func(_ context.Context, _ string, _ string, systemPrompt string, userPrompt string) (string, error) {
		calls++
		if !strings.Contains(userPrompt, "<<<UT_RUN_DATA") {
			t.Fatalf("expected the delimited payload in the user prompt, got %q", userPrompt)
		}
		if systemPrompt != utDocsSystemPrompt {
			t.Fatal("expected the utdocs system prompt")
		}
		return validGeneratedJSON(), nil
	}}

	generated, err := client.Generate(context.Background(), "user-1", "DeepSeek-V4 Flash", "Payment API", documentRequestFixture())
	if err != nil {
		t.Fatalf("expected the document to be generated, got %v", err)
	}
	if calls != 1 {
		t.Fatalf("expected a single model call, got %d", calls)
	}
	if generated.Title != "Unit tests for Payment API" || generated.Caption != "UT run for micro-payment" {
		t.Fatalf("expected the parsed content, got %+v", generated)
	}
	if len(generated.Rows) != 1 || generated.Rows[0].TestFunctionName != "TestFind" {
		t.Fatalf("expected the parsed rows, got %+v", generated.Rows)
	}
}

func TestGenerateRepairsInvalidJSONOnce(t *testing.T) {
	calls := 0
	client := &LLMClient{completion: func(_ context.Context, _ string, _ string, _ string, userPrompt string) (string, error) {
		calls++
		if calls == 1 {
			return "this is not JSON", nil
		}
		if !strings.Contains(userPrompt, "<<<REJECTED_ANSWER") {
			t.Fatalf("expected the rejected answer in the repair prompt, got %q", userPrompt)
		}
		return "```json\n" + validGeneratedJSON() + "\n```", nil
	}}

	generated, err := client.Generate(context.Background(), "user-1", "DeepSeek-V4 Flash", "Payment API", documentRequestFixture())
	if err != nil {
		t.Fatalf("expected the repair to succeed, got %v", err)
	}
	if calls != 2 {
		t.Fatalf("expected exactly one repair round, got %d calls", calls)
	}
	if generated.Title != "Unit tests for Payment API" {
		t.Fatalf("expected the repaired title, got %+v", generated)
	}
}

func TestGenerateFailsAfterOneRepair(t *testing.T) {
	calls := 0
	client := &LLMClient{completion: func(_ context.Context, _ string, _ string, _ string, _ string) (string, error) {
		calls++
		return "still not JSON", nil
	}}

	_, err := client.Generate(context.Background(), "user-1", "DeepSeek-V4 Flash", "Payment API", documentRequestFixture())
	if !errors.Is(err, sharedErrors.ErrBadGateway) {
		t.Fatalf("expected a bad gateway error, got %v", err)
	}
	if calls != 2 {
		t.Fatalf("expected exactly one repair round, got %d calls", calls)
	}
}

func TestGeneratePropagatesCompletionFailures(t *testing.T) {
	client := &LLMClient{completion: func(_ context.Context, _ string, _ string, _ string, _ string) (string, error) {
		return "", errors.New("the provider is down")
	}}

	if _, err := client.Generate(context.Background(), "user-1", "DeepSeek-V4 Flash", "Payment API", documentRequestFixture()); err == nil {
		t.Fatal("expected the provider failure to propagate")
	}
}

func TestGenerateWithoutACompletionSeamFails(t *testing.T) {
	client := &LLMClient{}
	if _, err := client.Generate(context.Background(), "user-1", "DeepSeek-V4 Flash", "Payment API", documentRequestFixture()); err == nil {
		t.Fatal("expected an unconfigured client to fail")
	}
}

func TestParseGeneratedContentHandlesFencesAndRejectsIncompleteObjects(t *testing.T) {
	fenced := "```json\n" + validGeneratedJSON() + "\n```"
	if _, err := ParseGeneratedContent(fenced); err != nil {
		t.Fatalf("expected the fenced object to parse, got %v", err)
	}

	for _, invalid := range []string{
		"",
		"not json",
		`{"summary":"missing title"}`,
		`{"title":"missing summary"}`,
		`{"title":"t","summary":"s","rows":[{"function":"no test function name"}]}`,
		`[{"title":"not an object"}]`,
	} {
		if _, err := ParseGeneratedContent(invalid); err == nil {
			t.Fatalf("expected %q to be rejected", invalid)
		}
	}
}
