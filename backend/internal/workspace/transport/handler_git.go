package transport

import (
	"errors"
	"net/http"

	sharedHttp "lunar/backend/internal/shared/http"
	"lunar/backend/internal/workspace/domain"
)

const (
	nameQueryParameter       = "name"
	branchNameMessage        = "the branch name is invalid"
	repositoryNameMessage    = "the repository name is invalid"
	workspacePathMessage     = "the workspace or repository path is not available"
	repositoryOutsideMessage = "the repository is outside the workspace root"
	notGitRepositoryMessage  = "the path is not a git repository"
	workingTreeDirtyMessage  = "the working tree has uncommitted changes; commit or stash them before switching branches"
	gitRejectedMessage       = "the repository rejected the requested git operation"
	gitNetworkMessage        = "the remote git operation failed; check network connectivity and credentials"
	gitDeadlineMessage       = "the request deadline could not be extended for the git operation"
	branchNotFoundMessage    = "the branch does not exist in this repository"
)

func (h *WorkspaceHandler) ListBranches(w http.ResponseWriter, r *http.Request) {
	if !h.extendGitWriteDeadline(w) {
		return
	}

	userID := h.getUserID(r)
	if userID == "" {
		sharedHttp.WriteError(w, http.StatusUnauthorized, unauthenticatedMessage)
		return
	}

	branches, err := h.service.ListBranches(r.Context(), userID, r.URL.Query().Get(nameQueryParameter))
	if err != nil {
		h.writeError(w, err)
		return
	}

	sharedHttp.WriteJSON(w, http.StatusOK, toBranchesResponse(branches))
}

func (h *WorkspaceHandler) CheckoutBranch(w http.ResponseWriter, r *http.Request) {
	if !h.extendGitWriteDeadline(w) {
		return
	}

	userID := h.getUserID(r)
	if userID == "" {
		sharedHttp.WriteError(w, http.StatusUnauthorized, unauthenticatedMessage)
		return
	}

	var request checkoutBranchRequest
	if err := sharedHttp.ParseJSON(r, &request); err != nil {
		sharedHttp.WriteError(w, http.StatusBadRequest, invalidBodyMessage)
		return
	}

	branch, err := h.service.CheckoutBranch(r.Context(), userID, request.Name, request.Branch)
	if err != nil {
		h.writeError(w, err)
		return
	}

	sharedHttp.WriteJSON(w, http.StatusOK, checkoutBranchResponse{Branch: branch})
}

func (h *WorkspaceHandler) FetchRepository(w http.ResponseWriter, r *http.Request) {
	userID, repoName, ok := h.readRepositoryOperation(w, r)
	if !ok {
		return
	}

	status, err := h.service.FetchRepository(r.Context(), userID, repoName)
	if err != nil {
		h.writeError(w, err)
		return
	}

	sharedHttp.WriteJSON(w, http.StatusOK, fetchResponse{Ahead: status.Ahead, Behind: status.Behind})
}

func (h *WorkspaceHandler) PullRepository(w http.ResponseWriter, r *http.Request) {
	userID, repoName, ok := h.readRepositoryOperation(w, r)
	if !ok {
		return
	}

	pullResult, err := h.service.PullRepository(r.Context(), userID, repoName)
	if err != nil {
		h.writeError(w, err)
		return
	}

	sharedHttp.WriteJSON(w, http.StatusOK, pullResponse{Branch: pullResult.Branch, UpdatedRelative: pullResult.UpdatedRelative})
}

func (h *WorkspaceHandler) StashRepository(w http.ResponseWriter, r *http.Request) {
	userID, repoName, ok := h.readRepositoryOperation(w, r)
	if !ok {
		return
	}

	stashCount, err := h.service.StashRepository(r.Context(), userID, repoName)
	if err != nil {
		h.writeError(w, err)
		return
	}

	sharedHttp.WriteJSON(w, http.StatusOK, stashResponse{StashCount: stashCount})
}

func (h *WorkspaceHandler) DiscardRepository(w http.ResponseWriter, r *http.Request) {
	userID, repoName, ok := h.readRepositoryOperation(w, r)
	if !ok {
		return
	}

	restoredCount, err := h.service.DiscardRepository(r.Context(), userID, repoName)
	if err != nil {
		h.writeError(w, err)
		return
	}

	sharedHttp.WriteJSON(w, http.StatusOK, discardResponse{RestoredCount: restoredCount})
}

func (h *WorkspaceHandler) readRepositoryOperation(w http.ResponseWriter, r *http.Request) (string, string, bool) {
	if !h.extendGitWriteDeadline(w) {
		return "", "", false
	}

	userID := h.getUserID(r)
	if userID == "" {
		sharedHttp.WriteError(w, http.StatusUnauthorized, unauthenticatedMessage)
		return "", "", false
	}

	var request repositoryNameRequest
	if err := sharedHttp.ParseJSON(r, &request); err != nil {
		sharedHttp.WriteError(w, http.StatusBadRequest, invalidBodyMessage)
		return "", "", false
	}

	return userID, request.Name, true
}

func (h *WorkspaceHandler) extendGitWriteDeadline(w http.ResponseWriter) bool {
	if err := sharedHttp.ExtendWriteDeadline(w, domain.GitWriteDeadline); err != nil {
		sharedHttp.WriteError(w, http.StatusInternalServerError, gitDeadlineMessage)
		return false
	}
	return true
}

func workspaceGitErrorResponse(err error) (int, string, bool) {
	switch {
	case errors.Is(err, domain.ErrInvalidBranchName):
		return http.StatusBadRequest, branchNameMessage, true
	case errors.Is(err, domain.ErrBranchNotFound):
		return http.StatusNotFound, branchNotFoundMessage, true
	case errors.Is(err, domain.ErrInvalidRepoName):
		return http.StatusBadRequest, repositoryNameMessage, true
	case errors.Is(err, domain.ErrInvalidRoot),
		errors.Is(err, domain.ErrRootNotFound),
		errors.Is(err, domain.ErrRootNotDirectory),
		errors.Is(err, domain.ErrRootNotAllowed),
		errors.Is(err, domain.ErrRootOutsideScope):
		return http.StatusBadRequest, workspacePathMessage, true
	case errors.Is(err, domain.ErrRepoOutsideRoot):
		return http.StatusBadRequest, repositoryOutsideMessage, true
	case errors.Is(err, domain.ErrNotGitRepository):
		return http.StatusBadRequest, notGitRepositoryMessage, true
	case errors.Is(err, domain.ErrWorkingTreeDirty):
		return http.StatusConflict, workingTreeDirtyMessage, true
	case errors.Is(err, domain.ErrGitOperationFailed):
		return http.StatusConflict, gitRejectedMessage, true
	case errors.Is(err, domain.ErrGitNetworkFailure), errors.Is(err, domain.ErrGitTimeout):
		return http.StatusGatewayTimeout, gitNetworkMessage, true
	default:
		return 0, "", false
	}
}
