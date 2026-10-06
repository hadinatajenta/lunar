package application

import (
	"context"
	"strings"
	"sync"
	"testing"

	jiraDomain "lunar/backend/internal/jira/domain"
	"lunar/backend/internal/workspace/domain"
)

type stubSuggestionRoots struct {
	rootPath string
	err      error
}

func (s stubSuggestionRoots) WorkspaceRootFor(_ context.Context, _ string) (string, error) {
	if s.err != nil {
		return "", s.err
	}

	return s.rootPath, nil
}

type stubSuggestionIssues struct {
	issue      *jiraDomain.JiraIssue
	backlog    *jiraDomain.JiraBacklogResponse
	issueErr   error
	backlogErr error
}

func (s stubSuggestionIssues) GetIssueDetail(_ context.Context, _ string, _ string) (*jiraDomain.JiraIssue, error) {
	if s.issueErr != nil {
		return nil, s.issueErr
	}

	return s.issue, nil
}

func (s stubSuggestionIssues) GetBacklog(_ context.Context, _ string, _ string) (*jiraDomain.JiraBacklogResponse, error) {
	if s.backlogErr != nil {
		return nil, s.backlogErr
	}

	return s.backlog, nil
}

type stubSuggestionBranches struct {
	matches map[string][]BranchMatch
	err     error
	keys    []string
}

func (s *stubSuggestionBranches) SprintBranchesAtRoot(_ context.Context, _ string, issueKeys []string) (*SprintBranches, error) {
	s.keys = append(s.keys, issueKeys...)
	if s.err != nil {
		return nil, s.err
	}

	matches := make(map[string][]BranchMatch, len(issueKeys))
	for _, key := range issueKeys {
		matches[key] = s.matches[key]
		if matches[key] == nil {
			matches[key] = []BranchMatch{}
		}
	}

	return &SprintBranches{Matches: matches}, nil
}

type stubCatalogueLocal struct {
	repositories []domain.RepositorySummary
	err          error
}

func (s stubCatalogueLocal) ListRepositories(_ context.Context, _ string) ([]domain.RepositorySummary, error) {
	if s.err != nil {
		return nil, s.err
	}

	return s.repositories, nil
}

type stubCatalogueRemote struct {
	repositories []RemoteRepository
	err          error
}

func (s stubCatalogueRemote) ListRemoteRepositories(_ context.Context, _ string) ([]RemoteRepository, error) {
	if s.err != nil {
		return nil, s.err
	}

	return s.repositories, nil
}

type stubSuggestionCode struct {
	matchesByTerm map[string][]CodeMatch
	err           error
}

func (s stubSuggestionCode) SearchCode(_ context.Context, _ string, term string, _ int) ([]CodeMatch, error) {
	if s.err != nil {
		return nil, s.err
	}

	return s.matchesByTerm[term], nil
}

type stubSuggestionPages struct {
	pages []LinkedPage
	err   error
}

func (s stubSuggestionPages) LinkedPages(_ context.Context, _ string, _ string) ([]LinkedPage, error) {
	if s.err != nil {
		return nil, s.err
	}

	return s.pages, nil
}

type stubSuggestionRanker struct {
	ranked   []RankedRepository
	err      error
	requests []RepositoryRankRequest
}

func (s *stubSuggestionRanker) RankRepositories(_ context.Context, request RepositoryRankRequest) ([]RankedRepository, error) {
	s.requests = append(s.requests, request)
	if s.err != nil {
		return nil, s.err
	}

	return s.ranked, nil
}

type stubSuggestionRemoteScanner struct {
	mutex                sync.Mutex
	branchesByRepository map[string][]string
	searchCalls          []string
	listCalls            []string
	err                  error
}

func (s *stubSuggestionRemoteScanner) SearchBranches(_ context.Context, _ string, repo string, filterText string) ([]string, error) {
	s.mutex.Lock()
	s.searchCalls = append(s.searchCalls, repo+"|"+filterText)
	stubErr := s.err
	s.mutex.Unlock()

	if stubErr != nil {
		return nil, stubErr
	}

	return s.branchesByRepository[repo], nil
}

func (s *stubSuggestionRemoteScanner) ListBranches(_ context.Context, _ string, repo string) ([]string, error) {
	s.mutex.Lock()
	s.listCalls = append(s.listCalls, repo)
	s.mutex.Unlock()

	return s.branchesByRepository[repo], nil
}

func (s *stubSuggestionRemoteScanner) calledSearches() []string {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	return append([]string{}, s.searchCalls...)
}

func (s *stubSuggestionRemoteScanner) calledLists() []string {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	return append([]string{}, s.listCalls...)
}

