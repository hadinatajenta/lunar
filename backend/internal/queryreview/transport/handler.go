package transport

import (
	"errors"
	"net/http"
	"time"

	"lunar/backend/internal/queryreview/application"
	"lunar/backend/internal/queryreview/domain"
	sharedAuth "lunar/backend/internal/shared/auth"
	sharedErrors "lunar/backend/internal/shared/errors"
	sharedHttp "lunar/backend/internal/shared/http"
)

const (
	queryReviewWriteDeadline = 3 * time.Minute

	messageUnauthenticated = "authenticated user is required"
	messageInvalidBody     = "invalid request body"
	messageDeadlineFailed  = "the request deadline could not be extended for the query review"
)

type QueryReviewHandler struct {
	service *application.QueryReviewService
}

func NewQueryReviewHandler(service *application.QueryReviewService) *QueryReviewHandler {
	return &QueryReviewHandler{service: service}
}

type queryReviewErrorBody struct {
	Error string `json:"error"`
	Code  string `json:"code,omitempty"`
}

func (h *QueryReviewHandler) Generate(w http.ResponseWriter, r *http.Request) {
	userID := queryReviewUserID(r)
	if userID == "" {
		sharedHttp.WriteError(w, http.StatusUnauthorized, messageUnauthenticated)
		return
	}

	var request domain.QueryReviewRequest
	if err := sharedHttp.ParseJSON(r, &request); err != nil {
		sharedHttp.WriteError(w, http.StatusBadRequest, messageInvalidBody)
		return
	}

	if err := sharedHttp.ExtendWriteDeadline(w, queryReviewWriteDeadline); err != nil {
		sharedHttp.WriteError(w, http.StatusInternalServerError, messageDeadlineFailed)
		return
	}

	result, err := h.service.GenerateQueryReview(r.Context(), userID, r.PathValue("id"), request)
	if err != nil {
		writeQueryReviewError(w, err)
		return
	}

	sharedHttp.WriteJSON(w, http.StatusOK, result)
}

func (h *QueryReviewHandler) Apply(w http.ResponseWriter, r *http.Request) {
	userID := queryReviewUserID(r)
	if userID == "" {
		sharedHttp.WriteError(w, http.StatusUnauthorized, messageUnauthenticated)
		return
	}

	var request domain.ApplyQueryReviewRequest
	if err := sharedHttp.ParseJSON(r, &request); err != nil {
		sharedHttp.WriteError(w, http.StatusBadRequest, messageInvalidBody)
		return
	}

	result, err := h.service.ApplyQueryReviewTable(r.Context(), userID, r.PathValue("id"), request)
	if err != nil {
		writeQueryReviewError(w, err)
		return
	}

	sharedHttp.WriteJSON(w, http.StatusOK, result)
}

func (h *QueryReviewHandler) MigTargets(w http.ResponseWriter, r *http.Request) {
	userID := queryReviewUserID(r)
	if userID == "" {
		sharedHttp.WriteError(w, http.StatusUnauthorized, messageUnauthenticated)
		return
	}

	result, err := h.service.MigTargets(r.Context(), userID, r.PathValue("id"))
	if err != nil {
		writeQueryReviewError(w, err)
		return
	}

	sharedHttp.WriteJSON(w, http.StatusOK, result)
}

func queryReviewUserID(r *http.Request) string {
	user, ok := sharedAuth.FromContext(r.Context())
	if !ok || user == nil {
		return ""
	}
	return user.UserID
}

func writeQueryReviewError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrNoCompareLink):
		sharedHttp.WriteJSON(w, http.StatusUnprocessableEntity, queryReviewErrorBody{
			Error: err.Error(),
			Code:  domain.ErrorCodeNoCompareLink,
		})
	case errors.Is(err, sharedErrors.ErrBadRequest):
		sharedHttp.WriteError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, sharedErrors.ErrUnauthorized):
		sharedHttp.WriteError(w, http.StatusUnauthorized, err.Error())
	case errors.Is(err, sharedErrors.ErrForbidden):
		sharedHttp.WriteError(w, http.StatusForbidden, err.Error())
	case errors.Is(err, sharedErrors.ErrNotFound):
		sharedHttp.WriteError(w, http.StatusNotFound, err.Error())
	default:
		sharedHttp.WriteError(w, http.StatusBadGateway, err.Error())
	}
}
