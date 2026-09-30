package transport

import (
	"errors"
	"net/http"
	"strings"

	"lunar/backend/internal/auth/application"
	"lunar/backend/internal/auth/domain"
	sharedAuth "lunar/backend/internal/shared/auth"
	sharedErrors "lunar/backend/internal/shared/errors"
	sharedHttp "lunar/backend/internal/shared/http"
)

type AuthHandler struct {
	authService *application.AuthService
}

func NewAuthHandler(authService *application.AuthService) *AuthHandler {
	return &AuthHandler{
		authService: authService,
	}
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sharedHttp.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req application.RegisterRequest
	if err := sharedHttp.ParseJSON(r, &req); err != nil {
		sharedHttp.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	res, err := h.authService.Register(r.Context(), req)
	if err != nil {
		if errors.Is(err, sharedErrors.ErrConflict) {
			sharedHttp.WriteError(w, http.StatusConflict, "user with this email already exists")
			return
		}
		if errors.Is(err, sharedErrors.ErrBadRequest) {
			sharedHttp.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		sharedHttp.WriteError(w, http.StatusInternalServerError, "failed to register user")
		return
	}
	sharedHttp.WriteJSON(w, http.StatusCreated, res)
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sharedHttp.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req application.LoginRequest
	if err := sharedHttp.ParseJSON(r, &req); err != nil {
		sharedHttp.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	res, err := h.authService.Login(r.Context(), req)
	if err != nil {
		if errors.Is(err, sharedErrors.ErrUnauthorized) {
			sharedHttp.WriteError(w, http.StatusUnauthorized, "invalid email or password")
			return
		}
		if errors.Is(err, sharedErrors.ErrBadRequest) {
			sharedHttp.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		sharedHttp.WriteError(w, http.StatusInternalServerError, "failed to authenticate")
		return
	}
	sharedHttp.WriteJSON(w, http.StatusOK, res)
}

func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		sharedHttp.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	userCtx, ok := sharedAuth.FromContext(r.Context())
	if !ok || userCtx == nil {
		sharedHttp.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	profile, err := h.authService.GetProfile(r.Context(), userCtx.UserID)
	if err != nil {
		if errors.Is(err, sharedErrors.ErrNotFound) {
			sharedHttp.WriteError(w, http.StatusNotFound, "user profile not found")
			return
		}
		sharedHttp.WriteError(w, http.StatusInternalServerError, "failed to fetch user profile")
		return
	}
	sharedHttp.WriteJSON(w, http.StatusOK, profile)
}

func (h *AuthHandler) GetSecrets(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		sharedHttp.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	userCtx, ok := sharedAuth.FromContext(r.Context())
	if !ok || userCtx == nil {
		sharedHttp.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	secrets, err := h.authService.GetRedactedSecrets(r.Context(), userCtx.UserID)
	if err != nil {
		sharedHttp.WriteError(w, http.StatusInternalServerError, "failed to retrieve secrets")
		return
	}
	sharedHttp.WriteJSON(w, http.StatusOK, secrets)
}

func (h *AuthHandler) SaveSecrets(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut && r.Method != http.MethodPost {
		sharedHttp.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	userCtx, ok := sharedAuth.FromContext(r.Context())
	if !ok || userCtx == nil {
		sharedHttp.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var input domain.SaveSecretsInput
	if err := sharedHttp.ParseJSON(r, &input); err != nil {
		sharedHttp.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.authService.SaveUserSecrets(r.Context(), userCtx.UserID, input); err != nil {
		sharedHttp.WriteError(w, http.StatusInternalServerError, "failed to save secrets")
		return
	}
	secrets, err := h.authService.GetRedactedSecrets(r.Context(), userCtx.UserID)
	if err != nil {
		sharedHttp.WriteJSON(w, http.StatusOK, map[string]string{"status": "updated"})
		return
	}
	sharedHttp.WriteJSON(w, http.StatusOK, secrets)
}

type verifyPATRequest struct {
	PAT string `json:"pat"`
}

type registerWithJiraRequest struct {
	PAT      string `json:"pat"`
	Password string `json:"password"`
}

func (h *AuthHandler) VerifyJiraPAT(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sharedHttp.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req verifyPATRequest
	if err := sharedHttp.ParseJSON(r, &req); err != nil {
		sharedHttp.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	if strings.TrimSpace(req.PAT) == "" {
		sharedHttp.WriteError(w, http.StatusBadRequest, "jira pat is required")
		return
	}

	res, err := h.authService.VerifyJiraPAT(r.Context(), req.PAT)
	if err != nil {
		if errors.Is(err, sharedErrors.ErrUnauthorized) {
			sharedHttp.WriteError(w, http.StatusUnauthorized, "invalid or expired Jira PAT")
			return
		}
		if errors.Is(err, sharedErrors.ErrBadGateway) {
			sharedHttp.WriteError(w, http.StatusBadGateway, "unable to reach Jira BRI API. Please check your VPN connection.")
			return
		}
		if errors.Is(err, sharedErrors.ErrBadRequest) {
			sharedHttp.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		sharedHttp.WriteError(w, http.StatusInternalServerError, "failed to verify jira pat")
		return
	}

	sharedHttp.WriteJSON(w, http.StatusOK, res)
}

func (h *AuthHandler) RegisterWithJira(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sharedHttp.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req registerWithJiraRequest
	if err := sharedHttp.ParseJSON(r, &req); err != nil {
		sharedHttp.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	if strings.TrimSpace(req.PAT) == "" {
		sharedHttp.WriteError(w, http.StatusBadRequest, "jira pat is required")
		return
	}

	if len(req.Password) < 6 {
		sharedHttp.WriteError(w, http.StatusBadRequest, "password must be at least 6 characters")
		return
	}

	res, err := h.authService.RegisterWithJira(r.Context(), req.PAT, req.Password)
	if err != nil {
		if errors.Is(err, sharedErrors.ErrConflict) {
			sharedHttp.WriteError(w, http.StatusConflict, "an account with this BRI email already exists")
			return
		}
		if errors.Is(err, sharedErrors.ErrUnauthorized) {
			sharedHttp.WriteError(w, http.StatusUnauthorized, "invalid or expired Jira PAT")
			return
		}
		if errors.Is(err, sharedErrors.ErrBadGateway) {
			sharedHttp.WriteError(w, http.StatusBadGateway, "unable to reach Jira BRI API. Please check your VPN connection.")
			return
		}
		if errors.Is(err, sharedErrors.ErrBadRequest) {
			sharedHttp.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		sharedHttp.WriteError(w, http.StatusInternalServerError, "failed to register with jira")
		return
	}

	sharedHttp.WriteJSON(w, http.StatusCreated, res)
}
