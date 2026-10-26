package infrastructure

import (
	"bytes"
	"fmt"
	"image/png"
	"strings"
	"testing"

	"lunar/backend/internal/utdocs/domain"
)

func TestPrepareScreenshotLinesCapsAndMarks(t *testing.T) {
	output := buildNumberedOutput(250)
	lines, dropped := PrepareScreenshotLines(output, maxScreenshotLines)

	if len(lines) != maxScreenshotLines {
		t.Fatalf("expected %d lines, got %d", maxScreenshotLines, len(lines))
	}
	if dropped != 51 {
		t.Fatalf("expected 51 dropped lines, got %d", dropped)
	}
	if lines[len(lines)-1] != TruncationMarker(51) {
		t.Fatalf("expected the truncation marker last, got %q", lines[len(lines)-1])
	}
	if !strings.Contains(lines[len(lines)-1], "51 lines were left out") {
		t.Fatalf("expected the marker to count the dropped lines, got %q", lines[len(lines)-1])
	}
}

func TestPrepareScreenshotLinesKeepsShortOutput(t *testing.T) {
	lines, dropped := PrepareScreenshotLines("first\nsecond\n", 200)
	if dropped != 0 {
		t.Fatalf("expected no dropped lines, got %d", dropped)
	}
	if len(lines) != 2 {
		t.Fatalf("expected two lines after trimming the trailing newline, got %d", len(lines))
	}

	emptyLines, emptyDropped := PrepareScreenshotLines("", 200)
	if emptyDropped != 0 || len(emptyLines) != 1 || emptyLines[0] != "" {
		t.Fatalf("expected one empty line for empty output, got %+v %d", emptyLines, emptyDropped)
	}
}

func TestPrepareScreenshotLinesNormalizesCarriageReturns(t *testing.T) {
	lines, dropped := PrepareScreenshotLines("first\r\nsecond\r\n", 200)
	if dropped != 0 || len(lines) != 2 || lines[0] != "first" || lines[1] != "second" {
		t.Fatalf("expected normalized lines, got %+v", lines)
	}
}

func TestTruncateScreenshotLine(t *testing.T) {
	long := strings.Repeat("x", 200)
	truncated := TruncateScreenshotLine(long, maxScreenshotColumns)
	if len([]rune(truncated)) != maxScreenshotColumns {
		t.Fatalf("expected %d runes, got %d", maxScreenshotColumns, len([]rune(truncated)))
	}
	if !strings.HasSuffix(truncated, "...") {
		t.Fatalf("expected the truncation suffix, got %q", truncated)
	}

	short := "short line"
	if TruncateScreenshotLine(short, maxScreenshotColumns) != short {
		t.Fatal("expected a short line to be unchanged")
	}
}

func TestExpandScreenshotTabs(t *testing.T) {
	expanded := ExpandScreenshotTabs("a\tb\tc", 4)
	if expanded != "a   b   c" {
		t.Fatalf("expected four column tab stops, got %q", expanded)
	}
	if ExpandScreenshotTabs("plain", 4) != "plain" {
		t.Fatal("expected a line without tabs to be unchanged")
	}
}

func TestRenderProducesADecodableTerminalPNG(t *testing.T) {
	renderer := NewScreenshotRenderer()
	request := domain.ScreenshotRequest{
		Repo:    "micro-payment",
		Branch:  "feat/CRMMS-123",
		Command: "go test -count=1 -timeout 120s ./internal/repo",
		Result:  "passed",
		Output:  "ok  \tlunar/backend/internal/repo\t0.120s\n",
	}

	small, err := renderer.Render(request)
	if err != nil {
		t.Fatalf("expected a screenshot, got %v", err)
	}
	if !bytes.HasPrefix(small, []byte{0x89, 0x50, 0x4e, 0x47}) {
		t.Fatal("expected a PNG signature")
	}

	smallImage, err := png.Decode(bytes.NewReader(small))
	if err != nil {
		t.Fatalf("expected a decodable PNG, got %v", err)
	}
	if smallImage.Bounds().Dx() <= maxScreenshotColumns*6 {
		t.Fatalf("expected room for %d columns, got width %d", maxScreenshotColumns, smallImage.Bounds().Dx())
	}
	if smallImage.Bounds().Dy() <= 0 {
		t.Fatal("expected a positive height")
	}

	request.Output = buildNumberedOutput(250)
	large, err := renderer.Render(request)
	if err != nil {
		t.Fatalf("expected a large screenshot, got %v", err)
	}
	largeImage, err := png.Decode(bytes.NewReader(large))
	if err != nil {
		t.Fatalf("expected a decodable large PNG, got %v", err)
	}

	if largeImage.Bounds().Dx() != smallImage.Bounds().Dx() {
		t.Fatalf("expected a fixed width, got %d and %d", smallImage.Bounds().Dx(), largeImage.Bounds().Dx())
	}
	if largeImage.Bounds().Dy() <= smallImage.Bounds().Dy() {
		t.Fatalf("expected the taller output to grow the image, got %d and %d", smallImage.Bounds().Dy(), largeImage.Bounds().Dy())
	}
	if largeImage.Bounds().Dy() > smallImage.Bounds().Dy()+maxScreenshotLines*screenshotLineHeight {
		t.Fatalf("expected the line cap to bound the height, got %d", largeImage.Bounds().Dy())
	}
}

func buildNumberedOutput(count int) string {
	builder := strings.Builder{}
	for index := 0; index < count; index++ {
		builder.WriteString(fmt.Sprintf("line %d", index))
		builder.WriteString("\n")
	}
	return builder.String()
}

func TestScreenshotResultLabelIncludesTheCounts(t *testing.T) {
	cases := []struct {
		request  domain.ScreenshotRequest
		expected string
	}{
		{request: domain.ScreenshotRequest{Result: "failed", Passed: 9, Total: 10}, expected: "failed (9/10 passed)"},
		{request: domain.ScreenshotRequest{Result: "passed", Passed: 4, Total: 4}, expected: "passed (4/4 passed)"},
		{request: domain.ScreenshotRequest{Result: "not_run"}, expected: "not_run"},
		{request: domain.ScreenshotRequest{Result: "blocked"}, expected: "blocked"},
	}

	for _, testCase := range cases {
		if actual := screenshotResultLabel(testCase.request); actual != testCase.expected {
			t.Fatalf("expected %q, got %q", testCase.expected, actual)
		}
	}
}