func (s *stubSuggestionRemoteScanner) DefaultBranch(_ context.Context, _ string, _ string) (string, error) {
	return "main", nil
}

func testLocalRepositories(names ...string) []domain.RepositorySummary {
	repositories := make([]domain.RepositorySummary, 0, len(names))
	for _, name := range names {
		repositories = append(repositories, domain.RepositorySummary{Name: name, IsGit: true})
	}

	return repositories
}

func testRemoteRepositories(project string, names ...string) []RemoteRepository {
	repositories := make([]RemoteRepository, 0, len(names))
	for _, name := range names {
		repositories = append(repositories, RemoteRepository{Repo: name, Project: project})
	}

	return repositories
}

func testCatalogueService(local LocalRepositorySource, remote RemoteRepositorySource) *RepositoryCatalogueService {
	return NewRepositoryCatalogueService(stubSuggestionRoots{rootPath: "/workspace"}, local, remote)
}

func testSuggestionService(catalogue *RepositoryCatalogueService, issues IssueContextSource, branches BranchMatchSource, options ...RepositorySuggestionOption) *RepositorySuggestionService {
	return NewRepositorySuggestionService(RepositorySuggestionDependencies{
		Catalogue:      catalogue,
		WorkspaceRoots: stubSuggestionRoots{rootPath: "/workspace"},
		Issues:         issues,
		Branches:       branches,
	}, options...)
}

func testIssue(key string, epicKey string, summary string, description string) *jiraDomain.JiraIssue {
	return &jiraDomain.JiraIssue{
		Key:        key,
		SprintName: "Sprint 21",
		Fields: jiraDomain.JiraIssueFields{
			Summary:     summary,
			Description: description,
			Parent:      &jiraDomain.JiraParent{Key: epicKey},
			IssueType:   &jiraDomain.JiraIssueType{Name: "Story"},
		},
	}
}

func testBacklog(siblingKeys ...string) *jiraDomain.JiraBacklogResponse {
	issues := make([]jiraDomain.JiraIssue, 0, len(siblingKeys)+1)
	for _, key := range siblingKeys {
		issues = append(issues, jiraDomain.JiraIssue{
			Key:        key,
			SprintName: "Sprint 21",
			Fields:     jiraDomain.JiraIssueFields{Parent: &jiraDomain.JiraParent{Key: "CRMMS-100"}},
		})
	}
	issues = append(issues, jiraDomain.JiraIssue{
		Key:        "CRMMS-77999",
		SprintName: "Sprint 21",
		Fields:     jiraDomain.JiraIssueFields{Parent: &jiraDomain.JiraParent{Key: "CRMMS-200"}},
	})

	return &jiraDomain.JiraBacklogResponse{Sprints: []jiraDomain.JiraSprint{{Name: "Sprint 21", State: "active", Issues: issues}}}
}

func assertEverySuggestionHasAReason(t *testing.T, suggestions *RepositorySuggestions) {
	t.Helper()

	for _, suggestion := range suggestions.Suggestions {
		if len(suggestion.Reasons) == 0 {
			t.Fatalf("a suggestion without a reason is a defect: %s", suggestion.Repo)
		}
	}
}

func assertReasonInvariants(t *testing.T, suggestions *RepositorySuggestions) {
	t.Helper()

	for _, suggestion := range suggestions.Suggestions {
		for _, reason := range suggestion.Reasons {
			weight, known := signalWeights[reason.Signal]
			if !known {
				t.Fatalf("unknown signal %q in %s", reason.Signal, suggestion.Repo)
			}
			if reason.Weight != weight {
				t.Fatalf("signal %q carries weight %v, expected %v", reason.Signal, reason.Weight, weight)
			}
			if reason.Strength <= 0 || reason.Strength > 1 {
				t.Fatalf("signal %q carries an out of range strength %v", reason.Signal, reason.Strength)
			}
			if strings.TrimSpace(reason.Text) == "" {
				t.Fatalf("signal %q carries no text", reason.Signal)
			}
		}
	}
}

func findReason(suggestion RepositorySuggestion, signal string) (SuggestionReason, bool) {
	for _, reason := range suggestion.Reasons {
		if reason.Signal == signal {
			return reason, true
		}
	}

	return SuggestionReason{}, false
}

func findUnavailable(suggestions *RepositorySuggestions, signal string) (UnavailableSignal, bool) {
	for _, unavailable := range suggestions.SignalsUnavailable {
		if unavailable.Signal == signal {
			return unavailable, true
		}
	}

	return UnavailableSignal{}, false
}
