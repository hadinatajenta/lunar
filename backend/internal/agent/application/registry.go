package application

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"lunar/backend/internal/agent/domain"
)

const (
	DomainServiceMap  = "servicemap"
	DomainJira        = "jira"
	DomainBitbucket   = "bitbucket"
	DomainQueryReview = "queryreview"
)

var toolDomainMap = map[string]string{
	"search_code":              DomainServiceMap,
	"get_file_content":         DomainServiceMap,
	"get_route_details":        DomainServiceMap,
	"get_service_contract":     DomainServiceMap,
	"find_symbol_references":   DomainServiceMap,
	"get_service_routes":       DomainServiceMap,
	"get_service_dependencies": DomainServiceMap,
	"list_services":            DomainServiceMap,
	"get_domain_knowledge":     DomainServiceMap,
	"get_git_diff":             DomainServiceMap,
	"grep_raw_fallback":        DomainServiceMap,

	"list_my_jira_issues":      DomainJira,
	"get_jira_issue_detail":    DomainJira,
	"get_active_sprint_issues": DomainJira,
	"create_jira_subtask":      DomainJira,
	"update_jira_subtask":      DomainJira,
	"transition_jira_issue":    DomainJira,

	"get_pull_request_diff":    DomainBitbucket,
	"get_query_review_context": DomainQueryReview,
}

type MapToolRegistry struct {
	mu          sync.RWMutex
	tools       map[string]domain.Tool
	definitions []domain.ToolDefinition
}

var _ domain.ToolRegistry = (*MapToolRegistry)(nil)

func NewMapToolRegistry() *MapToolRegistry {
	return &MapToolRegistry{tools: make(map[string]domain.Tool)}
}

func (r *MapToolRegistry) Register(tool domain.Tool) {
	if tool == nil {
		return
	}
	definition := tool.Definition()
	name := definition.Function.Name

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.tools[name]; exists {
		r.tools[name] = tool
		for index, registered := range r.definitions {
			if registered.Function.Name == name {
				r.definitions[index] = definition
				return
			}
		}
		return
	}

	r.tools[name] = tool
	r.definitions = append(r.definitions, definition)
}

func (r *MapToolRegistry) RegisterAll(tools ...domain.Tool) {
	for _, tool := range tools {
		r.Register(tool)
	}
}

func (r *MapToolRegistry) Definitions() []domain.ToolDefinition {
	r.mu.RLock()
	defer r.mu.RUnlock()

	definitions := make([]domain.ToolDefinition, len(r.definitions))
	copy(definitions, r.definitions)
	return definitions
}

func (r *MapToolRegistry) ScopedDefinitions(domains []string) []domain.ToolDefinition {
	if len(domains) == 0 {
		return r.Definitions()
	}

	allowedDomains := make(map[string]bool, len(domains))
	for _, domainName := range domains {
		allowedDomains[strings.ToLower(strings.TrimSpace(domainName))] = true
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	var scoped []domain.ToolDefinition
	for _, definition := range r.definitions {
		toolDomain, exists := toolDomainMap[definition.Function.Name]
		if exists && allowedDomains[toolDomain] {
			scoped = append(scoped, definition)
		}
	}
	return scoped
}

func (r *MapToolRegistry) Execute(ctx context.Context, name string, args json.RawMessage) (domain.ToolResult, error) {
	tool, exists := r.Get(name)
	if !exists {
		return domain.ToolResult{}, fmt.Errorf("tool not found: %s", name)
	}

	result, err := tool.Execute(ctx, args)
	if err != nil {
		return domain.ToolResult{}, fmt.Errorf("execute tool %s: %w", name, err)
	}
	return result, nil
}

func (r *MapToolRegistry) Get(name string) (domain.Tool, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tool, exists := r.tools[name]
	return tool, exists
}
