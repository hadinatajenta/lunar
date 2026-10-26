package application

import (
	"html"
	"regexp"
	"strconv"
	"strings"

	"lunar/backend/internal/utdocs/domain"
)

const (
	normalTableMarkerText = "Normal Test Scenario"
	normalTableMarker     = `<p><em>Normal Test Scenario</em></p>`

	normalTableHeader = `<tr><th>No</th><th>Function</th><th>Scenario</th><th>Expected Result</th><th>Input Data</th><th>Result</th><th>Screen Capture</th></tr>`

	noScenarioMessage = "Belum ada skenario pengujian pada pengembangan kali ini."

	maxDocumentRepositories = 50
	maxDocumentRows         = 500
	maxCellFunctionLength   = 300
	maxCellScenarioLength   = 4000
	maxCellExpectedLength   = 4000
	maxCellInputLength      = 4000
	maxCellResultLength     = 20
	maxCellRepositoryLength = 200
	maxScreenCaptureLength  = 400
	maxAttachmentNameLength = 200
)

var (
	normalParagraphPattern = regexp.MustCompile(`(?is)<p\b[^>]*>.*?</p>`)
	normalTablePattern     = regexp.MustCompile(`(?is)^\s*<table\b[^>]*>.*?</table>`)

	structuredCodeMacroPattern = regexp.MustCompile(`(?is)^\s*<ac:structured-macro\b[^>]*ac:name="code"[^>]*>.*?</ac:structured-macro>`)

	attachmentFilenamePattern = regexp.MustCompile(`(?i)^[a-z0-9][a-z0-9._ -]*\.png$`)
	attachmentCellPattern     = regexp.MustCompile(`(?i)^<ac:image(?:\s+ac:width="[0-9]+")?><ri:attachment ri:filename="[a-z0-9][a-z0-9._ -]*\.png"/></ac:image>$`)
)

const screenCaptureWidth = "400"

func BuildNormalTestScenarioTable(rows []domain.DocumentRow) string {
	rows = boundDocumentRows(rows)
	grouped := distinctRepositories(rows) > 1

	var builder strings.Builder
	builder.WriteString(normalTableMarker)
	builder.WriteString("\n")
	builder.WriteString(`<table class="wrapped relative-table" style="width: 100%;">`)
	builder.WriteString("\n<tbody>\n")
	builder.WriteString(normalTableHeader)
	builder.WriteString("\n")

	if len(rows) == 0 {
		builder.WriteString(`<tr><td colspan="7">` + html.EscapeString(noScenarioMessage) + `</td></tr>`)
		builder.WriteString("\n")
	}

	currentRepository := ""
	number := 0
	for _, row := range rows {
		repository := strings.TrimSpace(row.Repository)
		if grouped && repository != currentRepository {
			builder.WriteString(buildNormalGroupRow(repository))
			builder.WriteString("\n")
			currentRepository = repository
			number = 0
		}
		number++
		builder.WriteString(buildNormalRow(number, row))
		builder.WriteString("\n")
	}

	builder.WriteString("</tbody>\n</table>")
	return builder.String()
}

func ReplaceNormalTestScenarioTable(storage string, table string) string {
	if strings.TrimSpace(table) == "" {
		return storage
	}
	if strings.TrimSpace(storage) == "" {
		return table
	}

	markerStart, markerEnd, found := findNormalMarker(storage)
	if !found {
		return appendNormalBlock(storage, table)
	}

	if tableEnd, ok := findNormalTableEnd(storage, markerEnd); ok {
		return storage[:markerStart] + table + storage[tableEnd:]
	}
	return storage[:markerStart] + table + storage[markerEnd:]
}

func ReplaceNormalTestScenarioBlock(storage string, table string, codeMacros string) string {
	if strings.TrimSpace(table) == "" {
		return storage
	}

	block := table
	if strings.TrimSpace(codeMacros) != "" {
		block = table + "\n" + codeMacros
	}
	if strings.TrimSpace(storage) == "" {
		return block
	}

	markerStart, markerEnd, found := findNormalMarker(storage)
	if !found {
		return appendNormalBlock(storage, block)
	}

	if tableEnd, ok := findNormalTableEnd(storage, markerEnd); ok {
		return storage[:markerStart] + block + storage[consumeFollowingCodeMacros(storage, tableEnd):]
	}
	return storage[:markerStart] + block + storage[markerEnd:]
}

func consumeFollowingCodeMacros(storage string, from int) int {
	position := from
	for {
		loc := structuredCodeMacroPattern.FindStringIndex(storage[position:])
		if loc == nil {
			return position
		}
		position += loc[1]
	}
}

func BuildScreenCaptureCell(filename string) string {
	trimmed := strings.TrimSpace(filename)
	if len(trimmed) > maxAttachmentNameLength || !attachmentFilenamePattern.MatchString(trimmed) {
		return ""
	}
	return `<ac:image ac:width="` + screenCaptureWidth + `"><ri:attachment ri:filename="` + trimmed + `"/></ac:image>`
}

