package main

import (
	"errors"
	"net/http"

	sharedHttp "lunar/backend/internal/shared/http"
	"lunar/backend/internal/workspace/domain"
)

const (
	nameQueryParameter = "name"

	messageInvalidCheckoutBody = "request body must be a JSON object with a branch field"
	messageBranchNameInvalid   = "the branch name is invalid"
	messageWorkingTreeDirty    = "the working tree has uncommitted changes; commit or stash them before switching branches"
	messageNotGitRepository    = "the path is not a git repository"
	messageRemoteGitFailure    = "the remote git operation failed; check network connectivity and credentials"
	messageGitOperationFailed  = "the repository could not complete the git operation"
	messageDeadlineFailed      = "the request deadline could not be extended for the git operation"
	messageBranchNotFound      = "the branch does not exist in this repository"
)

type branchResponse struct {
	Name            string `json:"name"`
	IsCurrent       bool   `json:"is_current"`
	UpdatedRelative string `json:"updated_relative"`
}

type branchesResponse struct {
	Branches []branchResponse `json:"branches"`
}

type checkoutRequest struct {
	Branch string `json:"branch"`
}

type checkoutResponse struct {
	Branch string `json:"branch"`
}

type fetchResponse struct {
	Ahead  int `json:"ahead"`
	Behind int `json:"behind"`
}

type pullResponse struct {
	Branch          string `json:"branch"`
	UpdatedRelative string `json:"updated_relative"`
}

type stashResponse struct {
	StashCount int `json:"stash_count"`
}

type discardResponse struct {
	RestoredCount int `json:"restored_count"`
}

func (s *helperServer) handleBranches(w http.ResponseWriter, r *http.Request) {
	if !canExtendGitDeadline(w) {
		return
	}

	branches, err := s.operator.ListBranches(r.Context(), rootQueryValue(r), nameQueryValue(r))
	if err != nil {
		writeGitOperationError(w, err)
		return
	}

	branchResponses := make([]branchResponse, 0, len(branches))
	for _, branch := range branches {
		branchResponses = append(branchResponses, branchResponse{
			Name:            branch.Name,
			IsCurrent:       branch.IsCurrent,
			UpdatedRelative: branch.UpdatedRelative,
		})
	}

	sharedHttp.WriteJSON(w, http.StatusOK, branchesResponse{Branches: branchResponses})
}

func (s *helperServer) handleCheckout(w http.ResponseWriter, r *http.Request) {
	if !canExtendGitDeadline(w) {
		return
	}

	var request checkoutRequest
	if err := sharedHttp.ParseJSON(r, &request); err != nil {
		sharedHttp.WriteError(w, http.StatusBadRequest, messageInvalidCheckoutBody)
		return
	}

	branch, err := s.operator.CheckoutBranch(r.Context(), rootQueryValue(r), nameQueryValue(r), request.Branch)
	if err != nil {
		writeGitOperationError(w, err)
		return
	}

	sharedHttp.WriteJSON(w, http.StatusOK, checkoutResponse{Branch: branch})
}

func (s *helperServer) handleFetch(w http.ResponseWriter, r *http.Request) {
	if !canExtendGitDeadline(w) {
		return
	}

	status, err := s.operator.FetchRepository(r.Context(), rootQueryValue(r), nameQueryValue(r))
	if err != nil {
		writeGitOperationError(w, err)
		return
	}

	sharedHttp.WriteJSON(w, http.StatusOK, fetchResponse{Ahead: status.Ahead, Behind: status.Behind})
}

func (s *helperServer) handlePull(w http.ResponseWriter, r *http.Request) {
	if !canExtendGitDeadline(w) {
		return
	}

	pullResult, err := s.operator.PullRepository(r.Context(), rootQueryValue(r), nameQueryValue(r))
	if err != nil {
		writeGitOperationError(w, err)
		return
	}

	sharedHttp.WriteJSON(w, http.StatusOK, pullResponse{Branch: pullResult.Branch, UpdatedRelative: pullResult.UpdatedRelative})
}

