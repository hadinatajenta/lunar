package transport

import (
	"errors"
	"net/http"
	"time"

	"lunar/backend/internal/jira/application"
	sharedErrors "lunar/backend/internal/shared/errors"
	sharedHttp "lunar/backend/internal/shared/http"
)

const (
	branchNameRoute         = "GET /api/jira/issues/{key}/branch-name"
	branchNameIssueKey      = "key"
	branchNameWriteDeadline = 60 * time.Second

	branchNameUnauthenticatedMessage = "authenticated user is required"
	branchNameUnauthorizedMessage    = "invalid or expired Jira PAT"
	branchNameNotFoundMessage        = "the issue was not found in Jira"
	branchNameUpstreamMessage        = "unable to reach Jira BRI API. Please check your VPN connection"
	branchNameDeadlineMessage        = "the request deadline could not be extended for the branch name"
)

type BranchNameHandler struct {
	service *application.BranchNameService
}

func RegisterBranchNameRoutes(mux *http.ServeMux, authMiddleware func(http.Handler) http.Handler, service *application.BranchNameService) {
	handler := &BranchNameHandler{service: service}
	mux.Handle(branchNameRoute, authMiddleware(http.HandlerFunc(handler.GetBranchName)))
}

func (h *BranchNameHandler) GetBranchName(w http.ResponseWriter, r *http.Request) {
	if err := sharedHttp.ExtendWriteDeadline(w, branchNameWriteDeadline); err != nil {
		sharedHttp.WriteError(w, http.StatusInternalServerError, branchNameDeadlineMessage)
		return
	}

	userID := issueBranchUserID(r)
	if userID == "" {
		sharedHttp.WriteError(w, http.StatusUnauthorized, branchNameUnauthenticatedMessage)
		return
	}

	suggestion, err := h.service.Suggest(r.Context(), userID, r.PathValue(branchNameIssueKey))
	if err != nil {
		writeBranchNameError(w, err)
		return
	}

	sharedHttp.WriteJSON(w, http.StatusOK, suggestion)
}

func writeBranchNameError(w http.ResponseWriter, err error) {
	status, message := branchNameErrorResponse(err)
	sharedHttp.WriteError(w, status, message)
}

func branchNameErrorResponse(err error) (int, string) {
	switch {
	case errors.Is(err, sharedErrors.ErrUnauthorized):
		return http.StatusUnauthorized, branchNameUnauthorizedMessage
	case errors.Is(err, sharedErrors.ErrNotFound):
		return http.StatusNotFound, branchNameNotFoundMessage
	case errors.Is(err, sharedErrors.ErrBadRequest):
		return http.StatusBadRequest, err.Error()
	default:
		return http.StatusBadGateway, branchNameUpstreamMessage
	}
}
