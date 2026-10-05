package application

import (
	"errors"
	"fmt"
	"net/http"
	"path"
	"strings"
	"unicode"

	"lunar/backend/internal/attachment/domain"
)

const (
	MaxFileBytes              = 10 << 20
	MaxMultipartOverheadBytes = 1 << 20
	MaxRequestBodyBytes       = MaxFileBytes + MaxMultipartOverheadBytes

	sniffLength       = 512
	maxFileNameLength = 200

	AcceptedTypesMessage = "accepted file types are png, jpeg, webp, pdf, xlsx, xls and docx"
)

var (
	ErrUnsupportedType    = errors.New("unsupported file type")
	ErrUnreadableDocument = errors.New("the attached document cannot be read")
)

var acceptedImageContentTypes = map[string]bool{
	"image/png":  true,
	"image/jpeg": true,
	"image/webp": true,
}

var acceptedDocumentContentTypes = map[string]string{
	".pdf":  "application/pdf",
	".xlsx": "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
	".xls":  "application/vnd.ms-excel",
	".docx": "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
}

type AcceptedUpload struct {
	Name        string
	Kind        domain.Kind
	ContentType string
}

func AcceptUpload(rawName string, content []byte) (AcceptedUpload, error) {
	name := SanitizeFileName(rawName)

	detectedContentType := http.DetectContentType(sniffHead(content))
	if acceptedImageContentTypes[detectedContentType] {
		return AcceptedUpload{Name: name, Kind: domain.KindImage, ContentType: detectedContentType}, nil
	}

	contentType, isAcceptedDocument := acceptedDocumentContentTypes[strings.ToLower(path.Ext(name))]
	if isAcceptedDocument {
		return AcceptedUpload{Name: name, Kind: domain.KindDocument, ContentType: contentType}, nil
	}

	return AcceptedUpload{}, fmt.Errorf("%w: %s", ErrUnsupportedType, AcceptedTypesMessage)
}

func sniffHead(content []byte) []byte {
	if len(content) <= sniffLength {
		return content
	}
	return content[:sniffLength]
}

func SanitizeFileName(rawName string) string {
	flattened := strings.ReplaceAll(rawName, "\\", "/")
	baseName := path.Base(flattened)

	var builder strings.Builder
	for _, character := range baseName {
		if unicode.IsControl(character) {
			continue
		}
		builder.WriteRune(character)
	}

	name := strings.TrimSpace(builder.String())
	if name == "" || name == "." || name == ".." || name == "/" {
		return "attachment"
	}

	runes := []rune(name)
	if len(runes) > maxFileNameLength {
		name = string(runes[:maxFileNameLength])
	}
	return name
}
