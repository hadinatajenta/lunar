package application

import (
	"strings"
	"testing"

	"lunar/backend/internal/sop/domain"
)

func scanWithCreateStatement() domain.RepositoryScan {
	return domain.RepositoryScan{
		Repository: "everest",
		Branch:     "feature/CRMMS-77914",
		DatabaseFiles: []domain.DatabaseFile{
			{
				Path: "migrations/2026_add_mdr.sql",
				Statements: []domain.DatabaseStatement{
					{Action: domain.ActionCreate, Table: "log_program_mdr", SQL: "CREATE TABLE log_program_mdr (id BIGINT)"},
				},
			},
		},
		Tables: []domain.DatabaseTable{{Name: "log_program_mdr", Action: domain.ActionCreate}},
	}
}

func TestBuildDeployBlockWithoutDatabaseChange(t *testing.T) {
	block := BuildDeployBlock(DeployBlockInput{IssueKey: "CRMMS-1", Scans: []domain.RepositoryScan{}})

	if !strings.Contains(block.DatabaseRegion, domain.DeployDatabaseMarker.Start) || !strings.Contains(block.DatabaseRegion, domain.DeployDatabaseMarker.End) {
		t.Fatalf("expected the database region markers, got %s", block.DatabaseRegion)
	}
	if !strings.Contains(block.RollbackRegion, domain.DeployRollbackMarker.Start) || !strings.Contains(block.RollbackRegion, domain.DeployRollbackMarker.End) {
		t.Fatalf("expected the rollback region markers, got %s", block.RollbackRegion)
	}
	if !strings.Contains(block.DatabaseRegion, noDatabaseChangeMessage) {
		t.Fatalf("expected the info macro fallback, got %s", block.DatabaseRegion)
	}
	if !strings.Contains(block.RollbackRegion, noCreatedTableMessage) {
		t.Fatalf("expected the rollback info fallback, got %s", block.RollbackRegion)
	}
	if strings.Contains(block.DatabaseRegion, "<th>IP</th>") {
		t.Fatalf("expected no database table, got %s", block.DatabaseRegion)
	}
	if !strings.Contains(block.Middle, babIIApplicationHeading) || !strings.Contains(block.Middle, babIIIRollbackHeading) {
		t.Fatalf("expected BAB II and BAB III in the middle, got %s", block.Middle)
	}
	if strings.Count(block.Middle, manualScreenshotNote) != 2 {
		t.Fatalf("expected one manual screenshot note per chapter, got %s", block.Middle)
	}
	if len(block.Warnings) != 0 {
		t.Fatalf("expected no warnings, got %+v", block.Warnings)
	}
	if block.DatabaseHash == "" || block.RollbackHash == "" {
		t.Fatal("expected a hash per region")
	}
}

func TestBuildDeployBlockWithTablesAndRollback(t *testing.T) {
	block := BuildDeployBlock(DeployBlockInput{
		IssueKey: "CRMMS-1",
		Scans:    []domain.RepositoryScan{scanWithCreateStatement()},
		BambooPlans: map[string]string{
			"everest": "https://bamboo.example.com/plan/EVEREST",
		},
		Databases: map[string]domain.DatabaseTarget{
			"log_program_mdr": {IP: "10.1.2.3", Database: "mms"},
		},
	})

	for _, expected := range []string{
		babIDatabaseHeading,
		"<th>IP</th>",
		"10.1.2.3",
		"mms",
		"Jalankan script <code>migrations/2026_add_mdr.sql</code>",
		"CREATE TABLE log_program_mdr (id BIGINT)",
	} {
		if !strings.Contains(block.DatabaseRegion, expected) {
			t.Fatalf("expected the database region to contain %q, got %s", expected, block.DatabaseRegion)
		}
	}
	for _, expected := range []string{
		babIIApplicationHeading,
		"https://bamboo.example.com/plan/EVEREST",
		releaseBranch,
		babIIIRollbackHeading,
		"promote existing",
		manualScreenshotNote,
	} {
		if !strings.Contains(block.Middle, expected) {
			t.Fatalf("expected the middle to contain %q, got %s", expected, block.Middle)
		}
	}
	for _, expected := range []string{
		babIVRollbackHeading,
		"DROP TABLE IF EXISTS log_program_mdr;",
	} {
		if !strings.Contains(block.RollbackRegion, expected) {
			t.Fatalf("expected the rollback region to contain %q, got %s", expected, block.RollbackRegion)
		}
	}
	if len(block.Warnings) != 0 {
		t.Fatalf("expected no warnings, got %+v", block.Warnings)
	}
}

func TestBuildDeployBlockReportsMissingMappings(t *testing.T) {
	block := BuildDeployBlock(DeployBlockInput{
		IssueKey: "CRMMS-1",
		Scans:    []domain.RepositoryScan{scanWithCreateStatement()},
	})

	if !strings.Contains(block.Middle, bambooPlaceholder) {
		t.Fatalf("expected the Bamboo placeholder, got %s", block.Middle)
	}
	if !strings.Contains(block.DatabaseRegion, databasePlaceholder) {
		t.Fatalf("expected the database placeholder, got %s", block.DatabaseRegion)
	}

	joined := strings.Join(block.Warnings, "|")
	if !strings.Contains(joined, "no Bamboo plan configured for everest") {
		t.Fatalf("expected the Bamboo warning, got %+v", block.Warnings)
	}
	if !strings.Contains(joined, "no database host configured for log_program_mdr") {
		t.Fatalf("expected the database warning, got %+v", block.Warnings)
	}
}

