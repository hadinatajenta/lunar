package application

import (
	"strings"
	"testing"
)

const (
	microCompareURL = "https://bitbucket.example.com/projects/NME/repos/micro-payment/compare/diff?sourceBranch=feat%2FCRMMS-1&targetBranch=development"
	otherCompareURL = "https://bitbucket.example.com/projects/NME/repos/other-service/compare/diff?sourceBranch=feat%2FCRMMS-1&targetBranch=development"
)

func TestParseMigTablesReadsColumnsByHeaderName(t *testing.T) {
	storage := `<table><tbody>
<tr><th>Branch</th><th>Compare</th><th>Repository</th></tr>
<tr><td></td><td><a href="` + microCompareURL + `">link</a></td><td>micro-payment</td></tr>
</tbody></table>`

	rows, skipped := parseMigTables(storage)
	if len(skipped) != 0 {
		t.Fatalf("expected no skipped rows, got %+v", skipped)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %+v", rows)
	}
	row := rows[0]
	if row.repository != "micro-payment" || row.projectKey != "NME" {
		t.Fatalf("unexpected repository identity %+v", row)
	}
	if row.sourceBranch != "feat/CRMMS-1" || row.targetBranch != "development" {
		t.Fatalf("expected the URL branches, got %s -> %s", row.sourceBranch, row.targetBranch)
	}
}

func TestParseMigTablesFillsEmptyCellsFromTheCompareURL(t *testing.T) {
	storage := `<table><tbody>
<tr><th>Repository</th><th>Branch</th><th>Compare link</th></tr>
<tr><td></td><td></td><td>` + microCompareURL + `</td></tr>
</tbody></table>`

	rows, skipped := parseMigTables(storage)
	if len(skipped) != 0 || len(rows) != 1 {
		t.Fatalf("expected 1 row and no skips, got rows=%+v skipped=%+v", rows, skipped)
	}
	if rows[0].repository != "micro-payment" || rows[0].sourceBranch != "feat/CRMMS-1" {
		t.Fatalf("expected the repository and branch from the URL, got %+v", rows[0])
	}
}

func TestParseMigTablesSkipsARowWithoutAParsableLink(t *testing.T) {
	storage := `<table><tbody>
<tr><th>Repository</th><th>Branch</th><th>Compare</th></tr>
<tr><td>micro-payment</td><td>feat/x</td><td>not a link</td></tr>
</tbody></table>`

	rows, skipped := parseMigTables(storage)
	if len(rows) != 0 {
		t.Fatalf("expected no rows, got %+v", rows)
	}
	if len(skipped) != 1 || skipped[0].Repository != "micro-payment" {
		t.Fatalf("expected the row reported as skipped, got %+v", skipped)
	}
}

func TestParseMigTablesSkipsAnIncompleteRow(t *testing.T) {
	storage := `<table><tbody>
<tr><th>Repository</th><th>Branch</th><th>Compare</th></tr>
<tr><td></td><td></td><td><a href="https://bitbucket.example.com/projects/NME/repos/micro-payment/pull-requests/12/overview">pr</a></td></tr>
</tbody></table>`

	rows, skipped := parseMigTables(storage)
	if len(rows) != 0 {
		t.Fatalf("expected no rows, got %+v", rows)
	}
	if len(skipped) != 1 {
		t.Fatalf("expected the incomplete row to be reported, got %+v", skipped)
	}
	if !strings.Contains(skipped[0].Reason, "incomplete") {
		t.Fatalf("expected an incomplete reason, got %q", skipped[0].Reason)
	}
}

func TestParseMigTablesIgnoresAnUnrelatedTableFirst(t *testing.T) {
	storage := `<table><tbody>
<tr><th>Name</th><th>Owner</th></tr>
<tr><td>Release plan</td><td>Jane</td></tr>
</tbody></table>
<table><tbody>
<tr><th>Repository</th><th>Branch</th><th>Compare</th></tr>
<tr><td>other-service</td><td></td><td>` + otherCompareURL + `</td></tr>
</tbody></table>`

	rows, skipped := parseMigTables(storage)
	if len(skipped) != 0 {
		t.Fatalf("expected no skipped rows, got %+v", skipped)
	}
	if len(rows) != 1 || rows[0].repository != "other-service" {
		t.Fatalf("expected only the MIG table row, got %+v", rows)
	}
}

func TestParseMigTablesPrefersTheCompareURLBranches(t *testing.T) {
	storage := `<table><tbody>
<tr><th>Repository</th><th>Branch</th><th>Compare</th></tr>
<tr><td>micro-payment</td><td>legacy-branch</td><td>` + microCompareURL + `</td></tr>
</tbody></table>`

	rows, _ := parseMigTables(storage)
	if len(rows) != 1 {
		t.Fatalf("expected the row to parse, got %+v", rows)
	}
	if rows[0].sourceBranch != "feat/CRMMS-1" || rows[0].targetBranch != "development" {
		t.Fatalf("expected the compare URL branches, got %s -> %s", rows[0].sourceBranch, rows[0].targetBranch)
	}
}

