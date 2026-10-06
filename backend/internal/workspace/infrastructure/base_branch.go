package infrastructure

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"lunar/backend/internal/workspace/domain"
)

const remoteDefaultReference = "refs/remotes/origin/HEAD"

func (i *GitInspector) ReadBaseBranchCandidates(ctx context.Context, rootPath string, repoName string) (domain.BaseBranchCandidates, error) {
	resolvedRoot, err := domain.ResolveWorkspaceRoot(rootPath, i.allowedRoots)
	if err != nil {
		return domain.BaseBranchCandidates{}, err
	}

	repositoryPath, err := domain.ResolveRepoPath(resolvedRoot, repoName)
	if err != nil {
		if errors.Is(err, domain.ErrRootNotFound) {
			return domain.BaseBranchCandidates{IsCloned: false}, nil
		}
		return domain.BaseBranchCandidates{}, err
	}

	if !isGitRepository(ctx, repositoryPath) {
		return domain.BaseBranchCandidates{IsCloned: false}, nil
	}

	currentBranch, err := readBranch(ctx, repositoryPath)
	if err != nil {
		return domain.BaseBranchCandidates{}, fmt.Errorf("read current branch: %w", err)
	}

	output, err := runGit(ctx, repositoryPath, "for-each-ref", "--format="+branchRefFormat, "refs/heads")
	if err != nil {
		return domain.BaseBranchCandidates{}, fmt.Errorf("list base branch candidates: %w", err)
	}

	refs := parseBranchRefs(output, currentBranch)
	candidates := make([]domain.BaseBranchCandidate, 0, len(refs))
	for _, ref := range refs {
		candidates = append(candidates, domain.BaseBranchCandidate{
			Name:               ref.summary.Name,
			CommitAt:           ref.commitAt,
			LastCommitRelative: ref.summary.UpdatedRelative,
		})
	}

	remoteDefault, err := readBaseBranchRemoteDefault(ctx, repositoryPath)
	if err != nil {
		return domain.BaseBranchCandidates{}, fmt.Errorf("read the remote default branch: %w", err)
	}

	return domain.BaseBranchCandidates{
		IsCloned:      true,
		RemoteDefault: remoteDefault,
		Branches:      candidates,
	}, nil
}

func readBaseBranchRemoteDefault(ctx context.Context, repositoryPath string) (string, error) {
	matches, err := runGit(ctx, repositoryPath, "for-each-ref", "--format=%(refname)", remoteDefaultReference)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(matches) == "" {
		return "", nil
	}

	output, err := runGit(ctx, repositoryPath, "symbolic-ref", "--short", remoteDefaultReference)
	if err != nil {
		return "", fmt.Errorf("resolve the remote default branch: %w", err)
	}

	return strings.TrimPrefix(strings.TrimSpace(output), "origin/"), nil
}
