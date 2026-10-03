package tools

import (
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"unicode/utf8"

	agentdomain "lunar/backend/internal/agent/domain"
	codeindexdomain "lunar/backend/internal/codeindex/domain"
	workspacedomain "lunar/backend/internal/workspace/domain"
)

const (
	defaultSearchLimit    = 8
	maxSearchLimit        = 20
	defaultSymbolLimit    = 15
	maxSymbolLimit        = 30
	maxChunkCharacters    = 1200
	maxSnippetCharacters  = 600
	maxResultCharacters   = 12000
	maxServiceLimit       = 100
	maxRouteLimit         = 100
	maxDependencyLimit    = 200
	defaultKnowledgeLimit = 5
	maxKnowledgeLimit     = 10
)

type WorkspaceRootProvider interface {
	WorkspaceRoot() string
}

type toolDependencies struct {
	store         codeindexdomain.IndexStore
	gitReader     CodeGitReader
	workspaceRoot string
}

func NewTools(store codeindexdomain.IndexStore, gitReader CodeGitReader) []agentdomain.Tool {
	dependencies := toolDependencies{store: store, gitReader: gitReader}
	if provider, isProvider := gitReader.(WorkspaceRootProvider); isProvider {
		dependencies.workspaceRoot = strings.TrimSpace(provider.WorkspaceRoot())
	}

	return []agentdomain.Tool{
		searchCodeTool{dependencies: dependencies},
		grepRawFallbackTool{dependencies: dependencies},
		fileContentTool{dependencies: dependencies},
		symbolReferencesTool{dependencies: dependencies},
		listServicesTool{dependencies: dependencies},
		serviceRoutesTool{dependencies: dependencies},
		serviceDependenciesTool{dependencies: dependencies},
		serviceContractTool{dependencies: dependencies},
		routeDetailsTool{dependencies: dependencies},
		domainKnowledgeTool{dependencies: dependencies},
		gitDiffTool{dependencies: dependencies},
	}
}

func toolDefinition(name string, description string, parameters string, strict bool) agentdomain.ToolDefinition {
	return agentdomain.ToolDefinition{
		Type: "function",
		Function: agentdomain.ToolFunction{
			Name:        name,
			Description: description,
			Parameters:  json.RawMessage(parameters),
			Strict:      strict,
		},
	}
}

func (d toolDependencies) requireStore() (codeindexdomain.IndexStore, error) {
	if d.store == nil {
		return nil, errors.New("the code index store is not configured")
	}
	return d.store, nil
}

func (d toolDependencies) requireWorkspaceRoot() (string, error) {
	if d.workspaceRoot == "" {
		return "", errors.New("the workspace root is not configured")
	}
	return d.workspaceRoot, nil
}

func (d toolDependencies) requireGitReader() (CodeGitReader, error) {
	if d.gitReader == nil {
		return nil, errors.New("the git reader is not configured")
	}
	return d.gitReader, nil
}

func clampLimit(requested int, fallback int, maximum int) int {
	if requested <= 0 {
		return fallback
	}
	if requested > maximum {
		return maximum
	}
	return requested
}

func requireArgument(value string, argumentName string) (string, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", fmt.Errorf("argument %q is required", argumentName)
	}
	return trimmed, nil
}

func normalizeRepoFilter(rawFilter string) (string, error) {
	trimmed := strings.TrimSpace(rawFilter)
	if trimmed == "" {
		return "", nil
	}

	validName, err := workspacedomain.ValidateRepoName(trimmed)
	if err != nil {
		return "", fmt.Errorf("invalid repository filter %q: %w", trimmed, err)
	}
	return validName, nil
}

func repositoryFilterSlice(repoFilter string) []string {
	if repoFilter == "" {
		return nil
	}
	return []string{repoFilter}
}

func truncateText(text string, maximum int) (string, bool) {
	if maximum <= 0 {
		return "", text != ""
	}

	runes := []rune(text)
	if len(runes) <= maximum {
		return text, false
	}
	return string(runes[:maximum]), true
}

func clipToBytes(text string, maximum int) string {
	if maximum <= 0 {
		return ""
	}
	if len(text) <= maximum {
		return text
	}

	clipped := text[:maximum]
	for len(clipped) > 0 && !utf8.ValidString(clipped) {
		clipped = clipped[:len(clipped)-1]
	}
	return clipped
}

