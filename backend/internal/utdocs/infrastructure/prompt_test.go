package infrastructure

import (
	"errors"
	"strings"
	"testing"

	"lunar/backend/internal/utdocs/domain"
)

func TestUTDocsSystemPromptCarriesTheInvariants(t *testing.T) {
	requiredPhrases := []string{
		"untrusted data",
		"exactly one row for every test function",
		"Never invent a test function",
		"must use mocks",
		"Never write an example that opens a database",
		"never described as passing or failing",
		"it was not run",
		"Never write a result in a row",
		"professional Indonesian",
		"Write the title, summary, caption, and function names in English",
		"Output exactly one JSON object",
		"\"title\"",
		"\"summary\"",
		"\"rows\"",
		"\"testFunctionName\"",
		"\"caption\"",
		"mockTests",
		"untested_functions",
		"complete Go test function",
		"\"code\"",
		"TestSave",
	}

	for _, phrase := range requiredPhrases {
		if !strings.Contains(utDocsSystemPrompt, phrase) {
			t.Fatalf("expected the system prompt to carry %q", phrase)
		}
	}
}

func TestBuildUTDocsUserPromptEmbedsTheDelimitedPayload(t *testing.T) {
	request := domain.DocumentRequest{
		Model: "DeepSeek-V4 Flash",
		Repositories: []domain.DocumentRepository{
			{
				Repository: "micro-payment",
				Branch:     "feat/CRMMS-123",
				BaseRef:    "development",
				TestFiles: []domain.TestFile{
					{Path: "internal/repo/dao_test.go", Package: "./internal/repo", Language: domain.LanguageGo, Classification: domain.ClassificationSafe, TestFunctions: []string{"TestFind"}},
				},
				BlockedFiles: []domain.TestFile{
					{Path: "internal/seed/reset_test.go", Classification: domain.ClassificationBlocked, Reason: "contains destructive or database markers: TRUNCATE"},
				},
				Run: domain.RunResult{Status: domain.RunStatusPassed, Total: 1, Passed: 1},
				UntestedFunctions: []domain.UntestedFunction{
					{Name: "Find", File: "internal/repo/dao.go", Package: "./internal/repo"},
				},
			},
		},
	}

	prompt, err := BuildUTDocsUserPrompt("Payment API", request)
	if err != nil {
		t.Fatalf("expected the prompt to be built, got %v", err)
	}

	requiredFragments := []string{
		"<<<UT_RUN_DATA",
		"UT_RUN_DATA>>>",
		"Document title: Payment API",
		"micro-payment",
		"feat/CRMMS-123",
		"internal/seed/reset_test.go",
		"TRUNCATE",
		"untested_functions",
		"internal/repo/dao.go",
	}
	for _, fragment := range requiredFragments {
		if !strings.Contains(prompt, fragment) {
			t.Fatalf("expected the prompt to contain %q", fragment)
		}
	}
}

func TestBuildUTDocsRepairPromptBoundsTheRejectedAnswer(t *testing.T) {
	rejected := strings.Repeat("a", repairContentLimit+100)
	prompt := BuildUTDocsRepairPrompt(errors.New("the answer is not valid JSON"), rejected)

	if !strings.Contains(prompt, "REJECTED_ANSWER") {
		t.Fatal("expected the rejected answer delimiters")
	}
	if !strings.Contains(prompt, "truncated for the repair request") {
		t.Fatal("expected the truncation marker")
	}
	if len(prompt) > repairContentLimit+len("truncated for the repair request")+600 {
		t.Fatalf("expected the rejected answer to be bounded, got %d characters", len(prompt))
	}
}
