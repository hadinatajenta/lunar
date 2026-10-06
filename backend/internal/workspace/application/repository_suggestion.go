package application

import (
	"context"
	"fmt"
	"math"

	sharedErrors "lunar/backend/internal/shared/errors"
	"lunar/backend/internal/workspace/domain"
)

const (
	SignalEpicSiblings = "epic_siblings"
	SignalExplicitHint = "explicit_hint"
	SignalCodeIndex    = "code_index"
	SignalConfluence   = "confluence"
	SignalModel        = "model"

	WeightEpicSiblings = 5.0
	WeightExplicitHint = 4.0
	WeightCodeIndex    = 2.0
	WeightConfluence   = 1.5
	WeightModel        = 1.0

	ConfidenceHigh   = "high"
	ConfidenceMedium = "medium"
	ConfidenceLow    = "low"

	maximumSuggestions       = 3
	lowConfidenceMarginRatio = 0.2
	highConfidenceLeadRatio  = 2.0
)

var signalOrder = []string{SignalEpicSiblings, SignalExplicitHint, SignalCodeIndex, SignalConfluence, SignalModel}

var signalWeights = map[string]float64{
	SignalEpicSiblings: WeightEpicSiblings,
	SignalExplicitHint: WeightExplicitHint,
	SignalCodeIndex:    WeightCodeIndex,
	SignalConfluence:   WeightConfluence,
	SignalModel:        WeightModel,
}

type SuggestionReason struct {
	Signal   string  `json:"signal"`
	Weight   float64 `json:"weight"`
	Strength float64 `json:"strength"`
	Text     string  `json:"text"`
}

type RepositorySuggestion struct {
	Repo             string             `json:"repo"`
	Score            float64            `json:"score"`
	IsCloned         bool               `json:"is_cloned"`
	BitbucketProject string             `json:"bitbucket_project"`
	Reasons          []SuggestionReason `json:"reasons"`
}

type UnavailableSignal struct {
	Signal string `json:"signal"`
	Reason string `json:"reason"`
}

type RepositorySuggestions struct {
	Key                string                 `json:"key"`
	Confidence         string                 `json:"confidence"`
	Suggestions        []RepositorySuggestion `json:"suggestions"`
	Catalogue          CatalogueCounts        `json:"catalogue"`
	SignalsUnavailable []UnavailableSignal    `json:"signals_unavailable"`
	Scope              WorkspaceScope         `json:"scope"`
}

type RepositorySuggestionDependencies struct {
	Catalogue      *RepositoryCatalogueService
	WorkspaceRoots WorkspaceRootSource
	Issues         IssueContextSource
	Branches       BranchMatchSource
}

type RepositorySuggestionService struct {
	catalogue      *RepositoryCatalogueService
	workspaceRoots WorkspaceRootSource
	issues         IssueContextSource
	branches       BranchMatchSource
	feedback       *RepositoryFeedbackStore
	codeSearch     CodeSearchSource
	linkedPages    LinkedPageSource
	ranker         RepositoryRanker
	remoteScanner  *RemoteCandidateScanner
}

type RepositorySuggestionOption func(*RepositorySuggestionService)

func WithCodeSearch(source CodeSearchSource) RepositorySuggestionOption {
	return func(service *RepositorySuggestionService) {
		service.codeSearch = source
	}
}

func WithLinkedPages(source LinkedPageSource) RepositorySuggestionOption {
	return func(service *RepositorySuggestionService) {
		service.linkedPages = source
	}
}

func WithRepositoryRanker(ranker RepositoryRanker) RepositorySuggestionOption {
	return func(service *RepositorySuggestionService) {
		service.ranker = ranker
	}
}

func WithRemoteCandidateScanner(scanner *RemoteCandidateScanner) RepositorySuggestionOption {
	return func(service *RepositorySuggestionService) {
		service.remoteScanner = scanner
	}
}

func NewRepositorySuggestionService(dependencies RepositorySuggestionDependencies, options ...RepositorySuggestionOption) *RepositorySuggestionService {
	service := &RepositorySuggestionService{
		catalogue:      dependencies.Catalogue,
		workspaceRoots: dependencies.WorkspaceRoots,
		issues:         dependencies.Issues,
		branches:       dependencies.Branches,
		feedback:       NewRepositoryFeedbackStore(),
	}

	for _, option := range options {
		option(service)
	}

	return service
}