func TestBuildDeployBlockEscapesEvidence(t *testing.T) {
	scan := scanWithCreateStatement()
	scan.Repository = "everest<script>"
	scan.DatabaseFiles[0].Path = "migrations/<x>.sql"

	block := BuildDeployBlock(DeployBlockInput{
		IssueKey:    "CRMMS-1",
		Scans:       []domain.RepositoryScan{scan},
		BambooPlans: map[string]string{"everest<script>": "https://bamboo/plan?a=1&b=2"},
		Databases:   map[string]domain.DatabaseTarget{"log_program_mdr": {IP: "10.1.2.3", Database: "mms"}},
	})

	if strings.Contains(block.FullBody(), "<script>") {
		t.Fatalf("expected the repository name to be escaped, got %s", block.FullBody())
	}
	if !strings.Contains(block.Middle, "everest&lt;script&gt;") {
		t.Fatalf("expected the escaped repository name, got %s", block.Middle)
	}
	if strings.Contains(block.Middle, `href="https://bamboo/plan?a=1&b=2"`) {
		t.Fatalf("expected the Bamboo link to be escaped, got %s", block.Middle)
	}
}

func TestReplaceRegionOnlyReplacesThatRegion(t *testing.T) {
	block := BuildDeployBlock(DeployBlockInput{
		IssueKey:    "CRMMS-1",
		Scans:       []domain.RepositoryScan{scanWithCreateStatement()},
		BambooPlans: map[string]string{"everest": "https://bamboo/plan/EVEREST"},
		Databases:   map[string]domain.DatabaseTarget{"log_program_mdr": {IP: "10.1.2.3", Database: "mms"}},
	})
	existing := block.FullBody() + "\n<p>Catatan manual</p>"

	updatedDatabase, replaced := ReplaceRegion(existing, "<p>NEW DATABASE</p>", domain.DeployDatabaseMarker)
	if !replaced {
		t.Fatal("expected the database region to be replaced")
	}
	if !strings.Contains(updatedDatabase, "<p>NEW DATABASE</p>") {
		t.Fatalf("expected the new database region, got %s", updatedDatabase)
	}
	if strings.Contains(updatedDatabase, block.DatabaseRegion) {
		t.Fatalf("expected the old database region to be gone, got %s", updatedDatabase)
	}
	if !strings.Contains(updatedDatabase, block.RollbackRegion) {
		t.Fatalf("expected the rollback region to be untouched, got %s", updatedDatabase)
	}
	if !strings.Contains(updatedDatabase, block.Middle) {
		t.Fatalf("expected BAB II and BAB III to be untouched, got %s", updatedDatabase)
	}
	if !strings.Contains(updatedDatabase, "<p>Catatan manual</p>") {
		t.Fatalf("expected unrelated content to be untouched, got %s", updatedDatabase)
	}
}

func TestPrependBlockKeepsExistingContent(t *testing.T) {
	existing := "<p>Screenshot manual</p>"
	body := "BODY"
	updated := PrependBlock(existing, body)

	if !strings.HasPrefix(updated, body) {
		t.Fatalf("expected the body to be prepended, got %s", updated)
	}
	if !strings.Contains(updated, existing) {
		t.Fatalf("expected the existing content to be kept, got %s", updated)
	}
}

func TestParseDeployBlockRoundTripsTheFullBody(t *testing.T) {
	block := BuildDeployBlock(DeployBlockInput{
		IssueKey:    "CRMMS-1",
		Scans:       []domain.RepositoryScan{scanWithCreateStatement()},
		BambooPlans: map[string]string{"everest": "https://bamboo/plan/EVEREST"},
		Databases:   map[string]domain.DatabaseTarget{"log_program_mdr": {IP: "10.1.2.3", Database: "mms"}},
	})

	parsed, ok := ParseDeployBlock(block.FullBody())
	if !ok {
		t.Fatal("expected the block to parse")
	}
	if parsed.DatabaseRegion != block.DatabaseRegion || parsed.RollbackRegion != block.RollbackRegion {
		t.Fatalf("expected the regions to round-trip, got %+v", parsed)
	}
	if parsed.DatabaseHash != block.DatabaseHash || parsed.RollbackHash != block.RollbackHash {
		t.Fatalf("expected the hashes to round-trip, got %+v", parsed)
	}
	if parsed.Middle != block.Middle {
		t.Fatalf("expected the middle to round-trip, got %q", parsed.Middle)
	}
}

func TestParseDeployBlockRejectsAMissingRegion(t *testing.T) {
	if _, ok := ParseDeployBlock("<p>no markers</p>"); ok {
		t.Fatal("expected the parse to fail without markers")
	}
}

func TestBuildOperasionalChildBlockWrapsMarkers(t *testing.T) {
	block := BuildOperasionalChildBlock(domain.OperasionalMaintenance)
	if !strings.HasPrefix(block, domain.OperasionalMarker.Start) || !strings.HasSuffix(block, domain.OperasionalMarker.End) {
		t.Fatalf("expected the child block to be wrapped in markers, got %s", block)
	}
	for _, expected := range []string{`ac:name="toc"`, "1. Backup", "2. Restore", "3. Housekeeping"} {
		if !strings.Contains(block, expected) {
			t.Fatalf("expected the maintenance skeleton to contain %q, got %s", expected, block)
		}
	}
}
