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

	"kpp.dev/kpfc/internal/domain"
	"kpp.dev/kpfc/internal/middleware"
	"kpp.dev/kpfc/internal/usecase/card"
)

type CardHandler struct {
	cardUseCase *card.UseCase
}

func NewCardHandler(cardUC *card.UseCase) *CardHandler {
	return &CardHandler{
		cardUseCase: cardUC,
	}
}

type CreateCardRequest struct {
	Front string `json:"front"`
	Back  string `json:"back"`
}

func (h *CardHandler) CreateCard(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	deckID := chi.URLParam(r, "deckId")

	var req CreateCardRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, r, err, http.StatusBadRequest)
		return
	}

	card, err := h.cardUseCase.CreateCard(userID, deckID, req.Front, req.Back)
	if err != nil {
		respondError(w, r, err, http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusCreated, card)
}

func (h *CardHandler) GetCards(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	deckID := chi.URLParam(r, "deckId")

	cards, err := h.cardUseCase.GetCardsByDeck(userID, deckID)
	if err != nil {
		respondError(w, r, err, http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusOK, cards)
}

func (h *CardHandler) GetDueCards(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	deckID := chi.URLParam(r, "deckId")

	cards, err := h.cardUseCase.GetDueCards(userID, deckID)
	if err != nil {
		respondError(w, r, err, http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusOK, cards)
}

func (h *CardHandler) GetCard(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	cardID := chi.URLParam(r, "cardId")

	card, err := h.cardUseCase.GetCard(userID, cardID)
	if err != nil {
		respondError(w, r, err, http.StatusNotFound)
		return
	}

	respondJSON(w, http.StatusOK, card)
}

func (h *CardHandler) UpdateCard(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	cardID := chi.URLParam(r, "cardId")

	var req CreateCardRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, r, err, http.StatusBadRequest)
		return
	}

	card, err := h.cardUseCase.UpdateCard(userID, cardID, req.Front, req.Back)
	if err != nil {
		respondError(w, r, err, http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusOK, card)
}

func (h *CardHandler) DeleteCard(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	cardID := chi.URLParam(r, "cardId")

	if err := h.cardUseCase.DeleteCard(userID, cardID); err != nil {
		respondError(w, r, err, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

type ReviewCardRequest struct {
	Quality int `json:"quality"`
}

func (h *CardHandler) ReviewCard(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	cardID := chi.URLParam(r, "cardId")

	var req ReviewCardRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, r, err, http.StatusBadRequest)
		return
	}

	card, err := h.cardUseCase.ReviewCard(userID, cardID, domain.ReviewQuality(req.Quality))
	if err != nil {
		respondError(w, r, err, http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusOK, card)
}
