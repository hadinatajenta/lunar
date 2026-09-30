package transport

import (
	"context"
	"errors"
	"net/http"

	"lunar/backend/internal/confluence/application"
	sharedAuth "lunar/backend/internal/shared/auth"
	"lunar/backend/internal/shared/cache"
	sharedErrors "lunar/backend/internal/shared/errors"
	sharedHttp "lunar/backend/internal/shared/http"
)

type ConfluenceHandler struct {
	service *application.ConfluenceService
}

func NewConfluenceHandler(service *application.ConfluenceService) *ConfluenceHandler {
	return &ConfluenceHandler{service: service}
}

func (h *ConfluenceHandler) getUserID(r *http.Request) string {
	user, ok := sharedAuth.FromContext(r.Context())
	if !ok || user == nil {
		return ""
	}
	return user.UserID
}

func (h *ConfluenceHandler) writeError(w http.ResponseWriter, err error) {
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

func (h *ConfluenceHandler) requestContext(r *http.Request) context.Context {
	if r.URL.Query().Get("refresh") == "1" {
		return cache.WithForceRefresh(r.Context())
	}
	return r.Context()
}

func (h *ConfluenceHandler) ListDocuments(w http.ResponseWriter, r *http.Request) {
	userID := h.getUserID(r)
	scope := r.URL.Query().Get("scope")
	response, err := h.service.GetDocuments(h.requestContext(r), userID, scope)
	if err != nil {
		h.writeError(w, err)
		return
	}

	sharedHttp.WriteJSON(w, http.StatusOK, response)
}

func (h *ConfluenceHandler) GetDocument(w http.ResponseWriter, r *http.Request) {
	userID := h.getUserID(r)
	document, err := h.service.GetDocumentDetail(r.Context(), userID, r.PathValue("id"))
	if err != nil {
		h.writeError(w, err)
		return
	}

	sharedHttp.WriteJSON(w, http.StatusOK, document)
}
