package tools

import (
	"context"
	"encoding/json"
	"fmt"

	agentdomain "lunar/backend/internal/agent/domain"
)

const knowledgeTruncationNote = "\n[result truncated]\n"

type domainKnowledgeArguments struct {
	Query string `json:"query"`
	Limit int    `json:"limit"`
}

type domainKnowledgeTool struct {
	dependencies toolDependencies
}

func (t domainKnowledgeTool) Definition() agentdomain.ToolDefinition {
	return toolDefinition(ToolNameGetDomainKnowledge, getDomainKnowledgeDescription, getDomainKnowledgeSchema, true)
}

func (t domainKnowledgeTool) Execute(ctx context.Context, args json.RawMessage) (agentdomain.ToolResult, error) {
	var arguments domainKnowledgeArguments
	if err := json.Unmarshal(args, &arguments); err != nil {
		return agentdomain.ToolResult{}, fmt.Errorf("get_domain_knowledge: invalid arguments: %w", err)
	}

	query, err := requireArgument(arguments.Query, "query")
	if err != nil {
		return agentdomain.ToolResult{}, fmt.Errorf("get_domain_knowledge: %w", err)
	}
	store, err := t.dependencies.requireStore()
	if err != nil {
		return agentdomain.ToolResult{}, fmt.Errorf("get_domain_knowledge: %w", err)
	}

	limit := clampLimit(arguments.Limit, defaultKnowledgeLimit, maxKnowledgeLimit)
	items, err := store.SearchKnowledge(ctx, query, limit)
	if err != nil {
		return agentdomain.ToolResult{}, fmt.Errorf("get_domain_knowledge: %w", err)
	}
	if len(items) == 0 {
		return emptyToolResult(fmt.Sprintf("No domain knowledge note matched the query %q.", query)), nil
	}

	builder := newBoundedResultBuilder(maxResultCharacters)
	builder.write(fmt.Sprintf("Domain knowledge for query %q (%d notes):\n\n", query, len(items)))
	for _, item := range items {
		builder.write(fmt.Sprintf("### %s\n%s\n\n", item.Term, item.Explanation))
	}
	return agentdomain.ToolResult{Content: builder.finish(knowledgeTruncationNote)}, nil
}
