package transport

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	sharedAuth "lunar/backend/internal/shared/auth"
	sharedErrors "lunar/backend/internal/shared/errors"
	sharedHttp "lunar/backend/internal/shared/http"
	sopApplication "lunar/backend/internal/sop/application"
	"lunar/backend/internal/sop/domain"
)

const (
	messageInvalidGenerateBody = "invalid SOP generate request body"
	messageInvalidApplyBody    = "invalid SOP apply request body"
	messageUnauthenticated     = "authenticated user is required"
	messageServiceUnavailable  = "the SOP service is not configured"

	maxApplyRequestBodyBytes = 8 << 20
)

type Handler struct {
	service *sopApplication.Service
}

func NewHandler(service *sopApplication.Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Generate(w http.ResponseWriter, r *http.Request) {
	if h.service == nil {
		sharedHttp.WriteError(w, http.StatusBadGateway, messageServiceUnavailable)
		return
	}
	userID := sopUserID(r)
	if userID == "" {
		sharedHttp.WriteError(w, http.StatusUnauthorized, messageUnauthenticated)
		return
	}

	var request domain.GenerateRequest
	if err := sharedHttp.ParseJSON(r, &request); err != nil {
		sharedHttp.WriteError(w, http.StatusBadRequest, messageInvalidGenerateBody)
		return
	}

	result, err := h.service.Generate(r.Context(), userID, r.PathValue("id"), request)
	if err != nil {
		writeSOPError(w, err)
		return
	}
	sharedHttp.WriteJSON(w, http.StatusOK, result)
}

func (h *Handler) Apply(w http.ResponseWriter, r *http.Request) {
	if h.service == nil {
		sharedHttp.WriteError(w, http.StatusBadGateway, messageServiceUnavailable)
		return
	}
	userID := sopUserID(r)
	if userID == "" {
		sharedHttp.WriteError(w, http.StatusUnauthorized, messageUnauthenticated)
		return
	}

	var request domain.ApplyRequest
	if err := decodeApplyRequest(r, &request); err != nil {
		sharedHttp.WriteError(w, http.StatusBadRequest, messageInvalidApplyBody)
		return
	}

	result, err := h.service.Apply(r.Context(), userID, r.PathValue("id"), request)
	if err != nil {
		writeSOPError(w, err)
		return
	}
	sharedHttp.WriteJSON(w, http.StatusOK, result)
}

func decodeApplyRequest(r *http.Request, destination *domain.ApplyRequest) error {
	if r.Body == nil {
		return errors.New("empty request body")
	}
	defer r.Body.Close()

	body, err := io.ReadAll(io.LimitReader(r.Body, maxApplyRequestBodyBytes+1))
	if err != nil {
		return err
	}
	if len(body) > maxApplyRequestBodyBytes {
		return errors.New("request body exceeds the maximum allowed size")
	}

	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return errors.New("request body must contain a single JSON object")
	}
	return nil
}

func sopUserID(r *http.Request) string {
	user, ok := sharedAuth.FromContext(r.Context())
	if !ok || user == nil {
		return ""
	}
	return user.UserID
}

func writeSOPError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, sharedErrors.ErrUnauthorized):
		sharedHttp.WriteError(w, http.StatusUnauthorized, err.Error())
	case errors.Is(err, sharedErrors.ErrForbidden):
		sharedHttp.WriteError(w, http.StatusForbidden, err.Error())
	case errors.Is(err, sharedErrors.ErrNotFound):
		sharedHttp.WriteError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, sharedErrors.ErrConflict):
		sharedHttp.WriteError(w, http.StatusConflict, err.Error())
	case errors.Is(err, sharedErrors.ErrBadRequest):
		sharedHttp.WriteError(w, http.StatusBadRequest, err.Error())
	default:
		sharedHttp.WriteError(w, http.StatusBadGateway, err.Error())
	}
}
