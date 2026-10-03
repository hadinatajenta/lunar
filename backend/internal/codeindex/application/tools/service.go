package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	agentdomain "lunar/backend/internal/agent/domain"
	codeindexdomain "lunar/backend/internal/codeindex/domain"
)

const (
	serviceRouteChunkType  = "route"
	serviceChunkQueryLimit = 100
	serviceTruncationNote  = "\n[result truncated]\n"
)

type serviceNameArguments struct {
	ServiceName string `json:"service_name"`
}

type routeDetailsArguments struct {
	ServiceName  string `json:"service_name"`
	EndpointPath string `json:"endpoint_path"`
	Method       string `json:"method"`
}

type listServicesTool struct {
	dependencies toolDependencies
}

func (t listServicesTool) Definition() agentdomain.ToolDefinition {
	return toolDefinition(ToolNameListServices, listServicesDescription, listServicesSchema, false)
}

func (t listServicesTool) Execute(ctx context.Context, _ json.RawMessage) (agentdomain.ToolResult, error) {
	store, err := t.dependencies.requireStore()
	if err != nil {
		return agentdomain.ToolResult{}, fmt.Errorf("list_services: %w", err)
	}

	services, err := store.GetAllServices(ctx)
	if err != nil {
		return agentdomain.ToolResult{}, fmt.Errorf("list_services: %w", err)
	}
	if len(services) == 0 {
		return emptyToolResult("No service has been indexed yet."), nil
	}
	if len(services) > maxServiceLimit {
		services = services[:maxServiceLimit]
	}

	builder := newBoundedResultBuilder(maxResultCharacters)
	builder.write(fmt.Sprintf("Indexed services (%d):\n\n", len(services)))
	for _, service := range services {
		builder.write(fmt.Sprintf("- %s (%s, %d code chunks)\n", service.RepoName, service.Language, service.ChunkCount))
	}
	return agentdomain.ToolResult{Content: builder.finish(serviceTruncationNote)}, nil
}

type serviceRoutesTool struct {
	dependencies toolDependencies
}

func (t serviceRoutesTool) Definition() agentdomain.ToolDefinition {
	return toolDefinition(ToolNameGetServiceRoutes, getServiceRoutesDescription, getServiceRoutesSchema, true)
}

func (t serviceRoutesTool) Execute(ctx context.Context, args json.RawMessage) (agentdomain.ToolResult, error) {
	var arguments serviceNameArguments
	if err := json.Unmarshal(args, &arguments); err != nil {
		return agentdomain.ToolResult{}, fmt.Errorf("get_service_routes: invalid arguments: %w", err)
	}

	serviceName, store, err := t.dependencies.resolveService(arguments, "get_service_routes")
	if err != nil {
		return agentdomain.ToolResult{}, err
	}

	chunks, err := loadServiceChunks(ctx, store, serviceName)
	if err != nil {
		return agentdomain.ToolResult{}, fmt.Errorf("get_service_routes: %w", err)
	}
	return routesResult(serviceName, filterRouteChunks(chunks)), nil
}

type serviceDependenciesTool struct {
	dependencies toolDependencies
}

func (t serviceDependenciesTool) Definition() agentdomain.ToolDefinition {
	return toolDefinition(ToolNameGetServiceDependencies, getServiceDependenciesDescription, getServiceDependenciesSchema, true)
}

func (t serviceDependenciesTool) Execute(ctx context.Context, args json.RawMessage) (agentdomain.ToolResult, error) {
	var arguments serviceNameArguments
	if err := json.Unmarshal(args, &arguments); err != nil {
		return agentdomain.ToolResult{}, fmt.Errorf("get_service_dependencies: invalid arguments: %w", err)
	}

	serviceName, store, err := t.dependencies.resolveService(arguments, "get_service_dependencies")
	if err != nil {
		return agentdomain.ToolResult{}, err
	}

	dependencies, err := store.GetServiceDependencies(ctx, serviceName)
	if err != nil {
		return agentdomain.ToolResult{}, fmt.Errorf("get_service_dependencies: %w", err)
	}
	if len(dependencies) > maxDependencyLimit {
		dependencies = dependencies[:maxDependencyLimit]
	}

	builder := newBoundedResultBuilder(maxResultCharacters)
	builder.write(fmt.Sprintf("Dependencies of service %q (%d):\n\n", serviceName, len(dependencies)))
	sources := make([]agentdomain.SourceReference, 0, len(dependencies))
	for _, dependency := range dependencies {
		if !builder.write(dependencySummary(dependency) + "\n") {
			break
		}
		if dependency.SourceFile != "" {
			sources = append(sources, dependencySourceReference(dependency))
		}
	}
	return agentdomain.ToolResult{Content: builder.finish(serviceTruncationNote), Sources: sources}, nil
}

