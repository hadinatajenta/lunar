package transport

import (
	"errors"
	"net/http"

	sharedAuth "lunar/backend/internal/shared/auth"
	sharedErrors "lunar/backend/internal/shared/errors"
	sharedHttp "lunar/backend/internal/shared/http"
	"lunar/backend/internal/workspace/application"
)

const (
	unauthenticatedMessage = "authenticated user is required"
	invalidBodyMessage     = "invalid request body"
	serverReaderMessage    = "server-side repository reader is not configured"
	repositoryReadMessage  = "cannot read repository data for this workspace"
	internalMessage        = "internal server error"
)

type WorkspaceHandler struct {
	service *application.WorkspaceService
}

func NewWorkspaceHandler(service *application.WorkspaceService) *WorkspaceHandler {
	return &WorkspaceHandler{service: service}
}

func (h *WorkspaceHandler) GetWorkspace(w http.ResponseWriter, r *http.Request) {
	userID := h.getUserID(r)
	if userID == "" {
		sharedHttp.WriteError(w, http.StatusUnauthorized, unauthenticatedMessage)
		return
	}

	config, err := h.service.GetWorkspace(r.Context(), userID)
	if err != nil {
		h.writeError(w, err)
		return
	}

	sharedHttp.WriteJSON(w, http.StatusOK, toWorkspaceResponse(config))
}

func (h *WorkspaceHandler) SaveWorkspace(w http.ResponseWriter, r *http.Request) {
	userID := h.getUserID(r)
	if userID == "" {
		sharedHttp.WriteError(w, http.StatusUnauthorized, unauthenticatedMessage)
		return
	}

	var request saveWorkspaceRequest
	if err := sharedHttp.ParseJSON(r, &request); err != nil {
		sharedHttp.WriteError(w, http.StatusBadRequest, invalidBodyMessage)
		return
	}

	config, err := h.service.SaveWorkspace(r.Context(), userID, application.SaveWorkspaceInput{
		Source:      request.Source,
		RootPath:    request.RootPath,
		HelperURL:   request.HelperURL,
		HelperToken: request.HelperToken,
	})
	if err != nil {
		h.writeError(w, err)
		return
	}

	sharedHttp.WriteJSON(w, http.StatusOK, toWorkspaceResponse(config))
}

func (h *WorkspaceHandler) ListRepositories(w http.ResponseWriter, r *http.Request) {
	userID := h.getUserID(r)
	if userID == "" {
		sharedHttp.WriteError(w, http.StatusUnauthorized, unauthenticatedMessage)
		return
	}

	repositoryList, err := h.service.ListRepositories(r.Context(), userID)
	if err != nil {
		h.writeError(w, err)
		return
	}

	sharedHttp.WriteJSON(w, http.StatusOK, toRepositoriesResponse(repositoryList))
}

func (h *WorkspaceHandler) ListServices(w http.ResponseWriter, r *http.Request) {
	userID := h.getUserID(r)
	if userID == "" {
		sharedHttp.WriteError(w, http.StatusUnauthorized, unauthenticatedMessage)
		return
	}

	serviceList, err := h.service.ListServices(r.Context(), userID)
	if err != nil {
		h.writeError(w, err)
		return
	}

	sharedHttp.WriteJSON(w, http.StatusOK, toServicesResponse(serviceList))
}

func (h *WorkspaceHandler) ReplaceSelections(w http.ResponseWriter, r *http.Request) {
	userID := h.getUserID(r)
	if userID == "" {
		sharedHttp.WriteError(w, http.StatusUnauthorized, unauthenticatedMessage)
		return
	}

	var request replaceSelectionsRequest
	if err := sharedHttp.ParseJSON(r, &request); err != nil {
		sharedHttp.WriteError(w, http.StatusBadRequest, invalidBodyMessage)
		return
	}

	selectedCount, err := h.service.ReplaceSelections(r.Context(), userID, request.Repos)
	if err != nil {
		h.writeError(w, err)
		return
	}

	sharedHttp.WriteJSON(w, http.StatusOK, selectionsResponse{SelectedCount: selectedCount})
}

func (h *WorkspaceHandler) getUserID(r *http.Request) string {
	user, ok := sharedAuth.FromContext(r.Context())
	if !ok || user == nil {
		return ""
	}
	return user.UserID
}

func (h *WorkspaceHandler) writeError(w http.ResponseWriter, err error) {
	if errors.Is(err, sharedErrors.ErrUnauthorized) {
		sharedHttp.WriteError(w, http.StatusUnauthorized, err.Error())
		return
	}
	if errors.Is(err, sharedErrors.ErrNotFound) {
		sharedHttp.WriteError(w, http.StatusNotFound, err.Error())
		return
	}
	if errors.Is(err, sharedErrors.ErrBadRequest) {
		sharedHttp.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	if errors.Is(err, application.ErrServerReaderUnavailable) {
		sharedHttp.WriteError(w, http.StatusServiceUnavailable, serverReaderMessage)
		return
	}
	if errors.Is(err, sharedErrors.ErrBadGateway) {
		sharedHttp.WriteError(w, http.StatusBadGateway, repositoryReadMessage)
		return
	}
	sharedHttp.WriteError(w, http.StatusInternalServerError, internalMessage)
}
