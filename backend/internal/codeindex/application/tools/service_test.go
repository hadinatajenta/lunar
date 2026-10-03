package tools

import (
	"context"
	"strings"
	"testing"

	codeindexdomain "lunar/backend/internal/codeindex/domain"
	codeindexinfrastructure "lunar/backend/internal/codeindex/infrastructure"
)

const serviceManifestFixture = `package handler

import "github.com/gin-gonic/gin"

func RegisterRoutes(router *gin.Engine) {
	router.GET("/api/payments", getPayments)
	router.POST("/api/payments", createPayment)
}
`

func seedServiceIndex(t *testing.T, store codeindexdomain.IndexStore, repoName string) {
	t.Helper()

	chunks, _ := codeindexinfrastructure.ExtractFile(
		serviceManifestFixture,
		"internal/handler/routes.go",
		repoName,
		".go",
	)
	if len(chunks) == 0 {
		t.Fatal("the service manifest fixture did not produce any chunk")
	}
	if err := store.SaveChunks(context.Background(), repoName, chunks); err != nil {
		t.Fatalf("cannot seed chunks for %s: %v", repoName, err)
	}

	dependencies := []codeindexdomain.ServiceDependency{
		{
			FromService: repoName,
			ToService:   "ledger-service",
			CallType:    "http_client",
			Endpoint:    "/api/ledger/entries",
			SourceFile:  "internal/client/ledger.go",
		},
		{
			FromService: repoName,
			ToService:   "notification-service",
			CallType:    "kafka_publish",
			Topic:       "payment-events",
			SourceFile:  "internal/event/publisher.go",
		},
		{
			FromService: "settlement-service",
			ToService:   repoName,
			CallType:    "rabbitmq_subscribe",
			Topic:       "payment-settled",
			SourceFile:  "internal/consumer/settlement.go",
		},
	}
	if err := store.SaveDependencies(context.Background(), repoName, dependencies); err != nil {
		t.Fatalf("cannot seed dependencies for %s: %v", repoName, err)
	}
}

func TestListServicesReturnsIndexedRepositories(t *testing.T) {
	store := openIndexTestStore(t)
	saveExtractedChunks(t, store, "payments-service", "main.go", "package main\n\nfunc main() {}\n", ".go")
	saveExtractedChunks(t, store, "ledger-service", "main.go", "package main\n\nfunc main() {}\n", ".go")
	tools := newTestTools(store, t.TempDir())

	result, err := executeTool(t, mustFindTool(t, tools, ToolNameListServices), map[string]any{})
	if err != nil {
		t.Fatalf("list_services failed: %v", err)
	}
	for _, expected := range []string{"payments-service", "ledger-service", "code chunks"} {
		if !strings.Contains(result.Content, expected) {
			t.Errorf("expected %q in the service list:\n%s", expected, result.Content)
		}
	}
}

func TestGetServiceRoutesListsOnlyRoutesWithLineRanges(t *testing.T) {
	store := openIndexTestStore(t)
	seedServiceIndex(t, store, "payments-service")
	tools := newTestTools(store, t.TempDir())

	result, err := executeTool(t, mustFindTool(t, tools, ToolNameGetServiceRoutes), map[string]any{"service_name": "payments-service"})
	if err != nil {
		t.Fatalf("get_service_routes failed: %v", err)
	}
	for _, expected := range []string{`router.GET("/api/payments", getPayments)`, `router.POST("/api/payments", createPayment)`} {
		if !strings.Contains(result.Content, expected) {
			t.Errorf("expected %q in the route list:\n%s", expected, result.Content)
		}
	}
	if len(result.Sources) != 2 {
		t.Fatalf("expected two route sources, got %d", len(result.Sources))
	}
	for _, source := range result.Sources {
		if source.Type != "route" {
			t.Errorf("expected a route source, got type %q", source.Type)
		}
		if source.StartLine < 1 || source.EndLine < source.StartLine {
			t.Errorf("expected a real line range for %s, got %d-%d", source.File, source.StartLine, source.EndLine)
		}
	}
}

