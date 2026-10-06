package main

import (
	"errors"
	"net/http"

	sharedErrors "lunar/backend/internal/shared/errors"
	sharedHttp "lunar/backend/internal/shared/http"
	"lunar/backend/internal/workspace/application"
	"lunar/backend/internal/workspace/domain"
)

const (
	repoPathParameter             = "repo"
	helperLocalUserID             = "local-helper"
	messageInvalidBranchCreation  = "request body must be a JSON object with a key, a branch, and targets"
	messageAllowedRootsOutside    = "the workspace root is outside LUNAR_WORKSPACE_ALLOWED_ROOTS"
	messageBranchOperationsAbsent = "git branch operations are not configured"
)

type baseBranchOptionResponse struct {
	Name               string  `json:"name"`
	LastCommitRelative *string `json:"last_commit_relative"`
	Source             string  `json:"source"`
}

type baseBranchOptionsResponse struct {
	Repo             string                     `json:"repo"`
	IsCloned         bool                       `json:"is_cloned"`
	Remembered       *string                    `json:"remembered"`
	BitbucketDefault *string                    `json:"bitbucket_default"`
	Options          []baseBranchOptionResponse `json:"options"`
	OtherBranchCount int                        `json:"other_branch_count"`
}

type createBranchTargetRequest struct {
	Repo         string `json:"repo"`
	BaseBranch   string `json:"base_branch"`
	RememberBase bool   `json:"remember_base"`
}

type createBranchesRequest struct {
	Key     string                      `json:"key"`
	Branch  string                      `json:"branch"`
	Targets []createBranchTargetRequest `json:"targets"`
}

type createdBranchResponse struct {
	Repo       string `json:"repo"`
	BaseBranch string `json:"base_branch"`
	HeadCommit string `json:"head_commit"`
}

type failedBranchResponse struct {
	Repo   string `json:"repo"`
	Reason string `json:"reason"`
}

type skippedBranchResponse struct {
	Repo   string `json:"repo"`
	Reason string `json:"reason"`
}

type createBranchesResponse struct {
	Key     string                  `json:"key"`
	Branch  string                  `json:"branch"`
	Created []createdBranchResponse `json:"created"`
	Failed  []failedBranchResponse  `json:"failed"`
	Skipped []skippedBranchResponse `json:"skipped"`
}

func (s *helperServer) handleCreateBranches(w http.ResponseWriter, r *http.Request) {
	if !canExtendGitDeadline(w) {
		return
	}

	var request createBranchesRequest
	if err := sharedHttp.ParseJSON(r, &request); err != nil {
		sharedHttp.WriteError(w, http.StatusBadRequest, messageInvalidBranchCreation)
		return
	}

	branchRequest := toBranchCreationRequest(request)
	branchName, err := application.ValidateBranchCreationRequest(branchRequest)
	if err != nil {
		writeBranchCreationValidationError(w, err)
		return
	}

	operations, ok := s.branchOperations()
	if !ok {
		sharedHttp.WriteError(w, http.StatusInternalServerError, messageBranchOperationsAbsent)
		return
	}

	rootPath := rootQueryValue(r)
	if err := application.ValidateWritableWorkspaceRoot(rootPath, s.allowedRoots); err != nil {
		writeBranchCreationRootError(w, err)
		return
	}

	outcome := application.RunBranchCreation(r.Context(), application.BranchCreationDependencies{
		Operations: operations,
		Memory:     s.baseBranchMemory,
		UserID:     helperLocalUserID,
		RootPath:   rootPath,
	}, branchRequest, branchName)

	sharedHttp.WriteJSON(w, http.StatusMultiStatus, toCreateBranchesResponse(branchRequest, branchName, outcome))
}

