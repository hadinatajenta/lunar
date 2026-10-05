package application

import (
	"errors"
	"strings"
	"testing"

	"lunar/backend/internal/attachment/domain"
	"lunar/backend/internal/attachment/testsupport"
)

func TestAcceptUpload_AcceptsAnImageByItsContentNotItsName(t *testing.T) {
	accepted, err := AcceptUpload("quarterly-notes.txt", testsupport.PNG(t))
	if err != nil {
		t.Fatalf("expected the png content to be accepted, got: %v", err)
	}
	if accepted.Kind != domain.KindImage {
		t.Errorf("expected kind %q, got %q", domain.KindImage, accepted.Kind)
	}
	if accepted.ContentType != "image/png" {
		t.Errorf("expected content type image/png, got %q", accepted.ContentType)
	}
	if accepted.Name != "quarterly-notes.txt" {
		t.Errorf("expected the sanitized name to be kept, got %q", accepted.Name)
	}
}

func TestAcceptUpload_RejectsTextContentWearingAnImageExtension(t *testing.T) {
	accepted, err := AcceptUpload("screenshot.png", []byte("this is plainly not an image"))
	if !errors.Is(err, ErrUnsupportedType) {
		t.Fatalf("expected ErrUnsupportedType for text content named .png, got %v", err)
	}
	if accepted.Kind != "" {
		t.Errorf("expected no accepted upload, got %+v", accepted)
	}
}

func TestAcceptUpload_AcceptsEveryAllowlistedDocumentExtension(t *testing.T) {
	for _, fileName := range []string{"report.pdf", "sales.XLSX", "legacy.xls", "spec.Docx"} {
		accepted, err := AcceptUpload(fileName, []byte("document body"))
		if err != nil {
			t.Fatalf("expected %q to be accepted, got: %v", fileName, err)
		}
		if accepted.Kind != domain.KindDocument {
			t.Errorf("expected %q to be a document, got %q", fileName, accepted.Kind)
		}
	}
}

func TestAcceptUpload_RejectsUnsupportedTypesAndNamesTheAcceptedOnes(t *testing.T) {
	for _, fileName := range []string{"archive.zip", "vector.gif", "installer.exe", "sheet.csv", "noextension"} {
		_, err := AcceptUpload(fileName, []byte("PK\x03\x04 not really supported"))
		if !errors.Is(err, ErrUnsupportedType) {
			t.Fatalf("expected %q to be rejected with ErrUnsupportedType, got %v", fileName, err)
		}
		for _, expectedType := range []string{"png", "jpeg", "webp", "pdf", "xlsx", "xls", "docx"} {
			if !strings.Contains(err.Error(), expectedType) {
				t.Errorf("expected the rejection of %q to name %q, got %q", fileName, expectedType, err.Error())
			}
		}
	}
}

func TestSanitizeFileName_StripsDirectoriesAndControlCharacters(t *testing.T) {
	cases := map[string]string{
		"../../etc/passwd":               "passwd",
		`..\..\windows\system32\cmd.exe`: "cmd.exe",
		"bad\x00name\n.pdf":              "badname.pdf",
		"..":                             "attachment",
		"":                               "attachment",
		"/":                              "attachment",
	}

	for input, expected := range cases {
		if actual := SanitizeFileName(input); actual != expected {
			t.Errorf("expected %q to sanitize to %q, got %q", input, expected, actual)
		}
	}
}

func TestSanitizeFileName_BoundsTheLength(t *testing.T) {
	sanitized := SanitizeFileName(strings.Repeat("a", 500) + ".pdf")
	if len([]rune(sanitized)) != maxFileNameLength {
		t.Errorf("expected the name to be capped at %d runes, got %d", maxFileNameLength, len([]rune(sanitized)))
	}
}