func (s *RepositorySuggestionService) Suggest(ctx context.Context, userID string, issueKey string) (*RepositorySuggestions, error) {
	canonicalKey, err := canonicalIssueKey(issueKey)
	if err != nil {
		return nil, err
	}
	if s.issues == nil || s.catalogue == nil || s.workspaceRoots == nil {
		return nil, ErrServerReaderUnavailable
	}

	issue, err := s.issues.GetIssueDetail(ctx, userID, canonicalKey)
	if err != nil {
		return nil, fmt.Errorf("read issue %s: %w", canonicalKey, err)
	}

	rootPath, err := s.workspaceRoots.WorkspaceRootFor(ctx, userID)
	if err != nil {
		return nil, err
	}

	catalogue, err := s.catalogue.Catalogue(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("build repository catalogue: %w", err)
	}

	state := newSuggestionState(catalogue, canonicalKey)
	if catalogue.RemoteUnavailable {
		state.markUnavailable(SignalEpicSiblings, bitbucketUnavailableReason)
	}

	s.collectSiblingEvidence(ctx, userID, issue, rootPath, state)
	s.collectExplicitHints(issue, state)
	s.collectCodeIndexEvidence(ctx, userID, issue, state)
	s.collectConfluenceEvidence(ctx, userID, canonicalKey, state)

	scored := state.rank(s.feedback, userID)
	verification, err := s.enrichSiblingsRemotely(ctx, userID, state, scored)
	if err != nil {
		return nil, err
	}
	if verification.checkedRepositories > 0 {
		state.rebuildSiblingReasons()
	}
	if verification.degraded {
		state.markSignalDegraded(SignalEpicSiblings, bitbucketPartialReason)
	}

	scored = state.rank(s.feedback, userID)
	if len(scored) == 0 {
		s.collectModelEvidence(ctx, userID, issue, state)
		scored = state.rank(s.feedback, userID)
	}

	return &RepositorySuggestions{
		Key:                canonicalKey,
		Confidence:         confidenceFor(scored),
		Suggestions:        state.suggestions(scored),
		Catalogue:          state.counts(),
		SignalsUnavailable: state.unavailableSignals(),
		Scope:              scopeFor(catalogue, verification),
	}, nil
}

func (s *RepositorySuggestionService) SearchRepositories(ctx context.Context, userID string, query string) (*RepositoryCatalogue, error) {
	if s.catalogue == nil {
		return nil, ErrServerReaderUnavailable
	}

	catalogue, err := s.catalogue.Search(ctx, userID, query)
	if err != nil {
		return nil, fmt.Errorf("search repository catalogue: %w", err)
	}

	return catalogue, nil
}

func (s *RepositorySuggestionService) RecordFeedback(ctx context.Context, userID string, issueKey string, repo string, action FeedbackAction) error {
	canonicalKey, err := canonicalIssueKey(issueKey)
	if err != nil {
		return err
	}

	epicKey, err := s.epicKeyFor(ctx, userID, canonicalKey)
	if err != nil {
		return err
	}

	if err := s.validateCatalogueRepository(ctx, userID, repo); err != nil {
		return err
	}

	if err := s.feedback.Record(userID, epicKey, canonicalKey, repo, action); err != nil {
		return fmt.Errorf("record repository feedback: %w", err)
	}

	return nil
}

func (s *RepositorySuggestionService) epicKeyFor(ctx context.Context, userID string, canonicalKey string) (string, error) {
	if s.issues == nil {
		return "", nil
	}

	issue, err := s.issues.GetIssueDetail(ctx, userID, canonicalKey)
	if err != nil {
		return "", fmt.Errorf("read issue %s: %w", canonicalKey, err)
	}

	return issueEpicKey(issue), nil
}

func (s *RepositorySuggestionService) validateCatalogueRepository(ctx context.Context, userID string, repo string) error {
	if s.catalogue == nil {
		return ErrServerReaderUnavailable
	}

	validatedRepo, err := validateRepositoryName(repo)
	if err != nil {
		return err
	}

	catalogue, err := s.catalogue.Catalogue(ctx, userID)
	if err != nil {
		return fmt.Errorf("build repository catalogue: %w", err)
	}
	if !catalogue.contains(validatedRepo) {
		return fmt.Errorf("%w: repository %s is not in the catalogue", sharedErrors.ErrNotFound, validatedRepo)
	}

	return nil
}

func validateRepositoryName(repo string) (string, error) {
	validatedRepo, err := domain.ValidateRepoName(repo)
	if err != nil {
		return "", fmt.Errorf("%w: %s", sharedErrors.ErrBadRequest, err)
	}

	return validatedRepo, nil
}

func roundScore(score float64) float64 {
	return math.Round(score*100) / 100
}
