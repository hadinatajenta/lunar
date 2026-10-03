package tools

import (
	"encoding/json"
	"slices"
	"strings"
	"testing"

	codeindexdomain "lunar/backend/internal/codeindex/domain"
)

func TestNewToolsExposesEveryRegisteredCodeTool(t *testing.T) {
	store := openIndexTestStore(t)
	tools := newTestTools(store, t.TempDir())

	expectedNames := []string{
		"search_code",
		"get_file_content",
		"get_route_details",
		"get_service_contract",
		"find_symbol_references",
		"get_service_routes",
		"get_service_dependencies",
		"list_services",
		"get_domain_knowledge",
		"get_git_diff",
		"grep_raw_fallback",
	}

	if len(tools) != len(expectedNames) {
		t.Fatalf("expected %d tools, got %d", len(expectedNames), len(tools))
	}

	seenNames := make(map[string]bool, len(tools))
	for _, tool := range tools {
		definition := tool.Definition()
		if definition.Type != "function" {
			t.Errorf("tool %q has type %q, expected function", definition.Function.Name, definition.Type)
		}
		if !slices.Contains(expectedNames, definition.Function.Name) {
			t.Errorf("unexpected tool name %q", definition.Function.Name)
		}
		if strings.TrimSpace(definition.Function.Description) == "" {
			t.Errorf("tool %q has an empty description", definition.Function.Name)
		}
		seenNames[definition.Function.Name] = true
	}

	for _, name := range expectedNames {
		if !seenNames[name] {
			t.Errorf("tool %q is missing from the registry", name)
		}
	}
}

func TestNewToolsDeclareObjectSchemasWithRequiredProperties(t *testing.T) {
	store := openIndexTestStore(t)
	tools := newTestTools(store, t.TempDir())

	expectedRequired := map[string][]string{
		"search_code":              {"query"},
		"get_file_content":         {"repo_name", "file_path"},
		"get_route_details":        {"service_name", "endpoint_path"},
		"get_service_contract":     {"service_name"},
		"find_symbol_references":   {"symbol"},
		"get_service_routes":       {"service_name"},
		"get_service_dependencies": {"service_name"},
		"list_services":            {},
		"get_domain_knowledge":     {"query"},
		"get_git_diff":             {"repo_name"},
		"grep_raw_fallback":        {"pattern"},
	}

	for name, required := range expectedRequired {
		definition := mustFindTool(t, tools, name).Definition()

		var schema struct {
			Type       string         `json:"type"`
			Properties map[string]any `json:"properties"`
			Required   []string       `json:"required"`
		}
		if err := json.Unmarshal(definition.Function.Parameters, &schema); err != nil {
			t.Fatalf("tool %q has invalid parameter JSON: %v", name, err)
		}
		if schema.Type != "object" {
			t.Errorf("tool %q declares schema type %q, expected object", name, schema.Type)
		}
		for _, property := range required {
			if _, exists := schema.Properties[property]; !exists {
				t.Errorf("tool %q requires %q but does not declare it", name, property)
			}
		}
		if len(required) == 0 {
			continue
		}
		if len(schema.Required) != len(required) {
			t.Errorf("tool %q declares required %v, expected %v", name, schema.Required, required)
		}
	}
}

func TestToolsWithoutWorkspaceRootFailWithClearErrors(t *testing.T) {
	store := openIndexTestStore(t)
	tools := NewTools(store, diffOnlyGitReader{})

	for _, name := range []string{ToolNameGetFileContent, ToolNameGrepRawFallback} {
		var arguments map[string]any
		switch name {
		case ToolNameGetFileContent:
			arguments = map[string]any{"repo_name": "payments-service", "file_path": "handler.go"}
		default:
			arguments = map[string]any{"pattern": "payment"}
		}

		_, err := executeTool(t, mustFindTool(t, tools, name), arguments)
		if err == nil {
			t.Fatalf("tool %q should fail when the workspace root is not configured", name)
		}
		if !strings.Contains(err.Error(), "workspace root is not configured") {
			t.Errorf("tool %q returned an unclear error: %v", name, err)
		}
	}
}

func TestToolsWithoutStoreFailWithClearErrors(t *testing.T) {
	tools := newTestTools(nil, t.TempDir())

	_, err := executeTool(t, mustFindTool(t, tools, ToolNameListServices), map[string]any{})
	if err == nil {
		t.Fatal("list_services should fail when the index store is not configured")
	}
	if !strings.Contains(err.Error(), "code index store is not configured") {
		t.Errorf("list_services returned an unclear error: %v", err)
	}
}

func TestToolResultLimitClampingNeverExceedsMaximum(t *testing.T) {
	cases := []struct {
		requested int
		fallback  int
		maximum   int
		expected  int
	}{
		{requested: 0, fallback: 8, maximum: 20, expected: 8},
		{requested: -3, fallback: 8, maximum: 20, expected: 8},
		{requested: 5, fallback: 8, maximum: 20, expected: 5},
		{requested: 1000, fallback: 8, maximum: 20, expected: 20},
	}

	for _, testCase := range cases {
		if clamped := clampLimit(testCase.requested, testCase.fallback, testCase.maximum); clamped != testCase.expected {
			t.Errorf("clampLimit(%d, %d, %d) = %d, expected %d",
				testCase.requested, testCase.fallback, testCase.maximum, clamped, testCase.expected)
		}
	}
}

func TestSourceReferenceKeepsChunkLineRange(t *testing.T) {
	chunk := codeindexdomain.CodeChunk{
		RepoName:   "payments-service",
		FilePath:   "internal/handler/routes.go",
		ChunkType:  "route",
		RawContent: "router.GET(\"/api/payments\", getPayments)",
		StartLine:  6,
		EndLine:    6,
	}

	source := sourceReferenceForChunk(chunk, chunk.RawContent)
	if source.StartLine != 6 || source.EndLine != 6 {
		t.Errorf("expected line range 6-6, got %d-%d", source.StartLine, source.EndLine)
	}

	chunkWithoutRange := codeindexdomain.CodeChunk{RepoName: "payments-service", FilePath: "routes.go", ChunkType: "route"}
	sourceWithoutRange := sourceReferenceForChunk(chunkWithoutRange, chunkWithoutRange.RawContent)
	if sourceWithoutRange.StartLine != 0 || sourceWithoutRange.EndLine != 0 {
		t.Errorf("expected an empty line range, got %d-%d", sourceWithoutRange.StartLine, sourceWithoutRange.EndLine)
	}
}
