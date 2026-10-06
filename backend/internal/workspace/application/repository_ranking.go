package application

import (
	"math"
	"sort"
	"strings"
)

const (
	bitbucketUnavailableReason = "Bitbucket could not be reached, so repositories outside the workspace were not searched"
	bitbucketPartialReason     = "the Bitbucket check did not complete, so not every candidate repository was verified"
	modelUnavailableReason     = "no model ranking is configured"
	modelFailedReason          = "the model ranking could not be produced"
)

type scoredRepository struct {
	repo     string
	score    float64
	isCloned bool
	project  string
	reasons  []SuggestionReason
}

type suggestionState struct {
	issueKey    string
	epicKey     string
	catalogue   *RepositoryCatalogue
	reasons     map[string]map[string]SuggestionReason
	unavailable map[string]string
	canonical   map[string]string
	siblings    *siblingEvidence
}

func newSuggestionState(catalogue *RepositoryCatalogue, issueKey string) *suggestionState {
	canonical := make(map[string]string, len(catalogue.Repositories))
	for _, entry := range catalogue.Repositories {
		canonical[strings.ToLower(entry.Repo)] = entry.Repo
	}

	return &suggestionState{
		issueKey:    issueKey,
		catalogue:   catalogue,
		reasons:     make(map[string]map[string]SuggestionReason),
		unavailable: make(map[string]string),
		canonical:   canonical,
	}
}

func (s *suggestionState) markUnavailable(signal string, reason string) {
	if s == nil {
		return
	}
	if _, known := signalWeights[signal]; !known {
		return
	}
	if _, reported := s.unavailable[signal]; reported {
		return
	}

	s.unavailable[signal] = reason
}

func (s *suggestionState) setReason(repo string, signal string, text string, strength float64) {
	name := s.canonicalRepoName(repo)
	weight, known := signalWeights[signal]
	if name == "" || !known || strength <= 0 {
		return
	}

	repositoryReasons := s.reasons[name]
	if repositoryReasons == nil {
		repositoryReasons = make(map[string]SuggestionReason)
		s.reasons[name] = repositoryReasons
	}

	repositoryReasons[signal] = SuggestionReason{
		Signal:   signal,
		Weight:   weight,
		Strength: roundScore(math.Min(strength, 1.0)),
		Text:     truncateReason(text),
	}
	delete(s.unavailable, signal)
}

func (s *suggestionState) canonicalRepoName(repo string) string {
	name := strings.TrimSpace(repo)
	if name == "" {
		return ""
	}
	if canonical, found := s.canonical[strings.ToLower(name)]; found {
		return canonical
	}

	return name
}

func (s *suggestionState) catalogueEntry(repo string) CataloguedRepository {
	for _, entry := range s.catalogue.Repositories {
		if strings.EqualFold(entry.Repo, repo) {
			return entry
		}
	}

	return CataloguedRepository{Repo: repo, IsCloned: false}
}

func (s *suggestionState) repositoryNames() []string {
	names := make([]string, 0, len(s.catalogue.Repositories))
	for _, entry := range s.catalogue.Repositories {
		names = append(names, entry.Repo)
	}

	return names
}

func (s *suggestionState) counts() CatalogueCounts {
	return CatalogueCounts{
		LocalRepositoryCount:  s.catalogue.LocalRepositoryCount,
		RemoteRepositoryCount: s.catalogue.RemoteRepositoryCount,
	}
}

