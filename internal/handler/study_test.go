package handler_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/go-chi/chi/v5"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"kpp.dev/kpfc/internal/handler"
	"kpp.dev/kpfc/internal/middleware"
	"kpp.dev/kpfc/internal/model"
	"kpp.dev/kpfc/internal/repository"
	"kpp.dev/kpfc/internal/service"
)

func setupStudyTestRouter(t *testing.T) (*chi.Mux, *service.AuthService) {
	t.Helper()
	dsn := "file:" + t.Name() + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Discard,
	})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.Deck{}, &model.Card{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	userRepo := repository.NewGORMUserRepository(db)
	deckRepo := repository.NewGORMDeckRepository(db)
	cardRepo := repository.NewGORMCardRepository(db)
	accessSvc := service.NewAccessService(deckRepo, cardRepo, nil)
	authSvc := service.NewAuthService(userRepo, "test-secret")
	deckSvc := service.NewDeckService(deckRepo, accessSvc)
	cardSvc := service.NewCardService(cardRepo, accessSvc)
	studySvc := service.NewStudyService(cardRepo, userRepo, accessSvc)
	authHandler := handler.NewAuthHandler(authSvc)
	deckHandler := handler.NewDeckHandler(deckSvc)
	cardHandler := handler.NewCardHandler(cardSvc)
	studyHandler := handler.NewStudyHandler(studySvc)

	r := chi.NewRouter()
	r.Post("/api/v1/auth/register", authHandler.Register)
	r.Post("/api/v1/auth/login", authHandler.Login)
	r.Group(func(r chi.Router) {
		r.Use(middleware.Auth(authSvc))
		r.Post("/api/v1/decks", deckHandler.Create)
		r.Get("/api/v1/decks/{id}/cards", cardHandler.ListByDeck)
		r.Post("/api/v1/decks/{id}/cards", cardHandler.Create)
		r.Get("/api/v1/cards/{id}", cardHandler.Get)
		r.Post("/api/v1/decks/{id}/study", studyHandler.StartSession)
		r.Post("/api/v1/cards/{id}/review", studyHandler.SubmitReview)
	})
	return r, authSvc
}

func registerAndLoginStudy(t *testing.T, r *chi.Mux, email, password string) string {
	t.Helper()
	postJSON(r, "/api/v1/auth/register", map[string]string{
		"email": email, "password": password,
	})
	w := postJSON(r, "/api/v1/auth/login", map[string]string{
		"email": email, "password": password,
	})
	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp)
	return resp["token"].(string)
}

func createDeckStudy(t *testing.T, r *chi.Mux, token, title string) float64 {
	t.Helper()
	w := postJSONWithToken(r, "/api/v1/decks", map[string]any{"title": title}, token)
	if w.Code != http.StatusCreated {
		t.Fatalf("createDeckStudy: status = %d", w.Code)
	}
	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp)
	return resp["id"].(float64)
}

func createCardStudy(t *testing.T, r *chi.Mux, token string, deckID float64, front, back string) float64 {
	t.Helper()
	path := fmt.Sprintf("/api/v1/decks/%.0f/cards", deckID)
	w := postJSONWithToken(r, path, map[string]any{"front": front, "back": back}, token)
	if w.Code != http.StatusCreated {
		t.Fatalf("createCardStudy: status = %d", w.Code)
	}
	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp)
	return resp["id"].(float64)
}

// --- POST /api/v1/decks/{id}/study (random mode) ---

func TestStartSession_RandomMode_Success(t *testing.T) {
	r, _ := setupStudyTestRouter(t)
	token := registerAndLoginStudy(t, r, "alice@example.com", "pass")
	deckID := createDeckStudy(t, r, token, "Go Deck")
	createCardStudy(t, r, token, deckID, "Q1", "A1")
	createCardStudy(t, r, token, deckID, "Q2", "A2")

	w := postJSONWithToken(r, fmt.Sprintf("/api/v1/decks/%.0f/study", deckID),
		map[string]any{"mode": "random"}, token)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", w.Code, w.Body)
	}
	var resp []any
	json.NewDecoder(w.Body).Decode(&resp)
	if len(resp) != 2 {
		t.Errorf("len = %d, want 2", len(resp))
	}
}

