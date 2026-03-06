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

	"github.com/go-chi/chi/v5"

	"kpp.dev/kpfc/internal/middleware"
	"kpp.dev/kpfc/internal/usecase/deck"
)

type DeckHandler struct {
	deckUseCase *deck.UseCase
}

func NewDeckHandler(deckUC *deck.UseCase) *DeckHandler {
	return &DeckHandler{
		deckUseCase: deckUC,
	}
}

type CreateDeckRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	IsPublic    bool   `json:"is_public"`
}

func (h *DeckHandler) CreateDeck(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())

	var req CreateDeckRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, r, err, http.StatusBadRequest)
		return
	}

	deck, err := h.deckUseCase.CreateDeck(userID, req.Title, req.Description, req.IsPublic)
	if err != nil {
		respondError(w, r, err, http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusCreated, deck)
}

func (h *DeckHandler) GetMyDecks(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())

	decks, err := h.deckUseCase.GetUserDecks(userID)
	if err != nil {
		respondError(w, r, err, http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusOK, decks)
}

func (h *DeckHandler) GetPublicDecks(w http.ResponseWriter, r *http.Request) {
	decks, err := h.deckUseCase.GetPublicDecks()
	if err != nil {
		respondError(w, r, err, http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusOK, decks)
}

func (h *DeckHandler) GetDeck(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	deckID := chi.URLParam(r, "deckId")

	deck, err := h.deckUseCase.GetDeck(userID, deckID)
	if err != nil {
		respondError(w, r, err, http.StatusNotFound)
		return
	}

	respondJSON(w, http.StatusOK, deck)
}

func (h *DeckHandler) UpdateDeck(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	deckID := chi.URLParam(r, "deckId")

	var req CreateDeckRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, r, err, http.StatusBadRequest)
		return
	}

	deck, err := h.deckUseCase.UpdateDeck(userID, deckID, req.Title, req.Description, req.IsPublic)
	if err != nil {
		respondError(w, r, err, http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusOK, deck)
}

func (h *DeckHandler) DeleteDeck(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	deckID := chi.URLParam(r, "deckId")

	if err := h.deckUseCase.DeleteDeck(userID, deckID); err != nil {
		respondError(w, r, err, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *DeckHandler) CloneDeck(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	deckID := chi.URLParam(r, "deckId")

	deck, err := h.deckUseCase.CloneDeck(userID, deckID)
	if err != nil {
		respondError(w, r, err, http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusCreated, deck)
}
