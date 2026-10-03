package infrastructure

import (
	"regexp"

	codeindexdomain "lunar/backend/internal/codeindex/domain"
)

var (
	phpRoutePattern = regexp.MustCompile(
		`Route::(get|post|put|delete|patch|any|match|resource)\s*\(\s*['"]([^'"]+)`,
	)
	phpFunctionPattern = regexp.MustCompile(
		`(?m)^[ \t]*(?:(?:public|protected|private|static|final)\s+)*function\s+([a-zA-Z0-9_]+)\s*\(`,
	)
	phpFieldAssignmentPattern = regexp.MustCompile(
		`(?m)^[ \t]*(\$\w+\[['"](\w+)['"]\]\s*=\s*(\d+|'[^']*'|"[^"]*")\s*;)`,
	)
)

func extractPHP(content string, filePath string, repoName string) ([]codeindexdomain.CodeChunk, []codeindexdomain.ServiceDependency) {
	var chunks []codeindexdomain.CodeChunk

	for _, match := range phpRoutePattern.FindAllStringSubmatchIndex(content, -1) {
		addChunkForRange(&chunks, repoName, filePath, string(codeindexdomain.ChunkTypeRoute), content, match[0], match[1])
	}

	chunks = append(chunks, extractFunctions(content, filePath, repoName, phpFunctionPattern)...)
	chunks = append(chunks, extractFieldAssignments(content, filePath, repoName, phpFieldAssignmentPattern)...)

	return chunks, nil
}
