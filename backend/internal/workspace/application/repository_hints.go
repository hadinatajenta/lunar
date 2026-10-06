package application

import (
	"context"
	"fmt"
	"strings"

	jiraDomain "lunar/backend/internal/jira/domain"
)

const (
	codeIndexTermLimit    = 5
	codeIndexResultLimit  = 50
	codeIndexMinimumTerms = 2
	fullHintStrength      = 1.0
	partialHintStrength   = 0.6

	hintNoRepositoryReason = "no repository is known, so the issue cannot name one"
	codeIndexMissingReason = "the code index is not available on this server"
	codeIndexNotCloned     = "the repository is not cloned, so its files cannot be searched"
	codeIndexTermReason    = "the issue carries fewer than two distinctive terms to search for"
	codeIndexFailedReason  = "the code index could not be searched"
	pageMissingReason      = "linked Confluence pages are not available on this server"
	pageNoneReason         = "the issue has no linked Confluence page"
	pageFailedReason       = "the linked Confluence pages could not be read"
)

func (s *RepositorySuggestionService) collectExplicitHints(issue *jiraDomain.JiraIssue, state *suggestionState) {
	if issue == nil {
		return
	}
	if len(state.catalogue.Repositories) == 0 {
		state.markUnavailable(SignalExplicitHint, hintNoRepositoryReason)
		return
	}

	textTokens := tokeniseText(issueText(issue))
	if len(textTokens) == 0 {
		return
	}

	for _, entry := range state.catalogue.Repositories {
		strength, matched := repositoryHintStrength(entry.Repo, textTokens)
		if strength <= 0 {
			continue
		}

		state.setReason(entry.Repo, SignalExplicitHint, hintReason(entry.Repo, matched), strength)
	}
}

func (s *RepositorySuggestionService) collectCodeIndexEvidence(ctx context.Context, userID string, issue *jiraDomain.JiraIssue, state *suggestionState) {
	if s.codeSearch == nil {
		state.markUnavailable(SignalCodeIndex, codeIndexMissingReason)
		return
	}

	clonedNames := state.clonedRepositoryNames()
	if len(clonedNames) == 0 {
		state.markUnavailable(SignalCodeIndex, codeIndexNotCloned)
		return
	}

	terms := distinctiveTerms(issueText(issue), codeIndexTermLimit)
	if len(terms) < codeIndexMinimumTerms {
		state.markUnavailable(SignalCodeIndex, codeIndexTermReason)
		return
	}

	repositoriesByTerm := make(map[string]map[string]struct{}, len(terms))
	filesByRepository := make(map[string]map[string]struct{})
	for _, term := range terms {
		matches, err := s.codeSearch.SearchCode(ctx, userID, term, codeIndexResultLimit)
		if err != nil {
			state.markUnavailable(SignalCodeIndex, codeIndexFailedReason)
			return
		}

		repositories := make(map[string]struct{})
		for _, match := range matches {
			name := state.cloneName(match.Repo)
			if name == "" {
				continue
			}

			repositories[name] = struct{}{}
			if filesByRepository[name] == nil {
				filesByRepository[name] = make(map[string]struct{})
			}
			filesByRepository[name][match.FilePath] = struct{}{}
		}
		repositoriesByTerm[term] = repositories
	}

	weights, totalWeight := codeIndexWeights(terms, repositoriesByTerm)
	if totalWeight == 0 {
		return
	}

	termsByRepository := make(map[string][]string)
	for _, term := range terms {
		for repository := range repositoriesByTerm[term] {
			termsByRepository[repository] = append(termsByRepository[repository], term)
		}
	}

	for repository, matchedTerms := range termsByRepository {
		if len(matchedTerms) < codeIndexMinimumTerms {
			continue
		}

		score := 0.0
		for _, term := range matchedTerms {
			score += weights[term]
		}

		state.setReason(repository, SignalCodeIndex, codeIndexReason(repository, len(matchedTerms), len(filesByRepository[repository])), score/totalWeight)
	}
}

func codeIndexWeights(terms []string, repositoriesByTerm map[string]map[string]struct{}) (map[string]float64, float64) {
	weights := make(map[string]float64, len(terms))
	totalWeight := 0.0
	for _, term := range terms {
		matchCount := len(repositoriesByTerm[term])
		if matchCount == 0 {
			continue
		}

		weights[term] = 1 / float64(matchCount)
		totalWeight += weights[term]
	}

	return weights, totalWeight
}

func (s *RepositorySuggestionService) collectConfluenceEvidence(ctx context.Context, userID string, issueKey string, state *suggestionState) {
	if s.linkedPages == nil {
		state.markUnavailable(SignalConfluence, pageMissingReason)
		return
	}

	pages, err := s.linkedPages.LinkedPages(ctx, userID, issueKey)
	if err != nil {
		state.markUnavailable(SignalConfluence, pageFailedReason)
		return
	}
	if len(pages) == 0 {
		state.markUnavailable(SignalConfluence, pageNoneReason)
		return
	}

	for _, entry := range state.catalogue.Repositories {
		strength, pageTitle := pageHintStrength(entry.Repo, pages)
		if strength <= 0 {
			continue
		}

		state.setReason(entry.Repo, SignalConfluence, fmt.Sprintf("the linked page %s names %s", pageTitle, entry.Repo), strength)
	}
}

func (s *RepositorySuggestionService) collectModelEvidence(ctx context.Context, userID string, issue *jiraDomain.JiraIssue, state *suggestionState) {
	if s.ranker == nil {
		state.markUnavailable(SignalModel, modelUnavailableReason)
		return
	}
	if issue == nil {
		return
	}

	ranked, err := s.ranker.RankRepositories(ctx, RepositoryRankRequest{
		UserID:      userID,
		IssueKey:    issue.Key,
		Summary:     issue.Fields.Summary,
		Description: issue.Fields.Description,
		Candidates:  state.repositoryNames(),
	})
	if err != nil {
		state.markUnavailable(SignalModel, modelFailedReason)
		return
	}

	for _, ranking := range ranked {
		if ranking.Rank <= 0 {
			continue
		}

		text := strings.TrimSpace(ranking.Reason)
		if text == "" {
			text = fmt.Sprintf("the model ranked %s at position %d", ranking.Repo, ranking.Rank)
		}

		state.setReason(ranking.Repo, SignalModel, text, 1/float64(ranking.Rank))
	}
}

func issueText(issue *jiraDomain.JiraIssue) string {
	if issue == nil {
		return ""
	}

	return strings.TrimSpace(issue.Fields.Summary + "\n" + issue.Fields.Description)
}

func (s *suggestionState) clonedRepositoryNames() map[string]string {
	cloned := make(map[string]string)
	for _, entry := range s.catalogue.Repositories {
		if entry.IsCloned {
			cloned[strings.ToLower(entry.Repo)] = entry.Repo
		}
	}

	return cloned
}

func (s *suggestionState) cloneName(repo string) string {
	name := strings.TrimSpace(repo)
	if name == "" {
		return ""
	}

	for _, entry := range s.catalogue.Repositories {
		if entry.IsCloned && strings.EqualFold(entry.Repo, name) {
			return entry.Repo
		}
	}

	return ""
}
