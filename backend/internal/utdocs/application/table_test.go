package application

import (
	"strings"
	"testing"

	"lunar/backend/internal/utdocs/domain"
)

func TestBuildNormalTestScenarioTableForASingleRepository(t *testing.T) {
	rows := []domain.DocumentRow{
		{Repository: "micro-payment", Function: "Find payment", Scenario: "Mencari data", ExpectedResult: "Data ditemukan", InputData: "ID", Result: domain.ResultPassed, ScreenCapture: domain.ScreenCaptureMarker},
		{Repository: "micro-payment", Function: "Save payment", Scenario: "Menyimpan data", ExpectedResult: "Data tersimpan", InputData: "Payload", Result: domain.ResultFailed, ScreenCapture: domain.ScreenCaptureMarker},
	}

	table := BuildNormalTestScenarioTable(rows)
	if !strings.HasPrefix(table, normalTableMarker) {
		t.Fatalf("expected the marker paragraph first, got %q", table)
	}
	if strings.Contains(table, "Layanan / Microservice") {
		t.Fatalf("expected no group row for a single repository, got %q", table)
	}
	if !strings.Contains(table, `<tr><td>1</td>`) || !strings.Contains(table, `<tr><td>2</td>`) {
		t.Fatalf("expected the sequential numbering, got %q", table)
	}
	for _, header := range []string{"<th>No</th>", "<th>Function</th>", "<th>Scenario</th>", "<th>Expected Result</th>", "<th>Input Data</th>", "<th>Result</th>", "<th>Screen Capture</th>"} {
		if !strings.Contains(table, header) {
			t.Fatalf("expected the header %q, got %q", header, table)
		}
	}
	if !strings.Contains(table, "Mencari data") || !strings.Contains(table, domain.ResultPassed) {
		t.Fatalf("expected the row values, got %q", table)
	}
}

func TestBuildNormalTestScenarioTableGroupsMultipleRepositoriesAndRestartsNumbering(t *testing.T) {
	rows := []domain.DocumentRow{
		{Repository: "micro-payment", Function: "A", Result: domain.ResultPassed},
		{Repository: "micro-payment", Function: "B", Result: domain.ResultPassed},
		{Repository: "micro-transfer", Function: "C", Result: domain.ResultSkipped},
	}

	table := BuildNormalTestScenarioTable(rows)
	if strings.Count(table, "Layanan / Microservice") != 2 {
		t.Fatalf("expected two group rows, got %q", table)
	}
	if strings.Count(table, `<tr><td>1</td>`) != 2 {
		t.Fatalf("expected the numbering to restart per group, got %q", table)
	}
	if !strings.Contains(table, "<code>micro-payment</code>") || !strings.Contains(table, "<code>micro-transfer</code>") {
		t.Fatalf("expected the repository group names, got %q", table)
	}
}

func TestBuildNormalTestScenarioTableForEmptyRows(t *testing.T) {
	table := BuildNormalTestScenarioTable(nil)
	if !strings.Contains(table, `<tr><td colspan="7">`+noScenarioMessage+`</td></tr>`) {
		t.Fatalf("expected the empty message row, got %q", table)
	}
	if !strings.Contains(table, `<tr><th>No</th>`) {
		t.Fatalf("expected the table header, got %q", table)
	}
}

func TestBuildNormalTestScenarioTableEscapesCells(t *testing.T) {
	rows := []domain.DocumentRow{
		{Repository: "micro-payment", Function: "<script>alert(1)</script>", Scenario: "a & b < c", Result: domain.ResultPassed},
	}

	table := BuildNormalTestScenarioTable(rows)
	if strings.Contains(table, "<script>") {
		t.Fatalf("expected the cell to be escaped, got %q", table)
	}
	if !strings.Contains(table, "&lt;script&gt;") || !strings.Contains(table, "a &amp; b &lt; c") {
		t.Fatalf("expected the escaped values, got %q", table)
	}
}

func TestBuildNormalTestScenarioTableKeepsTheAttachmentMarkupUnescaped(t *testing.T) {
	cell := BuildScreenCaptureCell("ut-micro-payment.png")
	if cell == "" {
		t.Fatal("expected a screen capture cell")
	}
	rows := []domain.DocumentRow{
		{Repository: "micro-payment", Function: "A", Result: domain.ResultPassed, ScreenCapture: cell},
	}

	table := BuildNormalTestScenarioTable(rows)
	if !strings.Contains(table, cell) {
		t.Fatalf("expected the attachment markup, got %q", table)
	}

	markerRows := []domain.DocumentRow{{Repository: "micro-payment", Function: "A", Result: domain.ResultPassed, ScreenCapture: domain.ScreenCaptureMarker}}
	markerTable := BuildNormalTestScenarioTable(markerRows)
	if !strings.Contains(markerTable, "[[screen-capture]]") || strings.Contains(markerTable, "<ac:image>") {
		t.Fatalf("expected the marker to be escaped text, got %q", markerTable)
	}
}

func TestBuildScreenCaptureCellRejectsUnsafeFilenames(t *testing.T) {
	for _, filename := range []string{"", "  ", "ut.png\" onload=\"x", "../../etc/passwd.png", "ut.txt", strings.Repeat("a", 300) + ".png"} {
		if cell := BuildScreenCaptureCell(filename); cell != "" {
			t.Fatalf("expected %q to be rejected, got %q", filename, cell)
		}
	}
	if cell := BuildScreenCaptureCell("ut-micro-payment.png"); cell != `<ac:image ac:width="400"><ri:attachment ri:filename="ut-micro-payment.png"/></ac:image>` {
		t.Fatalf("expected the attachment cell, got %q", cell)
	}
}