func TestGetServiceDependenciesDescribesOutboundAndInboundEdges(t *testing.T) {
	store := openIndexTestStore(t)
	seedServiceIndex(t, store, "payments-service")
	tools := newTestTools(store, t.TempDir())

	result, err := executeTool(t, mustFindTool(t, tools, ToolNameGetServiceDependencies), map[string]any{"service_name": "payments-service"})
	if err != nil {
		t.Fatalf("get_service_dependencies failed: %v", err)
	}
	for _, expected := range []string{
		"payments-service -> ledger-service [http_client] /api/ledger/entries",
		"payments-service -> notification-service [kafka_publish] payment-events",
		"settlement-service -> payments-service [rabbitmq_subscribe] payment-settled",
	} {
		if !strings.Contains(result.Content, expected) {
			t.Errorf("expected %q in the dependency list:\n%s", expected, result.Content)
		}
	}
	if findSource(result.Sources, "internal/client/ledger.go", "dependency") == nil {
		t.Errorf("expected a dependency source for the ledger client, got %+v", result.Sources)
	}
}

func TestGetServiceContractGroupsRoutesAndMessaging(t *testing.T) {
	store := openIndexTestStore(t)
	seedServiceIndex(t, store, "payments-service")
	tools := newTestTools(store, t.TempDir())

	result, err := executeTool(t, mustFindTool(t, tools, ToolNameGetServiceContract), map[string]any{"service_name": "payments-service"})
	if err != nil {
		t.Fatalf("get_service_contract failed: %v", err)
	}
	for _, expected := range []string{
		"# Service contract: `payments-service`",
		"## 1. Inbound HTTP routes (2)",
		"ledger-service",
		"## 2. Outbound HTTP calls",
		"## 3. Outbound messaging",
		"payment-events",
		"## 4. Inbound messaging",
		"payment-settled",
	} {
		if !strings.Contains(result.Content, expected) {
			t.Errorf("expected %q in the service contract:\n%s", expected, result.Content)
		}
	}
	if len(result.Sources) != 2 {
		t.Errorf("expected two route sources, got %d", len(result.Sources))
	}
}

func TestGetRouteDetailsReturnsDefinitionExcerptAndDependencies(t *testing.T) {
	workspaceRoot := t.TempDir()
	writeWorkspaceFile(t, workspaceRoot, "payments-service/internal/handler/routes.go", serviceManifestFixture)

	store := openIndexTestStore(t)
	seedServiceIndex(t, store, "payments-service")
	tools := newTestTools(store, workspaceRoot)

	result, err := executeTool(t, mustFindTool(t, tools, ToolNameGetRouteDetails), map[string]any{
		"service_name":  "payments-service",
		"endpoint_path": "/api/payments",
		"method":        "get",
	})
	if err != nil {
		t.Fatalf("get_route_details failed: %v", err)
	}
	for _, expected := range []string{
		"## Route details: GET /api/payments",
		"### Route definition",
		"### Router file excerpt",
		"### Downstream and messaging dependencies",
		"ledger-service",
	} {
		if !strings.Contains(result.Content, expected) {
			t.Errorf("expected %q in the route details:\n%s", expected, result.Content)
		}
	}
	if findSource(result.Sources, "internal/handler/routes.go", "route") == nil {
		t.Errorf("expected a route source, got %+v", result.Sources)
	}
	if findSource(result.Sources, "internal/handler/routes.go", "file_content") == nil {
		t.Errorf("expected a router file excerpt source, got %+v", result.Sources)
	}
}

func TestGetRouteDetailsReportsUnknownRoute(t *testing.T) {
	store := openIndexTestStore(t)
	seedServiceIndex(t, store, "payments-service")
	tools := newTestTools(store, t.TempDir())

	result, err := executeTool(t, mustFindTool(t, tools, ToolNameGetRouteDetails), map[string]any{
		"service_name":  "payments-service",
		"endpoint_path": "/api/unknown",
	})
	if err != nil {
		t.Fatalf("get_route_details failed: %v", err)
	}
	if !strings.Contains(result.Content, "No route definition") {
		t.Errorf("expected a missing-route message, got:\n%s", result.Content)
	}
}

func TestServiceToolsRejectBlankServiceName(t *testing.T) {
	store := openIndexTestStore(t)
	tools := newTestTools(store, t.TempDir())

	for _, name := range []string{
		ToolNameGetServiceRoutes,
		ToolNameGetServiceDependencies,
		ToolNameGetServiceContract,
		ToolNameGetRouteDetails,
	} {
		arguments := map[string]any{"service_name": "   "}
		if name == ToolNameGetRouteDetails {
			arguments["endpoint_path"] = "/api/payments"
		}
		if _, err := executeTool(t, mustFindTool(t, tools, name), arguments); err == nil {
			t.Errorf("tool %q should reject a blank service name", name)
		}
	}
}
