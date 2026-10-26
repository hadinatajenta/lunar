package application

import (
	"strings"
	"testing"

	"lunar/backend/internal/utdocs/domain"
)

func TestClassifyCandidatesMarksSqlmockTestsWithoutDatabaseMarkersSafe(t *testing.T) {
	content := "package repo\n\nimport (\n\t\"github.com/DATA-DOG/go-sqlmock\"\n)\n\nfunc TestFind(t *testing.T) {\n\tdb, mock, _ := sqlmock.New()\n\tmock.ExpectQuery(\"SELECT id FROM users WHERE id = ?\")\n\t_ = db\n}\n"

	outcome := ClassifyCandidates([]domain.CandidateTestFile{{
		Path:     "internal/repo/dao_test.go",
		Package:  "./internal/repo",
		Language: domain.LanguageGo,
		Content:  content,
	}})

	if len(outcome.TestFiles) != 1 {
		t.Fatalf("expected one test file, got %+v", outcome.TestFiles)
	}
	if outcome.TestFiles[0].Classification != domain.ClassificationSafe {
		t.Fatalf("expected a safe classification for a sqlmock test, got %+v", outcome.TestFiles[0])
	}
	if len(outcome.BlockedFiles) != 0 {
		t.Fatalf("expected no blocked files, got %+v", outcome.BlockedFiles)
	}
	if len(outcome.ExecutedPackages) != 1 || outcome.ExecutedPackages[0] != "./internal/repo" {
		t.Fatalf("expected the package to run, got %+v", outcome.ExecutedPackages)
	}
}

func TestClassifyCandidatesBlocksSqlmockTestsThatStillTouchADatabase(t *testing.T) {
	content := "package repo\n\nimport (\n\t\"database/sql\"\n\n\t\"github.com/DATA-DOG/go-sqlmock\"\n)\n\nfunc TestReset(t *testing.T) {\n\tdb, mock, _ := sqlmock.New()\n\tmock.ExpectExec(\"TRUNCATE orders\")\n\treal, _ := sql.Open(\"postgres\", dsn)\n\t_ = db\n\t_ = real\n}\n"

	outcome := ClassifyCandidates([]domain.CandidateTestFile{{
		Path:     "internal/repo/dao_test.go",
		Package:  "./internal/repo",
		Language: domain.LanguageGo,
		Content:  content,
	}})

	if len(outcome.TestFiles) != 1 || outcome.TestFiles[0].Classification != domain.ClassificationBlocked {
		t.Fatalf("expected a sqlmock test that opens a database to be blocked, got %+v", outcome.TestFiles)
	}
	if len(outcome.BlockedFiles) != 1 {
		t.Fatalf("expected one blocked file, got %+v", outcome.BlockedFiles)
	}
	if len(outcome.ExecutedPackages) != 0 {
		t.Fatalf("expected no executed packages, got %+v", outcome.ExecutedPackages)
	}
}

func TestClassifyCandidatesBlocksRealDriverImportsWithoutSqlmock(t *testing.T) {
	content := "package repo\n\nimport (\n\t_ \"github.com/lib/pq\"\n)\n"

	outcome := ClassifyCandidates([]domain.CandidateTestFile{{
		Path:     "internal/repo/dao_test.go",
		Package:  "./internal/repo",
		Language: domain.LanguageGo,
		Content:  content,
	}})

	if len(outcome.BlockedFiles) != 1 {
		t.Fatalf("expected one blocked file, got %+v", outcome.BlockedFiles)
	}
	if !strings.Contains(outcome.BlockedFiles[0].Reason, "github.com/lib/pq") {
		t.Fatalf("expected the driver in the reason, got %q", outcome.BlockedFiles[0].Reason)
	}
	if len(outcome.ExecutedPackages) != 0 {
		t.Fatalf("expected no executed packages, got %+v", outcome.ExecutedPackages)
	}
}

