package application

import (
	"fmt"
	"sort"
	"strings"

	"lunar/backend/internal/utdocs/domain"
)

const (
	integrationBuildTag       = "//go:build integration"
	legacyIntegrationBuildTag = "// +build integration"
	mockLibraryMarker         = "go-sqlmock"

	phpNotSupportedReason       = "PHP test files are not executed by the unit test runner"
	nodeRunnerUnavailableReason = "the node test runner is not available yet"
	integrationTestReason       = "integration test"

	maxSelectedTestFunctions = 500
)

var databaseConnectionMarkers = []string{
	"sql.Open",
	"gorm.Open",
	"sqlx.Open",
	"sqlx.Connect",
	"pgx.Connect",
	"mongo.Connect",
	"redis.NewClient",
	"OpenDB(",
	"AutoMigrate",
	"Migrate(",
	"testcontainers",
	"dockertest",
}

var destructiveMarkers = []string{
	"DROP DATABASE",
	"DROP TABLE",
	"TRUNCATE",
	"DELETE FROM",
	"FLUSHDB",
	"FLUSHALL",
	"flushall",
	"RefreshDatabase",
	"DatabaseMigrations",
	"migrate:fresh",
	"migrate:refresh",
	"db:wipe",
	"schema:drop",
	"resetDB",
	"ResetDB",
	"refreshDB",
	"RefreshDB",
	"flushDB",
	"FlushDB",
	"dropAll",
	"DropAll",
	"cleanDB",
	"CleanDB",
	"truncate(",
}

var realDatabaseDriverImports = []string{
	"github.com/lib/pq",
	"github.com/jackc/pgx",
	"github.com/go-sql-driver/mysql",
	"gorm.io/gorm",
	"gorm.io/driver",
	"github.com/jmoiron/sqlx",
	"go.mongodb.org/mongo-driver",
	"github.com/redis/go-redis",
	"github.com/gomodule/redigo",
	"modernc.org/sqlite",
	"github.com/mattn/go-sqlite3",
	"github.com/microsoft/go-mssqldb",
	"github.com/denisenkom/go-mssqldb",
}

type ClassificationOutcome struct {
	TestFiles        []domain.TestFile
	BlockedFiles     []domain.BlockedFile
	ExecutedPackages []string
}

func ClassifyCandidates(candidates []domain.CandidateTestFile) ClassificationOutcome {
	outcome := ClassificationOutcome{
		TestFiles:        make([]domain.TestFile, 0, len(candidates)),
		BlockedFiles:     make([]domain.BlockedFile, 0),
		ExecutedPackages: make([]string, 0),
	}

	packageOrder := make([]string, 0)
	candidatesByPackage := make(map[string][]domain.TestFile)
	blockedReasonByPackage := make(map[string]string)

	for _, candidate := range candidates {
		if candidate.Language == domain.LanguagePHP {
			outcome.BlockedFiles = append(outcome.BlockedFiles, domain.BlockedFile{
				Path:           candidate.Path,
				Classification: domain.ClassificationBlocked,
				Reason:         phpNotSupportedReason,
			})
			continue
		}
		if candidate.Language == domain.LanguageNode {
			testFile := domain.TestFile{
				Path:           candidate.Path,
				Package:        candidate.Package,
				Language:       candidate.Language,
				Classification: domain.ClassificationBlocked,
				Reason:         nodeRunnerUnavailableReason,
			}
			outcome.TestFiles = append(outcome.TestFiles, testFile)
			outcome.BlockedFiles = append(outcome.BlockedFiles, domain.BlockedFile{
				Path:           candidate.Path,
				Classification: domain.ClassificationBlocked,
				Reason:         nodeRunnerUnavailableReason,
			})
			continue
		}

		classification, reason := classifyGoTest(candidate.Content)
		testFile := domain.TestFile{
			Path:           candidate.Path,
			Package:        candidate.Package,
			Language:       candidate.Language,
			Classification: classification,
			Reason:         reason,
			TestFunctions:  candidate.TestFunctions,
		}
		outcome.TestFiles = append(outcome.TestFiles, testFile)

		if _, tracked := candidatesByPackage[candidate.Package]; !tracked {
			packageOrder = append(packageOrder, candidate.Package)
		}
		candidatesByPackage[candidate.Package] = append(candidatesByPackage[candidate.Package], testFile)
		if classification == domain.ClassificationBlocked && blockedReasonByPackage[candidate.Package] == "" {
			blockedReasonByPackage[candidate.Package] = reason
		}
	}

	for _, testFile := range outcome.TestFiles {
		if testFile.Language == domain.LanguageGo && testFile.Classification == domain.ClassificationBlocked {
			outcome.BlockedFiles = append(outcome.BlockedFiles, domain.BlockedFile{
				Path:           testFile.Path,
				Classification: domain.ClassificationBlocked,
				Reason:         testFile.Reason,
			})
		}
	}

	for _, packagePath := range packageOrder {
		if _, isBlocked := blockedReasonByPackage[packagePath]; isBlocked {
			for _, testFile := range candidatesByPackage[packagePath] {
				if testFile.Classification == domain.ClassificationBlocked {
					continue
				}
				outcome.BlockedFiles = append(outcome.BlockedFiles, domain.BlockedFile{
					Path:           testFile.Path,
					Classification: domain.ClassificationBlocked,
					Reason:         fmt.Sprintf("its package %s does not run because the package contains a blocked test file", packagePath),
				})
			}
			continue
		}
		outcome.ExecutedPackages = append(outcome.ExecutedPackages, packagePath)
	}

	return outcome
}