func (s *helperServer) handleBaseBranchOptions(w http.ResponseWriter, r *http.Request) {
	if !canExtendGitDeadline(w) {
		return
	}

	operations, ok := s.branchOperations()
	if !ok {
		sharedHttp.WriteError(w, http.StatusInternalServerError, messageBranchOperationsAbsent)
		return
	}

	repoName := r.PathValue(repoPathParameter)
	candidates, err := operations.ReadBaseBranchCandidates(r.Context(), rootQueryValue(r), repoName)
	if err != nil {
		writeBranchCreationRootError(w, err)
		return
	}

	options := application.BuildBaseBranchOptions(repoName, candidates, s.baseBranchMemory.Recall(helperLocalUserID, repoName))
	sharedHttp.WriteJSON(w, http.StatusOK, toBaseBranchOptionsResponse(options))
}

func (s *helperServer) branchOperations() (domain.BranchOperations, bool) {
	operations, ok := s.operator.(domain.BranchOperations)
	return operations, ok
}

func writeBranchCreationRootError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrWritesRequireAllowlist):
		sharedHttp.WriteError(w, http.StatusForbidden, messageWriteAllowlist)
	case errors.Is(err, domain.ErrRootOutsideScope):
		sharedHttp.WriteError(w, http.StatusForbidden, messageAllowedRootsOutside)
	default:
		writeGitOperationError(w, err)
	}
}

func writeBranchCreationValidationError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrInvalidBranchName):
		sharedHttp.WriteError(w, http.StatusBadRequest, messageBranchNameInvalid)
	case errors.Is(err, domain.ErrInvalidRepoName):
		sharedHttp.WriteError(w, http.StatusBadRequest, messageRepositoryNameInvalid)
	case errors.Is(err, sharedErrors.ErrBadRequest):
		sharedHttp.WriteError(w, http.StatusBadRequest, err.Error())
	default:
		writeGitOperationError(w, err)
	}
}

func toBranchCreationRequest(request createBranchesRequest) application.BranchCreationRequest {
	targets := make([]application.CreateBranchTarget, 0, len(request.Targets))
	for _, target := range request.Targets {
		targets = append(targets, application.CreateBranchTarget{
			Repo:         target.Repo,
			BaseBranch:   target.BaseBranch,
			RememberBase: target.RememberBase,
		})
	}

	return application.BranchCreationRequest{
		Key:     request.Key,
		Branch:  request.Branch,
		Targets: targets,
	}
}

func toBaseBranchOptionsResponse(options domain.BaseBranchOptions) baseBranchOptionsResponse {
	optionResponses := make([]baseBranchOptionResponse, 0, len(options.Options))
	for _, option := range options.Options {
		optionResponses = append(optionResponses, baseBranchOptionResponse{
			Name:               option.Name,
			LastCommitRelative: optionalString(option.LastCommitRelative),
			Source:             string(option.Source),
		})
	}

	return baseBranchOptionsResponse{
		Repo:             options.Repo,
		IsCloned:         options.IsCloned,
		Remembered:       optionalString(options.Remembered),
		BitbucketDefault: optionalString(options.BitbucketDefault),
		Options:          optionResponses,
		OtherBranchCount: options.OtherBranchCount,
	}
}

func toCreateBranchesResponse(request application.BranchCreationRequest, branchName string, outcome application.BranchCreationOutcome) createBranchesResponse {
	created := make([]createdBranchResponse, 0, len(outcome.Created))
	for _, branch := range outcome.Created {
		created = append(created, createdBranchResponse{
			Repo:       branch.Repo,
			BaseBranch: branch.BaseBranch,
			HeadCommit: branch.HeadCommit,
		})
	}

	failed := make([]failedBranchResponse, 0, len(outcome.Failed))
	for _, branch := range outcome.Failed {
		failed = append(failed, failedBranchResponse{Repo: branch.Repo, Reason: branch.Reason})
	}

	skipped := make([]skippedBranchResponse, 0, len(outcome.Skipped))
	for _, branch := range outcome.Skipped {
		skipped = append(skipped, skippedBranchResponse{Repo: branch.Repo, Reason: branch.Reason})
	}

	return createBranchesResponse{
		Key:     request.Key,
		Branch:  branchName,
		Created: created,
		Failed:  failed,
		Skipped: skipped,
	}
}

func optionalString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}
