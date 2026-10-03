package infrastructure

import (
	"regexp"

	codeindexdomain "lunar/backend/internal/codeindex/domain"
)

var (
	javaSpringRoutePattern = regexp.MustCompile(
		`@(Get|Post|Put|Delete|Patch|Request)Mapping(?:\s*\(\s*(?:(?:value|path)\s*=\s*)?["'{]([^"'{}]+)["'}]?\s*\))?`,
	)
	javaHTTPClientPattern = regexp.MustCompile(
		`(?i)(restTemplate|webClient|openFeign)\s*\.\s*\w+\s*\(\s*["']([^"']+)`,
	)
	javaFunctionPattern = regexp.MustCompile(
		`(?m)^[ \t]*(?:(?:public|protected|private|static|final|synchronized)\s+)+[\w<>\[\],\s]+\s+([a-zA-Z0-9_]+)\s*\(`,
	)
	javaFieldAssignmentPattern = regexp.MustCompile(
		`(?m)^[ \t]*([a-zA-Z0-9_.]+\.set([a-zA-Z0-9_]+)\s*\(\s*(\d+|"[^"]*")\s*\))`,
	)
)

func extractJava(content string, filePath string, repoName string) ([]codeindexdomain.CodeChunk, []codeindexdomain.ServiceDependency) {
	var chunks []codeindexdomain.CodeChunk
	var dependencies []codeindexdomain.ServiceDependency

	for _, match := range javaSpringRoutePattern.FindAllStringSubmatchIndex(content, -1) {
		addChunkForRange(&chunks, repoName, filePath, string(codeindexdomain.ChunkTypeRoute), content, match[0], match[1])
	}

	for _, match := range javaHTTPClientPattern.FindAllStringSubmatchIndex(content, -1) {
		addChunkForRange(&chunks, repoName, filePath, string(codeindexdomain.ChunkTypeHTTPClient), content, match[0], match[1])
		if len(match) < 6 {
			continue
		}
		if dependency, isFound := httpClientDependency(repoName, filePath, content[match[4]:match[5]]); isFound {
			dependencies = append(dependencies, dependency)
		}
	}

	chunks = append(chunks, extractFunctions(content, filePath, repoName, javaFunctionPattern)...)
	chunks = append(chunks, extractFieldAssignments(content, filePath, repoName, javaFieldAssignmentPattern)...)

	return chunks, dependencies
}
