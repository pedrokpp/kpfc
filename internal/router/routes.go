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
	"time"

	"github.com/go-chi/chi/v5"
)

// setupRoutes configures all application routes
func setupRoutes(r chi.Router, cfg Config) {
	// Health check endpoint (public, no auth required)
	r.Get("/health", healthCheckHandler)

	// API v1 routes
	r.Route("/api/v1", func(r chi.Router) {
		// Auth routes (public)
		r.Post("/auth/register", cfg.AuthHandler.Register)
		r.Post("/auth/login", cfg.AuthHandler.Login)

		// Authenticated routes
		r.Group(func(r chi.Router) {
			r.Use(cfg.JWTMiddleware.Authenticate)

			// User routes
			r.Get("/users/me", cfg.UserHandler.GetMe)
			r.Put("/users/me", cfg.UserHandler.UpdateMe)
			r.Delete("/users/me", cfg.UserHandler.DeleteMe)

			// Deck routes
			r.Get("/decks", cfg.DeckHandler.GetMyDecks)
			r.Post("/decks", cfg.DeckHandler.CreateDeck)
			r.Get("/decks/public", cfg.DeckHandler.GetPublicDecks)
			r.Get("/decks/{deckId}", cfg.DeckHandler.GetDeck)
			r.Put("/decks/{deckId}", cfg.DeckHandler.UpdateDeck)
			r.Delete("/decks/{deckId}", cfg.DeckHandler.DeleteDeck)
			r.Post("/decks/{deckId}/clone", cfg.DeckHandler.CloneDeck)

			// Card routes
			r.Get("/decks/{deckId}/cards", cfg.CardHandler.GetCards)
			r.Post("/decks/{deckId}/cards", cfg.CardHandler.CreateCard)
			r.Get("/decks/{deckId}/cards/due", cfg.CardHandler.GetDueCards)
			r.Get("/decks/{deckId}/cards/{cardId}", cfg.CardHandler.GetCard)
			r.Put("/decks/{deckId}/cards/{cardId}", cfg.CardHandler.UpdateCard)
			r.Delete("/decks/{deckId}/cards/{cardId}", cfg.CardHandler.DeleteCard)
			r.Post("/decks/{deckId}/cards/{cardId}/review", cfg.CardHandler.ReviewCard)
		})
	})
}

// HealthCheckResponse represents the health check response
type HealthCheckResponse struct {
	Status    string `json:"status"`
	Version   string `json:"version"`
	Timestamp string `json:"timestamp"`
}

// healthCheckHandler handles GET /health requests
func healthCheckHandler(w http.ResponseWriter, r *http.Request) {
	response := HealthCheckResponse{
		Status:    "ok",
		Version:   "v1",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}
