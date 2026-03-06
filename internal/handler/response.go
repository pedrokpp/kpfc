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
	"errors"
	"net/http"

	"kpp.dev/kpfc/internal/domain"
)

type ProblemDetail struct {
	Type     string `json:"type"`
	Title    string `json:"title"`
	Status   int    `json:"status"`
	Detail   string `json:"detail"`
	Instance string `json:"instance"`
}

func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func respondError(w http.ResponseWriter, r *http.Request, err error, defaultStatus int) {
	status := defaultStatus
	title := http.StatusText(status)
	detail := err.Error()

	if errors.Is(err, domain.ErrNotFound) {
		status = http.StatusNotFound
		title = "Not Found"
	} else if errors.Is(err, domain.ErrAlreadyExists) {
		status = http.StatusConflict
		title = "Conflict"
	} else if errors.Is(err, domain.ErrUnauthorized) || errors.Is(err, domain.ErrInvalidCredentials) {
		status = http.StatusUnauthorized
		title = "Unauthorized"
	} else if errors.Is(err, domain.ErrForbidden) {
		status = http.StatusForbidden
		title = "Forbidden"
	} else if errors.Is(err, domain.ErrInvalidInput) {
		status = http.StatusBadRequest
		title = "Bad Request"
	}

	problem := ProblemDetail{
		Type:     "about:blank",
		Title:    title,
		Status:   status,
		Detail:   detail,
		Instance: r.RequestURI,
	}

	respondJSON(w, status, problem)
}
