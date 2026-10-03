package infrastructure

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestParseDSMLToolCalls_FullwidthDelimiters(t *testing.T) {
	snippet := `<｜｜DSML｜｜ calls>
<｜｜DSML｜｜ invoke name="get_file_content">
<｜｜DSML｜｜ parameter name="file_path" string="true">src/programMdr/delivery.go</｜｜DSML｜｜ parameter>
<｜｜DSML｜｜ parameter name="repo_name" string="true">aurora</｜｜DSML｜｜ parameter>
<｜｜DSML｜｜ parameter name="start_line" string="false">19</｜｜DSML｜｜ parameter>
<｜｜DSML｜｜ parameter name="end_line" string="false">62</｜｜DSML｜｜ parameter>
</｜｜DSML｜｜ invoke>
<｜｜DSML｜｜ invoke name="get_file_content">
<｜｜DSML｜｜ parameter name="file_path" string="true">src/programMdr/const.go</｜｜DSML｜｜ parameter>
<｜｜DSML｜｜ parameter name="repo_name" string="true">aurora</｜｜DSML｜｜ parameter>
</｜｜DSML｜｜ invoke>
<｜｜DSML｜｜ invoke name="get_file_content">
<｜｜DSML｜｜ parameter name="file_path" string="true">src/pages/mdr_maintenance/InputPerubahanMdrCcQris.vue</｜｜DSML｜｜ parameter>
<｜｜DSML｜｜ parameter name="repo_name" string="true">fe-mms</｜｜DSML｜｜ parameter>
</｜｜DSML｜｜ invoke>
</｜｜DSML｜｜ calls>`

	if !ContainsDSML(snippet) {
		t.Fatal("expected ContainsDSML to detect the fullwidth block")
	}

	calls, remaining := ParseDSMLToolCalls(snippet)
	if len(calls) != 3 {
		t.Fatalf("tool calls = %d, want 3", len(calls))
	}
	if remaining != "" {
		t.Errorf("remaining content = %q, want empty", remaining)
	}

	for index, call := range calls {
		if call.Function.Name != "get_file_content" {
			t.Errorf("call[%d] name = %q, want get_file_content", index, call.Function.Name)
		}
		if call.Type != "function" {
			t.Errorf("call[%d] type = %q, want function", index, call.Type)
		}
		if !strings.HasPrefix(call.ID, "call_dsml_") {
			t.Errorf("call[%d] id = %q, want the dsml prefix", index, call.ID)
		}
	}

	var firstArguments struct {
		FilePath  string `json:"file_path"`
		RepoName  string `json:"repo_name"`
		StartLine int    `json:"start_line"`
		EndLine   int    `json:"end_line"`
	}
	if err := json.Unmarshal([]byte(calls[0].Function.Arguments), &firstArguments); err != nil {
		t.Fatalf("unmarshal first arguments: %v", err)
	}
	if firstArguments.FilePath != "src/programMdr/delivery.go" || firstArguments.RepoName != "aurora" {
		t.Errorf("first arguments = %+v, want the file and repository", firstArguments)
	}
	if firstArguments.StartLine != 19 || firstArguments.EndLine != 62 {
		t.Errorf("first arguments = %+v, want numeric line bounds", firstArguments)
	}

	var thirdArguments struct {
		FilePath string `json:"file_path"`
		RepoName string `json:"repo_name"`
	}
	if err := json.Unmarshal([]byte(calls[2].Function.Arguments), &thirdArguments); err != nil {
		t.Fatalf("unmarshal third arguments: %v", err)
	}
	if thirdArguments.FilePath != "src/pages/mdr_maintenance/InputPerubahanMdrCcQris.vue" || thirdArguments.RepoName != "fe-mms" {
		t.Errorf("third arguments = %+v, want the vue file and fe-mms repository", thirdArguments)
	}
}

