package application

import (
	"context"
	"fmt"
	"sort"
	"strings"

	jiraDomain "lunar/backend/internal/jira/domain"
)

const (
	siblingBacklogReason  = "the issue's epic and sprint could not be read from Jira"
	siblingNoIssueReason  = "no other issue was found in this epic or sprint"
	siblingBranchReason   = "the workspace branches could not be read"
	siblingNoBranchReason = "no other issue in this epic has a branch in the workspace"
)

type siblingEvidence struct {
	epicKey        string
	sprintFallback bool
	siblingKeys    map[string]struct{}
	keysByRepo     map[string]map[string]struct{}
	workedKeys     map[string]struct{}
}

func newSiblingEvidence(epicKey string, sprintFallback bool) *siblingEvidence {
	return &siblingEvidence{
		epicKey:        epicKey,
		sprintFallback: sprintFallback,
		siblingKeys:    make(map[string]struct{}),
		keysByRepo:     make(map[string]map[string]struct{}),
		workedKeys:     make(map[string]struct{}),
	}
}

func (e *siblingEvidence) add(repo string, issueKey string) {
	name := strings.TrimSpace(repo)
	key := strings.ToUpper(strings.TrimSpace(issueKey))
	if name == "" || key == "" {
		return
	}

	repositoryKeys := e.keysByRepo[name]
	if repositoryKeys == nil {
		repositoryKeys = make(map[string]struct{})
		e.keysByRepo[name] = repositoryKeys
	}
	repositoryKeys[key] = struct{}{}
	e.workedKeys[key] = struct{}{}
}

func (e *siblingEvidence) reasonText(repo string, count int) string {
	location := "this epic"
	if e.sprintFallback {
		location = "this sprint"
	}

	subject := "issues"
	verb := "were"
	if count == 1 {
		subject = "issue"
		verb = "was"
	}

	return fmt.Sprintf("%d other %s in %s %s worked in %s", count, subject, location, verb, repo)
}

func (s *RepositorySuggestionService) collectSiblingEvidence(ctx context.Context, userID string, issue *jiraDomain.JiraIssue, rootPath string, state *suggestionState) {
	state.epicKey = issueEpicKey(issue)

	siblings, sprintFallback, err := s.siblingIssueKeys(ctx, userID, issue)
	if err != nil {
		state.markUnavailable(SignalEpicSiblings, siblingBacklogReason)
		return
	}
	if len(siblings) == 0 {
		state.markUnavailable(SignalEpicSiblings, siblingNoIssueReason)
		return
	}
	if s.branches == nil {
		state.markUnavailable(SignalEpicSiblings, siblingBranchReason)
		return
	}

	matches, err := s.branches.SprintBranchesAtRoot(ctx, rootPath, siblings)
	if err != nil {
		state.markUnavailable(SignalEpicSiblings, siblingBranchReason)
		return
	}

	evidence := newSiblingEvidence(state.epicKey, sprintFallback)
	for _, siblingKey := range siblings {
		evidence.siblingKeys[siblingKey] = struct{}{}
		for _, match := range matches.Matches[siblingKey] {
			evidence.add(match.Repository, siblingKey)
		}
	}

	state.siblings = evidence
	state.rebuildSiblingReasons()
}

func (s *RepositorySuggestionService) siblingIssueKeys(ctx context.Context, userID string, issue *jiraDomain.JiraIssue) ([]string, bool, error) {
	if s.issues == nil || issue == nil {
		return nil, false, nil
	}

	backlog, err := s.issues.GetBacklog(ctx, userID, "")
	if err != nil {
		return nil, false, fmt.Errorf("read backlog: %w", err)
	}
	if backlog == nil {
		return nil, false, nil
	}

	epicKey := issueEpicKey(issue)
	sprintFallback := epicKey == ""
	issueKey := strings.ToUpper(strings.TrimSpace(issue.Key))
	issueSprint := strings.TrimSpace(issue.SprintName)

	siblings := make([]string, 0, 8)
	seen := make(map[string]struct{}, 8)
	for _, sprint := range backlog.Sprints {
		for _, candidate := range sprint.Issues {
			candidateKey := strings.ToUpper(strings.TrimSpace(candidate.Key))
			if candidateKey == "" || candidateKey == issueKey {
				continue
			}
			if epicKey != "" && !strings.EqualFold(issueParentKey(candidate), epicKey) {
				continue
			}
			if sprintFallback && issueSprint != "" && !strings.EqualFold(strings.TrimSpace(candidate.SprintName), issueSprint) {
				continue
			}
			if _, duplicate := seen[candidateKey]; duplicate {
				continue
			}

			seen[candidateKey] = struct{}{}
			siblings = append(siblings, candidateKey)
		}
	}

	sort.Strings(siblings)
	return siblings, sprintFallback, nil
}

func (s *suggestionState) rebuildSiblingReasons() {
	if s.siblings == nil {
		return
	}
	for repo := range s.reasons {
		delete(s.reasons[repo], SignalEpicSiblings)
	}

	totalWorked := len(s.siblings.workedKeys)
	if totalWorked == 0 {
		s.markUnavailable(SignalEpicSiblings, siblingNoBranchReason)
		return
	}

	repositories := make([]string, 0, len(s.siblings.keysByRepo))
	for repo := range s.siblings.keysByRepo {
		repositories = append(repositories, repo)
	}
	sort.Strings(repositories)

	for _, repo := range repositories {
		count := len(s.siblings.keysByRepo[repo])
		if count == 0 {
			continue
		}

		s.setReason(repo, SignalEpicSiblings, s.siblings.reasonText(repo, count), float64(count)/float64(totalWorked))
	}
}

func issueEpicKey(issue *jiraDomain.JiraIssue) string {
	if issue == nil {
		return ""
	}
	if key := issueParentKey(*issue); key != "" {
		return key
	}
	if isEpicIssue(issue) {
		return strings.ToUpper(strings.TrimSpace(issue.Key))
	}

	return ""
}

func issueParentKey(issue jiraDomain.JiraIssue) string {
	if issue.Fields.Parent == nil {
		return ""
	}

	return strings.ToUpper(strings.TrimSpace(issue.Fields.Parent.Key))
}

func isEpicIssue(issue *jiraDomain.JiraIssue) bool {
	if issue == nil || issue.Fields.IssueType == nil {
		return false
	}

	return strings.EqualFold(strings.TrimSpace(issue.Fields.IssueType.Name), "epic")
}
