package transport

import (
	"errors"
	"net/http"
	"strings"

	"lunar/backend/internal/copilot/application"
	"lunar/backend/internal/copilot/domain"
	sharedAuth "lunar/backend/internal/shared/auth"
	sharedErrors "lunar/backend/internal/shared/errors"
	sharedHttp "lunar/backend/internal/shared/http"
)

type CopilotHandler struct {
	service *application.CopilotService
}

func NewCopilotHandler(service *application.CopilotService) *CopilotHandler {
	return &CopilotHandler{service: service}
}

func (h *CopilotHandler) getUserID(r *http.Request) string {
	userCtx, ok := sharedAuth.FromContext(r.Context())
	if !ok || userCtx == nil {
		return ""
	}
	return userCtx.UserID
}

func (h *CopilotHandler) ListModels(w http.ResponseWriter, r *http.Request) {
	userID := h.getUserID(r)
	if userID == "" {
		sharedHttp.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	models, err := h.service.ListModels(r.Context(), userID)
	if err != nil {
		sharedHttp.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	sharedHttp.WriteJSON(w, http.StatusOK, models)
}

func (h *CopilotHandler) Chat(w http.ResponseWriter, r *http.Request) {
	userID := h.getUserID(r)
	if userID == "" {
		sharedHttp.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req domain.ChatRequest
	if err := sharedHttp.ParseJSON(r, &req); err != nil {
		sharedHttp.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	res, err := h.service.Chat(r.Context(), userID, req)
	if err != nil {
		if errors.Is(err, sharedErrors.ErrBadRequest) {
			sharedHttp.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		if errors.Is(err, sharedErrors.ErrUnauthorized) {
			sharedHttp.WriteError(w, http.StatusUnauthorized, err.Error())
			return
		}
		if errors.Is(err, sharedErrors.ErrForbidden) {
			sharedHttp.WriteError(w, http.StatusForbidden, err.Error())
			return
		}
		sharedHttp.WriteError(w, http.StatusBadGateway, err.Error())
		return
	}

	sharedHttp.WriteJSON(w, http.StatusOK, res)
}

type providerKeyResponse struct {
	Provider string `json:"provider"`
	APIKey   string `json:"api_key"`
	BaseURL  string `json:"base_url"`
	Model    string `json:"model"`
}

func (h *CopilotHandler) ProviderKey(w http.ResponseWriter, r *http.Request) {
	userID := h.getUserID(r)
	if userID == "" {
		sharedHttp.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	model := strings.TrimSpace(r.URL.Query().Get("model"))
	if model == "" {
		sharedHttp.WriteError(w, http.StatusBadRequest, "a model is required")
		return
	}

	credentials, err := h.service.ResolveProviderCredentials(r.Context(), userID, model)
	if err != nil {
		if errors.Is(err, sharedErrors.ErrBadRequest) {
			sharedHttp.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		if errors.Is(err, sharedErrors.ErrUnauthorized) {
			sharedHttp.WriteError(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		sharedHttp.WriteError(w, http.StatusInternalServerError, "cannot resolve the provider credentials")
		return
	}

	sharedHttp.WriteJSON(w, http.StatusOK, providerKeyResponse{
		Provider: credentials.Provider,
		APIKey:   credentials.APIKey,
		BaseURL:  credentials.BaseURL,
		Model:    credentials.Model,
	})
}

func (h *CopilotHandler) SaveTranscript(w http.ResponseWriter, r *http.Request) {
	userID := h.getUserID(r)
	if userID == "" {
		sharedHttp.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req domain.TranscriptRequest
	if err := sharedHttp.ParseJSON(r, &req); err != nil {
		sharedHttp.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	res, err := h.service.SaveTranscript(r.Context(), userID, req)
	if err != nil {
		if errors.Is(err, sharedErrors.ErrBadRequest) {
			sharedHttp.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		if errors.Is(err, sharedErrors.ErrNotFound) {
			sharedHttp.WriteError(w, http.StatusNotFound, "session not found")
			return
		}
		if errors.Is(err, sharedErrors.ErrForbidden) {
			sharedHttp.WriteError(w, http.StatusForbidden, "access denied")
			return
		}
		sharedHttp.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	sharedHttp.WriteJSON(w, http.StatusCreated, res)
}

func (h *CopilotHandler) ListSessions(w http.ResponseWriter, r *http.Request) {
	userID := h.getUserID(r)
	if userID == "" {
		sharedHttp.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	sessions, err := h.service.GetSessions(r.Context(), userID)
	if err != nil {
		sharedHttp.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if sessions == nil {
		sessions = []domain.ChatSession{}
	}
	sharedHttp.WriteJSON(w, http.StatusOK, sessions)
}

func (h *CopilotHandler) DeleteSession(w http.ResponseWriter, r *http.Request) {
	userID := h.getUserID(r)
	if userID == "" {
		sharedHttp.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	sessionID := r.PathValue("id")
	if sessionID == "" {
		sharedHttp.WriteError(w, http.StatusBadRequest, "missing session id")
		return
	}

	if err := h.service.DeleteSession(r.Context(), userID, sessionID); err != nil {
		if errors.Is(err, sharedErrors.ErrNotFound) {
			sharedHttp.WriteError(w, http.StatusNotFound, "session not found")
			return
		}
		sharedHttp.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	sharedHttp.WriteJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (h *CopilotHandler) ListMessages(w http.ResponseWriter, r *http.Request) {
	userID := h.getUserID(r)
	if userID == "" {
		sharedHttp.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	sessionID := r.PathValue("id")
	if sessionID == "" {
		sharedHttp.WriteError(w, http.StatusBadRequest, "missing session id")
		return
	}

	messages, err := h.service.GetMessages(r.Context(), userID, sessionID)
	if err != nil {
		if errors.Is(err, sharedErrors.ErrNotFound) {
			sharedHttp.WriteError(w, http.StatusNotFound, "session not found")
			return
		}
		if errors.Is(err, sharedErrors.ErrForbidden) {
			sharedHttp.WriteError(w, http.StatusForbidden, "access denied")
			return
		}
		sharedHttp.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if messages == nil {
		messages = []domain.ChatMessage{}
	}
	sharedHttp.WriteJSON(w, http.StatusOK, messages)
}
