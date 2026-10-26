package infrastructure

import (
	"fmt"
	"strings"
	"testing"

	"lunar/backend/internal/queryreview/domain"
)

const extractionDiff = `diff --git a/internal/repo/dao.go b/internal/repo/dao.go
index 1111111..2222222 100644
--- a/internal/repo/dao.go
+++ b/internal/repo/dao.go
@@ -10,6 +10,8 @@ func LoadUser(db *gorm.DB) {
     db := open()
-    old := db.Where("id = 1")
+    rows := db.Table("user_data").Where("pn = ?", pn).Find(&users)
+    if rows.Error != nil {
+        return
+    }
     return
 }
`

func addedOnlyDiff(filePath string, addedLines ...string) string {
	var builder strings.Builder
	builder.WriteString("diff --git a/" + filePath + " b/" + filePath + "\n")
	builder.WriteString("--- a/" + filePath + "\n")
	builder.WriteString("+++ b/" + filePath + "\n")
	builder.WriteString(fmt.Sprintf("@@ -0,0 +1,%d @@\n", len(addedLines)))
	for _, line := range addedLines {
		builder.WriteString("+" + line + "\n")
	}
	return builder.String()
}

func TestDiffExtractorKeepsOnlyAddedLinesWithTheirNewLineNumbers(t *testing.T) {
	analysis := NewDiffExtractor().Extract(extractionDiff)

	if len(analysis.Files) != 1 {
		t.Fatalf("expected 1 analyzed file, got %+v", analysis)
	}
	file := analysis.Files[0]
	if file.Path != "internal/repo/dao.go" {
		t.Fatalf("expected the dao path, got %q", file.Path)
	}
	if file.Language != "go" {
		t.Fatalf("expected language go, got %q", file.Language)
	}

	expectedLines := []domain.PromptLine{
		{Number: 11, Content: `    rows := db.Table("user_data").Where("pn = ?", pn).Find(&users)`},
		{Number: 12, Content: "    if rows.Error != nil {"},
		{Number: 13, Content: "        return"},
		{Number: 14, Content: "    }"},
	}
	if len(file.AddedLines) != len(expectedLines) {
		t.Fatalf("expected %d added lines, got %+v", len(expectedLines), file.AddedLines)
	}
	for index, expected := range expectedLines {
		if file.AddedLines[index] != expected {
			t.Fatalf("expected added line %d to be %+v, got %+v", index, expected, file.AddedLines[index])
		}
	}

	for _, line := range file.AddedLines {
		if strings.Contains(line.Content, "old :=") {
			t.Fatalf("removed line leaked into the prompt lines: %+v", line)
		}
		if strings.Contains(line.Content, "db := open()") {
			t.Fatalf("context line leaked into the prompt lines: %+v", line)
		}
	}

	if len(analysis.Analyzed) != 1 || analysis.Analyzed[0].AddedLines != 4 {
		t.Fatalf("expected 1 analyzed file with 4 added lines, got %+v", analysis.Analyzed)
	}
	if len(analysis.Skipped) != 0 {
		t.Fatalf("expected no skipped files, got %+v", analysis.Skipped)
	}
	if analysis.Truncated {
		t.Fatal("expected truncated to be false")
	}
}

func TestDiffExtractorDropsRemovedAndContextQueryLines(t *testing.T) {
	diffText := `diff --git a/internal/handler/user_handler.go b/internal/handler/user_handler.go
--- a/internal/handler/user_handler.go
+++ b/internal/handler/user_handler.go
@@ -1,3 +1,3 @@
 package handler
-rows := db.Raw("select id from user_data")
+func handle() {}
 
`

	analysis := NewDiffExtractor().Extract(diffText)

	if len(analysis.Files) != 0 {
		t.Fatalf("expected the removed query line to leave no prompt files, got %+v", analysis.Files)
	}
	if len(analysis.Skipped) != 1 || analysis.Skipped[0].Reason != skipReasonNotDatabase {
		t.Fatalf("expected the file to be skipped as not database relevant, got %+v", analysis.Skipped)
	}
}

func TestDiffExtractorKeepsDatabaseRelevantPathsWithoutQuerySyntax(t *testing.T) {
	diffText := addedOnlyDiff("internal/repository/user_repository.go", "func (r *UserRepository) Save() {}")

	analysis := NewDiffExtractor().Extract(diffText)

	if len(analysis.Files) != 1 || analysis.Files[0].Path != "internal/repository/user_repository.go" {
		t.Fatalf("expected the repository file to be analyzed, got %+v", analysis)
	}
	if analysis.Files[0].Language != "go" {
		t.Fatalf("expected language go, got %q", analysis.Files[0].Language)
	}
}

func TestDiffExtractorKeepsNonDatabasePathsOnlyWhenAnAddedLineHasQuerySyntax(t *testing.T) {
	withQuery := NewDiffExtractor().Extract(addedOnlyDiff(
		"internal/handler/user_handler.go",
		`rows, err := db.Raw("SELECT id FROM user_data WHERE pn = ?").Rows()`,
	))
	if len(withQuery.Files) != 1 {
		t.Fatalf("expected the query line to keep the handler file, got %+v", withQuery)
	}

	withoutQuery := NewDiffExtractor().Extract(addedOnlyDiff("internal/handler/user_handler.go", "func handle() {}"))
	if len(withoutQuery.Files) != 0 {
		t.Fatalf("expected the handler file without query syntax to be dropped, got %+v", withoutQuery)
	}
	if len(withoutQuery.Skipped) != 1 || withoutQuery.Skipped[0].Reason != skipReasonNotDatabase {
		t.Fatalf("expected a not database relevant reason, got %+v", withoutQuery.Skipped)
	}
}

