package application

import (
	"fmt"
	"strings"
	"sync"

	sharedErrors "lunar/backend/internal/shared/errors"
)

const (
	feedbackAdditionLevel   = 2.0
	feedbackRemovalLevel    = -1.0
	feedbackUntickLevel     = -0.5
	feedbackKeepLevel       = 0.25
	feedbackAdjustmentLimit = 2.0

	epicFeedbackScope  = "epic"
	issueFeedbackScope = "issue"
)

type FeedbackAction string

const (
	FeedbackAdded    FeedbackAction = "added"
	FeedbackRemoved  FeedbackAction = "removed"
	FeedbackUnticked FeedbackAction = "unticked"
	FeedbackKept     FeedbackAction = "kept"
)

func ParseFeedbackAction(rawAction string) (FeedbackAction, error) {
	action := FeedbackAction(strings.ToLower(strings.TrimSpace(rawAction)))
	switch action {
	case FeedbackAdded, FeedbackRemoved, FeedbackUnticked, FeedbackKept:
		return action, nil
	default:
		return "", fmt.Errorf("%w: feedback action %q is not one of added, removed, unticked, kept", sharedErrors.ErrBadRequest, rawAction)
	}
}

func (a FeedbackAction) IsIssueScoped() bool {
	return a == FeedbackUnticked
}

func feedbackLevel(action FeedbackAction) float64 {
	switch action {
	case FeedbackAdded:
		return feedbackAdditionLevel
	case FeedbackRemoved:
		return feedbackRemovalLevel
	case FeedbackUnticked:
		return feedbackUntickLevel
	case FeedbackKept:
		return feedbackKeepLevel
	default:
		return 0
	}
}

type RepositoryFeedbackStore struct {
	mutex  sync.Mutex
	counts map[string]map[FeedbackAction]int
}

func NewRepositoryFeedbackStore() *RepositoryFeedbackStore {
	return &RepositoryFeedbackStore{counts: make(map[string]map[FeedbackAction]int)}
}

func (s *RepositoryFeedbackStore) Record(userID string, epicKey string, issueKey string, repo string, action FeedbackAction) error {
	user := strings.TrimSpace(userID)
	name := strings.TrimSpace(repo)
	if user == "" {
		return fmt.Errorf("%w: the user is required", sharedErrors.ErrBadRequest)
	}
	if name == "" {
		return fmt.Errorf("%w: the repository is required", sharedErrors.ErrBadRequest)
	}
	if feedbackLevel(action) == 0 {
		return fmt.Errorf("%w: feedback action %q is not one of added, removed, unticked, kept", sharedErrors.ErrBadRequest, action)
	}
	if s == nil {
		return fmt.Errorf("%w: repository feedback is not configured", sharedErrors.ErrInternal)
	}

	key := feedbackRecordKey(epicKey, issueKey, user, name, action)

	s.mutex.Lock()
	defer s.mutex.Unlock()

	actionCounts := s.counts[key]
	if actionCounts == nil {
		actionCounts = make(map[FeedbackAction]int)
		s.counts[key] = actionCounts
	}
	actionCounts[action]++

	return nil
}

func (s *RepositoryFeedbackStore) Adjustment(userID string, epicKey string, issueKey string, repo string) float64 {
	if s == nil {
		return 0
	}

	user := strings.TrimSpace(userID)
	name := strings.TrimSpace(repo)
	if user == "" || name == "" {
		return 0
	}

	total := s.scopedAdjustment(issueScopeKey(user, issueKey, name))

	epic := strings.TrimSpace(epicKey)
	issue := strings.TrimSpace(issueKey)
	if epic != "" && !strings.EqualFold(epic, issue) {
		total += s.scopedAdjustment(epicScopeKey(user, epic, name))
	}

	return clampFeedbackAdjustment(total)
}

func (s *RepositoryFeedbackStore) scopedAdjustment(key string) float64 {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	total := 0.0
	for action, count := range s.counts[key] {
		total += feedbackLevel(action) * float64(count)
	}

	return total
}

func feedbackRecordKey(epicKey string, issueKey string, userID string, repo string, action FeedbackAction) string {
	epic := strings.TrimSpace(epicKey)
	issue := strings.TrimSpace(issueKey)

	if action.IsIssueScoped() || epic == "" || strings.EqualFold(epic, issue) {
		return issueScopeKey(userID, issue, repo)
	}

	return epicScopeKey(userID, epic, repo)
}

func issueScopeKey(userID string, issueKey string, repo string) string {
	return feedbackKey(issueFeedbackScope, userID, issueKey, repo)
}

func epicScopeKey(userID string, epicKey string, repo string) string {
	return feedbackKey(epicFeedbackScope, userID, epicKey, repo)
}

func feedbackKey(scope string, userID string, identifier string, repo string) string {
	return scope + "\x00" + userID + "\x00" + strings.TrimSpace(identifier) + "\x00" + repo
}

func clampFeedbackAdjustment(adjustment float64) float64 {
	if adjustment > feedbackAdjustmentLimit {
		return feedbackAdjustmentLimit
	}
	if adjustment < -feedbackAdjustmentLimit {
		return -feedbackAdjustmentLimit
	}

	return adjustment
}
