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

package router

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"kpp.dev/kpfc/internal/handler"
	"kpp.dev/kpfc/internal/middleware"
)

func TestHealthCheck(t *testing.T) {
	// Create minimal config for testing health endpoint
	cfg := Config{
		AuthHandler:    &handler.AuthHandler{},
		UserHandler:    &handler.UserHandler{},
		DeckHandler:    &handler.DeckHandler{},
		CardHandler:    &handler.CardHandler{},
		JWTMiddleware:  &middleware.JWTMiddleware{},
		CORSMiddleware: middleware.NewCORSMiddleware([]string{"*"}),
		AGPLSourceURL:  "https://github.com/test/test",
	}

	r := New(cfg)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response HealthCheckResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.Status != "ok" {
		t.Errorf("expected status 'ok', got '%s'", response.Status)
	}

	if response.Version != "v1" {
		t.Errorf("expected version 'v1', got '%s'", response.Version)
	}

	if response.Timestamp == "" {
		t.Error("expected non-empty timestamp")
	}

	// Validate timestamp format
	_, err := time.Parse(time.RFC3339, response.Timestamp)
	if err != nil {
		t.Errorf("invalid timestamp format: %v", err)
	}
}

func TestHealthCheckContentType(t *testing.T) {
	cfg := Config{
		AuthHandler:    &handler.AuthHandler{},
		UserHandler:    &handler.UserHandler{},
		DeckHandler:    &handler.DeckHandler{},
		CardHandler:    &handler.CardHandler{},
		JWTMiddleware:  &middleware.JWTMiddleware{},
		CORSMiddleware: middleware.NewCORSMiddleware([]string{"*"}),
		AGPLSourceURL:  "https://github.com/test/test",
	}

	r := New(cfg)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("expected Content-Type 'application/json', got '%s'", contentType)
	}
}
