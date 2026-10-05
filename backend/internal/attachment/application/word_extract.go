package application

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"strings"
)

const (
	docxDocumentPath  = "word/document.xml"
	docxCellSeparator = "\t"
)

type docxTextCollector struct {
	builder   strings.Builder
	isTextRun bool
}

func extractDOCXText(content []byte) (text string, err error) {
	archive, err := zip.NewReader(bytes.NewReader(content), int64(len(content)))
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrUnreadableDocument, err)
	}

	documentEntry := findZipEntry(archive, docxDocumentPath)
	if documentEntry == nil {
		return "", fmt.Errorf("%w: %s is missing", ErrUnreadableDocument, docxDocumentPath)
	}

	reader, err := documentEntry.Open()
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrUnreadableDocument, err)
	}
	defer func() {
		if closeErr := reader.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("%w: %v", ErrUnreadableDocument, closeErr)
		}
	}()

	collector := &docxTextCollector{}
	if err := collector.collect(reader); err != nil {
		return "", fmt.Errorf("%w: %v", ErrUnreadableDocument, err)
	}

	return capExtractedText(collector.builder.String()), nil
}

func findZipEntry(archive *zip.Reader, name string) *zip.File {
	for _, entry := range archive.File {
		if entry.Name == name {
			return entry
		}
	}
	return nil
}

func (c *docxTextCollector) collect(reader io.Reader) error {
	decoder := xml.NewDecoder(reader)
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		c.handleToken(token)
	}
}

func (c *docxTextCollector) handleToken(token xml.Token) {
	switch element := token.(type) {
	case xml.StartElement:
		switch element.Name.Local {
		case "t":
			c.isTextRun = true
		case "tab":
			c.builder.WriteString(docxCellSeparator)
		case "br":
			c.builder.WriteString(rowSeparator)
		}
	case xml.EndElement:
		switch element.Name.Local {
		case "t":
			c.isTextRun = false
		case "p":
			c.builder.WriteString(rowSeparator)
		case "tc":
			c.builder.WriteString(docxCellSeparator)
		}
	case xml.CharData:
		if c.isTextRun {
			c.builder.Write([]byte(element))
		}
	}
}
