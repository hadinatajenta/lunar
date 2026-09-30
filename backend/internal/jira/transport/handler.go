package transport

import (
	"context"
	"errors"
	"net/http"

	"lunar/backend/internal/jira/application"
	sharedAuth "lunar/backend/internal/shared/auth"
	sharedErrors "lunar/backend/internal/shared/errors"
	sharedHttp "lunar/backend/internal/shared/http"
)

type JiraHandler struct {
	service *application.JiraService
}

func NewJiraHandler(service *application.JiraService) *JiraHandler {
	return &JiraHandler{service: service}
}

func (h *JiraHandler) getUserID(r *http.Request) string {
	user, ok := sharedAuth.FromContext(r.Context())
	if !ok || user == nil {
		return ""
	}
	return user.UserID
}

func (h *JiraHandler) writeError(w http.ResponseWriter, err error) {
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
	sharedHttp.WriteError(w, http.StatusBadGateway, err.Error())
}

func (h *JiraHandler) requestContext(r *http.Request) context.Context {
	if r.URL.Query().Get("refresh") == "1" {
		return application.WithForceRefresh(r.Context())
	}
	return r.Context()
}

func (h *JiraHandler) ListMyIssues(w http.ResponseWriter, r *http.Request) {
	userID := h.getUserID(r)
	issues, err := h.service.GetMyIssues(h.requestContext(r), userID)
	if err != nil {
		h.writeError(w, err)
		return
	}

	sharedHttp.WriteJSON(w, http.StatusOK, issues)
}

func (h *JiraHandler) GetBacklog(w http.ResponseWriter, r *http.Request) {
	userID := h.getUserID(r)
	squad := r.URL.Query().Get("squad")
	backlog, err := h.service.GetBacklog(h.requestContext(r), userID, squad)
	if err != nil {
		h.writeError(w, err)
		return
	}

	sharedHttp.WriteJSON(w, http.StatusOK, backlog)
}

func (h *JiraHandler) GetIssueDetail(w http.ResponseWriter, r *http.Request) {
	issueKey := r.PathValue("key")
	if issueKey == "" {
		issueKey = r.URL.Query().Get("key")
	}

	userID := h.getUserID(r)
	issue, err := h.service.GetIssueDetail(r.Context(), userID, issueKey)
	if err != nil {
		h.writeError(w, err)
		return
	}

	sharedHttp.WriteJSON(w, http.StatusOK, issue)
}
