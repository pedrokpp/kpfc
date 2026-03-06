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

type AuthHandler struct {
	userUseCase *user.UseCase
	jwtMid      *middleware.JWTMiddleware
}

func NewAuthHandler(userUC *user.UseCase, jwtMid *middleware.JWTMiddleware) *AuthHandler {
	return &AuthHandler{
		userUseCase: userUC,
		jwtMid:      jwtMid,
	}
}

type RegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Name     string `json:"name"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type AuthResponse struct {
	Token string      `json:"token"`
	User  interface{} `json:"user"`
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, r, err, http.StatusBadRequest)
		return
	}

	user, err := h.userUseCase.Register(req.Email, req.Password, req.Name)
	if err != nil {
		respondError(w, r, err, http.StatusInternalServerError)
		return
	}

	token, err := h.jwtMid.GenerateToken(user.ID)
	if err != nil {
		respondError(w, r, err, http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusCreated, AuthResponse{Token: token, User: user})
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, r, err, http.StatusBadRequest)
		return
	}

	user, err := h.userUseCase.Login(req.Email, req.Password)
	if err != nil {
		respondError(w, r, err, http.StatusUnauthorized)
		return
	}

	token, err := h.jwtMid.GenerateToken(user.ID)
	if err != nil {
		respondError(w, r, err, http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusOK, AuthResponse{Token: token, User: user})
}