type serviceContractTool struct {
	dependencies toolDependencies
}

func (t serviceContractTool) Definition() agentdomain.ToolDefinition {
	return toolDefinition(ToolNameGetServiceContract, getServiceContractDescription, getServiceContractSchema, true)
}

func (t serviceContractTool) Execute(ctx context.Context, args json.RawMessage) (agentdomain.ToolResult, error) {
	var arguments serviceNameArguments
	if err := json.Unmarshal(args, &arguments); err != nil {
		return agentdomain.ToolResult{}, fmt.Errorf("get_service_contract: invalid arguments: %w", err)
	}

	serviceName, store, err := t.dependencies.resolveService(arguments, "get_service_contract")
	if err != nil {
		return agentdomain.ToolResult{}, err
	}

	chunks, err := loadServiceChunks(ctx, store, serviceName)
	if err != nil {
		return agentdomain.ToolResult{}, fmt.Errorf("get_service_contract: %w", err)
	}
	dependencies, err := store.GetServiceDependencies(ctx, serviceName)
	if err != nil {
		return agentdomain.ToolResult{}, fmt.Errorf("get_service_contract: %w", err)
	}

	routes := filterRouteChunks(chunks)
	if len(routes) > maxRouteLimit {
		routes = routes[:maxRouteLimit]
	}

	builder := newBoundedResultBuilder(maxResultCharacters)
	builder.write(fmt.Sprintf("# Service contract: `%s`\n\n", serviceName))
	builder.write(fmt.Sprintf("## 1. Inbound HTTP routes (%d)\n", len(routes)))
	if len(routes) == 0 {
		builder.write("No HTTP route is indexed; the service may be a background worker or a consumer.\n\n")
	} else {
		for _, route := range routes {
			builder.write(fmt.Sprintf("- `%s` (file: `%s`)\n", compactText(route.RawContent), route.FilePath))
		}
		builder.write("\n")
	}
	writeDependencySection(builder, "## 2. Outbound HTTP calls", dependencies, serviceName, dependencyCategoryOutboundHTTP)
	writeDependencySection(builder, "## 3. Outbound messaging", dependencies, serviceName, dependencyCategoryRabbitMQOut, dependencyCategoryKafkaOut)
	writeDependencySection(builder, "## 4. Inbound messaging", dependencies, serviceName, dependencyCategoryRabbitMQIn, dependencyCategoryKafkaIn)

	return agentdomain.ToolResult{Content: builder.finish(serviceTruncationNote), Sources: routeSources(routes)}, nil
}

type routeDetailsTool struct {
	dependencies toolDependencies
}

func (t routeDetailsTool) Definition() agentdomain.ToolDefinition {
	return toolDefinition(ToolNameGetRouteDetails, getRouteDetailsDescription, getRouteDetailsSchema, true)
}

