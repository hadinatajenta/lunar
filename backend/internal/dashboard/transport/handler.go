package transport

import (
	"net/http"

	"lunar/backend/internal/dashboard/application"
	sharedAuth "lunar/backend/internal/shared/auth"
	sharedHttp "lunar/backend/internal/shared/http"
)

type DashboardHandler struct {
	service *application.DashboardService
}

func NewDashboardHandler(service *application.DashboardService) *DashboardHandler {
	return &DashboardHandler{service: service}
}

func (h *DashboardHandler) getUserID(r *http.Request) string {
	user, ok := sharedAuth.FromContext(r.Context())
	if !ok || user == nil {
		return ""
	}
	return user.UserID
}

func (h *DashboardHandler) GetSummary(w http.ResponseWriter, r *http.Request) {
	userID := h.getUserID(r)
	summary := h.service.GetSummary(r.Context(), userID)
	sharedHttp.WriteJSON(w, http.StatusOK, summary)
}
