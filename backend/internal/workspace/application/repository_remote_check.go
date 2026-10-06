package application

import (
	"context"
	"fmt"
	"strings"
)

type remoteVerification struct {
	checkedRepositories int
	verified            bool
	degraded            bool
}

func (s *RepositorySuggestionService) verifyCandidatesRemotely(ctx context.Context, userID string, issueKey string, state *suggestionState, scored []scoredRepository) (remoteVerification, error) {
	if s.remoteScanner == nil || len(scored) == 0 {
		return remoteVerification{}, nil
	}

	repositories := state.remoteCandidateRepositories(scored)
	if len(repositories) == 0 {
		return remoteVerification{}, nil
	}

	result, err := s.remoteScanner.ScanIssueKey(ctx, userID, issueKey, repositories)
	if err != nil {
		return remoteVerification{}, fmt.Errorf("scan remote candidate repositories: %w", err)
	}

	return remoteVerification{
		checkedRepositories: result.RepositoriesChecked,
		verified:            result.RepositoriesChecked > 0 && !result.TimedOut && len(result.Failures) == 0,
		degraded:            result.TimedOut || len(result.Failures) > 0,
	}, nil
}

func (s *suggestionState) remoteCandidateRepositories(scored []scoredRepository) []string {
	repositories := make([]string, 0, len(scored))
	for _, entry := range scored {
		project := strings.TrimSpace(entry.project)
		if project == "" {
			continue
		}

		repositories = append(repositories, project+"/"+entry.repo)
	}

	return repositories
}
