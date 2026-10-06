package application

import (
	"fmt"
	"testing"

	"lunar/backend/internal/workspace/domain"
)

func TestClassifyWorkspaceReadinessTreatsAnUnsetRootAsNotConfigured(t *testing.T) {
	readiness := ClassifyWorkspaceReadiness(WorkspaceReadinessFacts{})

	if readiness.State != ReadinessNotConfigured {
		t.Fatalf("expected not-configured, got %q", readiness.State)
	}
	if readiness.Message != "No workspace folder is configured. Choose the folder that holds your service repositories." {
		t.Fatalf("unexpected message %q", readiness.Message)
	}
	if readiness.MissingRepositories == nil {
		t.Fatalf("expected an empty missing repository list, got nil")
	}
}

func TestClassifyWorkspaceReadinessReportsARefusedRootAsInvalid(t *testing.T) {
	readiness := ClassifyWorkspaceReadiness(WorkspaceReadinessFacts{
		RootPath:  "/Users/erendt/BCA",
		RootError: fmt.Errorf("%w: /Users/erendt/BCA", domain.ErrRootNotFound),
	})

	if readiness.State != ReadinessInvalid {
		t.Fatalf("expected invalid, got %q", readiness.State)
	}
	if readiness.Message != "workspace root does not exist" {
		t.Fatalf("unexpected message %q", readiness.Message)
	}
}

func TestClassifyWorkspaceReadinessReportsAnEmptyWorkspace(t *testing.T) {
	readiness := ClassifyWorkspaceReadiness(WorkspaceReadinessFacts{RootPath: "/Users/erendt/BRI"})

	if readiness.State != ReadinessEmpty {
		t.Fatalf("expected empty, got %q", readiness.State)
	}
	if readiness.Message != "No repositories were found in /Users/erendt/BRI." {
		t.Fatalf("unexpected message %q", readiness.Message)
	}
}

func TestClassifyWorkspaceReadinessReportsARootThatIsItselfARepository(t *testing.T) {
	readiness := ClassifyWorkspaceReadiness(WorkspaceReadinessFacts{
		RootPath:              "/Users/erendt/mms-api",
		RootIsRepository:      true,
		RemoteRepositoryCount: 65,
	})

	if readiness.State != ReadinessSingleRepo {
		t.Fatalf("expected single-repo, got %q", readiness.State)
	}
	expectedMessage := "/Users/erendt/mms-api is itself a repository. Point Lunar at the folder that contains your repositories instead."
	if readiness.Message != expectedMessage {
		t.Fatalf("unexpected message %q", readiness.Message)
	}
}

func TestClassifyWorkspaceReadinessReportsAWorkspaceWithoutABranchForTheProjectInView(t *testing.T) {
	readiness := ClassifyWorkspaceReadiness(WorkspaceReadinessFacts{
		RootPath:             "/Users/erendt/BCA",
		LocalRepositoryCount: 3,
		BranchNames:          []string{"feature/OTHERPROJ-14-login", "bugfix/ABC-2-fix", "main"},
		ProjectKey:           "CRMMS",
	})

	if readiness.State != ReadinessForeign {
		t.Fatalf("expected foreign, got %q", readiness.State)
	}
	expectedMessage := "The repositories in /Users/erendt/BCA carry branches for other Jira projects, but none for the project you are viewing."
	if readiness.Message != expectedMessage {
		t.Fatalf("unexpected message %q", readiness.Message)
	}
}

func TestClassifyWorkspaceReadinessNeverTreatsAWorkspaceWithoutKeyedBranchesAsForeign(t *testing.T) {
	readiness := ClassifyWorkspaceReadiness(WorkspaceReadinessFacts{
		RootPath:              "/Users/erendt/BRI",
		LocalRepositoryCount:  3,
		BranchNames:           []string{"main", "develop", "release/production"},
		ProjectKey:            "CRMMS",
		RemoteRepositoryCount: 3,
	})

	if readiness.State != ReadinessReady {
		t.Fatalf("expected a fresh workspace to be ready, got %q", readiness.State)
	}
}

func TestClassifyWorkspaceReadinessReportsPartialCoverage(t *testing.T) {
	readiness := ClassifyWorkspaceReadiness(WorkspaceReadinessFacts{
		RootPath:              "/Users/erendt/BRI",
		LocalRepositoryCount:  2,
		BranchNames:           []string{"feature/CRMMS-77911-enhancement", "main"},
		ProjectKey:            "CRMMS",
		RemoteRepositoryCount: 5,
	})

	if readiness.State != ReadinessPartial {
		t.Fatalf("expected partial, got %q", readiness.State)
	}
	if readiness.Message != "Searching 2 of 5 repositories. 3 are not cloned locally." {
		t.Fatalf("unexpected message %q", readiness.Message)
	}
	if readiness.MissingRepositoryCount != 3 {
		t.Fatalf("expected 3 missing repositories, got %d", readiness.MissingRepositoryCount)
	}
}

