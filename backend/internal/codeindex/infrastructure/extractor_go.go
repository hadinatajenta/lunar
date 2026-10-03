package infrastructure

import (
	"regexp"

	codeindexdomain "lunar/backend/internal/codeindex/domain"
)

var (
	goRoutePattern = regexp.MustCompile(
		`\.(GET|POST|PUT|DELETE|PATCH|OPTIONS|HEAD)\s*\(\s*["` + "`" + `]([^"` + "`" + `\s]+)`,
	)
	goMuxRoutePattern = regexp.MustCompile(
		`HandleFunc\s*\(\s*["` + "`" + `]([^"` + "`" + `\s]+)`,
	)
	goHTTPClientPattern = regexp.MustCompile(
		`http\.(Get|Post|NewRequest|NewRequestWithContext)\s*\([^,)]*[,\s]*["` + "`" + `](https?://[^"` + "`" + `\s]+)`,
	)
	goKafkaPublishPattern = regexp.MustCompile(
		`(?i)(\.Produce|\.Publish|\.WriteMessages|\.SendMessage|\.PublishMessage)\s*\([^)]*["` + "`" + `]([a-z][a-z0-9._-]{2,60})["` + "`" + `]`,
	)
	goKafkaSubscribePattern = regexp.MustCompile(
		`(?i)(\.Subscribe|\.Consume|\.ReadMessages|kafka\.NewReader)\s*\([^)]*["` + "`" + `]([a-z][a-z0-9._-]{2,60})["` + "`" + `]`,
	)
	goFunctionPattern = regexp.MustCompile(
		`(?m)^func\s+(?:\([^)]+\)\s+)?([a-zA-Z0-9_]+)\s*\(`,
	)
	goFieldAssignmentPattern = regexp.MustCompile(
		`(?m)^[ \t]*([a-zA-Z0-9_]+\.[a-zA-Z0-9_]+\s*=\s*(\d+|"[^"]*"))\s*$`,
	)
	goMapAssignmentPattern = regexp.MustCompile(
		`(?m)^[ \t]*(\w+\["(\w+)"\]\s*=\s*(\d+|"[^"]*"))\s*$`,
	)
	goConstBlockPattern  = regexp.MustCompile(`(?m)^[ \t]*const[ \t]*\([^)]*\)`)
	goConstSinglePattern = regexp.MustCompile(`(?m)^[ \t]*const[ \t]+[A-Za-z_][A-Za-z0-9_]*[ \t]*=[ \t]*("[^"]*"|-?[0-9]+)`)
)

var goConstPatterns = []*regexp.Regexp{goConstBlockPattern, goConstSinglePattern}

func extractGo(content string, filePath string, repoName string) ([]codeindexdomain.CodeChunk, []codeindexdomain.ServiceDependency) {
	var chunks []codeindexdomain.CodeChunk
	var dependencies []codeindexdomain.ServiceDependency

	for _, match := range goRoutePattern.FindAllStringSubmatchIndex(content, -1) {
		addChunkForRange(&chunks, repoName, filePath, string(codeindexdomain.ChunkTypeRoute), content, match[0], match[1])
	}

	for _, match := range goMuxRoutePattern.FindAllStringSubmatchIndex(content, -1) {
		addChunkForRange(&chunks, repoName, filePath, string(codeindexdomain.ChunkTypeRoute), content, match[0], match[1])
	}

	for _, match := range goHTTPClientPattern.FindAllStringSubmatchIndex(content, -1) {
		addChunkForRange(&chunks, repoName, filePath, string(codeindexdomain.ChunkTypeHTTPClient), content, match[0], match[1])
		if len(match) < 6 {
			continue
		}
		if dependency, isFound := httpClientDependency(repoName, filePath, content[match[4]:match[5]]); isFound {
			dependencies = append(dependencies, dependency)
		}
	}

	for _, match := range goKafkaPublishPattern.FindAllStringSubmatchIndex(content, -1) {
		addChunkForRange(&chunks, repoName, filePath, string(codeindexdomain.ChunkTypeKafkaTopic), content, match[0], match[1])
		if len(match) < 6 {
			continue
		}
		dependencies = append(dependencies, kafkaDependency(repoName, filePath, content[match[4]:match[5]], codeindexdomain.CallTypeKafkaPublish))
	}

	for _, match := range goKafkaSubscribePattern.FindAllStringSubmatchIndex(content, -1) {
		addChunkForRange(&chunks, repoName, filePath, string(codeindexdomain.ChunkTypeKafkaTopic), content, match[0], match[1])
		if len(match) < 6 {
			continue
		}
		dependencies = append(dependencies, kafkaDependency(repoName, filePath, content[match[4]:match[5]], codeindexdomain.CallTypeKafkaSubscribe))
	}

	chunks = append(chunks, extractFunctions(content, filePath, repoName, goFunctionPattern)...)
	chunks = append(chunks, extractFieldAssignments(content, filePath, repoName, goFieldAssignmentPattern)...)
	chunks = append(chunks, extractFieldAssignments(content, filePath, repoName, goMapAssignmentPattern)...)
	chunks = append(chunks, extractDeclarationChunks(content, filePath, repoName, string(codeindexdomain.ChunkTypeEnumDef), goConstPatterns)...)

	return chunks, dependencies
}