func TestClassifyCandidatesBlocksDestructiveAndConnectionMarkers(t *testing.T) {
	testCases := []struct {
		name    string
		content string
		marker  string
	}{
		{name: "sql open", content: "db, _ := sql.Open(\"postgres\", dsn)", marker: "sql.Open"},
		{name: "open db helper", content: "database := OpenDB(config)", marker: "OpenDB("},
		{name: "auto migrate", content: "db.AutoMigrate(&User{})", marker: "AutoMigrate"},
		{name: "gorm open", content: "db, _ := gorm.Open(postgres.Open(dsn))", marker: "gorm.Open"},
		{name: "mongo connect", content: "client, _ := mongo.Connect(ctx, options)", marker: "mongo.Connect"},
		{name: "redis client", content: "client := redis.NewClient(&redis.Options{})", marker: "redis.NewClient"},
		{name: "testcontainers", content: "req := testcontainers.ContainerRequest{}", marker: "testcontainers"},
		{name: "dockertest", content: "pool, _ := dockertest.NewPool(\"\")", marker: "dockertest"},
		{name: "drop database", content: "DROP DATABASE lunar;", marker: "DROP DATABASE"},
		{name: "drop table", content: "DROP TABLE users;", marker: "DROP TABLE"},
		{name: "truncate", content: "TRUNCATE TABLE users;", marker: "TRUNCATE"},
		{name: "truncate call", content: "truncate(users)", marker: "truncate("},
		{name: "delete from", content: "DELETE FROM users WHERE 1=1", marker: "DELETE FROM"},
		{name: "flushdb", content: "FLUSHDB", marker: "FLUSHDB"},
		{name: "flushall", content: "cache:flushall", marker: "flushall"},
		{name: "refresh database", content: "use RefreshDatabase;", marker: "RefreshDatabase"},
		{name: "database migrations", content: "use DatabaseMigrations;", marker: "DatabaseMigrations"},
		{name: "migrate fresh", content: "php artisan migrate:fresh", marker: "migrate:fresh"},
		{name: "migrate refresh", content: "php artisan migrate:refresh", marker: "migrate:refresh"},
		{name: "db wipe", content: "php artisan db:wipe", marker: "db:wipe"},
		{name: "schema drop", content: "php artisan schema:drop", marker: "schema:drop"},
		{name: "reset db", content: "resetDB(t)", marker: "resetDB"},
		{name: "refresh db", content: "RefreshDB(t)", marker: "RefreshDB"},
		{name: "flush db", content: "FlushDB()", marker: "FlushDB"},
		{name: "drop all", content: "dropAll()", marker: "dropAll"},
		{name: "clean db", content: "CleanDB()", marker: "CleanDB"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			outcome := ClassifyCandidates([]domain.CandidateTestFile{{
				Path:     "internal/repo/dao_test.go",
				Package:  "./internal/repo",
				Language: domain.LanguageGo,
				Content:  "package repo\n\nfunc TestDao(t *testing.T) {\n\t" + testCase.content + "\n}\n",
			}})

			if len(outcome.BlockedFiles) != 1 {
				t.Fatalf("expected one blocked file, got %+v", outcome.BlockedFiles)
			}
			if !strings.Contains(outcome.BlockedFiles[0].Reason, testCase.marker) {
				t.Fatalf("expected the reason to name %q, got %q", testCase.marker, outcome.BlockedFiles[0].Reason)
			}
			if len(outcome.ExecutedPackages) != 0 {
				t.Fatalf("expected no executed packages, got %+v", outcome.ExecutedPackages)
			}
		})
	}
}

func TestClassifyCandidatesBlocksIntegrationBuildTag(t *testing.T) {
	for _, tag := range []string{"//go:build integration", "// +build integration"} {
		content := tag + "\n\npackage repo\n\nfunc TestIntegration(t *testing.T) {}\n"

		outcome := ClassifyCandidates([]domain.CandidateTestFile{{
			Path:     "internal/repo/dao_test.go",
			Package:  "./internal/repo",
			Language: domain.LanguageGo,
			Content:  content,
		}})

		if len(outcome.BlockedFiles) != 1 {
			t.Fatalf("expected one blocked file for %q, got %+v", tag, outcome.BlockedFiles)
		}
		if outcome.BlockedFiles[0].Reason != integrationTestReason {
			t.Fatalf("expected the integration reason for %q, got %q", tag, outcome.BlockedFiles[0].Reason)
		}
	}
}

