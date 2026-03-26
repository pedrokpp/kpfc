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

// DeckHandler handles deck CRUD endpoints.
type DeckHandler struct {
	decks *service.DeckService
}

func NewDeckHandler(decks *service.DeckService) *DeckHandler {
	return &DeckHandler{decks: decks}
}

func deckResponse(d *model.Deck) map[string]any {
	return map[string]any{
		"id":           d.ID,
		"user_id":      d.UserID,
		"title":        d.Title,
		"description":  d.Description,
		"is_public":    d.IsPublic,
		"upvote_count": d.UpvoteCount,
		"created_at":   d.CreatedAt,
		"updated_at":   d.UpdatedAt,
	}
}

func parseDeckID(r *http.Request) (uint, bool) {
	raw := chi.URLParam(r, "id")
	n, err := strconv.ParseUint(raw, 10, 64)
	if err != nil {
		return 0, false
	}
	return uint(n), true
}

// List handles GET /api/v1/decks
func (h *DeckHandler) List(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	decks, err := h.decks.ListByUser(userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list decks")
		return
	}

	out := make([]map[string]any, len(decks))
	for i := range decks {
		out[i] = deckResponse(&decks[i])
	}
	writeJSON(w, http.StatusOK, out)
}

// Create handles POST /api/v1/decks
func (h *DeckHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req struct {
		Title       string `json:"title"`
		Description string `json:"description"`
		IsPublic    bool   `json:"is_public"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Title == "" {
		writeError(w, http.StatusBadRequest, "title is required")
		return
	}

	deck, err := h.decks.Create(userID, req.Title, req.Description, req.IsPublic)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create deck")
		return
	}

	writeJSON(w, http.StatusCreated, deckResponse(deck))
}

// Get handles GET /api/v1/decks/{id}
func (h *DeckHandler) Get(w http.ResponseWriter, r *http.Request) {
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

	deck, err := h.decks.GetByID(userID, deckID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			writeError(w, http.StatusNotFound, "deck not found")
			return
		}
		if errors.Is(err, service.ErrForbidden) {
			writeError(w, http.StatusForbidden, "forbidden")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to get deck")
		return
	}

	writeJSON(w, http.StatusOK, deckResponse(deck))
}

// Update handles PUT /api/v1/decks/{id}
func (h *DeckHandler) Update(w http.ResponseWriter, r *http.Request) {
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
		Title       string `json:"title"`
		Description string `json:"description"`
		IsPublic    bool   `json:"is_public"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Title == "" {
		writeError(w, http.StatusBadRequest, "title is required")
		return
	}

	deck, err := h.decks.Update(userID, deckID, req.Title, req.Description, req.IsPublic)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			writeError(w, http.StatusNotFound, "deck not found")
			return
		}
		if errors.Is(err, service.ErrForbidden) {
			writeError(w, http.StatusForbidden, "forbidden")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to update deck")
		return
	}

	writeJSON(w, http.StatusOK, deckResponse(deck))
}

// Delete handles DELETE /api/v1/decks/{id}
func (h *DeckHandler) Delete(w http.ResponseWriter, r *http.Request) {
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

	err := h.decks.Delete(userID, deckID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			writeError(w, http.StatusNotFound, "deck not found")
			return
		}
		if errors.Is(err, service.ErrForbidden) {
			writeError(w, http.StatusForbidden, "forbidden")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to delete deck")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
