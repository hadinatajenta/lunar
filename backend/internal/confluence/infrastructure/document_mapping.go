package infrastructure

import (
	"html"
	"net/url"
	"regexp"
	"strings"

	"lunar/backend/internal/confluence/domain"
)

const descriptionMaxLength = 2000

var (
	htmlTagPattern       = regexp.MustCompile(`<[^>]*>`)
	standaloneSOPPattern = regexp.MustCompile(`\bsop\b`)
	standaloneUTPattern  = regexp.MustCompile(`\but\b`)
)

func buildDocumentFromContent(content confluenceContent, fallbackBase string) domain.ConfluenceDocument {
	labelNames := collectLabelNames(content)
	documentType := resolveDocumentType(labelNames, content.Title)
	return domain.ConfluenceDocument{
		ID:          content.ID,
		Type:        documentType,
		TypeLabel:   typeLabelFor(documentType),
		Title:       content.Title,
		Status:      resolveStatus(labelNames),
		Owner:       content.Version.By.DisplayName,
		Updated:     content.Version.When,
		Space:       resolveSpaceName(content.Space),
		Description: plainTextFromStorage(content.Body.Storage.Value),
		URL:         buildDocumentURL(content, fallbackBase),
	}
}

func collectLabelNames(content confluenceContent) []string {
	labelResults := content.Metadata.Labels.Results
	labelNames := make([]string, 0, len(labelResults))
	for _, label := range labelResults {
		labelNames = append(labelNames, strings.ToLower(label.Name))
	}
	return labelNames
}

func resolveDocumentType(labelNames []string, title string) string {
	if containsAnyLabel(labelNames, "ut", "unit-test", "unit test") {
		return "ut"
	}
	if containsAnyLabel(labelNames, "query", "query-review", "qr") {
		return "query"
	}
	if containsAnyLabel(labelNames, "sop") {
		return "sop"
	}
	return resolveDocumentTypeFromTitle(title)
}

func resolveDocumentTypeFromTitle(title string) string {
	normalizedTitle := strings.ToLower(title)
	if strings.Contains(normalizedTitle, "query review") {
		return "query"
	}
	if standaloneSOPPattern.MatchString(normalizedTitle) {
		return "sop"
	}
	if standaloneUTPattern.MatchString(normalizedTitle) || strings.Contains(normalizedTitle, "unit test") {
		return "ut"
	}
	return "doc"
}

func resolveStatus(labelNames []string) string {
	if containsAnyLabel(labelNames, "done", "closed", "resolved") {
		return "done"
	}
	if containsAnyLabel(labelNames, "in-progress", "in progress", "progress") {
		return "progress"
	}
	return "open"
}

func containsAnyLabel(labelNames []string, needles ...string) bool {
	for _, label := range labelNames {
		for _, needle := range needles {
			if strings.Contains(label, needle) {
				return true
			}
		}
	}
	return false
}

func typeLabelFor(documentType string) string {
	switch documentType {
	case "ut":
		return "UT"
	case "query":
		return "QR"
	case "sop":
		return "SOP"
	default:
		return "DOC"
	}
}

func resolveSpaceName(space confluenceSpace) string {
	if space.Name != "" {
		return space.Name
	}
	return space.Key
}

func buildDocumentURL(content confluenceContent, fallbackBase string) string {
	base := strings.TrimRight(content.Links.Base, "/")
	if base == "" {
		base = strings.TrimRight(fallbackBase, "/")
	}
	if base == "" {
		return ""
	}
	webUI := strings.TrimSpace(content.Links.WebUI)
	if webUI == "" {
		return base + "/pages/viewpage.action?pageId=" + url.PathEscape(content.ID)
	}
	if !strings.HasPrefix(webUI, "/") {
		webUI = "/" + webUI
	}
	return base + webUI
}

func plainTextFromStorage(storageHTML string) string {
	tagStripped := htmlTagPattern.ReplaceAllString(storageHTML, " ")
	decoded := html.UnescapeString(tagStripped)
	collapsed := strings.Join(strings.Fields(decoded), " ")
	return truncateRunes(collapsed, descriptionMaxLength)
}

func truncateRunes(text string, limit int) string {
	runes := []rune(text)
	if len(runes) <= limit {
		return text
	}
	return string(runes[:limit])
}
