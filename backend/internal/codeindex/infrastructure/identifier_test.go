package infrastructure

import (
	"strings"
	"testing"

	codeindexdomain "lunar/backend/internal/codeindex/domain"
)

func TestSplitCamelCase(t *testing.T) {
	testCases := []struct {
		input    string
		expected []string
	}{
		{"GetPaymentFlow", []string{"Get", "Payment", "Flow"}},
		{"processOrder", []string{"process", "Order"}},
		{"XMLParser", []string{"XML", "Parser"}},
		{"getHTTPResponse", []string{"get", "HTTP", "Response"}},
		{"simple", []string{"simple"}},
		{"A", []string{"A"}},
	}

	for _, testCase := range testCases {
		t.Run(testCase.input, func(t *testing.T) {
			words := splitCamelCase(testCase.input)
			if strings.Join(words, "|") != strings.Join(testCase.expected, "|") {
				t.Errorf("splitCamelCase(%q) = %v, want %v", testCase.input, words, testCase.expected)
			}
		})
	}
}

func TestSplitIdentifier(t *testing.T) {
	testCases := []struct {
		input    string
		expected []string
	}{
		{"GetServiceDependencies", []string{"get", "service", "dependencies"}},
		{"kafka_topic_payment", []string{"kafka", "topic", "payment"}},
		{"micro-cm-edc", []string{"micro", "cm", "edc"}},
		{"ProcessPaymentRequest", []string{"process", "payment", "request"}},
		{"ok", []string{}},
		{"HTTPClient", []string{"http", "client"}},
	}

	for _, testCase := range testCases {
		t.Run(testCase.input, func(t *testing.T) {
			words := splitIdentifier(testCase.input)
			if strings.Join(words, "|") != strings.Join(testCase.expected, "|") {
				t.Errorf("splitIdentifier(%q) = %v, want %v", testCase.input, words, testCase.expected)
			}
		})
	}
}

func TestPrepareContentForFTSExpandsCamelCase(t *testing.T) {
	expanded := prepareContentForFTS("func calculateLateFee(ctx context.Context) error {")

	if expanded != strings.ToLower(expanded) {
		t.Errorf("expected expanded content to be lowercased, got %q", expanded)
	}
	if !strings.Contains(expanded, "calculate late fee") {
		t.Errorf("expected camelCase expansion tokens in %q", expanded)
	}
	if !strings.Contains(expanded, "calculatelatefee") {
		t.Errorf("expected original identifier token in %q", expanded)
	}
}

func TestPrepareContentForFTSWithoutExpandableIdentifier(t *testing.T) {
	expanded := prepareContentForFTS("the quick brown fox")
	if expanded != "the quick brown fox" {
		t.Errorf("expected unchanged lowercase content, got %q", expanded)
	}
}

func TestSanitizeFTSQuery(t *testing.T) {
	testCases := []struct {
		input    string
		expected string
	}{
		{"payment flow", "payment flow"},
		{"GetPayment()", "GetPayment"},
		{"micro:cm:edc", "micro cm edc"},
		{"[route] /api/payments", "route /api/payments"},
	}

	for _, testCase := range testCases {
		if sanitized := sanitizeFTSQuery(testCase.input); sanitized != testCase.expected {
			t.Errorf("sanitizeFTSQuery(%q) = %q, want %q", testCase.input, sanitized, testCase.expected)
		}
	}
}

func TestAddChunkForRangeReportsSourceLines(t *testing.T) {
	content := "alpha\nbeta\ngamma\n"

	var chunks []codeindexdomain.CodeChunk
	addChunkForRange(&chunks, "sample-repo", "sample.go", string(codeindexdomain.ChunkTypeCodeBlock), content, 6, 16)

	if len(chunks) != 1 {
		t.Fatalf("expected 1 chunk, got %d", len(chunks))
	}
	if chunks[0].StartLine != 2 || chunks[0].EndLine != 3 {
		t.Errorf("expected range 2-3, got %d-%d", chunks[0].StartLine, chunks[0].EndLine)
	}
	if chunks[0].RawContent != "beta\ngamma" {
		t.Errorf("unexpected raw content %q", chunks[0].RawContent)
	}
	if chunks[0].Content == "" {
		t.Error("expected FTS content to be populated")
	}
}

func TestAddChunkSkipsBlankContent(t *testing.T) {
	var chunks []codeindexdomain.CodeChunk
	addChunk(&chunks, "sample-repo", "sample.go", string(codeindexdomain.ChunkTypeCodeBlock), "   \n\t", 4, 5)

	if len(chunks) != 0 {
		t.Errorf("expected no chunks for blank content, got %d", len(chunks))
	}
}
