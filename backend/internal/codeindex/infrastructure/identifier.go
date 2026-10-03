package infrastructure

import (
	"regexp"
	"strings"
	"unicode"

	codeindexdomain "lunar/backend/internal/codeindex/domain"
)

var identifierPattern = regexp.MustCompile(`[A-Za-z][A-Za-z0-9_]{2,}`)

var ftsQueryReplacer = strings.NewReplacer(
	"(", " ",
	")", " ",
	"[", " ",
	"]", " ",
	"{", " ",
	"}", " ",
	":", " ",
	";", " ",
	"'", " ",
	"`", " ",
)

func prepareContentForFTS(rawContent string) string {
	base := strings.ToLower(rawContent)
	seenTokens := map[string]bool{base: true}
	extras := make([]string, 0, 16)

	for _, identifier := range identifierPattern.FindAllString(rawContent, -1) {
		parts := splitIdentifier(identifier)
		if len(parts) < 2 {
			continue
		}
		joined := strings.Join(parts, " ")
		if seenTokens[joined] {
			continue
		}
		seenTokens[joined] = true
		extras = append(extras, joined)
	}

	if len(extras) == 0 {
		return base
	}
	return base + " " + strings.Join(extras, " ")
}

func splitIdentifier(value string) []string {
	if len(value) <= 2 {
		return nil
	}

	var words []string
	for _, part := range strings.FieldsFunc(value, func(r rune) bool { return r == '_' || r == '-' }) {
		for _, word := range splitCamelCase(part) {
			lowered := strings.ToLower(word)
			if len(lowered) >= 2 {
				words = append(words, lowered)
			}
		}
	}
	return words
}

func splitCamelCase(value string) []string {
	if value == "" {
		return nil
	}

	runes := []rune(value)
	words := make([]string, 0, 4)
	start := 0

	for index := 1; index < len(runes); index++ {
		current := runes[index]
		if !unicode.IsUpper(current) {
			continue
		}
		previousIsLower := unicode.IsLower(runes[index-1])
		nextIsLower := index+1 < len(runes) && unicode.IsLower(runes[index+1])
		if previousIsLower || nextIsLower {
			words = append(words, string(runes[start:index]))
			start = index
		}
	}

	if start < len(runes) {
		words = append(words, string(runes[start:]))
	}
	return words
}

func sanitizeFTSQuery(query string) string {
	return strings.Join(strings.Fields(ftsQueryReplacer.Replace(query)), " ")
}

func addChunk(chunks *[]codeindexdomain.CodeChunk, repoName string, filePath string, chunkType string, rawContent string, startLine int, endLine int) {
	if strings.TrimSpace(rawContent) == "" {
		return
	}
	*chunks = append(*chunks, codeindexdomain.CodeChunk{
		RepoName:   repoName,
		FilePath:   filePath,
		ChunkType:  chunkType,
		RawContent: rawContent,
		Content:    prepareContentForFTS(rawContent),
		StartLine:  startLine,
		EndLine:    endLine,
	})
}

func addChunkForRange(chunks *[]codeindexdomain.CodeChunk, repoName string, filePath string, chunkType string, content string, startOffset int, endOffset int) {
	startLine, endLine := lineRangeForOffsets(content, startOffset, endOffset)
	if startLine == 0 {
		return
	}
	addChunk(chunks, repoName, filePath, chunkType, sourceLines(content, startLine, endLine), startLine, endLine)
}

func lineRangeForOffsets(content string, startOffset int, endOffset int) (int, int) {
	if startOffset < 0 || endOffset <= startOffset || startOffset >= len(content) {
		return 0, 0
	}
	if endOffset > len(content) {
		endOffset = len(content)
	}

	startLine := lineNumberAt(content, startOffset)
	endLine := startLine + strings.Count(content[startOffset:endOffset], "\n")
	if endLine > startLine && content[endOffset-1] == '\n' {
		endLine--
	}
	return startLine, endLine
}

func lineNumberAt(content string, offset int) int {
	if offset <= 0 {
		return 1
	}
	if offset > len(content) {
		offset = len(content)
	}
	return strings.Count(content[:offset], "\n") + 1
}

func sourceLines(content string, startLine int, endLine int) string {
	if startLine < 1 || endLine < startLine {
		return ""
	}

	lines := strings.Split(content, "\n")
	if startLine > len(lines) {
		return ""
	}
	if endLine > len(lines) {
		endLine = len(lines)
	}
	return strings.Join(lines[startLine-1:endLine], "\n")
}