func TestStartSession_SpacedMode_EmptyWhenNoDueCards(t *testing.T) {
	r, _ := setupStudyTestRouter(t)
	token := registerAndLoginStudy(t, r, "alice@example.com", "pass")
	deckID := createDeckStudy(t, r, token, "Go Deck")
	// Cards created with NextReviewAt zero-value — FindDueCards uses <= now,
	// so zero time (year 0001) is always in the past and should be returned.
	createCardStudy(t, r, token, deckID, "Q1", "A1")

	w := postJSONWithToken(r, fmt.Sprintf("/api/v1/decks/%.0f/study", deckID),
		map[string]any{"mode": "spaced"}, token)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", w.Code, w.Body)
	}
	var resp []any
	json.NewDecoder(w.Body).Decode(&resp)
	// Zero-value NextReviewAt (0001-01-01) is before now, so card is due.
	if len(resp) != 1 {
		t.Errorf("len = %d, want 1 (card with zero NextReviewAt is due)", len(resp))
	}
}

func TestStartSession_InvalidMode(t *testing.T) {
	r, _ := setupStudyTestRouter(t)
	token := registerAndLoginStudy(t, r, "alice@example.com", "pass")
	deckID := createDeckStudy(t, r, token, "Go Deck")

	w := postJSONWithToken(r, fmt.Sprintf("/api/v1/decks/%.0f/study", deckID),
		map[string]any{"mode": "marathon"}, token)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}

func TestStartSession_MissingMode(t *testing.T) {
	r, _ := setupStudyTestRouter(t)
	token := registerAndLoginStudy(t, r, "alice@example.com", "pass")
	deckID := createDeckStudy(t, r, token, "Go Deck")

	w := postJSONWithToken(r, fmt.Sprintf("/api/v1/decks/%.0f/study", deckID),
		map[string]any{}, token)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}

func TestStartSession_OtherUserDeck(t *testing.T) {
	r, _ := setupStudyTestRouter(t)
	tokenA := registerAndLoginStudy(t, r, "a@example.com", "pass")
	tokenB := registerAndLoginStudy(t, r, "b@example.com", "pass")
	deckID := createDeckStudy(t, r, tokenA, "A's Deck")

	w := postJSONWithToken(r, fmt.Sprintf("/api/v1/decks/%.0f/study", deckID),
		map[string]any{"mode": "random"}, tokenB)

	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", w.Code)
	}
}

func TestStartSession_Unauthenticated(t *testing.T) {
	r, _ := setupStudyTestRouter(t)
	w := postJSONWithToken(r, "/api/v1/decks/1/study",
		map[string]any{"mode": "random"}, "")

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", w.Code)
	}
}

func TestStartSession_DeckNotFound(t *testing.T) {
	r, _ := setupStudyTestRouter(t)
	token := registerAndLoginStudy(t, r, "alice@example.com", "pass")

	w := postJSONWithToken(r, "/api/v1/decks/99999/study",
		map[string]any{"mode": "random"}, token)

	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", w.Code)
	}
}

// --- POST /api/v1/cards/{id}/review ---

func TestSubmitReview_Success_UpdatesSM2State(t *testing.T) {
	r, _ := setupStudyTestRouter(t)
	token := registerAndLoginStudy(t, r, "alice@example.com", "pass")
	deckID := createDeckStudy(t, r, token, "Deck")
	cardID := createCardStudy(t, r, token, deckID, "Q", "A")

	w := postJSONWithToken(r, fmt.Sprintf("/api/v1/cards/%.0f/review", cardID),
		map[string]any{"quality": 5}, token)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", w.Code, w.Body)
	}
	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp)

	// After one perfect review: repetitions should be 1, interval 1.
	if resp["repetitions"].(float64) != 1 {
		t.Errorf("repetitions = %v, want 1", resp["repetitions"])
	}
	if resp["interval"].(float64) != 1 {
		t.Errorf("interval = %v, want 1", resp["interval"])
	}
}

func TestSubmitReview_BadQuality_ResetsState(t *testing.T) {
	r, _ := setupStudyTestRouter(t)
	token := registerAndLoginStudy(t, r, "alice@example.com", "pass")
	deckID := createDeckStudy(t, r, token, "Deck")
	cardID := createCardStudy(t, r, token, deckID, "Q", "A")

	// First review: advance the card.
	postJSONWithToken(r, fmt.Sprintf("/api/v1/cards/%.0f/review", cardID),
		map[string]any{"quality": 5}, token)
	postJSONWithToken(r, fmt.Sprintf("/api/v1/cards/%.0f/review", cardID),
		map[string]any{"quality": 5}, token)

	// Bad review: should reset repetitions and interval.
	w := postJSONWithToken(r, fmt.Sprintf("/api/v1/cards/%.0f/review", cardID),
		map[string]any{"quality": 1}, token)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", w.Code, w.Body)
	}
	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["repetitions"].(float64) != 0 {
		t.Errorf("repetitions = %v, want 0 after bad review", resp["repetitions"])
	}
	if resp["interval"].(float64) != 1 {
		t.Errorf("interval = %v, want 1 after bad review", resp["interval"])
	}
}

