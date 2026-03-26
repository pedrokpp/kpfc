package handler

import (
	"net/http"

	"kpp.dev/kpfc/internal/service"
)

// PublicHandler handles unauthenticated public endpoints.
type PublicHandler struct {
	decks *service.DeckService
}

func NewPublicHandler(decks *service.DeckService) *PublicHandler {
	return &PublicHandler{decks: decks}
}

// ListDecks handles GET /api/v1/public/decks
// Query param: ?sort=upvotes (default: newest first)
func (h *PublicHandler) ListDecks(w http.ResponseWriter, r *http.Request) {
	sortBy := r.URL.Query().Get("sort")

	decks, err := h.decks.ListPublic(sortBy)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list public decks")
		return
	}

	out := make([]map[string]any, len(decks))
	for i := range decks {
		out[i] = deckResponse(&decks[i])
	}
	writeJSON(w, http.StatusOK, out)
}