func (t routeDetailsTool) Execute(ctx context.Context, args json.RawMessage) (agentdomain.ToolResult, error) {
	var arguments routeDetailsArguments
	if err := json.Unmarshal(args, &arguments); err != nil {
		return agentdomain.ToolResult{}, fmt.Errorf("get_route_details: invalid arguments: %w", err)
	}

	serviceName, store, err := t.dependencies.resolveService(serviceNameArguments{ServiceName: arguments.ServiceName}, "get_route_details")
	if err != nil {
		return agentdomain.ToolResult{}, err
	}
	endpointPath, err := requireArgument(arguments.EndpointPath, "endpoint_path")
	if err != nil {
		return agentdomain.ToolResult{}, fmt.Errorf("get_route_details: %w", err)
	}
	method := strings.ToUpper(strings.TrimSpace(arguments.Method))
	if method == "" {
		method = "HTTP"
	}

	searched, err := store.SearchChunksIn(ctx, endpointPath, []string{serviceName}, 10)
	if err != nil {
		return agentdomain.ToolResult{}, fmt.Errorf("get_route_details: %w", err)
	}
	target := findRoute(filterRouteChunks(searched), endpointPath)
	if target == nil {
		allChunks, err := loadServiceChunks(ctx, store, serviceName)
		if err != nil {
			return agentdomain.ToolResult{}, fmt.Errorf("get_route_details: %w", err)
		}
		target = findRoute(filterRouteChunks(allChunks), endpointPath)
	}

	builder := newBoundedResultBuilder(maxResultCharacters)
	builder.write(fmt.Sprintf("## Route details: %s %s (service: %s)\n\n", method, endpointPath, serviceName))

	var sources []agentdomain.SourceReference
	if target == nil {
		builder.write(fmt.Sprintf("No route definition for `%s` was found in the index of service `%s`.\n\n", endpointPath, serviceName))
	} else {
		text, _ := truncateText(target.RawContent, maxChunkCharacters)
		builder.write(fmt.Sprintf("### Route definition\n- file: `%s`%s\n```\n%s\n```\n\n",
			target.FilePath, lineRangeLabel(target.StartLine, target.EndLine), target.RawContent))
		sources = append(sources, sourceReferenceForChunk(*target, text))
		sources = append(sources, writeRouterExcerpt(ctx, builder, t.dependencies.workspaceRoot, serviceName, target.FilePath)...)
	}

	dependencies, err := store.GetServiceDependencies(ctx, serviceName)
	if err != nil {
		return agentdomain.ToolResult{}, fmt.Errorf("get_route_details: %w", err)
	}
	writeDependencySection(builder, "### Downstream and messaging dependencies", dependencies, serviceName,
		dependencyCategoryOutboundHTTP, dependencyCategoryRabbitMQOut,
		dependencyCategoryKafkaOut, dependencyCategoryRabbitMQIn, dependencyCategoryKafkaIn)

	return agentdomain.ToolResult{Content: builder.finish(serviceTruncationNote), Sources: sources}, nil
}

func (d toolDependencies) resolveService(arguments serviceNameArguments, toolName string) (string, codeindexdomain.IndexStore, error) {
	serviceName, err := requireArgument(arguments.ServiceName, "service_name")
	if err != nil {
		return "", nil, fmt.Errorf("%s: %w", toolName, err)
	}
	store, err := d.requireStore()
	if err != nil {
		return "", nil, fmt.Errorf("%s: %w", toolName, err)
	}
	return serviceName, store, nil
}

func loadServiceChunks(ctx context.Context, store codeindexdomain.IndexStore, serviceName string) ([]codeindexdomain.CodeChunk, error) {
	return store.SearchChunksIn(ctx, "", []string{serviceName}, serviceChunkQueryLimit)
}

func routesResult(serviceName string, routes []codeindexdomain.CodeChunk) agentdomain.ToolResult {
	if len(routes) == 0 {
		return emptyToolResult(fmt.Sprintf("No HTTP route is indexed for service %q.", serviceName))
	}
	if len(routes) > maxRouteLimit {
		routes = routes[:maxRouteLimit]
	}

	builder := newBoundedResultBuilder(maxResultCharacters)
	builder.write(fmt.Sprintf("Routes exposed by service %q (%d):\n\n", serviceName, len(routes)))
	sources := make([]agentdomain.SourceReference, 0, len(routes))
	for _, route := range routes {
		snippet := compactText(route.RawContent)
		entry := fmt.Sprintf("- %s (file: %s)%s\n", snippet, route.FilePath, lineRangeLabel(route.StartLine, route.EndLine))
		if !builder.write(entry) {
			break
		}
		sources = append(sources, sourceReferenceForChunk(route, snippet))
	}
	return agentdomain.ToolResult{Content: builder.finish(serviceTruncationNote), Sources: sources}
}

func routeSources(routes []codeindexdomain.CodeChunk) []agentdomain.SourceReference {
	sources := make([]agentdomain.SourceReference, 0, len(routes))
	for _, route := range routes {
		sources = append(sources, sourceReferenceForChunk(route, compactText(route.RawContent)))
	}
	return sources
}

func compactText(text string) string {
	return strings.Join(strings.Fields(text), " ")
}
