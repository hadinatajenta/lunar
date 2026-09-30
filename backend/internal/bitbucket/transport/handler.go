package transport

import (
	"errors"
	"net/http"
	"strings"

	"lunar/backend/internal/bitbucket/application"
	"lunar/backend/internal/bitbucket/domain"
	sharedAuth "lunar/backend/internal/shared/auth"
	sharedErrors "lunar/backend/internal/shared/errors"
	sharedHttp "lunar/backend/internal/shared/http"
)

type BitbucketHandler struct {
	service *application.BitbucketService
}

func NewBitbucketHandler(service *application.BitbucketService) *BitbucketHandler {
	return &BitbucketHandler{service: service}
}

func (h *BitbucketHandler) getUserID(r *http.Request) string {
	user, ok := sharedAuth.FromContext(r.Context())
	if !ok || user == nil {
		return ""
	}
	return user.UserID
}

func (h *BitbucketHandler) ListPushes(w http.ResponseWriter, r *http.Request) {
	filter := r.URL.Query().Get("filter")
	userID := h.getUserID(r)

	pushes, err := h.service.ListPushes(r.Context(), userID, filter)
	if err != nil {
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

	sharedHttp.WriteJSON(w, http.StatusOK, pushes)
}

func (h *BitbucketHandler) ListPullRequests(w http.ResponseWriter, r *http.Request) {
	filter := r.URL.Query().Get("filter")
	userID := h.getUserID(r)

	prs, err := h.service.ListPullRequests(r.Context(), userID, filter)
	if err != nil {
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

	sharedHttp.WriteJSON(w, http.StatusOK, prs)
}

func (h *BitbucketHandler) GetDiff(w http.ResponseWriter, r *http.Request) {
	prID := r.PathValue("id")
	repo := r.URL.Query().Get("repo")
	userID := h.getUserID(r)

	diff, err := h.service.GetPullRequestDiff(r.Context(), userID, repo, prID)
	if err != nil {
		if errors.Is(err, sharedErrors.ErrBadRequest) {
			sharedHttp.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		sharedHttp.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	sharedHttp.WriteJSON(w, http.StatusOK, diff)
}

func (h *BitbucketHandler) CreatePullRequest(w http.ResponseWriter, r *http.Request) {
	var req domain.CreatePRRequest
	if err := sharedHttp.ParseJSON(r, &req); err != nil {
		sharedHttp.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	userID := h.getUserID(r)
	pr, err := h.service.CreatePullRequest(r.Context(), userID, req)
	if err != nil {
		if errors.Is(err, sharedErrors.ErrBadRequest) {
			sharedHttp.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		sharedHttp.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	sharedHttp.WriteJSON(w, http.StatusCreated, pr)
}

func (h *BitbucketHandler) GenerateAIReview(w http.ResponseWriter, r *http.Request) {
	prID := r.PathValue("id")
	repo := r.URL.Query().Get("repo")
	model := r.URL.Query().Get("model")
	userID := h.getUserID(r)

	var reqBody struct {
		Model string `json:"model"`
	}
	_ = sharedHttp.ParseJSON(r, &reqBody)
	if reqBody.Model != "" {
		model = reqBody.Model
	}

	review, err := h.service.GenerateAIReview(r.Context(), userID, repo, prID, model)
	if err != nil {
		if errors.Is(err, sharedErrors.ErrBadRequest) {
			sharedHttp.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		if errors.Is(err, sharedErrors.ErrUnauthorized) {
			sharedHttp.WriteError(w, http.StatusUnauthorized, err.Error())
			return
		}
		sharedHttp.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	sharedHttp.WriteJSON(w, http.StatusOK, review)
}

func (h *BitbucketHandler) PostComment(w http.ResponseWriter, r *http.Request) {
	prID := r.PathValue("id")
	repo := r.URL.Query().Get("repo")
	userID := h.getUserID(r)

	var req domain.PRCommentRequest
	if err := sharedHttp.ParseJSON(r, &req); err != nil {
		sharedHttp.WriteError(w, http.StatusBadRequest, "invalid comment request body")
		return
	}

	comment, err := h.service.PostPRComment(r.Context(), userID, repo, prID, req.Content)
	if err != nil {
		if errors.Is(err, sharedErrors.ErrBadRequest) {
			sharedHttp.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		sharedHttp.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	sharedHttp.WriteJSON(w, http.StatusCreated, comment)
}

func (h *BitbucketHandler) ApplyAction(w http.ResponseWriter, r *http.Request) {
	prID := r.PathValue("id")
	repo := r.URL.Query().Get("repo")
	userID := h.getUserID(r)

	var req domain.ReviewActionRequest
	if err := sharedHttp.ParseJSON(r, &req); err != nil {
		sharedHttp.WriteError(w, http.StatusBadRequest, "invalid action request body")
		return
	}

	if err := h.service.ApplyReviewAction(r.Context(), userID, repo, prID, strings.TrimSpace(req.Action)); err != nil {
		if errors.Is(err, sharedErrors.ErrBadRequest) {
			sharedHttp.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		if errors.Is(err, sharedErrors.ErrNotFound) {
			sharedHttp.WriteError(w, http.StatusNotFound, "pull request not found")
			return
		}
		sharedHttp.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	sharedHttp.WriteJSON(w, http.StatusOK, map[string]string{
		"status": "success",
		"action": req.Action,
		"pr_id":  prID,
	})
}

func (h *BitbucketHandler) Reset(w http.ResponseWriter, r *http.Request) {
	h.service.ResetSeedData()
	sharedHttp.WriteJSON(w, http.StatusOK, map[string]string{"status": "reset"})
}
