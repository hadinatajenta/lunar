package application

import (
	"errors"
	"os"
	"strings"
	"testing"

	"lunar/backend/internal/attachment/testsupport"
)

func TestExtractDocumentText_ReadsPDFText(t *testing.T) {
	text, err := ExtractDocumentText("report.pdf", testsupport.PDF("Hello Lunar Attachment"))
	if err != nil {
		t.Fatalf("cannot extract the pdf text: %v", err)
	}
	if !strings.Contains(text, "Hello Lunar Attachment") {
		t.Errorf("expected the pdf line to be extracted, got %q", text)
	}
}

func TestExtractDocumentText_ReadsXLSXText(t *testing.T) {
	workbook := testsupport.XLSX(t, [][]string{
		{"Region", "Revenue"},
		{"APAC", "4200"},
	})

	text, err := ExtractDocumentText("revenue.xlsx", workbook)
	if err != nil {
		t.Fatalf("cannot extract the xlsx text: %v", err)
	}
	for _, expected := range []string{"Region", "Revenue", "APAC", "4200"} {
		if !strings.Contains(text, expected) {
			t.Errorf("expected %q in the extracted xlsx text, got %q", expected, text)
		}
	}
	if !strings.Contains(text, "Region\tRevenue") {
		t.Errorf("expected columns to be separated by a tab, got %q", text)
	}
}

func TestExtractDocumentText_ReadsLegacyXLSText(t *testing.T) {
	content, err := os.ReadFile("testdata/table.xls")
	if err != nil {
		t.Fatalf("cannot read the legacy xls fixture: %v", err)
	}

	text, err := ExtractDocumentText("table.xls", content)
	if err != nil {
		t.Fatalf("cannot extract the xls text: %v", err)
	}
	for _, expected := range []string{"Code", "Description", "code1", "name1", "description1"} {
		if !strings.Contains(text, expected) {
			t.Errorf("expected %q in the extracted xls text, got %q", expected, text)
		}
	}
}

func TestExtractDocumentText_ReadsDOCXTextWithParagraphBreaks(t *testing.T) {
	document := testsupport.DOCX(t, []string{"Release plan", "Ship the attachment backend"})

	text, err := ExtractDocumentText("plan.docx", document)
	if err != nil {
		t.Fatalf("cannot extract the docx text: %v", err)
	}
	if !strings.Contains(text, "Release plan\nShip the attachment backend") {
		t.Errorf("expected both paragraphs on their own line, got %q", text)
	}
}

func TestExtractDocumentText_RejectsADocxThatIsNotAZipArchive(t *testing.T) {
	_, err := ExtractDocumentText("broken.docx", []byte("this is not a zip archive"))
	if !errors.Is(err, ErrUnreadableDocument) {
		t.Fatalf("expected ErrUnreadableDocument, got %v", err)
	}
}

func TestExtractDocumentText_RejectsAnUnknownExtension(t *testing.T) {
	_, err := ExtractDocumentText("notes.txt", []byte("plain text"))
	if !errors.Is(err, ErrUnsupportedType) {
		t.Fatalf("expected ErrUnsupportedType, got %v", err)
	}
}

func TestCapExtractedText_TruncatesAndMarksTheCut(t *testing.T) {
	oversized := strings.Repeat("x", MaxExtractedCharacters+500)

	capped := capExtractedText(oversized)
	if len([]rune(capped)) != MaxExtractedCharacters+len([]rune(truncationNotice)) {
		t.Fatalf("expected %d runes, got %d", MaxExtractedCharacters+len([]rune(truncationNotice)), len([]rune(capped)))
	}
	if !strings.HasSuffix(capped, truncationNotice) {
		t.Errorf("expected the truncation notice at the end, got %q", capped[len(capped)-40:])
	}

	untouched := capExtractedText("short document")
	if untouched != "short document" {
		t.Errorf("expected a short document to survive unchanged, got %q", untouched)
	}
}

func TestBuildDocumentPreview_CollapsesWhitespaceAndBoundsTheLength(t *testing.T) {
	preview := BuildDocumentPreview("line one\n\n   line two")
	if preview != "line one line two" {
		t.Errorf("expected the whitespace to be collapsed, got %q", preview)
	}

	longPreview := BuildDocumentPreview(strings.Repeat("y", documentPreviewCharacters+50))
	if len([]rune(longPreview)) != documentPreviewCharacters+len([]rune(truncationSuffix)) {
		t.Errorf("expected the preview to be bounded, got %d runes", len([]rune(longPreview)))
	}
}
