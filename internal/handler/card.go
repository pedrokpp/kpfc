package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"kpp.dev/kpfc/internal/middleware"
	"kpp.dev/kpfc/internal/model"
	"kpp.dev/kpfc/internal/repository"
	"kpp.dev/kpfc/internal/service"
)

// CardHandler handles card CRUD endpoints.
type CardHandler struct {
	cards *service.CardService
}

func NewCardHandler(cards *service.CardService) *CardHandler {
	return &CardHandler{cards: cards}
}

func cardResponse(c *model.Card) map[string]any {
	return map[string]any{
		"id":             c.ID,
		"deck_id":        c.DeckID,
		"front":          c.Front,
		"back":           c.Back,
		"card_type":      c.CardType,
		"cloze_index":    c.ClozeIndex,
		"extra":          c.Extra,
		"interval":       c.Interval,
		"repetitions":    c.Repetitions,
		"ease_factor":    c.EaseFactor,
		"next_review_at": c.NextReviewAt,
		"created_at":     c.CreatedAt,
		"updated_at":     c.UpdatedAt,
	}
}

func parseCardID(r *http.Request) (uint, bool) {
	raw := chi.URLParam(r, "id")
	n, err := strconv.ParseUint(raw, 10, 64)
	if err != nil {
		return 0, false
	}
	return uint(n), true
}

func cardError(w http.ResponseWriter, err error) {
	if errors.Is(err, repository.ErrNotFound) {
		writeError(w, http.StatusNotFound, "card not found")
		return
	}
	if errors.Is(err, service.ErrForbidden) {
		writeError(w, http.StatusForbidden, "forbidden")
		return
	}
	writeError(w, http.StatusInternalServerError, "internal error")
}

// ListByDeck handles GET /api/v1/decks/{id}/cards
func (h *CardHandler) ListByDeck(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	deckID, ok := parseDeckID(r)
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid deck id")
		return
	}

	cards, err := h.cards.ListByDeck(userID, deckID)
	if err != nil {
		cardError(w, err)
		return
	}

	out := make([]map[string]any, len(cards))
	for i := range cards {
		out[i] = cardResponse(&cards[i])
	}
	writeJSON(w, http.StatusOK, out)
}

// Create handles POST /api/v1/decks/{id}/cards
func (h *CardHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	deckID, ok := parseDeckID(r)
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid deck id")
		return
	}

	var req struct {
		Front      string `json:"front"`
		Back       string `json:"back"`
		CardType   string `json:"card_type"`
		ClozeIndex int    `json:"cloze_index"`
		Extra      string `json:"extra"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.CardType == "" {
		req.CardType = "basic"
	}

	var card *model.Card
	var err error

	if req.CardType == "basic" {
		if req.Front == "" || req.Back == "" {
			writeError(w, http.StatusBadRequest, "front and back are required")
			return
		}
		card, err = h.cards.Create(userID, deckID, req.Front, req.Back)
	} else {
		card, err = h.cards.CreateAdvanced(userID, deckID, service.CardCreateOpts{
			Front:      req.Front,
			Back:       req.Back,
			CardType:   req.CardType,
			ClozeIndex: req.ClozeIndex,
			Extra:      req.Extra,
		})
		if errors.Is(err, service.ErrInvalidCloze) || errors.Is(err, service.ErrInvalidCardType) {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
	}
	if err != nil {
		cardError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, cardResponse(card))
}

// Get handles GET /api/v1/cards/{id}
func (h *CardHandler) Get(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	cardID, ok := parseCardID(r)
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid card id")
		return
	}

	card, err := h.cards.GetByID(userID, cardID)
	if err != nil {
		cardError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, cardResponse(card))
}

// Update handles PUT /api/v1/cards/{id}
func (h *CardHandler) Update(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	cardID, ok := parseCardID(r)
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid card id")
		return
	}

	var req struct {
		Front      string `json:"front"`
		Back       string `json:"back"`
		CardType   string `json:"card_type"`
		ClozeIndex int    `json:"cloze_index"`
		Extra      string `json:"extra"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.CardType == "" {
		req.CardType = "basic"
	}

	var card *model.Card
	var err error

	if req.CardType == "basic" {
		if req.Front == "" || req.Back == "" {
			writeError(w, http.StatusBadRequest, "front and back are required")
			return
		}
		card, err = h.cards.Update(userID, cardID, req.Front, req.Back)
	} else {
		card, err = h.cards.UpdateAdvanced(userID, cardID, service.CardCreateOpts{
			Front:      req.Front,
			Back:       req.Back,
			CardType:   req.CardType,
			ClozeIndex: req.ClozeIndex,
			Extra:      req.Extra,
		})
		if errors.Is(err, service.ErrInvalidCloze) || errors.Is(err, service.ErrInvalidCardType) {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
	}
	if err != nil {
		cardError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, cardResponse(card))
}

// Delete handles DELETE /api/v1/cards/{id}
func (h *CardHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	cardID, ok := parseCardID(r)
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid card id")
		return
	}

	err := h.cards.Delete(userID, cardID)
	if err != nil {
		cardError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