func ValidAttachmentFilename(filename string) bool {
	trimmed := strings.TrimSpace(filename)
	return len(trimmed) <= maxAttachmentNameLength && attachmentFilenamePattern.MatchString(trimmed)
}

func buildNormalGroupRow(repository string) string {
	escaped := html.EscapeString(boundCellValue(repository, maxCellRepositoryLength))
	return `<tr><td colspan="7"><strong>Layanan / Microservice: <code>` + escaped + `</code></strong></td></tr>`
}

func buildNormalRow(number int, row domain.DocumentRow) string {
	return "<tr><td>" + strconv.Itoa(number) + "</td><td>" +
		html.EscapeString(boundCellValue(row.Function, maxCellFunctionLength)) + "</td><td>" +
		html.EscapeString(boundCellValue(row.Scenario, maxCellScenarioLength)) + "</td><td>" +
		html.EscapeString(boundCellValue(row.ExpectedResult, maxCellExpectedLength)) + "</td><td>" +
		html.EscapeString(boundCellValue(row.InputData, maxCellInputLength)) + "</td><td>" +
		html.EscapeString(boundCellValue(row.Result, maxCellResultLength)) + "</td><td>" +
		screenCaptureCell(row.ScreenCapture) + "</td></tr>"
}

func screenCaptureCell(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}
	if attachmentCellPattern.MatchString(trimmed) {
		return trimmed
	}
	return html.EscapeString(boundCellValue(trimmed, maxScreenCaptureLength))
}

func renderMarkdownTable(rows []domain.DocumentRow) string {
	rows = boundDocumentRows(rows)
	grouped := distinctRepositories(rows) > 1

	builder := strings.Builder{}
	builder.WriteString("| No | Function | Scenario | Expected Result | Input Data | Result | Screen Capture |\n")
	builder.WriteString("| --- | --- | --- | --- | --- | --- | --- |\n")

	if len(rows) == 0 {
		builder.WriteString("| | " + escapeMarkdownCell(noScenarioMessage) + " | | | | | |\n")
		return builder.String()
	}

	currentRepository := ""
	number := 0
	for _, row := range rows {
		repository := strings.TrimSpace(row.Repository)
		if grouped && repository != currentRepository {
			builder.WriteString("\n**Layanan / Microservice: " + escapeMarkdownCell(boundCellValue(repository, maxCellRepositoryLength)) + "**\n\n")
			currentRepository = repository
			number = 0
		}
		number++
		builder.WriteString("| " + strconv.Itoa(number) +
			" | " + escapeMarkdownCell(boundCellValue(row.Function, maxCellFunctionLength)) +
			" | " + escapeMarkdownCell(boundCellValue(row.Scenario, maxCellScenarioLength)) +
			" | " + escapeMarkdownCell(boundCellValue(row.ExpectedResult, maxCellExpectedLength)) +
			" | " + escapeMarkdownCell(boundCellValue(row.InputData, maxCellInputLength)) +
			" | " + escapeMarkdownCell(boundCellValue(row.Result, maxCellResultLength)) +
			" | " + escapeMarkdownCell(row.ScreenCapture) + " |\n")
	}
	return builder.String()
}

func escapeMarkdownCell(value string) string {
	replaced := strings.ReplaceAll(value, "|", `\|`)
	replaced = strings.ReplaceAll(replaced, "\n", "<br/>")
	return replaced
}

func findNormalMarker(storage string) (int, int, bool) {
	marker := strings.ToLower(normalTableMarkerText)
	for _, match := range normalParagraphPattern.FindAllStringIndex(storage, -1) {
		if strings.Contains(strings.ToLower(storage[match[0]:match[1]]), marker) {
			return match[0], match[1], true
		}
	}
	return 0, 0, false
}

func findNormalTableEnd(storage string, from int) (int, bool) {
	loc := normalTablePattern.FindStringIndex(storage[from:])
	if loc == nil {
		return 0, false
	}
	return from + loc[1], true
}

func appendNormalBlock(storage string, table string) string {
	if strings.HasSuffix(storage, "\n") {
		return storage + table
	}
	return storage + "\n" + table
}

func boundCellValue(value string, maxLength int) string {
	trimmed := strings.TrimSpace(value)
	runes := []rune(trimmed)
	if len(runes) <= maxLength {
		return trimmed
	}
	return string(runes[:maxLength])
}

func distinctRepositories(rows []domain.DocumentRow) int {
	seen := make(map[string]struct{})
	for _, row := range rows {
		repository := strings.TrimSpace(row.Repository)
		if repository == "" {
			continue
		}
		seen[repository] = struct{}{}
	}
	return len(seen)
}

func boundDocumentRows(rows []domain.DocumentRow) []domain.DocumentRow {
	if len(rows) <= maxDocumentRows {
		return rows
	}
	return rows[:maxDocumentRows]
}
