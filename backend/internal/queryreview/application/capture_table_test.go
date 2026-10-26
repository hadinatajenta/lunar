package application

import (
	"strings"
	"testing"

	"lunar/backend/internal/queryreview/domain"
)

func singleRepositoryReviews() []domain.RepositoryQueryReview {
	return []domain.RepositoryQueryReview{
		{
			Repository: "micro-payment",
			Queries: []domain.QueryItem{
				{No: 1, Feature: "[micro-payment] Get user by PN", QueryStatement: "explain select id from user_data where pn = ?", Index: "pn"},
			},
		},
	}
}

func TestBuildCaptureExplainTableHasNoGroupHeaderForASingleRepository(t *testing.T) {
	table := BuildCaptureExplainTable(singleRepositoryReviews())

	if strings.Count(table, captureExplainMarkerText) != 1 {
		t.Fatalf("expected exactly one marker, got %q", table)
	}
	if strings.Contains(table, "Layanan / Microservice:") {
		t.Fatalf("expected no group header for a single repository, got %q", table)
	}
	if !strings.Contains(table, `<table class="wrapped relative-table" style="width: 100%;">`) {
		t.Fatalf("expected the wrapped table, got %q", table)
	}
	if !strings.Contains(table, `<tr><th>No</th><th>Fitur</th><th>Query Statement</th><th>Index</th><th>Capture Explain / Execution Plan</th></tr>`) {
		t.Fatalf("expected the header row, got %q", table)
	}
	expectedRow := `<tr><td>1</td><td>[micro-payment] Get user by PN</td><td>explain select id from user_data where pn = ?</td><td>pn</td><td></td></tr>`
	if !strings.Contains(table, expectedRow) {
		t.Fatalf("expected the query row %q, got %q", expectedRow, table)
	}
}

func TestBuildCaptureExplainTableGroupsAndRestartsNumberingForMultipleRepositories(t *testing.T) {
	reviews := []domain.RepositoryQueryReview{
		{
			Repository: "micro-payment",
			Queries: []domain.QueryItem{
				{No: 9, Feature: "[micro-payment] First", QueryStatement: "explain select 1", Index: "a"},
				{No: 10, Feature: "[micro-payment] Second", QueryStatement: "explain select 2", Index: "b"},
			},
		},
		{
			Repository: "other-service",
			Queries: []domain.QueryItem{
				{No: 7, Feature: "[other-service] Third", QueryStatement: "explain select 3", Index: "c"},
			},
		},
	}

	table := BuildCaptureExplainTable(reviews)

	microGroup := `<tr><td colspan="5"><strong>Layanan / Microservice: <code>micro-payment</code></strong></td></tr>`
	otherGroup := `<tr><td colspan="5"><strong>Layanan / Microservice: <code>other-service</code></strong></td></tr>`
	if !strings.Contains(table, microGroup) || !strings.Contains(table, otherGroup) {
		t.Fatalf("expected both group rows, got %q", table)
	}
	if strings.Index(table, microGroup) > strings.Index(table, otherGroup) {
		t.Fatalf("expected the group order to follow the input, got %q", table)
	}

	if !strings.Contains(table, `<tr><td>1</td><td>[micro-payment] First</td>`) {
		t.Fatalf("expected micro-payment numbering to start at 1, got %q", table)
	}
	if !strings.Contains(table, `<tr><td>2</td><td>[micro-payment] Second</td>`) {
		t.Fatalf("expected micro-payment numbering to continue at 2, got %q", table)
	}
	if !strings.Contains(table, `<tr><td>1</td><td>[other-service] Third</td>`) {
		t.Fatalf("expected other-service numbering to restart at 1, got %q", table)
	}
	if strings.Contains(table, `<tr><td>7</td>`) || strings.Contains(table, `<tr><td>9</td>`) {
		t.Fatalf("expected the incoming No to be replaced by per-group numbering, got %q", table)
	}
}

func TestBuildCaptureExplainTableEscapesStatementMarkup(t *testing.T) {
	reviews := []domain.RepositoryQueryReview{
		{
			Repository: "micro-payment",
			Queries: []domain.QueryItem{
				{No: 1, Feature: "[micro-payment] Injection", QueryStatement: `select <script>alert(1)</script>`, Index: "pn"},
			},
		},
	}

	table := BuildCaptureExplainTable(reviews)

	if strings.Contains(table, "<script>") {
		t.Fatalf("expected the statement markup to be escaped, got %q", table)
	}
	if !strings.Contains(table, `&lt;script&gt;alert(1)&lt;/script&gt;`) {
		t.Fatalf("expected the escaped statement, got %q", table)
	}
}

func TestBuildCaptureExplainTableRendersTheCreateIndexOnASecondLineInTheIndexCell(t *testing.T) {
	reviews := []domain.RepositoryQueryReview{
		{
			Repository: "micro-payment",
			Queries: []domain.QueryItem{
				{No: 1, Feature: "[micro-payment] Get order", QueryStatement: "EXPLAIN select id from orders where mid = ?", Index: "PRIMARY", CreateIndex: "CREATE INDEX idx_orders_mid ON orders (mid);"},
				{No: 2, Feature: "[micro-payment] Get user", QueryStatement: "EXPLAIN select id from user_data where pn = ?", Index: "pn"},
			},
		},
	}

	table := BuildCaptureExplainTable(reviews)

	expectedWithCreate := `<td>PRIMARY<br/>CREATE INDEX idx_orders_mid ON orders (mid);</td>`
	if !strings.Contains(table, expectedWithCreate) {
		t.Fatalf("expected the index cell with the create index on a second line, got %q", table)
	}
	expectedWithoutCreate := `<td>pn</td>`
	if !strings.Contains(table, expectedWithoutCreate) {
		t.Fatalf("expected the index cell with only the possible key, got %q", table)
	}
}