func TestClassifyCandidatesKeepsMockOnlyTestsSafe(t *testing.T) {
	content := "package repo\n\nimport \"errors\"\n\ntype storeMock struct{}\n\nfunc (m storeMock) Find(id string) (string, error) { return \"\", errors.New(\"missing\") }\n\nfunc TestDao(t *testing.T) {\n\tvar store storeMock\n\t_, _ = store.Find(\"1\")\n}\n"

	outcome := ClassifyCandidates([]domain.CandidateTestFile{{
		Path:     "internal/repo/dao_test.go",
		Package:  "./internal/repo",
		Language: domain.LanguageGo,
		Content:  content,
	}})

	if len(outcome.TestFiles) != 1 || outcome.TestFiles[0].Classification != domain.ClassificationSafe {
		t.Fatalf("expected a mock only test to be safe, got %+v", outcome.TestFiles)
	}
}

func TestClassifyCandidatesExcludesPHPFiles(t *testing.T) {
	outcome := ClassifyCandidates([]domain.CandidateTestFile{{
		Path:     "tests/Feature/UserTest.php",
		Language: domain.LanguagePHP,
		Content:  "<?php\n\nuse RefreshDatabase;\n",
	}})

	if len(outcome.TestFiles) != 0 {
		t.Fatalf("expected PHP files to be excluded from the test file list, got %+v", outcome.TestFiles)
	}
	if len(outcome.BlockedFiles) != 1 {
		t.Fatalf("expected the PHP file to be reported as not run, got %+v", outcome.BlockedFiles)
	}
	if outcome.BlockedFiles[0].Reason != phpNotSupportedReason {
		t.Fatalf("expected the PHP reason, got %q", outcome.BlockedFiles[0].Reason)
	}
	if len(outcome.ExecutedPackages) != 0 {
		t.Fatalf("expected no executed packages, got %+v", outcome.ExecutedPackages)
	}
}

func TestClassifyCandidatesBlocksNodeTestsWithoutARunner(t *testing.T) {
	outcome := ClassifyCandidates([]domain.CandidateTestFile{{
		Path:     "web/util.test.ts",
		Language: domain.LanguageNode,
		Content:  "test('util', () => {})\n",
	}})

	if len(outcome.TestFiles) != 1 || outcome.TestFiles[0].Classification != domain.ClassificationBlocked {
		t.Fatalf("expected the node test to be blocked, got %+v", outcome.TestFiles)
	}
	if outcome.BlockedFiles[0].Reason != nodeRunnerUnavailableReason {
		t.Fatalf("expected the node runner reason, got %q", outcome.BlockedFiles[0].Reason)
	}
}

func TestClassifyCandidatesBlocksTheWholePackage(t *testing.T) {
	outcome := ClassifyCandidates([]domain.CandidateTestFile{
		{
			Path:     "internal/seed/reset_test.go",
			Package:  "./internal/seed",
			Language: domain.LanguageGo,
			Content:  "package seed\n\nfunc TestReset(t *testing.T) {\n\tTRUNCATE TABLE users\n}\n",
		},
		{
			Path:     "internal/seed/seed_test.go",
			Package:  "./internal/seed",
			Language: domain.LanguageGo,
			Content:  "package seed\n\nfunc TestSeed(t *testing.T) {}\n",
		},
	})

	if len(outcome.ExecutedPackages) != 0 {
		t.Fatalf("expected the package to be blocked, got %+v", outcome.ExecutedPackages)
	}
	if len(outcome.BlockedFiles) != 2 {
		t.Fatalf("expected both files to be named, got %+v", outcome.BlockedFiles)
	}

	reasons := map[string]string{}
	for _, blocked := range outcome.BlockedFiles {
		reasons[blocked.Path] = blocked.Reason
	}
	if !strings.Contains(reasons["internal/seed/reset_test.go"], "TRUNCATE") {
		t.Fatalf("expected the marker reason for the blocking file, got %q", reasons["internal/seed/reset_test.go"])
	}
	if !strings.Contains(reasons["internal/seed/seed_test.go"], "does not run") {
		t.Fatalf("expected the blocked package reason for the safe file, got %q", reasons["internal/seed/seed_test.go"])
	}
}

