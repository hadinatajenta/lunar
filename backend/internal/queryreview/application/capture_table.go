package application

import (
	"html"
	"regexp"
	"strconv"
	"strings"

	"lunar/backend/internal/queryreview/domain"
)

const (
	captureExplainMarkerText = "Capture explain (mysql)"
	captureExplainMarker     = `<p><em>Capture explain (mysql)</em></p>`

	captureExplainTableHeader = `<tr><th>No</th><th>Fitur</th><th>Query Statement</th><th>Index</th><th>Capture Explain / Execution Plan</th></tr>`

	noQueryChangesMessage = "Tidak ada penambahan atau perubahan query pada pengembangan kali ini."

	maxCaptureRepositories         = 50
	maxCaptureQueriesPerRepository = 200
	maxCaptureRepositoryLength     = 200
	maxCaptureFeatureLength        = 500
	maxCaptureStatementLength      = 8000
	maxCaptureIndexLength          = 500
	maxCaptureCreateIndexLength    = 2000
)

var (
	captureParagraphPattern = regexp.MustCompile(`(?is)<p\b[^>]*>.*?</p>`)
	captureTablePattern     = regexp.MustCompile(`(?is)^\s*<table\b[^>]*>.*?</table>`)
)

func BuildCaptureExplainTable(repositories []domain.RepositoryQueryReview) string {
	populated := make([]domain.RepositoryQueryReview, 0, len(repositories))
	for _, repository := range repositories {
		if len(repository.Queries) == 0 {
			continue
		}
		populated = append(populated, repository)
	}
	repositories = populated

	if len(repositories) > maxCaptureRepositories {
		repositories = repositories[:maxCaptureRepositories]
	}

	grouped := len(repositories) > 1

	var builder strings.Builder
	builder.WriteString(captureExplainMarker)
	builder.WriteString("\n")
	builder.WriteString(`<table class="wrapped relative-table" style="width: 100%;">`)
	builder.WriteString("\n<tbody>\n")
	builder.WriteString(captureExplainTableHeader)
	builder.WriteString("\n")

	if len(repositories) == 0 {
		builder.WriteString(`<tr><td colspan="5">` + html.EscapeString(noQueryChangesMessage) + `</td></tr>`)
		builder.WriteString("\n")
	}

	for _, repository := range repositories {
		if grouped {
			builder.WriteString(buildCaptureGroupRow(repository.Repository))
			builder.WriteString("\n")
		}

		queries := repository.Queries
		if len(queries) > maxCaptureQueriesPerRepository {
			queries = queries[:maxCaptureQueriesPerRepository]
		}
		for index, query := range queries {
			builder.WriteString(buildCaptureQueryRow(index+1, query))
			builder.WriteString("\n")
		}
	}

	builder.WriteString("</tbody>\n</table>")
	return builder.String()
}

func ReplaceCaptureExplainTable(storage string, table string) string {
	if strings.TrimSpace(table) == "" {
		return storage
	}
	if strings.TrimSpace(storage) == "" {
		return table
	}

	markerStart, markerEnd, found := findCaptureMarker(storage)
	if !found {
		return appendCaptureBlock(storage, table)
	}

	if tableEnd, ok := findCaptureTableEnd(storage, markerEnd); ok {
		return storage[:markerStart] + table + storage[tableEnd:]
	}
	return storage[:markerStart] + table + storage[markerEnd:]
}

func buildCaptureGroupRow(repository string) string {
	escaped := html.EscapeString(boundCaptureValue(repository, maxCaptureRepositoryLength))
	return `<tr><td colspan="5"><strong>Layanan / Microservice: <code>` + escaped + `</code></strong></td></tr>`
}

func buildCaptureQueryRow(number int, query domain.QueryItem) string {
	return "<tr><td>" + strconv.Itoa(number) + "</td><td>" +
		html.EscapeString(boundCaptureValue(query.Feature, maxCaptureFeatureLength)) + "</td><td>" +
		html.EscapeString(boundCaptureValue(query.QueryStatement, maxCaptureStatementLength)) + "</td><td>" +
		buildCaptureIndexCell(query) + "</td><td></td></tr>"
}

func buildCaptureIndexCell(query domain.QueryItem) string {
	parts := make([]string, 0, 2)
	if indexName := boundCaptureValue(query.Index, maxCaptureIndexLength); indexName != "" {
		parts = append(parts, html.EscapeString(indexName))
	}
	if createIndex := boundCaptureValue(query.CreateIndex, maxCaptureCreateIndexLength); createIndex != "" {
		parts = append(parts, html.EscapeString(createIndex))
	}
	return strings.Join(parts, "<br/>")
}

func findCaptureMarker(storage string) (int, int, bool) {
	marker := strings.ToLower(captureExplainMarkerText)
	for _, match := range captureParagraphPattern.FindAllStringIndex(storage, -1) {
		if strings.Contains(strings.ToLower(storage[match[0]:match[1]]), marker) {
			return match[0], match[1], true
		}
	}
	return 0, 0, false
}

func findCaptureTableEnd(storage string, from int) (int, bool) {
	loc := captureTablePattern.FindStringIndex(storage[from:])
	if loc == nil {
		return 0, false
	}
	return from + loc[1], true
}

func appendCaptureBlock(storage string, table string) string {
	if strings.HasSuffix(storage, "\n") {
		return storage + table
	}
	return storage + "\n" + table
}

func boundCaptureValue(value string, maxLength int) string {
	trimmed := strings.TrimSpace(value)
	runes := []rune(trimmed)
	if len(runes) <= maxLength {
		return trimmed
	}
	return string(runes[:maxLength])
}