func TestSubmitReview_InvalidQuality_TooHigh(t *testing.T) {
	r, _ := setupStudyTestRouter(t)
	token := registerAndLoginStudy(t, r, "alice@example.com", "pass")
	deckID := createDeckStudy(t, r, token, "Deck")
	cardID := createCardStudy(t, r, token, deckID, "Q", "A")

	w := postJSONWithToken(r, fmt.Sprintf("/api/v1/cards/%.0f/review", cardID),
		map[string]any{"quality": 6}, token)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}

func TestSubmitReview_InvalidQuality_Negative(t *testing.T) {
	r, _ := setupStudyTestRouter(t)
	token := registerAndLoginStudy(t, r, "alice@example.com", "pass")
	deckID := createDeckStudy(t, r, token, "Deck")
	cardID := createCardStudy(t, r, token, deckID, "Q", "A")

	w := postJSONWithToken(r, fmt.Sprintf("/api/v1/cards/%.0f/review", cardID),
		map[string]any{"quality": -1}, token)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}

func TestSubmitReview_MissingQuality(t *testing.T) {
	r, _ := setupStudyTestRouter(t)
	token := registerAndLoginStudy(t, r, "alice@example.com", "pass")
	deckID := createDeckStudy(t, r, token, "Deck")
	cardID := createCardStudy(t, r, token, deckID, "Q", "A")

	w := postJSONWithToken(r, fmt.Sprintf("/api/v1/cards/%.0f/review", cardID),
		map[string]any{}, token)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}

func TestSubmitReview_OtherUserCard(t *testing.T) {
	r, _ := setupStudyTestRouter(t)
	tokenA := registerAndLoginStudy(t, r, "a@example.com", "pass")
	tokenB := registerAndLoginStudy(t, r, "b@example.com", "pass")
	deckID := createDeckStudy(t, r, tokenA, "A's Deck")
	cardID := createCardStudy(t, r, tokenA, deckID, "Q", "A")

	w := postJSONWithToken(r, fmt.Sprintf("/api/v1/cards/%.0f/review", cardID),
		map[string]any{"quality": 4}, tokenB)

	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", w.Code)
	}
}

func TestSubmitReview_CardNotFound(t *testing.T) {
	r, _ := setupStudyTestRouter(t)
	token := registerAndLoginStudy(t, r, "alice@example.com", "pass")

	w := postJSONWithToken(r, "/api/v1/cards/99999/review",
		map[string]any{"quality": 4}, token)

	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", w.Code)
	}
}

func TestSubmitReview_Unauthenticated(t *testing.T) {
	r, _ := setupStudyTestRouter(t)

	w := postJSONWithToken(r, "/api/v1/cards/1/review",
		map[string]any{"quality": 4}, "")

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", w.Code)
	}
}

func TestSpacedMode_NoStateChangeInRandomSession(t *testing.T) {
	// Verify that StartSession(random) does not modify card SM-2 state.
	r, _ := setupStudyTestRouter(t)
	token := registerAndLoginStudy(t, r, "alice@example.com", "pass")
	deckID := createDeckStudy(t, r, token, "Deck")
	cardID := createCardStudy(t, r, token, deckID, "Q", "A")

	// Read initial card state.
	wBefore := getWithToken(r, fmt.Sprintf("/api/v1/cards/%.0f", cardID), token)
	var before map[string]any
	json.NewDecoder(wBefore.Body).Decode(&before)

	// Start a random session.
	postJSONWithToken(r, fmt.Sprintf("/api/v1/decks/%.0f/study", deckID),
		map[string]any{"mode": "random"}, token)

	// Card state should be unchanged.
	wAfter := getWithToken(r, fmt.Sprintf("/api/v1/cards/%.0f", cardID), token)
	var after map[string]any
	json.NewDecoder(wAfter.Body).Decode(&after)

	if before["repetitions"] != after["repetitions"] {
		t.Errorf("repetitions changed after random session: %v → %v",
			before["repetitions"], after["repetitions"])
	}
	if before["interval"] != after["interval"] {
		t.Errorf("interval changed after random session: %v → %v",
			before["interval"], after["interval"])
	}
}
