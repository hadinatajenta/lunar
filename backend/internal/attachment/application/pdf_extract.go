package application

import (
	"bytes"
	"fmt"
	"io"

	"github.com/ledongthuc/pdf"
)

func extractPDFText(content []byte) (string, error) {
	reader, err := pdf.NewReader(bytes.NewReader(content), int64(len(content)))
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrUnreadableDocument, err)
	}

	plainText, err := reader.GetPlainText()
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrUnreadableDocument, err)
	}

	rawText, err := io.ReadAll(plainText)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrUnreadableDocument, err)
	}

	return capExtractedText(string(rawText)), nil
}