func SelectedTestFunctions(outcome ClassificationOutcome) ([]string, map[string]string) {
	executedPackages := make(map[string]struct{}, len(outcome.ExecutedPackages))
	for _, packagePath := range outcome.ExecutedPackages {
		executedPackages[packagePath] = struct{}{}
	}

	filesByFunction := make(map[string]string)
	for _, testFile := range outcome.TestFiles {
		if testFile.Language != domain.LanguageGo || testFile.Classification != domain.ClassificationSafe {
			continue
		}
		if _, executed := executedPackages[testFile.Package]; !executed {
			continue
		}
		for _, function := range testFile.TestFunctions {
			trimmed := strings.TrimSpace(function)
			if trimmed == "" {
				continue
			}
			if _, exists := filesByFunction[trimmed]; !exists {
				filesByFunction[trimmed] = testFile.Path
			}
		}
	}

	functions := make([]string, 0, len(filesByFunction))
	for function := range filesByFunction {
		functions = append(functions, function)
	}
	sort.Strings(functions)
	if len(functions) > maxSelectedTestFunctions {
		functions = functions[:maxSelectedTestFunctions]
	}
	return functions, filesByFunction
}

func ClassifyNonTestFiles(files []domain.NonTestFile) []domain.BlockedFile {
	blockedFiles := make([]domain.BlockedFile, 0)
	for _, file := range files {
		markers := FindMarkers(file.Content)
		if len(markers) == 0 {
			continue
		}
		blockedFiles = append(blockedFiles, domain.BlockedFile{
			Path:           file.Path,
			Classification: domain.ClassificationBlocked,
			Reason:         "changed non-test file matches destructive or database markers: " + strings.Join(markers, ", "),
		})
	}
	return blockedFiles
}

func classifyGoTest(content string) (domain.Classification, string) {
	if strings.Contains(content, integrationBuildTag) || strings.Contains(content, legacyIntegrationBuildTag) {
		return domain.ClassificationBlocked, integrationTestReason
	}
	if markers := FindMarkers(content); len(markers) > 0 {
		return domain.ClassificationBlocked, "contains destructive or database markers: " + strings.Join(markers, ", ")
	}
	if strings.Contains(content, mockLibraryMarker) {
		return domain.ClassificationSafe, ""
	}
	if driver, found := findRealDriverImport(content); found {
		return domain.ClassificationBlocked, "imports a real database driver without go-sqlmock: " + driver
	}
	return domain.ClassificationSafe, ""
}

func FindMarkers(content string) []string {
	matched := make([]string, 0)
	for _, marker := range databaseConnectionMarkers {
		if strings.Contains(content, marker) {
			matched = append(matched, marker)
		}
	}
	for _, marker := range destructiveMarkers {
		if strings.Contains(content, marker) {
			matched = append(matched, marker)
		}
	}
	return matched
}

func findRealDriverImport(content string) (string, bool) {
	for _, driver := range realDatabaseDriverImports {
		if strings.Contains(content, driver) {
			return driver, true
		}
	}
	return "", false
}
