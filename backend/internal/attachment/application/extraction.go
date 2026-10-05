package application

import (
	"fmt"
	"path"
	"strings"
)

const (
	MaxExtractedCharacters    = 60000
	documentPreviewCharacters = 200
	truncationSuffix          = "..."
	truncationNotice          = "\n[attachment text truncated]"
)

func ExtractDocumentText(name string, content []byte) (text string, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			text = ""
			err = fmt.Errorf("%w: %v", ErrUnreadableDocument, recovered)
		}
	}()

	switch strings.ToLower(path.Ext(name)) {
	case ".pdf":
		return extractPDFText(content)
	case ".xlsx":
		return extractXLSXText(content)
	case ".xls":
		return extractXLSText(content)
	case ".docx":
		return extractDOCXText(content)
	default:
		return "", fmt.Errorf("%w: %s", ErrUnsupportedType, AcceptedTypesMessage)
	}
}

func BuildDocumentPreview(text string) string {
	collapsed := collapseWhitespace(text)
	runes := []rune(collapsed)
	if len(runes) <= documentPreviewCharacters {
		return collapsed
	}
	return strings.TrimSpace(string(runes[:documentPreviewCharacters])) + truncationSuffix
}

func capExtractedText(text string) string {
	normalized := strings.ReplaceAll(text, "\r\n", "\n")
	normalized = strings.TrimSpace(normalized)

	runes := []rune(normalized)
	if len(runes) <= MaxExtractedCharacters {
		return normalized
	}
	return strings.TrimSpace(string(runes[:MaxExtractedCharacters])) + truncationNotice
}

func collapseWhitespace(text string) string {
	return strings.Join(strings.Fields(text), " ")
}
