// Copyright 2026 kpp.dev
//
// This file is part of kpfc.
//
// kpfc is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// kpfc is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with kpfc. If not, see <https://www.gnu.org/licenses/>.

package handler

import (
	"encoding/json"
	"net/http"

	"kpp.dev/kpfc/internal/middleware"
	"kpp.dev/kpfc/internal/usecase/user"
)

type UserHandler struct {
	userUseCase *user.UseCase
}

func NewUserHandler(userUC *user.UseCase) *UserHandler {
	return &UserHandler{
		userUseCase: userUC,
	}
}

func (h *UserHandler) GetMe(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		respondError(w, r, http.ErrAbortHandler, http.StatusUnauthorized)
		return
	}

	user, err := h.userUseCase.GetUserByID(userID)
	if err != nil {
		respondError(w, r, err, http.StatusNotFound)
		return
	}

	respondJSON(w, http.StatusOK, user)
}

type UpdateUserRequest struct {
	Name string `json:"name"`
}

func (h *UserHandler) UpdateMe(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())

	var req UpdateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, r, err, http.StatusBadRequest)
		return
	}

	user, err := h.userUseCase.UpdateUser(userID, req.Name)
	if err != nil {
		respondError(w, r, err, http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusOK, user)
}

func (h *UserHandler) DeleteMe(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())

	if err := h.userUseCase.DeleteUser(userID); err != nil {
		respondError(w, r, err, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
