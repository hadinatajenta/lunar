package application

import (
	"context"
	"errors"
	"fmt"
	"strings"

	jiraDomain "lunar/backend/internal/jira/domain"
	sharedErrors "lunar/backend/internal/shared/errors"
	"lunar/backend/internal/workspace/domain"
)

const (
	branchReasonAlreadyExists         = "the branch already exists"
	branchReasonWorkingTreeDirty      = "the working tree has uncommitted changes; commit or stash them before switching branches"
	branchReasonNotCloned             = "the repository is not cloned"
	branchReasonBaseBranchMissing     = "the base branch was not found in this repository"
	branchReasonNotGitRepository      = "the repository is not a git repository"
	branchReasonRepositoryUnreadable  = "the repository could not be read"
	branchReasonRemoteCheckFailed     = "the existing branch could not be checked on Bitbucket, so no branch was created"
	branchReasonGitRejected           = "the repository rejected the requested branch creation"
	branchReasonGitTimeout            = "the git operation did not finish in time"
	branchReasonWritesDisabled        = "branch creation is disabled until LUNAR_WORKSPACE_ALLOWED_ROOTS names the workspace root"
	branchReasonInvalidBranchName     = "the branch name is invalid"
	branchReasonOperationsUnavailable = "server-side git operations are not configured"

	branchCreationKeyRequiredMessage     = "the issue key is required"
	branchCreationTargetsRequiredMessage = "at least one target repository is required"
	branchCreationBaseRequiredMessage    = "every target requires a base branch"
)

type RemoteBranchFinder interface {
	SearchBranches(ctx context.Context, userID string, repo string, filterText string) ([]string, error)
}

type CreateBranchTarget struct {
	Repo         string
	BaseBranch   string
	RememberBase bool
}

type BranchCreationRequest struct {
	Key     string
	Branch  string
	Targets []CreateBranchTarget
}

type CreatedBranch struct {
	Repo       string
	BaseBranch string
	HeadCommit string
}

type FailedBranch struct {
	Repo   string
	Reason string
}

type SkippedBranch struct {
	Repo   string
	Reason string
}

type BranchCreationOutcome struct {
	Created []CreatedBranch
	Failed  []FailedBranch
	Skipped []SkippedBranch
}

type RememberedBaseStore interface {
	Remember(userID string, repoName string, baseBranch string)
}

type BranchCreationDependencies struct {
	Operations domain.BranchOperations
	Finder     RemoteBranchFinder
	Memory     RememberedBaseStore
	UserID     string
	RootPath   string
}

type remoteBranchCheck struct {
	Skipped bool
	Failed  bool
	Reason  string
}

func ValidateBranchCreationRequest(request BranchCreationRequest) (string, error) {
	validBranchName, err := domain.ValidateBranchName(request.Branch)
	if err != nil {
		return "", err
	}

	if strings.TrimSpace(request.Key) == "" {
		return "", sharedErrors.BadRequest(branchCreationKeyRequiredMessage)
	}
	if len(request.Targets) == 0 {
		return "", sharedErrors.BadRequest(branchCreationTargetsRequiredMessage)
	}

	for _, target := range request.Targets {
		if _, err := domain.ValidateRepoName(target.Repo); err != nil {
			return "", err
		}
		if strings.TrimSpace(target.BaseBranch) == "" {
			return "", sharedErrors.BadRequest(branchCreationBaseRequiredMessage)
		}
		if _, err := domain.ValidateBranchName(target.BaseBranch); err != nil {
			return "", fmt.Errorf("base branch: %w", err)
		}
	}

	return validBranchName, nil
}

func RunBranchCreation(ctx context.Context, dependencies BranchCreationDependencies, request BranchCreationRequest, branchName string) BranchCreationOutcome {
	outcome := BranchCreationOutcome{
		Created: []CreatedBranch{},
		Failed:  []FailedBranch{},
		Skipped: []SkippedBranch{},
	}

	for _, target := range request.Targets {
		if dependencies.Operations == nil {
			outcome.Failed = append(outcome.Failed, FailedBranch{Repo: strings.TrimSpace(target.Repo), Reason: branchReasonOperationsUnavailable})
			continue
		}
		createTargetBranch(ctx, dependencies, request.Key, branchName, target, &outcome)
	}

	return outcome
}

func BuildBaseBranchOptions(repoName string, candidates domain.BaseBranchCandidates, remembered string) domain.BaseBranchOptions {
	options := domain.OrderBaseBranchOptions(candidates.Branches, remembered, candidates.RemoteDefault)

	return domain.BaseBranchOptions{
		Repo:             repoName,
		IsCloned:         candidates.IsCloned,
		Remembered:       presentBaseBranch(options, remembered),
		BitbucketDefault: candidates.RemoteDefault,
		Options:          options,
		OtherBranchCount: 0,
	}
}