type boundedResultBuilder struct {
	builder   strings.Builder
	maximum   int
	truncated bool
}

func newBoundedResultBuilder(maximum int) *boundedResultBuilder {
	return &boundedResultBuilder{maximum: maximum}
}

func (b *boundedResultBuilder) write(text string) bool {
	if b.truncated {
		return false
	}

	remaining := b.maximum - b.builder.Len()
	if remaining <= 0 {
		b.truncated = true
		return false
	}
	if len(text) <= remaining {
		b.builder.WriteString(text)
		return true
	}

	b.builder.WriteString(clipToBytes(text, remaining))
	b.truncated = true
	return false
}

func (b *boundedResultBuilder) finish(note string) string {
	if !b.truncated {
		return b.builder.String()
	}
	return clipToBytes(b.builder.String(), b.maximum-len(note)) + note
}

func emptyToolResult(message string) agentdomain.ToolResult {
	return agentdomain.ToolResult{Content: message}
}

func sourceReferenceForChunk(chunk codeindexdomain.CodeChunk, snippet string) agentdomain.SourceReference {
	return agentdomain.SourceReference{
		Repo:      chunk.RepoName,
		File:      chunk.FilePath,
		Type:      chunk.ChunkType,
		Snippet:   snippet,
		StartLine: chunk.StartLine,
		EndLine:   chunk.EndLine,
	}
}

func lineRangeLabel(startLine int, endLine int) string {
	if startLine <= 0 {
		return ""
	}
	if endLine <= startLine {
		return fmt.Sprintf(" lines=%d", startLine)
	}
	return fmt.Sprintf(" lines=%d-%d", startLine, endLine)
}

func truncationLabel(isTruncated bool) string {
	if !isTruncated {
		return ""
	}
	return " truncated=true"
}

const (
	dependencyCategoryOutboundHTTP = "outbound_http"
	dependencyCategoryInboundHTTP  = "inbound_http"
	dependencyCategoryRabbitMQOut  = "rabbitmq_publish"
	dependencyCategoryRabbitMQIn   = "rabbitmq_consume"
	dependencyCategoryKafkaOut     = "kafka_publish"
	dependencyCategoryKafkaIn      = "kafka_subscribe"
	dependencyCategoryUnknown      = ""
)

func dependencySummary(dependency codeindexdomain.ServiceDependency) string {
	target := dependency.Endpoint
	if strings.TrimSpace(target) == "" {
		target = dependency.Topic
	}
	return fmt.Sprintf("- %s -> %s [%s] %s (file: %s)", dependency.FromService, dependency.ToService, dependency.CallType, target, dependency.SourceFile)
}

func dependencySourceReference(dependency codeindexdomain.ServiceDependency) agentdomain.SourceReference {
	return agentdomain.SourceReference{
		Repo:    dependency.FromService,
		File:    dependency.SourceFile,
		Type:    "dependency",
		Snippet: dependencySummary(dependency),
	}
}

func dependencyCategory(dependency codeindexdomain.ServiceDependency, serviceName string) string {
	isOutbound := dependency.FromService == serviceName
	switch dependency.CallType {
	case "http_client":
		if isOutbound {
			return dependencyCategoryOutboundHTTP
		}
		return dependencyCategoryInboundHTTP
	case "rabbitmq_publish":
		return dependencyCategoryRabbitMQOut
	case "rabbitmq_subscribe":
		return dependencyCategoryRabbitMQIn
	case "kafka_publish":
		return dependencyCategoryKafkaOut
	case "kafka_subscribe":
		return dependencyCategoryKafkaIn
	default:
		return dependencyCategoryUnknown
	}
}

func writeDependencySection(
	builder *boundedResultBuilder,
	title string,
	dependencies []codeindexdomain.ServiceDependency,
	serviceName string,
	categories ...string,
) {
	builder.write(title + "\n")
	written := 0
	for _, dependency := range dependencies {
		if !slices.Contains(categories, dependencyCategory(dependency, serviceName)) {
			continue
		}
		if !builder.write(dependencySummary(dependency) + "\n") {
			return
		}
		written++
	}
	if written == 0 {
		builder.write("None detected.\n")
	}
	builder.write("\n")
}
