package infrastructure

import (
	"strconv"
	"strings"

	"lunar/backend/internal/queryreview/application"
	"lunar/backend/internal/queryreview/domain"
)

const (
	queryReviewFileCharBudget  = 12000
	queryReviewTotalCharBudget = 120000
	queryReviewLineOverhead    = 24
)

type DiffExtractor struct{}

var _ application.DiffExtractor = (*DiffExtractor)(nil)

func NewDiffExtractor() *DiffExtractor {
	return &DiffExtractor{}
}

func (e *DiffExtractor) Extract(diffText string) domain.DiffAnalysis {
	analysis := domain.DiffAnalysis{
		Files:    []domain.PromptFile{},
		Analyzed: []domain.AnalyzedFile{},
		Skipped:  []domain.SkippedFile{},
	}

	totalPayloadSize := 0

	for _, file := range splitUnifiedDiff(diffText) {
		reason := classifyIgnoredPath(file.path)
		if reason == "" && len(file.addedLines) == 0 {
			reason = skipReasonNoAddedLines
		}
		if reason == "" && !isDatabaseRelevantPath(file.path) && !containsQuerySyntax(file.addedLines) {
			reason = skipReasonNotDatabase
		}
		if reason != "" {
			analysis.Skipped = append(analysis.Skipped, domain.SkippedFile{Path: file.path, Reason: reason})
			continue
		}

		payloadSize := promptFilePayloadSize(file.addedLines)
		if payloadSize > queryReviewFileCharBudget {
			analysis.Skipped = append(analysis.Skipped, domain.SkippedFile{Path: file.path, Reason: skipReasonOverBudget})
			analysis.Truncated = true
			continue
		}
		if totalPayloadSize+payloadSize > queryReviewTotalCharBudget {
			analysis.Skipped = append(analysis.Skipped, domain.SkippedFile{Path: file.path, Reason: skipReasonOverTotalBudget})
			analysis.Truncated = true
			continue
		}
		totalPayloadSize += payloadSize

		language := languageForPath(file.path)
		analysis.Files = append(analysis.Files, domain.PromptFile{
			Path:       file.path,
			Language:   language,
			AddedLines: file.addedLines,
		})
		analysis.Analyzed = append(analysis.Analyzed, domain.AnalyzedFile{
			Path:       file.path,
			AddedLines: len(file.addedLines),
			Language:   language,
		})
	}

	return analysis
}

type parsedDiffFile struct {
	path       string
	addedLines []domain.PromptLine
}

func splitUnifiedDiff(diffText string) []parsedDiffFile {
	files := make([]parsedDiffFile, 0)
	current := parsedDiffFile{addedLines: []domain.PromptLine{}}
	hasCurrent := false
	inHunk := false
	newLine := 0

	flush := func() {
		if hasCurrent {
			files = append(files, current)
			current = parsedDiffFile{addedLines: []domain.PromptLine{}}
			hasCurrent = false
		}
	}

	for _, raw := range strings.Split(diffText, "\n") {
		if strings.HasPrefix(raw, "diff --git ") {
			flush()
			current = parsedDiffFile{path: pathFromGitDiffHeader(raw), addedLines: []domain.PromptLine{}}
			hasCurrent = true
			inHunk = false
			continue
		}
		if !hasCurrent {
			continue
		}

		if !inHunk {
			if strings.HasPrefix(raw, "+++ ") {
				if updated := pathFromUnifiedMarker(raw); updated != "" {
					current.path = updated
				}
				continue
			}
			if hasFileHeaderPrefix(raw) {
				continue
			}
		}

		if strings.HasPrefix(raw, "@@") {
			newLine = parseNewHunkStart(raw)
			inHunk = true
			continue
		}
		if !inHunk {
			continue
		}

		if strings.HasPrefix(raw, "\\") {
			continue
		}
		if strings.HasPrefix(raw, "+") {
			current.addedLines = append(current.addedLines, domain.PromptLine{
				Number:  newLine,
				Content: strings.TrimPrefix(raw, "+"),
			})
			newLine++
			continue
		}
		if strings.HasPrefix(raw, "-") {
			continue
		}
		if strings.HasPrefix(raw, " ") {
			newLine++
		}
	}

	flush()
	return files
}

func hasFileHeaderPrefix(raw string) bool {
	prefixes := []string{
		"--- ",
		"index ",
		"new file mode ",
		"deleted file mode ",
		"similarity index ",
		"rename from ",
		"rename to ",
		"old mode ",
		"new mode ",
	}
	for _, prefix := range prefixes {
		if strings.HasPrefix(raw, prefix) {
			return true
		}
	}
	return false
}

func pathFromGitDiffHeader(line string) string {
	parts := strings.Fields(line)
	if len(parts) < 4 {
		return ""
	}
	return strings.TrimPrefix(parts[3], "b/")
}

func pathFromUnifiedMarker(line string) string {
	value := strings.TrimPrefix(line, "+++ ")
	if index := strings.IndexByte(value, '\t'); index >= 0 {
		value = value[:index]
	}
	value = strings.TrimSpace(value)
	if value == "/dev/null" {
		return ""
	}
	return strings.TrimPrefix(value, "b/")
}

func parseNewHunkStart(header string) int {
	fields := strings.Fields(header)
	if len(fields) < 3 {
		return 1
	}

	field := strings.TrimPrefix(fields[2], "+")
	if index := strings.IndexByte(field, ','); index >= 0 {
		field = field[:index]
	}

	value, err := strconv.Atoi(field)
	if err != nil || value < 0 {
		return 1
	}
	return value
}

func containsQuerySyntax(lines []domain.PromptLine) bool {
	for _, line := range lines {
		lowered := strings.ToLower(line.Content)
		for _, marker := range querySyntaxMarkers {
			if strings.Contains(lowered, marker) {
				return true
			}
		}
	}
	return false
}

func promptFilePayloadSize(lines []domain.PromptLine) int {
	size := 0
	for _, line := range lines {
		size += len(line.Content) + queryReviewLineOverhead
	}
	return size
}
