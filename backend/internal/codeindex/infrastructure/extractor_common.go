package infrastructure

import (
	"regexp"
	"strings"

	codeindexdomain "lunar/backend/internal/codeindex/domain"
)

const (
	maxSingleChunkLines    = 250
	functionChunkSizeLines = 100
	functionChunkStepLines = 80
)

var lineCommentPrefix = "/" + "/"

type scanState int

const (
	stateCode scanState = iota
	stateLineComment
	stateBlockComment
	stateDoubleQuote
	stateSingleQuote
	stateBacktick
)

func ExtractFile(content string, filePath string, repoName string, extension string) ([]codeindexdomain.CodeChunk, []codeindexdomain.ServiceDependency) {
	switch strings.ToLower(extension) {
	case ".go":
		return extractGo(content, filePath, repoName)
	case ".java":
		return extractJava(content, filePath, repoName)
	case ".js", ".ts":
		return extractNode(content, filePath, repoName)
	case ".php":
		return extractPHP(content, filePath, repoName)
	default:
		return nil, nil
	}
}

func extractFunctions(content string, filePath string, repoName string, functionPattern *regexp.Regexp) []codeindexdomain.CodeChunk {
	var chunks []codeindexdomain.CodeChunk

	for _, match := range functionPattern.FindAllStringSubmatchIndex(content, -1) {
		if !hasCapturedGroup(match) {
			continue
		}

		openBraceIndex := findNextOpenBrace(content, match[1])
		if openBraceIndex < 0 {
			continue
		}

		closeBraceIndex, isFound := findMatchingBrace(content, openBraceIndex)
		if !isFound || closeBraceIndex <= openBraceIndex {
			continue
		}

		appendFunctionChunks(&chunks, content, filePath, repoName, match[0], closeBraceIndex+1)
	}
	return chunks
}

func hasCapturedGroup(match []int) bool {
	for index := 2; index+1 < len(match); index += 2 {
		if match[index] >= 0 && match[index+1] >= 0 {
			return true
		}
	}
	return false
}

func appendFunctionChunks(chunks *[]codeindexdomain.CodeChunk, content string, filePath string, repoName string, startOffset int, endOffset int) {
	startLine, endLine := lineRangeForOffsets(content, startOffset, endOffset)
	if startLine == 0 {
		return
	}

	chunkType := string(codeindexdomain.ChunkTypeCodeBlock)
	if endLine-startLine+1 <= maxSingleChunkLines {
		addChunk(chunks, repoName, filePath, chunkType, sourceLines(content, startLine, endLine), startLine, endLine)
		return
	}

	functionLines := strings.Split(sourceLines(content, startLine, endLine), "\n")
	for blockStart := 0; blockStart < len(functionLines); blockStart += functionChunkStepLines {
		blockEnd := blockStart + functionChunkSizeLines
		if blockEnd > len(functionLines) {
			blockEnd = len(functionLines)
		}
		addChunk(
			chunks,
			repoName,
			filePath,
			chunkType,
			strings.Join(functionLines[blockStart:blockEnd], "\n"),
			startLine+blockStart,
			startLine+blockEnd-1,
		)
		if blockEnd >= len(functionLines) {
			break
		}
	}
}

func extractFieldAssignments(content string, filePath string, repoName string, assignmentPattern *regexp.Regexp) []codeindexdomain.CodeChunk {
	var chunks []codeindexdomain.CodeChunk
	chunkType := string(codeindexdomain.ChunkTypeCodeBlock)

	for _, match := range assignmentPattern.FindAllStringIndex(content, -1) {
		addChunkForRange(&chunks, repoName, filePath, chunkType, content, match[0], match[1])
	}
	return chunks
}

func extractDeclarationChunks(content string, filePath string, repoName string, chunkType string, patterns []*regexp.Regexp) []codeindexdomain.CodeChunk {
	var chunks []codeindexdomain.CodeChunk
	lines := strings.Split(content, "\n")

	for _, pattern := range patterns {
		for _, match := range pattern.FindAllStringIndex(content, -1) {
			startLine, endLine := lineRangeForOffsets(content, match[0], match[1])
			if startLine == 0 {
				continue
			}
			for startLine > 1 && isLineComment(lines[startLine-2]) {
				startLine--
			}
			addChunk(&chunks, repoName, filePath, chunkType, sourceLines(content, startLine, endLine), startLine, endLine)
		}
	}
	return chunks
}

func isLineComment(line string) bool {
	return strings.HasPrefix(strings.TrimSpace(line), lineCommentPrefix)
}

func findMatchingBrace(content string, openBraceIndex int) (int, bool) {
	if openBraceIndex < 0 || openBraceIndex >= len(content) || content[openBraceIndex] != '{' {
		return -1, false
	}

	depth := 0
	currentState := stateCode

	for index := openBraceIndex; index < len(content); index++ {
		current := content[index]
		var next byte
		if index+1 < len(content) {
			next = content[index+1]
		}

		switch currentState {
		case stateLineComment:
			if current == '\n' {
				currentState = stateCode
			}
			continue
		case stateBlockComment:
			if current == '*' && next == '/' {
				currentState = stateCode
				index++
			}
			continue
		case stateDoubleQuote:
			if current == '\\' {
				index++
				continue
			}
			if current == '"' {
				currentState = stateCode
			}
			continue
		case stateSingleQuote:
			if current == '\\' {
				index++
				continue
			}
			if current == '\'' {
				currentState = stateCode
			}
			continue
		case stateBacktick:
			if current == '`' {
				currentState = stateCode
			}
			continue
		}

		switch {
		case current == '/' && next == '/':
			currentState = stateLineComment
			index++
		case current == '#':
			currentState = stateLineComment
		case current == '/' && next == '*':
			currentState = stateBlockComment
			index++
		case current == '"':
			currentState = stateDoubleQuote
		case current == '\'':
			currentState = stateSingleQuote
		case current == '`':
			currentState = stateBacktick
		case current == '{':
			depth++
		case current == '}':
			depth--
			if depth == 0 {
				return index, true
			}
		}
	}

	return -1, false
}

func findNextOpenBrace(content string, startIndex int) int {
	parenthesisDepth := 0

	for index := startIndex; index < len(content); index++ {
		switch content[index] {
		case '(':
			parenthesisDepth++
		case ')':
			if parenthesisDepth > 0 {
				parenthesisDepth--
			}
		case '{':
			if parenthesisDepth == 0 {
				return index
			}
		case ';':
			if parenthesisDepth == 0 {
				return -1
			}
		}
	}
	return -1
}
