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
	"lunar/backend/internal/utdocs/application"
	"lunar/backend/internal/utdocs/domain"
)

const (
	messageInvalidRequestBody = "invalid unit test document request"
	messageInvalidApplyBody   = "invalid unit test document apply request"
	messageApplyUnavailable   = "the unit test document apply service is not configured"

	maxApplyRequestBodyBytes = 8 << 20
)

type Handler struct {
	service *application.DocumentService
	apply   *application.ApplyService
}

func NewHandler(service *application.DocumentService, apply ...*application.ApplyService) *Handler {
	handler := &Handler{service: service}
	if len(apply) > 0 {
		handler.apply = apply[0]
	}
	return handler
}

func (h *Handler) Generate(w http.ResponseWriter, r *http.Request) {
	var request domain.DocumentRequest
	if err := sharedHttp.ParseJSON(r, &request); err != nil {
		sharedHttp.WriteError(w, http.StatusBadRequest, messageInvalidRequestBody)
		return
	}

	result, err := h.service.Generate(r.Context(), authenticatedUserID(r), r.PathValue("id"), request)
	if err != nil {
		writeDocumentError(w, err)
		return
	}

	sharedHttp.WriteJSON(w, http.StatusOK, result)
}

func (h *Handler) Apply(w http.ResponseWriter, r *http.Request) {
	if h.apply == nil {
		sharedHttp.WriteError(w, http.StatusBadGateway, messageApplyUnavailable)
		return
	}

	var request domain.ApplyRequest
	if err := decodeApplyRequest(r, &request); err != nil {
		sharedHttp.WriteError(w, http.StatusBadRequest, messageInvalidApplyBody)
		return
	}

	result, err := h.apply.Apply(r.Context(), authenticatedUserID(r), r.PathValue("id"), request)
	if err != nil {
		writeDocumentError(w, err)
		return
	}

	sharedHttp.WriteJSON(w, http.StatusOK, result)
}

func decodeApplyRequest(r *http.Request, dst *domain.ApplyRequest) error {
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
	if err := decoder.Decode(dst); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return errors.New("request body must contain a single JSON object")
	}
	return nil
}

func authenticatedUserID(r *http.Request) string {
	user, ok := sharedAuth.FromContext(r.Context())
	if !ok || user == nil {
		return ""
	}
	return user.UserID
}

func writeDocumentError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, sharedErrors.ErrUnauthorized):
		sharedHttp.WriteError(w, http.StatusUnauthorized, err.Error())
	case errors.Is(err, sharedErrors.ErrNotFound):
		sharedHttp.WriteError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, sharedErrors.ErrBadRequest):
		sharedHttp.WriteError(w, http.StatusBadRequest, err.Error())
	default:
		sharedHttp.WriteError(w, http.StatusBadGateway, err.Error())
	}
}