func TestDiffExtractorReportsSkipReasonsForIgnoredPaths(t *testing.T) {
	cases := []struct {
		path   string
		reason string
	}{
		{path: "internal/repo/user_dao_test.go", reason: skipReasonTestFile},
		{path: "internal/tests/user_dao.go", reason: skipReasonTestFile},
		{path: "internal/mocks/user_mock.go", reason: skipReasonMockFile},
		{path: "api/generated/user.pb.go", reason: skipReasonGeneratedFile},
		{path: "vendor/github.com/stretchr/testify/assert.go", reason: skipReasonVendorFile},
		{path: "docs/query-review.md", reason: skipReasonDocumentation},
		{path: "assets/logo.svg", reason: skipReasonAssetFile},
	}

	for _, testCase := range cases {
		analysis := NewDiffExtractor().Extract(addedOnlyDiff(testCase.path, `rows := db.Raw("select 1")`))
		if len(analysis.Files) != 0 {
			t.Fatalf("expected %s to be dropped, got %+v", testCase.path, analysis.Files)
		}
		if len(analysis.Skipped) != 1 {
			t.Fatalf("expected one skip entry for %s, got %+v", testCase.path, analysis.Skipped)
		}
		if analysis.Skipped[0].Reason != testCase.reason {
			t.Fatalf("expected reason %q for %s, got %q", testCase.reason, testCase.path, analysis.Skipped[0].Reason)
		}
	}
}

func TestDiffExtractorDropsFilesWithoutAddedLines(t *testing.T) {
	diffText := `diff --git a/internal/repo/dao.go b/internal/repo/dao.go
--- a/internal/repo/dao.go
+++ /dev/null
@@ -1,2 +0,0 @@
-package repo
-const removed = true
`

	analysis := NewDiffExtractor().Extract(diffText)

	if len(analysis.Files) != 0 {
		t.Fatalf("expected no prompt files for a deletion only diff, got %+v", analysis.Files)
	}
	if len(analysis.Skipped) != 1 || analysis.Skipped[0].Reason != skipReasonNoAddedLines {
		t.Fatalf("expected the no added lines reason, got %+v", analysis.Skipped)
	}
}

func TestDiffExtractorNumbersAddedLinesPerHunk(t *testing.T) {
	diffText := `diff --git a/internal/repo/dao.go b/internal/repo/dao.go
--- a/internal/repo/dao.go
+++ b/internal/repo/dao.go
@@ -1,2 +1,3 @@
 package repo
+import "gorm.io/gorm"
 
@@ -20,2 +21,3 @@
 func a() {}
+db.Exec("update user_data set x = 1")
 func b() {}
`

	analysis := NewDiffExtractor().Extract(diffText)

	if len(analysis.Files) != 1 {
		t.Fatalf("expected 1 analyzed file, got %+v", analysis)
	}
	expectedLines := []domain.PromptLine{
		{Number: 2, Content: `import "gorm.io/gorm"`},
		{Number: 22, Content: `db.Exec("update user_data set x = 1")`},
	}
	if len(analysis.Files[0].AddedLines) != len(expectedLines) {
		t.Fatalf("expected %d added lines, got %+v", len(expectedLines), analysis.Files[0].AddedLines)
	}
	for index, expected := range expectedLines {
		if analysis.Files[0].AddedLines[index] != expected {
			t.Fatalf("expected added line %d to be %+v, got %+v", index, expected, analysis.Files[0].AddedLines[index])
		}
	}
}

func TestDiffExtractorDropsAFileOverThePerFileBudgetAndFlagsTruncation(t *testing.T) {
	diffText := addedOnlyDiff("internal/repo/huge_dao.go", strings.Repeat("x", queryReviewFileCharBudget))

	analysis := NewDiffExtractor().Extract(diffText)

	if len(analysis.Files) != 0 {
		t.Fatalf("expected the oversized file to be dropped, got %+v", analysis.Files)
	}
	if len(analysis.Skipped) != 1 || analysis.Skipped[0].Reason != skipReasonOverBudget {
		t.Fatalf("expected the over budget reason, got %+v", analysis.Skipped)
	}
	if !analysis.Truncated {
		t.Fatal("expected truncated to be true when the budget drops a whole file")
	}
}

func TestDiffExtractorDropsFilesBeyondTheTotalBudgetAndFlagsTruncation(t *testing.T) {
	perFilePayload := queryReviewFileCharBudget
	contentLength := perFilePayload - queryReviewLineOverhead

	var builder strings.Builder
	for index := 0; index < 11; index++ {
		builder.WriteString(addedOnlyDiff(
			fmt.Sprintf("internal/repo/dao_%d.go", index),
			strings.Repeat("x", contentLength),
		))
	}

	analysis := NewDiffExtractor().Extract(builder.String())

	if len(analysis.Files) != 10 {
		t.Fatalf("expected the first ten files to fit the total budget, got %d", len(analysis.Files))
	}
	if len(analysis.Skipped) != 1 {
		t.Fatalf("expected the eleventh file to be skipped, got %+v", analysis.Skipped)
	}
	if analysis.Skipped[0].Reason != skipReasonOverTotalBudget {
		t.Fatalf("expected the total budget reason, got %q", analysis.Skipped[0].Reason)
	}
	if !analysis.Truncated {
		t.Fatal("expected truncated to be true when the total budget drops a whole file")
	}
}

func TestDiffExtractorReturnsEmptySlicesForAnEmptyDiff(t *testing.T) {
	analysis := NewDiffExtractor().Extract("")

	if len(analysis.Files) != 0 || len(analysis.Analyzed) != 0 || len(analysis.Skipped) != 0 {
		t.Fatalf("expected empty analysis, got %+v", analysis)
	}
	if analysis.Truncated {
		t.Fatal("expected truncated to be false for an empty diff")
	}
}