func (s *helperServer) handleStash(w http.ResponseWriter, r *http.Request) {
	if !canExtendGitDeadline(w) {
		return
	}

	stashCount, err := s.operator.StashRepository(r.Context(), rootQueryValue(r), nameQueryValue(r))
	if err != nil {
		writeGitOperationError(w, err)
		return
	}

	sharedHttp.WriteJSON(w, http.StatusOK, stashResponse{StashCount: stashCount})
}

func (s *helperServer) handleDiscard(w http.ResponseWriter, r *http.Request) {
	if !canExtendGitDeadline(w) {
		return
	}

	restoredCount, err := s.operator.DiscardRepository(r.Context(), rootQueryValue(r), nameQueryValue(r))
	if err != nil {
		writeGitOperationError(w, err)
		return
	}

	sharedHttp.WriteJSON(w, http.StatusOK, discardResponse{RestoredCount: restoredCount})
}

func canExtendGitDeadline(w http.ResponseWriter) bool {
	if err := sharedHttp.ExtendWriteDeadline(w, domain.GitWriteDeadline); err != nil {
		sharedHttp.WriteError(w, http.StatusInternalServerError, messageDeadlineFailed)
		return false
	}
	return true
}

func rootQueryValue(r *http.Request) string {
	return r.URL.Query().Get(rootQueryParameter)
}

func nameQueryValue(r *http.Request) string {
	return r.URL.Query().Get(nameQueryParameter)
}

func writeGitOperationError(w http.ResponseWriter, err error) {
	sharedHttp.WriteError(w, statusForGitOperationError(err), gitOperationErrorMessage(err))
}

func statusForGitOperationError(err error) int {
	switch {
	case errors.Is(err, domain.ErrWorkingTreeDirty),
		errors.Is(err, domain.ErrGitOperationFailed):
		return http.StatusConflict
	case errors.Is(err, domain.ErrGitNetworkFailure),
		errors.Is(err, domain.ErrGitTimeout):
		return http.StatusGatewayTimeout
	case errors.Is(err, domain.ErrBranchNotFound):
		return http.StatusNotFound
	case errors.Is(err, domain.ErrInvalidBranchName),
		errors.Is(err, domain.ErrInvalidRepoName),
		errors.Is(err, domain.ErrNotGitRepository),
		isWorkspaceRootError(err):
		return http.StatusBadRequest
	default:
		return http.StatusConflict
	}
}

func gitOperationErrorMessage(err error) string {
	switch {
	case errors.Is(err, domain.ErrInvalidBranchName):
		return messageBranchNameInvalid
	case errors.Is(err, domain.ErrBranchNotFound):
		return messageBranchNotFound
	case errors.Is(err, domain.ErrInvalidRepoName):
		return messageRepositoryNameInvalid
	case errors.Is(err, domain.ErrRepoOutsideRoot):
		return messageRepositoryOutsideRoot
	case errors.Is(err, domain.ErrNotGitRepository):
		return messageNotGitRepository
	case isWorkspaceRootError(err):
		return rootErrorMessage(err)
	case errors.Is(err, domain.ErrWorkingTreeDirty):
		return messageWorkingTreeDirty
	case errors.Is(err, domain.ErrGitNetworkFailure), errors.Is(err, domain.ErrGitTimeout):
		return messageRemoteGitFailure
	default:
		return messageGitOperationFailed
	}
}

func isWorkspaceRootError(err error) bool {
	return errors.Is(err, domain.ErrInvalidRoot) ||
		errors.Is(err, domain.ErrRootNotFound) ||
		errors.Is(err, domain.ErrRootNotDirectory) ||
		errors.Is(err, domain.ErrRootNotAllowed) ||
		errors.Is(err, domain.ErrRootOutsideScope)
}
