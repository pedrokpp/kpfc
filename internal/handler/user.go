package handler

import (
	"encoding/json"
	"net/http"

	"kpp.dev/kpfc/internal/middleware"
	"kpp.dev/kpfc/internal/model"
	"kpp.dev/kpfc/internal/service"
)

// UserHandler handles user profile endpoints.
type UserHandler struct {
	users *service.UserService
}

func NewUserHandler(users *service.UserService) *UserHandler {
	return &UserHandler{users: users}
}

func userResponse(u *model.User) map[string]any {
	return map[string]any{
		"id":           u.ID,
		"email":        u.Email,
		"display_name": u.DisplayName,
		"login_streak": u.LoginStreak,
		"total_points": u.TotalPoints,
	}
}

// GetMe handles GET /api/v1/users/me
func (h *UserHandler) GetMe(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	user, err := h.users.GetProfile(userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get profile")
		return
	}

	writeJSON(w, http.StatusOK, userResponse(user))
}

// UpdateMe handles PUT /api/v1/users/me
// Body: {"display_name"}
func (h *UserHandler) UpdateMe(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req struct {
		DisplayName string `json:"display_name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	user, err := h.users.UpdateProfile(userID, req.DisplayName)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update profile")
		return
	}

	writeJSON(w, http.StatusOK, userResponse(user))
}