func TestReplaceNormalTestScenarioTableReplacesTheFollowingTable(t *testing.T) {
	storage := `<p>intro</p>` + "\n" + normalTableMarker + "\n" + `<table class="old"><tbody><tr><td>old</td></tr></tbody></table>` + "\n<p>after</p>"
	table := BuildNormalTestScenarioTable([]domain.DocumentRow{{Repository: "micro-payment", Function: "A", Result: domain.ResultPassed}})

	result := ReplaceNormalTestScenarioTable(storage, table)
	if strings.Contains(result, "old") {
		t.Fatalf("expected the old table to be replaced, got %q", result)
	}
	if !strings.Contains(result, table) {
		t.Fatalf("expected the new table, got %q", result)
	}
	if !strings.Contains(result, "after") || !strings.Contains(result, "intro") {
		t.Fatalf("expected the surrounding content to be kept, got %q", result)
	}
	if strings.Count(result, normalTableMarker) != 1 {
		t.Fatalf("expected a single marker, got %q", result)
	}
}

func TestReplaceNormalTestScenarioTableInsertsWhenTheTableIsMissing(t *testing.T) {
	storage := normalTableMarker + "\n<p>after</p>"
	table := BuildNormalTestScenarioTable(nil)

	result := ReplaceNormalTestScenarioTable(storage, table)
	if !strings.Contains(result, "after") || !strings.Contains(result, table) {
		t.Fatalf("expected the table to be inserted after the marker, got %q", result)
	}
}

func TestReplaceNormalTestScenarioTableAppendsWithoutAMarker(t *testing.T) {
	storage := "<p>hello</p>"
	table := BuildNormalTestScenarioTable(nil)

	result := ReplaceNormalTestScenarioTable(storage, table)
	if result != storage+"\n"+table {
		t.Fatalf("expected the table to be appended, got %q", result)
	}

	if got := ReplaceNormalTestScenarioTable(storage, "  "); got != storage {
		t.Fatalf("expected an empty table to be ignored, got %q", got)
	}
	if got := ReplaceNormalTestScenarioTable("", table); got != table {
		t.Fatalf("expected an empty storage to become the table, got %q", got)
	}
}

func TestReplaceNormalTestScenarioBlockReplacesTableAndFollowingCodeMacros(t *testing.T) {
	table := BuildNormalTestScenarioTable([]domain.DocumentRow{{Repository: "micro-payment", Function: "A", Result: domain.ResultPassed}})
	macro := BuildMockTestCodeMacro("func TestSave(t *testing.T) {}")
	storage := "<p>intro</p>\n" + normalTableMarker + "\n" + `<table class="old"><tbody><tr><td>old</td></tr></tbody></table>` + "\n" + macro + "\n<p>after</p>"

	result := ReplaceNormalTestScenarioBlock(storage, table, macro)
	if strings.Contains(result, "old") {
		t.Fatalf("expected the old table to be replaced, got %q", result)
	}
	if !strings.Contains(result, table) {
		t.Fatalf("expected the new table, got %q", result)
	}
	if strings.Count(result, `<ac:structured-macro ac:name="code"`) != 1 {
		t.Fatalf("expected a single code macro, got %q", result)
	}
	if !strings.Contains(result, "intro") || !strings.Contains(result, "after") {
		t.Fatalf("expected the surrounding content to be kept, got %q", result)
	}
}

func TestReplaceNormalTestScenarioBlockRemovesPreviousCodeMacrosWhenThereAreNone(t *testing.T) {
	table := BuildNormalTestScenarioTable([]domain.DocumentRow{{Repository: "micro-payment", Function: "A", Result: domain.ResultPassed}})
	macro := BuildMockTestCodeMacro("func TestSave(t *testing.T) {}")
	storage := normalTableMarker + "\n<table><tbody><tr><td>old</td></tr></tbody></table>\n" + macro + "\n<p>after</p>"

	result := ReplaceNormalTestScenarioBlock(storage, table, "")
	if strings.Contains(result, "<ac:structured-macro") {
		t.Fatalf("expected the previous code macro to be removed, got %q", result)
	}
	if !strings.Contains(result, "after") || !strings.Contains(result, table) {
		t.Fatalf("expected the table and the trailing content, got %q", result)
	}
}

func TestReplaceNormalTestScenarioBlockKeepsAFollowingParagraph(t *testing.T) {
	table := BuildNormalTestScenarioTable([]domain.DocumentRow{{Repository: "micro-payment", Function: "A", Result: domain.ResultPassed}})
	storage := "<p>intro</p>\n" + normalTableMarker + "\n<table><tbody><tr><td>old</td></tr></tbody></table>\n<p>after</p>"

	result := ReplaceNormalTestScenarioBlock(storage, table, BuildMockTestCodeMacro("func TestSave(t *testing.T) {}"))
	if !strings.Contains(result, "<p>after</p>") {
		t.Fatalf("expected the following paragraph to be kept, got %q", result)
	}
	if !strings.Contains(result, `<ac:structured-macro ac:name="code"`) {
		t.Fatalf("expected the code macro to be appended, got %q", result)
	}
	if !strings.Contains(result, "intro") {
		t.Fatalf("expected the leading content to be kept, got %q", result)
	}
}
