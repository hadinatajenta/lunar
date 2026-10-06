package application

import (
	"fmt"
	"sort"
	"strings"

	jiraDomain "lunar/backend/internal/jira/domain"
)

type WorkspaceReadinessState string

const (
	ReadinessNotConfigured WorkspaceReadinessState = "not-configured"
	ReadinessInvalid       WorkspaceReadinessState = "invalid"
	ReadinessEmpty         WorkspaceReadinessState = "empty"
	ReadinessSingleRepo    WorkspaceReadinessState = "single-repo"
	ReadinessForeign       WorkspaceReadinessState = "foreign"
	ReadinessPartial       WorkspaceReadinessState = "partial"
	ReadinessReady         WorkspaceReadinessState = "ready"
)

const (
	messageNoWorkspaceConfigured = "No workspace folder is configured. Choose the folder that holds your service repositories."
	messageNoRepositoriesFound   = "No repositories were found in %s."
	messageSingleRepository      = "%s is itself a repository. Point Lunar at the folder that contains your repositories instead."
	messageForeignProject        = "The repositories in %s carry branches for other Jira projects, but none for the project you are viewing."
	messagePartialCoverage       = "Searching %d of %d repositories. %d are not cloned locally."
	messageReady                 = "Ready"
	maximumMissingRepositories   = 50
)

type WorkspaceReadiness struct {
	State                  WorkspaceReadinessState `json:"state"`
	RootPath               string                  `json:"root_path"`
	Message                string                  `json:"message"`
	LocalRepositoryCount   int                     `json:"local_repository_count"`
	RemoteRepositoryCount  int                     `json:"remote_repository_count"`
	MissingRepositoryCount int                     `json:"missing_repository_count"`
	MissingRepositories    []string                `json:"missing_repositories"`
}

type WorkspaceReadinessFacts struct {
	RootPath              string
	RootError             error
	RootIsRepository      bool
	LocalRepositoryCount  int
	BranchNames           []string
	ProjectKey            string
	RemoteRepositoryCount int
	MissingRepositories   []string
}

func ClassifyWorkspaceReadiness(facts WorkspaceReadinessFacts) WorkspaceReadiness {
	boundedMissing := boundedMissingRepositories(facts.MissingRepositories)
	readiness := WorkspaceReadiness{
		RootPath:               strings.TrimSpace(facts.RootPath),
		LocalRepositoryCount:   facts.LocalRepositoryCount,
		RemoteRepositoryCount:  facts.RemoteRepositoryCount,
		MissingRepositories:    boundedMissing,
		MissingRepositoryCount: missingRepositoryCount(facts, boundedMissing),
	}

	if readiness.RootPath == "" {
		readiness.State = ReadinessNotConfigured
		readiness.Message = messageNoWorkspaceConfigured
		return readiness
	}

	if facts.RootError != nil {
		readiness.State = ReadinessInvalid
		readiness.Message = describeRootError(facts.RootError)
		return readiness
	}

	if facts.LocalRepositoryCount == 0 && facts.RootIsRepository {
		readiness.State = ReadinessSingleRepo
		readiness.Message = fmt.Sprintf(messageSingleRepository, readiness.RootPath)
		return readiness
	}

	if facts.LocalRepositoryCount == 0 {
		readiness.State = ReadinessEmpty
		readiness.Message = fmt.Sprintf(messageNoRepositoriesFound, readiness.RootPath)
		return readiness
	}

	if isForeignWorkspace(facts) {
		readiness.State = ReadinessForeign
		readiness.Message = fmt.Sprintf(messageForeignProject, readiness.RootPath)
		return readiness
	}

	if facts.RemoteRepositoryCount > facts.LocalRepositoryCount {
		readiness.State = ReadinessPartial
		readiness.Message = fmt.Sprintf(messagePartialCoverage, facts.LocalRepositoryCount, facts.RemoteRepositoryCount, readiness.MissingRepositoryCount)
		return readiness
	}

	readiness.State = ReadinessReady
	readiness.Message = messageReady
	return readiness
}

func isForeignWorkspace(facts WorkspaceReadinessFacts) bool {
	projectKey := strings.ToUpper(strings.TrimSpace(facts.ProjectKey))
	if projectKey == "" {
		return false
	}

	hasKeyedBranch := false
	hasProjectKey := false
	for _, branchName := range facts.BranchNames {
		for _, issueKey := range jiraDomain.IssueKeysIn(branchName) {
			hasKeyedBranch = true
			if projectOfIssueKey(issueKey) == projectKey {
				hasProjectKey = true
			}
		}
	}

	return hasKeyedBranch && !hasProjectKey
}

func projectOfIssueKey(issueKey string) string {
	separator := strings.Index(issueKey, "-")
	if separator <= 0 {
		return ""
	}

	return strings.ToUpper(strings.TrimSpace(issueKey[:separator]))
}

func missingRepositoryCount(facts WorkspaceReadinessFacts, boundedMissing []string) int {
	if facts.RemoteRepositoryCount > 0 {
		missing := facts.RemoteRepositoryCount - facts.LocalRepositoryCount
		if missing < 0 {
			return 0
		}
		return missing
	}

	return len(boundedMissing)
}

func boundedMissingRepositories(repositories []string) []string {
	seen := make(map[string]struct{}, len(repositories))
	names := make([]string, 0, len(repositories))

	for _, repository := range repositories {
		trimmed := strings.TrimSpace(repository)
		if trimmed == "" {
			continue
		}
		if _, duplicate := seen[trimmed]; duplicate {
			continue
		}
		seen[trimmed] = struct{}{}
		names = append(names, trimmed)
	}

	sort.Strings(names)
	if len(names) > maximumMissingRepositories {
		names = names[:maximumMissingRepositories]
	}

	return names
}