func TestParseDSMLToolCalls_SinglePipeAndSurroundingText(t *testing.T) {
	text := `I will check the following files first:
<|DSML|calls>
<|DSML|invoke name="search_code">
<|DSML|parameter name="query" string="true">payment process</|DSML|parameter>
<|DSML|parameter name="limit" string="false">5</|DSML|parameter>
</|DSML|invoke>
</|DSML|calls>
One moment...`

	calls, remaining := ParseDSMLToolCalls(text)
	if len(calls) != 1 {
		t.Fatalf("tool calls = %d, want 1", len(calls))
	}
	if calls[0].Function.Name != "search_code" {
		t.Errorf("name = %q, want search_code", calls[0].Function.Name)
	}

	var arguments struct {
		Query string `json:"query"`
		Limit int    `json:"limit"`
	}
	if err := json.Unmarshal([]byte(calls[0].Function.Arguments), &arguments); err != nil {
		t.Fatalf("unmarshal arguments: %v", err)
	}
	if arguments.Query != "payment process" {
		t.Errorf("query = %q, want the string parameter preserved", arguments.Query)
	}
	if arguments.Limit != 5 {
		t.Errorf("limit = %d, want the numeric parameter coerced", arguments.Limit)
	}

	if remaining != "I will check the following files first:\n\nOne moment..." {
		t.Errorf("remaining = %q, want the surrounding text preserved", remaining)
	}
}

func TestParseDSMLToolCalls_InvokeWithoutBlockWrapper(t *testing.T) {
	text := `Intro
<|DSML|invoke name="list_services">
</|DSML|invoke>
Outro`

	calls, remaining := ParseDSMLToolCalls(text)
	if len(calls) != 1 {
		t.Fatalf("tool calls = %d, want 1", len(calls))
	}
	if calls[0].Function.Name != "list_services" {
		t.Errorf("name = %q, want list_services", calls[0].Function.Name)
	}
	if calls[0].Function.Arguments != "{}" {
		t.Errorf("arguments = %q, want an empty object", calls[0].Function.Arguments)
	}
	if !strings.Contains(remaining, "Intro") || !strings.Contains(remaining, "Outro") {
		t.Errorf("remaining = %q, want the surrounding text preserved", remaining)
	}
}

func TestParseDSMLToolCalls_NonDSMLTextIsUntouched(t *testing.T) {
	text := "Plain answer with no tool markup."
	if ContainsDSML(text) {
		t.Fatal("plain text must not be detected as DSML")
	}
	calls, remaining := ParseDSMLToolCalls(text)
	if len(calls) != 0 {
		t.Errorf("tool calls = %d, want none", len(calls))
	}
	if remaining != text {
		t.Errorf("remaining = %q, want the input unchanged", remaining)
	}
}

func TestParseDSMLToolCalls_ScalarCoercion(t *testing.T) {
	text := `<|DSML|calls>
<|DSML|invoke name="search_code">
<|DSML|parameter name="strict" string="false">true</|DSML|parameter>
<|DSML|parameter name="ratio" string="false">1.5</|DSML|parameter>
<|DSML|parameter name="filters" string="false">{"repo":"aurora"}</|DSML|parameter>
<|DSML|parameter name="raw" string="false">not-json</|DSML|parameter>
<|DSML|parameter name="label">plain</|DSML|parameter>
</|DSML|invoke>
</|DSML|calls>`

	calls, _ := ParseDSMLToolCalls(text)
	if len(calls) != 1 {
		t.Fatalf("tool calls = %d, want 1", len(calls))
	}

	var arguments struct {
		Strict  bool                   `json:"strict"`
		Ratio   float64                `json:"ratio"`
		Filters map[string]interface{} `json:"filters"`
		Raw     string                 `json:"raw"`
		Label   string                 `json:"label"`
	}
	if err := json.Unmarshal([]byte(calls[0].Function.Arguments), &arguments); err != nil {
		t.Fatalf("unmarshal arguments: %v", err)
	}
	if !arguments.Strict {
		t.Error("strict must be coerced to a boolean")
	}
	if arguments.Ratio != 1.5 {
		t.Errorf("ratio = %v, want 1.5", arguments.Ratio)
	}
	if arguments.Filters["repo"] != "aurora" {
		t.Errorf("filters = %v, want the nested JSON object", arguments.Filters)
	}
	if arguments.Raw != "not-json" {
		t.Errorf("raw = %q, want the unparsable scalar kept as text", arguments.Raw)
	}
	if arguments.Label != "plain" {
		t.Errorf("label = %q, want the string parameter preserved", arguments.Label)
	}
}

func TestContainsDSML(t *testing.T) {
	if ContainsDSML("no markup here") {
		t.Error("plain text must not be detected")
	}
	if ContainsDSML("DSML marker without a verb") {
		t.Error("a lone DSML marker must not be detected")
	}
	if !ContainsDSML(`<|DSML|invoke name="x">`) {
		t.Error("an invoke tag must be detected")
	}
	if !ContainsDSML(`<|DSML|calls>`) {
		t.Error("a calls tag must be detected")
	}
}