func (s *suggestionState) rank(feedback *RepositoryFeedbackStore, userID string) []scoredRepository {
	scored := make([]scoredRepository, 0, len(s.reasons))
	for repo, repositoryReasons := range s.reasons {
		total := 0.0
		reasons := make([]SuggestionReason, 0, len(repositoryReasons))
		for _, signal := range signalOrder {
			reason, found := repositoryReasons[signal]
			if !found {
				continue
			}

			total += reason.Weight * reason.Strength
			reasons = append(reasons, reason)
		}
		if len(reasons) == 0 {
			continue
		}

		total += feedback.Adjustment(userID, s.epicKey, s.issueKey, repo)
		if total <= 0 {
			continue
		}

		entry := s.catalogueEntry(repo)
		scored = append(scored, scoredRepository{
			repo:     repo,
			score:    total,
			isCloned: entry.IsCloned,
			project:  entry.Project,
			reasons:  reasons,
		})
	}

	sortScoredRepositories(scored)
	return scored
}

func (s *suggestionState) suggestions(scored []scoredRepository) []RepositorySuggestion {
	limit := len(scored)
	if limit > maximumSuggestions {
		limit = maximumSuggestions
	}

	suggestions := make([]RepositorySuggestion, 0, limit)
	for _, entry := range scored[:limit] {
		suggestions = append(suggestions, RepositorySuggestion{
			Repo:             entry.repo,
			Score:            roundScore(entry.score),
			IsCloned:         entry.isCloned,
			BitbucketProject: entry.project,
			Reasons:          entry.reasons,
		})
	}

	return suggestions
}

func (s *suggestionState) unavailableSignals() []UnavailableSignal {
	signals := make([]UnavailableSignal, 0, len(s.unavailable))
	for _, signal := range signalOrder {
		if reason, found := s.unavailable[signal]; found {
			signals = append(signals, UnavailableSignal{Signal: signal, Reason: reason})
		}
	}

	return signals
}

func sortScoredRepositories(scored []scoredRepository) {
	sort.SliceStable(scored, func(first int, second int) bool {
		if scored[first].score != scored[second].score {
			return scored[first].score > scored[second].score
		}
		if !strings.EqualFold(scored[first].repo, scored[second].repo) {
			return strings.ToLower(scored[first].repo) < strings.ToLower(scored[second].repo)
		}
		return scored[first].project < scored[second].project
	})
}

func confidenceFor(scored []scoredRepository) string {
	if len(scored) == 0 {
		return ConfidenceLow
	}
	if len(scored) > 1 && withinLowMargin(scored[0].score, scored[1].score) {
		return ConfidenceLow
	}
	if isModelOnly(scored[0].reasons) {
		return ConfidenceLow
	}
	if !hasStrongSignal(scored[0].reasons) {
		return ConfidenceMedium
	}
	if len(scored) == 1 || scored[0].score >= highConfidenceLeadRatio*scored[1].score {
		return ConfidenceHigh
	}

	return ConfidenceMedium
}

func withinLowMargin(leaderScore float64, runnerUpScore float64) bool {
	if leaderScore <= 0 {
		return true
	}

	return (leaderScore-runnerUpScore)/leaderScore < lowConfidenceMarginRatio
}

func isModelOnly(reasons []SuggestionReason) bool {
	return len(reasons) == 1 && reasons[0].Signal == SignalModel
}

func hasStrongSignal(reasons []SuggestionReason) bool {
	for _, reason := range reasons {
		if reason.Signal == SignalEpicSiblings || reason.Signal == SignalExplicitHint {
			return true
		}
	}

	return false
}

func scopeFor(catalogue *RepositoryCatalogue, verification remoteVerification) WorkspaceScope {
	localCount := catalogue.LocalRepositoryCount
	remoteCount := catalogue.RemoteRepositoryCount

	coveragePercent := 0
	switch {
	case remoteCount > 0 && localCount*100/remoteCount > 100:
		coveragePercent = 100
	case remoteCount > 0:
		coveragePercent = localCount * 100 / remoteCount
	case localCount > 0:
		coveragePercent = 100
	}

	return WorkspaceScope{
		LocalRepositoryCount:      localCount,
		RemoteRepositoryCount:     remoteCount,
		CoveragePercent:           coveragePercent,
		VerifiedRemotely:          verification.verified,
		RemoteRepositoriesChecked: verification.checkedRepositories,
	}
}
