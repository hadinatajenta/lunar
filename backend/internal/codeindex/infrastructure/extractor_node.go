package infrastructure

import (
	"regexp"

	codeindexdomain "lunar/backend/internal/codeindex/domain"
)

var (
	nodeRoutePattern = regexp.MustCompile(
		`(?:app|router|server)\.(get|post|put|delete|patch|options)\s*\(\s*['"]([^'"]+)`,
	)
	nodeHTTPClientPattern = regexp.MustCompile(
		`(?i)(axios|fetch|got|request)\s*[.(]\s*(?:get|post|put|delete|patch)?\s*\(\s*[` + "`" + `'"]([^` + "`" + `'"]+)`,
	)
	nodeFunctionPattern = regexp.MustCompile(
		`(?m)^[ \t]*(?:export\s+)?(?:async\s+)?function\s+([a-zA-Z0-9_]+)\s*\(|^[ \t]*(?:const|let|var)\s+([a-zA-Z0-9_]+)\s*=\s*(?:async\s*)?\([^)]*\)\s*(?::\s*[^{]+)?=>\s*\{`,
	)
	nodeFieldAssignmentPattern = regexp.MustCompile(
		`(?m)^[ \t]*([a-zA-Z0-9_.]+(?:\.([a-zA-Z0-9_]+)|\[['"](\w+)['"]\])\s*=\s*(\d+|'[^']*'|"[^"]*"))\s*;?\s*$`,
	)
)

func extractNode(content string, filePath string, repoName string) ([]codeindexdomain.CodeChunk, []codeindexdomain.ServiceDependency) {
	var chunks []codeindexdomain.CodeChunk
	var dependencies []codeindexdomain.ServiceDependency

	for _, match := range nodeRoutePattern.FindAllStringSubmatchIndex(content, -1) {
		addChunkForRange(&chunks, repoName, filePath, string(codeindexdomain.ChunkTypeRoute), content, match[0], match[1])
	}

	for _, match := range nodeHTTPClientPattern.FindAllStringSubmatchIndex(content, -1) {
		addChunkForRange(&chunks, repoName, filePath, string(codeindexdomain.ChunkTypeHTTPClient), content, match[0], match[1])
		if len(match) < 6 {
			continue
		}
		if dependency, isFound := httpClientDependency(repoName, filePath, content[match[4]:match[5]]); isFound {
			dependencies = append(dependencies, dependency)
		}
	}

	chunks = append(chunks, extractFunctions(content, filePath, repoName, nodeFunctionPattern)...)
	chunks = append(chunks, extractFieldAssignments(content, filePath, repoName, nodeFieldAssignmentPattern)...)

	return chunks, dependencies
}