func TestParseMigTablesKeepsTheRepositoryCellOverTheURL(t *testing.T) {
	storage := `<table><tbody>
<tr><th>Repository</th><th>Branch</th><th>Compare</th></tr>
<tr><td>explicit-repo</td><td></td><td>` + microCompareURL + `</td></tr>
</tbody></table>`

	rows, _ := parseMigTables(storage)
	if len(rows) != 1 {
		t.Fatalf("expected the row to parse, got %+v", rows)
	}
	if rows[0].repository != "explicit-repo" || rows[0].projectKey != "NME" {
		t.Fatalf("expected the repository cell with the URL project key, got %+v", rows[0])
	}
}

func TestParseMigTablesStripsTheRefsHeadsPrefixFromTheBranchCell(t *testing.T) {
	storage := `<table><tbody>
<tr><th>Repository</th><th>Branch</th><th>Compare</th></tr>
<tr><td>micro-payment</td><td>refs/heads/feature/CRMMS-1</td><td><a href="https://bitbucket.example.com/projects/NME/repos/micro-payment/pull-requests/12/overview">pr</a></td></tr>
</tbody></table>`

	rows, _ := parseMigTables(storage)
	if len(rows) != 1 {
		t.Fatalf("expected one row, got %+v", rows)
	}
	if rows[0].sourceBranch != "feature/CRMMS-1" || rows[0].targetBranch != "feature/CRMMS-1" {
		t.Fatalf("expected the stripped branch, got %s -> %s", rows[0].sourceBranch, rows[0].targetBranch)
	}
}

func TestParseMigTablesIgnoresATableWithoutTheRequiredColumns(t *testing.T) {
	storage := `<table><tbody>
<tr><th>Repository</th><th>Owner</th></tr>
<tr><td>micro-payment</td><td>Jane</td></tr>
</tbody></table>`

	rows, skipped := parseMigTables(storage)
	if len(rows) != 0 || len(skipped) != 0 {
		t.Fatalf("expected the table to be ignored, got rows=%+v skipped=%+v", rows, skipped)
	}
}

func TestParseMigTablesSkipsABrowseOnlyRow(t *testing.T) {
	storage := `<table><tbody>
<tr><th>Repository</th><th>Branch</th><th>Link Compare</th></tr>
<tr><td><a href="https://bitbucket.example.com/projects/NME/repos/everest/browse">everest</a></td><td>development</td><td><a href="https://bitbucket.example.com/projects/NME/repos/everest/browse">browse</a></td></tr>
</tbody></table>`

	rows, skipped := parseMigTables(storage)
	if len(rows) != 0 {
		t.Fatalf("expected a browse only link to produce no row, got %+v", rows)
	}
	if len(skipped) != 1 || !strings.Contains(skipped[0].Reason, "no parsable compare link") {
		t.Fatalf("expected the row to be reported with no parsable compare link, got %+v", skipped)
	}
}

func TestParseMigTablesFindsACompareLinkOutsideTheLinkColumn(t *testing.T) {
	storage := `<table><tbody>
<tr><th>Repository</th><th>Branch</th><th>Link Compare</th></tr>
<tr><td>micro-payment</td><td>feat/x</td><td><a href="https://bitbucket.example.com/projects/NME/repos/micro-payment/browse">browse</a> ` + microCompareURL + `</td></tr>
</tbody></table>`

	rows, skipped := parseMigTables(storage)
	if len(skipped) != 0 {
		t.Fatalf("expected no skipped rows, got %+v", skipped)
	}
	if len(rows) != 1 || rows[0].repoSlug != "micro-payment" {
		t.Fatalf("expected the compare link to be found in the row, got %+v", rows)
	}
	if rows[0].sourceBranch != "feat/CRMMS-1" || rows[0].targetBranch != "development" {
		t.Fatalf("expected the compare URL branches, got %s -> %s", rows[0].sourceBranch, rows[0].targetBranch)
	}
}

func TestFilterExcludedRepositoriesMatchesTheURLSlugWhenTheCellTextDiffers(t *testing.T) {
	rows := []migTableRow{
		{repository: "Everest Service", repoSlug: "everest"},
		{repository: "Merchant Management System", repoSlug: "fe-mms"},
		{repository: "micro-payment", repoSlug: "micro-payment"},
	}

	included, skipped := filterExcludedRepositories(rows)
	if len(included) != 2 {
		t.Fatalf("expected everest and micro-payment to remain, got %+v", included)
	}
	if len(skipped) != 1 || skipped[0].Repository != "Merchant Management System" {
		t.Fatalf("expected the fe-mms row to be excluded by its slug, got %+v", skipped)
	}
}

func TestFilterExcludedRepositoriesComparesCaseInsensitively(t *testing.T) {
	rows := []migTableRow{
		{repository: "micro-payment"},
		{repository: "FE-MMS"},
		{repository: "mms"},
	}

	included, skipped := filterExcludedRepositories(rows)
	if len(included) != 1 || included[0].repository != "micro-payment" {
		t.Fatalf("expected only micro-payment to remain, got %+v", included)
	}
	if len(skipped) != 2 {
		t.Fatalf("expected the two excluded repositories to be reported, got %+v", skipped)
	}
	for _, entry := range skipped {
		if entry.Reason != "excluded repository" {
			t.Fatalf("expected the excluded reason, got %q", entry.Reason)
		}
	}
}