func BranchCreationFailureReason(err error) string {
	switch {
	case errors.Is(err, domain.ErrWorkingTreeDirty):
		return branchReasonWorkingTreeDirty
	case errors.Is(err, domain.ErrInvalidBranchName):
		return branchReasonInvalidBranchName
	case errors.Is(err, domain.ErrBranchNotFound):
		return branchReasonBaseBranchMissing
	case errors.Is(err, domain.ErrRootNotFound):
		return branchReasonNotCloned
	case errors.Is(err, domain.ErrNotGitRepository):
		return branchReasonNotGitRepository
	case errors.Is(err, domain.ErrWritesRequireAllowlist), errors.Is(err, domain.ErrRootOutsideScope):
		return branchReasonWritesDisabled
	case errors.Is(err, domain.ErrGitTimeout), errors.Is(err, domain.ErrGitNetworkFailure):
		return branchReasonGitTimeout
	case errors.Is(err, domain.ErrGitOperationFailed):
		return branchReasonGitRejected
	default:
		return branchReasonRepositoryUnreadable
	}
}

func createTargetBranch(ctx context.Context, dependencies BranchCreationDependencies, key string, branchName string, target CreateBranchTarget, outcome *BranchCreationOutcome) {
	repoName := strings.TrimSpace(target.Repo)
	baseBranch := strings.TrimSpace(target.BaseBranch)

	check := checkRemoteBranch(ctx, dependencies, repoName, key)
	if check.Failed {
		outcome.Failed = append(outcome.Failed, FailedBranch{Repo: repoName, Reason: check.Reason})
		return
	}
	if check.Skipped {
		outcome.Skipped = append(outcome.Skipped, SkippedBranch{Repo: repoName, Reason: check.Reason})
		return
	}

	created, err := dependencies.Operations.CreateBranch(ctx, dependencies.RootPath, repoName, branchName, baseBranch)
	if err != nil {
		if errors.Is(err, domain.ErrBranchAlreadyExists) {
			outcome.Skipped = append(outcome.Skipped, SkippedBranch{Repo: repoName, Reason: branchReasonAlreadyExists})
			return
		}
		outcome.Failed = append(outcome.Failed, FailedBranch{Repo: repoName, Reason: BranchCreationFailureReason(err)})
		return
	}

	rememberBaseBranch(dependencies, repoName, baseBranch, target.RememberBase)
	outcome.Created = append(outcome.Created, CreatedBranch{
		Repo:       repoName,
		BaseBranch: baseBranch,
		HeadCommit: created.HeadCommit,
	})
}

func checkRemoteBranch(ctx context.Context, dependencies BranchCreationDependencies, repoName string, key string) remoteBranchCheck {
	if dependencies.Finder == nil {
		return remoteBranchCheck{}
	}

	coordinates, err := dependencies.Operations.ReadRepositoryCoordinates(ctx, dependencies.RootPath, repoName)
	if err != nil {
		return remoteBranchCheck{Failed: true, Reason: BranchCreationFailureReason(err)}
	}
	if !coordinates.IsKnown() {
		return remoteBranchCheck{}
	}

	existingBranches, err := dependencies.Finder.SearchBranches(ctx, dependencies.UserID, coordinates.String(), strings.TrimSpace(key))
	if err != nil {
		return remoteBranchCheck{Failed: true, Reason: branchReasonRemoteCheckFailed}
	}
	carryingKey := make([]string, 0, len(existingBranches))
	for _, branchName := range existingBranches {
		if jiraDomain.ContainsIssueKey(branchName, key) {
			carryingKey = append(carryingKey, branchName)
		}
	}
	if len(carryingKey) == 0 {
		return remoteBranchCheck{}
	}

	return remoteBranchCheck{Skipped: true, Reason: branchReasonAlreadyExists + ": " + carryingKey[0]}
}

func rememberBaseBranch(dependencies BranchCreationDependencies, repoName string, baseBranch string, shouldRemember bool) {
	if !shouldRemember || dependencies.Memory == nil {
		return
	}
	dependencies.Memory.Remember(dependencies.UserID, repoName, baseBranch)
}

func presentBaseBranch(options []domain.BaseBranchOption, remembered string) string {
	if remembered == "" {
		return ""
	}
	for _, option := range options {
		if option.Name == remembered {
			return remembered
		}
	}
	return ""
}