func TestBuildCaptureExplainTableEscapesCreateIndexMarkup(t *testing.T) {
	reviews := []domain.RepositoryQueryReview{
		{
			Repository: "micro-payment",
			Queries: []domain.QueryItem{
				{No: 1, Feature: "[micro-payment] Injection", QueryStatement: "EXPLAIN select 1", Index: "pn", CreateIndex: `CREATE INDEX idx ON t (<script>);`},
			},
		},
	}

	table := BuildCaptureExplainTable(reviews)

	if strings.Contains(table, "<script>") {
		t.Fatalf("expected the create index markup to be escaped, got %q", table)
	}
	if !strings.Contains(table, `&lt;script&gt;`) {
		t.Fatalf("expected the escaped create index, got %q", table)
	}
}

func TestReplaceCaptureExplainTableReplacesTheMarkerAndTable(t *testing.T) {
	storage := `<p>Intro</p>` + "\n" +
		`<p><em>Capture explain (mysql)</em></p>` + "\n" +
		`<table class="old"><tbody><tr><td>old</td></tr></tbody></table>` + "\n" +
		`<p>Outro</p>`
	table := BuildCaptureExplainTable(singleRepositoryReviews())

	result := ReplaceCaptureExplainTable(storage, table)

	if !strings.HasPrefix(result, `<p>Intro</p>`) {
		t.Fatalf("expected the intro to be preserved, got %q", result)
	}
	if !strings.HasSuffix(result, `<p>Outro</p>`) {
		t.Fatalf("expected the outro to be preserved, got %q", result)
	}
	if strings.Contains(result, "old") {
		t.Fatalf("expected the previous table to be replaced, got %q", result)
	}
	if strings.Count(result, captureExplainMarkerText) != 1 {
		t.Fatalf("expected exactly one marker after replacement, got %q", result)
	}
	if !strings.Contains(result, `[micro-payment] Get user by PN`) {
		t.Fatalf("expected the new table content, got %q", result)
	}
}

func TestReplaceCaptureExplainTableInsertsAfterAMarkerWithoutTable(t *testing.T) {
	storage := `<p>Intro</p>` + "\n" +
		`<p><em>Capture explain (mysql)</em></p>` + "\n" +
		`<p>Tail</p>`
	table := BuildCaptureExplainTable(singleRepositoryReviews())

	result := ReplaceCaptureExplainTable(storage, table)

	if strings.Count(result, captureExplainMarkerText) != 1 {
		t.Fatalf("expected exactly one marker, got %q", result)
	}
	markerIndex := strings.Index(result, captureExplainMarkerText)
	tableIndex := strings.Index(result, "<table")
	tailIndex := strings.Index(result, "<p>Tail</p>")
	if markerIndex == -1 || tableIndex == -1 || tailIndex == -1 {
		t.Fatalf("expected the marker, table, and tail, got %q", result)
	}
	if !(markerIndex < tableIndex && tableIndex < tailIndex) {
		t.Fatalf("expected the table between the marker and the tail, got %q", result)
	}
}

func TestReplaceCaptureExplainTableAppendsWhenThereIsNoMarker(t *testing.T) {
	storage := `<p>Only intro</p>`
	table := BuildCaptureExplainTable(singleRepositoryReviews())

	result := ReplaceCaptureExplainTable(storage, table)

	if !strings.HasPrefix(result, storage) {
		t.Fatalf("expected the original storage to be preserved, got %q", result)
	}
	if !strings.Contains(result, captureExplainMarkerText) {
		t.Fatalf("expected the marker to be appended, got %q", result)
	}
	if strings.Index(result, captureExplainMarkerText) < len(storage) {
		t.Fatalf("expected the appended block after the storage, got %q", result)
	}
}

func TestReplaceCaptureExplainTableMatchesAPlainMarker(t *testing.T) {
	storage := `<p>Capture explain (mysql)</p>` + "\n" +
		`<table class="old"><tbody><tr><td>old</td></tr></tbody></table>`
	table := BuildCaptureExplainTable(singleRepositoryReviews())

	result := ReplaceCaptureExplainTable(storage, table)

	if strings.Contains(result, "old") {
		t.Fatalf("expected the previous table to be replaced, got %q", result)
	}
	if !strings.Contains(result, `<em>Capture explain (mysql)</em>`) {
		t.Fatalf("expected the canonical marker, got %q", result)
	}
}

func TestReplaceCaptureExplainTableReturnsTheTableWhenStorageIsEmpty(t *testing.T) {
	table := BuildCaptureExplainTable(singleRepositoryReviews())

	if result := ReplaceCaptureExplainTable("   ", table); result != table {
		t.Fatalf("expected the table for empty storage, got %q", result)
	}
}

func TestBuildCaptureExplainTableFallsBackToASingleMessageRow(t *testing.T) {
	table := BuildCaptureExplainTable(nil)

	if !strings.Contains(table, `colspan="5"`) {
		t.Fatalf("expected a single message row, got %q", table)
	}
	if !strings.Contains(table, noQueryChangesMessage) {
		t.Fatalf("expected the fallback message, got %q", table)
	}
	if strings.Count(table, "<tr>") != 2 {
		t.Fatalf("expected only the header and the message row, got %q", table)
	}
}
