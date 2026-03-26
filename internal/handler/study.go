package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"kpp.dev/kpfc/internal/middleware"
	"kpp.dev/kpfc/internal/repository"
	"kpp.dev/kpfc/internal/service"
)

// StudyHandler handles study session and review endpoints.
type StudyHandler struct {
	study *service.StudyService
}

func NewStudyHandler(study *service.StudyService) *StudyHandler {
	return &StudyHandler{study: study}
}

func studyError(w http.ResponseWriter, err error) {
	if errors.Is(err, repository.ErrNotFound) {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	if errors.Is(err, service.ErrForbidden) {
		writeError(w, http.StatusForbidden, "forbidden")
		return
	}
	if errors.Is(err, service.ErrInvalidQuality) {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeError(w, http.StatusInternalServerError, "internal error")
}

// StartSession handles POST /api/v1/decks/{id}/study
func (h *StudyHandler) StartSession(w http.ResponseWriter, r *http.Request) {
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
		Mode string `json:"mode"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Mode == "" {
		writeError(w, http.StatusBadRequest, "mode is required")
		return
	}

	cards, err := h.study.StartSession(userID, deckID, req.Mode)
	if err != nil {
		if err.Error() == "mode must be \"spaced\" or \"random\"" {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		studyError(w, err)
		return
	}

	out := make([]map[string]any, len(cards))
	for i := range cards {
		out[i] = cardResponse(&cards[i])
	}
	writeJSON(w, http.StatusOK, out)
}

// SubmitReview handles POST /api/v1/cards/{id}/review
func (h *StudyHandler) SubmitReview(w http.ResponseWriter, r *http.Request) {
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
		Quality *int `json:"quality"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Quality == nil {
		writeError(w, http.StatusBadRequest, "quality is required")
		return
	}

	card, err := h.study.SubmitReview(userID, cardID, *req.Quality)
	if err != nil {
		studyError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, cardResponse(card))
}
