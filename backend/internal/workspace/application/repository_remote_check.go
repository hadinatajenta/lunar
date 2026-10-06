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

func (s *RepositorySuggestionService) enrichSiblingsRemotely(ctx context.Context, userID string, state *suggestionState, scored []scoredRepository) (remoteVerification, error) {
	if s.remoteScanner == nil || state.siblings == nil || len(state.siblings.siblingKeys) == 0 {
		return remoteVerification{}, nil
	}

	siblingKeys := state.siblings.remoteLookupKeys()
	if len(siblingKeys) == 0 {
		return remoteVerification{}, nil
	}

	repositories := s.remoteCandidateRepositories(state, scored)
	if len(repositories) == 0 {
		return remoteVerification{}, nil
	}

	result, err := s.remoteScanner.ScanIssueKeys(ctx, userID, siblingKeys, repositories)
	if err != nil {
		return remoteVerification{}, fmt.Errorf("scan sibling branches on Bitbucket: %w", err)
	}

	for repository, branchesByKey := range result.MatchesByRepositoryAndKey {
		repositoryName := state.canonicalRepoName(remoteRepositorySlug(repository))
		for siblingKey, branches := range branchesByKey {
			if len(branches) == 0 {
				continue
			}

			state.siblings.addRemote(repositoryName, siblingKey)
		}
	}

	return remoteVerification{
		checkedRepositories: result.RepositoriesChecked,
		verified:            result.RepositoriesChecked > 0 && !result.TimedOut && len(result.Failures) == 0,
		degraded:            result.TimedOut || len(result.Failures) > 0,
	}, nil
}

func (s *RepositorySuggestionService) remoteCandidateRepositories(state *suggestionState, scored []scoredRepository) []string {
	repositories := make([]string, 0, RemoteCandidateLimit)
	seen := make(map[string]struct{})

	for _, entry := range scored {
		if entry.isCloned {
			continue
		}
		appendRemoteCandidate(&repositories, seen, entry.project, entry.repo)
	}
	for _, entry := range state.catalogue.Repositories {
		if entry.IsCloned {
			continue
		}
		appendRemoteCandidate(&repositories, seen, entry.Project, entry.Repo)
	}

	return repositories
}

func appendRemoteCandidate(repositories *[]string, seen map[string]struct{}, project string, repo string) {
	trimmedProject := strings.TrimSpace(project)
	name := strings.TrimSpace(repo)
	if trimmedProject == "" || name == "" {
		return
	}

	identity := strings.ToLower(name)
	if _, duplicate := seen[identity]; duplicate {
		return
	}
	seen[identity] = struct{}{}

	*repositories = append(*repositories, trimmedProject+"/"+name)
}

func remoteRepositorySlug(repository string) string {
	trimmed := strings.TrimSpace(repository)
	if separator := strings.LastIndex(trimmed, "/"); separator >= 0 {
		return strings.TrimSpace(trimmed[separator+1:])
	}

	return trimmed
}
