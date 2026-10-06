package transport

import (
	"errors"
	"net/http"

	sharedAuth "lunar/backend/internal/shared/auth"
	sharedErrors "lunar/backend/internal/shared/errors"
	sharedHttp "lunar/backend/internal/shared/http"
	workspaceApp "lunar/backend/internal/workspace/application"
	"lunar/backend/internal/workspace/domain"
)

const (
	issueKeyPathParameter = "key"

	noWorkspaceMessage         = "no workspace is configured"
	upstreamUnreachableMessage = "Jira or Bitbucket could not be reached"
	repositoryNotFoundMessage  = "the repository does not exist"
	repositoryDeadlineMessage  = "the request deadline could not be extended for the repository lookup"
)

type repositoryFeedbackRequest struct {
	Repo   string `json:"repo"`
	Action string `json:"action"`
}

type repositoryFeedbackResponse struct {
	Recorded bool `json:"recorded"`
}

type RepositorySuggestionHandler struct {
	service *workspaceApp.RepositorySuggestionService
}

func NewRepositorySuggestionHandler(service *workspaceApp.RepositorySuggestionService) *RepositorySuggestionHandler {
	return &RepositorySuggestionHandler{service: service}
}

func RegisterRepositoryRoutes(mux *http.ServeMux, authMiddleware func(http.Handler) http.Handler, workspaceRoots workspaceApp.WorkspaceRootSource, local workspaceApp.LocalRepositorySource, issues workspaceApp.IssueContextSource, branches workspaceApp.BranchMatchSource, ranker workspaceApp.RepositoryRanker) {
	catalogue := workspaceApp.NewRepositoryCatalogueService(workspaceRoots, local, nil)
	service := workspaceApp.NewRepositorySuggestionService(workspaceApp.RepositorySuggestionDependencies{
		Catalogue:      catalogue,
		WorkspaceRoots: workspaceRoots,
		Issues:         issues,
		Branches:       branches,
	}, workspaceApp.WithRepositoryRanker(ranker))
	handler := NewRepositorySuggestionHandler(service)

	mux.Handle("GET /api/jira/issues/{key}/repository-suggestions", authMiddleware(http.HandlerFunc(handler.GetRepositorySuggestions)))
	mux.Handle("GET /api/jira/repositories", authMiddleware(http.HandlerFunc(handler.ListRepositories)))
	mux.Handle("POST /api/jira/issues/{key}/repository-feedback", authMiddleware(http.HandlerFunc(handler.RecordRepositoryFeedback)))
}

func (h *RepositorySuggestionHandler) GetRepositorySuggestions(w http.ResponseWriter, r *http.Request) {
	if !extendRepositoryDeadline(w) {
		return
	}

	userID := repositoryRequestUserID(r)
	if userID == "" {
		sharedHttp.WriteError(w, http.StatusUnauthorized, unauthenticatedMessage)
		return
	}

	suggestions, err := h.service.Suggest(r.Context(), userID, r.PathValue(issueKeyPathParameter))
	if err != nil {
		writeRepositoryError(w, err)
		return
	}

	sharedHttp.WriteJSON(w, http.StatusOK, suggestions)
}

func (h *RepositorySuggestionHandler) ListRepositories(w http.ResponseWriter, r *http.Request) {
	userID := repositoryRequestUserID(r)
	if userID == "" {
		sharedHttp.WriteError(w, http.StatusUnauthorized, unauthenticatedMessage)
		return
	}

	catalogue, err := h.service.SearchRepositories(r.Context(), userID, r.URL.Query().Get("search"))
	if err != nil {
		writeRepositoryError(w, err)
		return
	}

	sharedHttp.WriteJSON(w, http.StatusOK, catalogue)
}

func (h *RepositorySuggestionHandler) RecordRepositoryFeedback(w http.ResponseWriter, r *http.Request) {
	userID := repositoryRequestUserID(r)
	if userID == "" {
		sharedHttp.WriteError(w, http.StatusUnauthorized, unauthenticatedMessage)
		return
	}

	var request repositoryFeedbackRequest
	if err := sharedHttp.ParseJSON(r, &request); err != nil {
		sharedHttp.WriteError(w, http.StatusBadRequest, invalidBodyMessage)
		return
	}

	action, err := workspaceApp.ParseFeedbackAction(request.Action)
	if err != nil {
		writeRepositoryError(w, err)
		return
	}

	if err := h.service.RecordFeedback(r.Context(), userID, r.PathValue(issueKeyPathParameter), request.Repo, action); err != nil {
		writeRepositoryError(w, err)
		return
	}

	sharedHttp.WriteJSON(w, http.StatusAccepted, repositoryFeedbackResponse{Recorded: true})
}

func repositoryRequestUserID(r *http.Request) string {
	user, ok := sharedAuth.FromContext(r.Context())
	if !ok || user == nil {
		return ""
	}

	return user.UserID
}

func extendRepositoryDeadline(w http.ResponseWriter) bool {
	if err := sharedHttp.ExtendWriteDeadline(w, domain.GitWriteDeadline); err != nil {
		sharedHttp.WriteError(w, http.StatusInternalServerError, repositoryDeadlineMessage)
		return false
	}

	return true
}

func writeRepositoryError(w http.ResponseWriter, err error) {
	status, message := repositoryErrorResponse(err)
	sharedHttp.WriteError(w, status, message)
}

func repositoryErrorResponse(err error) (int, string) {
	switch {
	case errors.Is(err, workspaceApp.ErrWorkspaceNotUsable):
		return http.StatusUnprocessableEntity, noWorkspaceMessage
	case errors.Is(err, sharedErrors.ErrUnauthorized):
		return http.StatusUnauthorized, unauthenticatedMessage
	case errors.Is(err, sharedErrors.ErrNotFound):
		return http.StatusNotFound, repositoryNotFoundMessage
	case errors.Is(err, sharedErrors.ErrBadRequest):
		return http.StatusBadRequest, err.Error()
	case errors.Is(err, workspaceApp.ErrServerReaderUnavailable):
		return http.StatusServiceUnavailable, serverReaderMessage
	case errors.Is(err, sharedErrors.ErrBadGateway):
		return http.StatusBadGateway, upstreamUnreachableMessage
	case errors.Is(err, domain.ErrGitTimeout):
		return http.StatusGatewayTimeout, repositoryReadMessage
	default:
		return http.StatusInternalServerError, internalMessage
	}
}