func TestClassifyCandidatesRunsOnlyFullySafePackages(t *testing.T) {
	outcome := ClassifyCandidates([]domain.CandidateTestFile{
		{
			Path:     "internal/repo/dao_test.go",
			Package:  "./internal/repo",
			Language: domain.LanguageGo,
			Content:  "package repo\n\nfunc TestDao(t *testing.T) {}\n",
		},
		{
			Path:     "internal/seed/seed_test.go",
			Package:  "./internal/seed",
			Language: domain.LanguageGo,
			Content:  "package seed\n\nfunc TestSeed(t *testing.T) {\n\tDROP TABLE users\n}\n",
		},
	})

	if len(outcome.ExecutedPackages) != 1 || outcome.ExecutedPackages[0] != "./internal/repo" {
		t.Fatalf("expected only the safe package to run, got %+v", outcome.ExecutedPackages)
	}
}

func TestClassifyNonTestFilesReportsMarkers(t *testing.T) {
	blockedFiles := ClassifyNonTestFiles([]domain.NonTestFile{
		{
			Path:    "internal/seed/reset.go",
			Content: "package seed\n\nfunc ResetAll(db *sql.DB) {\n\tdb.Exec(\"TRUNCATE users\")\n}\n",
		},
		{
			Path:    "internal/repo/dao.go",
			Content: "package repo\n\nfunc Find(id string) string { return id }\n",
		},
	})

	if len(blockedFiles) != 1 {
		t.Fatalf("expected one reported non-test file, got %+v", blockedFiles)
	}
	if blockedFiles[0].Path != "internal/seed/reset.go" {
		t.Fatalf("expected the reset helper, got %+v", blockedFiles[0])
	}
	if !strings.Contains(blockedFiles[0].Reason, "TRUNCATE") {
		t.Fatalf("expected the marker in the reason, got %q", blockedFiles[0].Reason)
	}
}

func TestFindMarkersReturnsEveryMatch(t *testing.T) {
	markers := FindMarkers("OpenDB(config)\nTRUNCATE users\n")
	if len(markers) != 2 {
		t.Fatalf("expected two markers, got %+v", markers)
	}
	if markers[0] != "OpenDB(" || markers[1] != "TRUNCATE" {
		t.Fatalf("expected the declared marker order, got %+v", markers)
	}
}

func TestSelectedTestFunctionsOnlyFromSafeExecutedPackages(t *testing.T) {
	outcome := ClassifyCandidates([]domain.CandidateTestFile{
		{
			Path:          "internal/repo/dao_test.go",
			Package:       "./internal/repo",
			Language:      domain.LanguageGo,
			Content:       "package repo\n\nfunc TestFind(t *testing.T) {}\n\nfunc TestSave(t *testing.T) {}\n",
			TestFunctions: []string{"TestSave", "TestFind"},
		},
		{
			Path:          "internal/seed/reset_test.go",
			Package:       "./internal/seed",
			Language:      domain.LanguageGo,
			Content:       "package seed\n\nfunc TestReset(t *testing.T) {\n\tTRUNCATE TABLE users\n}\n",
			TestFunctions: []string{"TestReset"},
		},
	})

	functions, files := SelectedTestFunctions(outcome)
	if strings.Join(functions, ",") != "TestFind,TestSave" {
		t.Fatalf("expected the sorted safe functions, got %+v", functions)
	}
	if files["TestFind"] != "internal/repo/dao_test.go" {
		t.Fatalf("expected the file mapping, got %+v", files)
	}
	if _, exists := files["TestReset"]; exists {
		t.Fatalf("expected the blocked function to be excluded, got %+v", files)
	}
}

func TestSelectedTestFunctionsIgnoresCandidatesWithoutFunctions(t *testing.T) {
	functions, files := SelectedTestFunctions(ClassificationOutcome{})
	if len(functions) != 0 || len(files) != 0 {
		t.Fatalf("expected no selected functions, got %+v %+v", functions, files)
	}
}