func TestClassifyWorkspaceReadinessReportsAReadyWorkspace(t *testing.T) {
	readiness := ClassifyWorkspaceReadiness(WorkspaceReadinessFacts{
		RootPath:              "/Users/erendt/BRI",
		LocalRepositoryCount:  4,
		BranchNames:           []string{"feature/CRMMS-77911-enhancement", "feature/OTHERPROJ-14-login"},
		ProjectKey:            "CRMMS",
		RemoteRepositoryCount: 4,
	})

	if readiness.State != ReadinessReady {
		t.Fatalf("expected ready, got %q", readiness.State)
	}
	if readiness.Message != "Ready" {
		t.Fatalf("unexpected message %q", readiness.Message)
	}
}

func TestClassifyWorkspaceReadinessMatchesTheProjectCaseInsensitively(t *testing.T) {
	readiness := ClassifyWorkspaceReadiness(WorkspaceReadinessFacts{
		RootPath:              "/Users/erendt/BRI",
		LocalRepositoryCount:  1,
		BranchNames:           []string{"feature/CRMMS-77911-enhancement"},
		ProjectKey:            "crmms",
		RemoteRepositoryCount: 1,
	})

	if readiness.State != ReadinessReady {
		t.Fatalf("expected ready, got %q", readiness.State)
	}
}

func TestClassifyWorkspaceReadinessDoesNotJudgeForeignWithoutAProjectInView(t *testing.T) {
	readiness := ClassifyWorkspaceReadiness(WorkspaceReadinessFacts{
		RootPath:              "/Users/erendt/BCA",
		LocalRepositoryCount:  2,
		BranchNames:           []string{"feature/OTHERPROJ-14-login"},
		RemoteRepositoryCount: 2,
	})

	if readiness.State != ReadinessReady {
		t.Fatalf("expected ready without a project in view, got %q", readiness.State)
	}
}

func TestClassifyWorkspaceReadinessCapsAndSortsTheMissingRepositories(t *testing.T) {
	missing := make([]string, 0, 60)
	for index := 60; index > 0; index-- {
		missing = append(missing, fmt.Sprintf("service-%02d", index))
	}
	missing = append(missing, "service-01", "  ")

	readiness := ClassifyWorkspaceReadiness(WorkspaceReadinessFacts{
		RootPath:              "/Users/erendt/BRI",
		LocalRepositoryCount:  5,
		RemoteRepositoryCount: 65,
		MissingRepositories:   missing,
	})

	if len(readiness.MissingRepositories) != maximumMissingRepositories {
		t.Fatalf("expected %d missing repositories, got %d", maximumMissingRepositories, len(readiness.MissingRepositories))
	}
	if readiness.MissingRepositories[0] != "service-01" || readiness.MissingRepositories[49] != "service-50" {
		t.Fatalf("expected a sorted, deduplicated list, got %q ... %q", readiness.MissingRepositories[0], readiness.MissingRepositories[49])
	}
	if readiness.MissingRepositoryCount != 60 {
		t.Fatalf("expected 60 missing repositories, got %d", readiness.MissingRepositoryCount)
	}
}

func TestClassifyWorkspaceReadinessNeverInventsAMissingRepositoryCount(t *testing.T) {
	readiness := ClassifyWorkspaceReadiness(WorkspaceReadinessFacts{
		RootPath:              "/Users/erendt/BRI",
		LocalRepositoryCount:  36,
		RemoteRepositoryCount: 36,
	})

	if readiness.MissingRepositoryCount != 0 {
		t.Fatalf("expected no missing repositories, got %d", readiness.MissingRepositoryCount)
	}
}

func TestClassifyWorkspaceReadinessIgnoresAnUnknownRemoteCatalogue(t *testing.T) {
	readiness := ClassifyWorkspaceReadiness(WorkspaceReadinessFacts{
		RootPath:             "/Users/erendt/BRI",
		LocalRepositoryCount: 36,
		BranchNames:          []string{"feature/CRMMS-77911-enhancement"},
		ProjectKey:           "CRMMS",
	})

	if readiness.State != ReadinessReady {
		t.Fatalf("expected ready when the remote catalogue is unavailable, got %q", readiness.State)
	}
	if readiness.RemoteRepositoryCount != 0 {
		t.Fatalf("expected an unknown remote count to stay zero, got %d", readiness.RemoteRepositoryCount)
	}
}

func TestClassifyWorkspaceReadinessReportsAnInvalidRootBeforeAnyCoverageState(t *testing.T) {
	readiness := ClassifyWorkspaceReadiness(WorkspaceReadinessFacts{
		RootPath:              "/Users/erendt/BCA",
		RootError:             fmt.Errorf("%w: /Users/erendt/BCA", domain.ErrRootOutsideScope),
		LocalRepositoryCount:  36,
		RemoteRepositoryCount: 65,
	})

	if readiness.State != ReadinessInvalid {
		t.Fatalf("expected invalid, got %q", readiness.State)
	}
	if readiness.Message != "workspace root is outside the allowed roots" {
		t.Fatalf("unexpected message %q", readiness.Message)
	}
}

func TestClassifyWorkspaceReadinessDescribesANonDirectoryRoot(t *testing.T) {
	readiness := ClassifyWorkspaceReadiness(WorkspaceReadinessFacts{
		RootPath:  "/Users/erendt/some-file",
		RootError: domain.ErrRootNotDirectory,
	})

	if readiness.State != ReadinessInvalid {
		t.Fatalf("expected invalid, got %q", readiness.State)
	}
	if readiness.Message != "workspace root is not a directory" {
		t.Fatalf("unexpected message %q", readiness.Message)
	}
}
