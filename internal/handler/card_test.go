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

func setupCardTestRouter(t *testing.T) (*chi.Mux, *service.AuthService) {
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
	authSvc := service.NewAuthService(userRepo, "test-secret")
	deckSvc := service.NewDeckService(deckRepo)
	cardSvc := service.NewCardService(cardRepo, deckRepo)
	authHandler := handler.NewAuthHandler(authSvc)
	deckHandler := handler.NewDeckHandler(deckSvc)
	cardHandler := handler.NewCardHandler(cardSvc)

	r := chi.NewRouter()
	r.Post("/api/v1/auth/register", authHandler.Register)
	r.Post("/api/v1/auth/login", authHandler.Login)
	r.Group(func(r chi.Router) {
		r.Use(middleware.Auth(authSvc))
		r.Post("/api/v1/decks", deckHandler.Create)
		r.Get("/api/v1/decks/{id}/cards", cardHandler.ListByDeck)
		r.Post("/api/v1/decks/{id}/cards", cardHandler.Create)
		r.Get("/api/v1/cards/{id}", cardHandler.Get)
		r.Put("/api/v1/cards/{id}", cardHandler.Update)
		r.Delete("/api/v1/cards/{id}", cardHandler.Delete)
	})
	return r, authSvc
}

// registerAndLoginCard returns a token for a newly registered user.
func registerAndLoginCard(t *testing.T, r *chi.Mux, email, password string) string {
	t.Helper()
	postJSON(r, "/api/v1/auth/register", map[string]string{
		"email":    email,
		"password": password,
	})
	w := postJSON(r, "/api/v1/auth/login", map[string]string{
		"email":    email,
		"password": password,
	})
	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp)
	return resp["token"].(string)
}

// createDeckCard creates a deck and returns its ID.
func createDeckCard(t *testing.T, r *chi.Mux, token, title string) float64 {
	t.Helper()
	w := postJSONWithToken(r, "/api/v1/decks", map[string]any{"title": title}, token)
	if w.Code != http.StatusCreated {
		t.Fatalf("createDeckCard: status = %d; body: %s", w.Code, w.Body)
	}
	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp)
	return resp["id"].(float64)
}

// createCard creates a card in deckID and returns its ID.
func createCard(t *testing.T, r *chi.Mux, token string, deckID float64, front, back string) float64 {
	t.Helper()
	path := fmt.Sprintf("/api/v1/decks/%.0f/cards", deckID)
	w := postJSONWithToken(r, path, map[string]any{"front": front, "back": back}, token)
	if w.Code != http.StatusCreated {
		t.Fatalf("createCard: status = %d; body: %s", w.Code, w.Body)
	}
	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp)
	return resp["id"].(float64)
}

// --- POST /api/v1/decks/{id}/cards ---

func TestCreateCard_Success(t *testing.T) {
	r, _ := setupCardTestRouter(t)
	token := registerAndLoginCard(t, r, "alice@example.com", "pass")
	deckID := createDeckCard(t, r, token, "Go Basics")

	w := postJSONWithToken(r, fmt.Sprintf("/api/v1/decks/%.0f/cards", deckID), map[string]any{
		"front": "What is a goroutine?",
		"back":  "A lightweight thread managed by the Go runtime.",
	}, token)

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201; body: %s", w.Code, w.Body)
	}
	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["front"] != "What is a goroutine?" {
		t.Errorf("front = %v", resp["front"])
	}
	if resp["deck_id"] == nil {
		t.Error("expected deck_id in response")
	}
}

func TestCreateCard_MissingFields(t *testing.T) {
	r, _ := setupCardTestRouter(t)
	token := registerAndLoginCard(t, r, "alice@example.com", "pass")
	deckID := createDeckCard(t, r, token, "Deck")

	w := postJSONWithToken(r, fmt.Sprintf("/api/v1/decks/%.0f/cards", deckID), map[string]any{
		"front": "only front",
	}, token)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}

func TestCreateCard_OtherUserDeck(t *testing.T) {
	r, _ := setupCardTestRouter(t)
	tokenA := registerAndLoginCard(t, r, "a@example.com", "pass")
	tokenB := registerAndLoginCard(t, r, "b@example.com", "pass")
	deckID := createDeckCard(t, r, tokenA, "A's Deck")

	w := postJSONWithToken(r, fmt.Sprintf("/api/v1/decks/%.0f/cards", deckID), map[string]any{
		"front": "Q",
		"back":  "A",
	}, tokenB)
	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", w.Code)
	}
}

func TestCreateCard_DeckNotFound(t *testing.T) {
	r, _ := setupCardTestRouter(t)
	token := registerAndLoginCard(t, r, "alice@example.com", "pass")

	w := postJSONWithToken(r, "/api/v1/decks/99999/cards", map[string]any{
		"front": "Q",
		"back":  "A",
	}, token)
	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", w.Code)
	}
}

func TestCreateCard_Unauthenticated(t *testing.T) {
	r, _ := setupCardTestRouter(t)

	w := postJSONWithToken(r, "/api/v1/decks/1/cards", map[string]any{
		"front": "Q",
		"back":  "A",
	}, "")
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", w.Code)
	}
}

// --- GET /api/v1/decks/{id}/cards ---

func TestListCards_Success(t *testing.T) {
	r, _ := setupCardTestRouter(t)
	token := registerAndLoginCard(t, r, "alice@example.com", "pass")
	deckID := createDeckCard(t, r, token, "Deck")
	createCard(t, r, token, deckID, "Q1", "A1")
	createCard(t, r, token, deckID, "Q2", "A2")

	w := getWithToken(r, fmt.Sprintf("/api/v1/decks/%.0f/cards", deckID), token)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", w.Code, w.Body)
	}
	var resp []any
	json.NewDecoder(w.Body).Decode(&resp)
	if len(resp) != 2 {
		t.Errorf("len = %d, want 2", len(resp))
	}
}

func TestListCards_OtherUserDeck(t *testing.T) {
	r, _ := setupCardTestRouter(t)
	tokenA := registerAndLoginCard(t, r, "a@example.com", "pass")
	tokenB := registerAndLoginCard(t, r, "b@example.com", "pass")
	deckID := createDeckCard(t, r, tokenA, "A's Deck")

	w := getWithToken(r, fmt.Sprintf("/api/v1/decks/%.0f/cards", deckID), tokenB)
	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", w.Code)
	}
}

func TestListCards_Unauthenticated(t *testing.T) {
	r, _ := setupCardTestRouter(t)
	w := getWithToken(r, "/api/v1/decks/1/cards", "")
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", w.Code)
	}
}

// --- GET /api/v1/cards/{id} ---

func TestGetCard_OwnCard(t *testing.T) {
	r, _ := setupCardTestRouter(t)
	token := registerAndLoginCard(t, r, "alice@example.com", "pass")
	deckID := createDeckCard(t, r, token, "Deck")
	cardID := createCard(t, r, token, deckID, "Q", "A")

	w := getWithToken(r, fmt.Sprintf("/api/v1/cards/%.0f", cardID), token)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", w.Code, w.Body)
	}
	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["front"] != "Q" {
		t.Errorf("front = %v, want Q", resp["front"])
	}
}

func TestGetCard_OtherUserCard(t *testing.T) {
	r, _ := setupCardTestRouter(t)
	tokenA := registerAndLoginCard(t, r, "a@example.com", "pass")
	tokenB := registerAndLoginCard(t, r, "b@example.com", "pass")
	deckID := createDeckCard(t, r, tokenA, "A's Deck")
	cardID := createCard(t, r, tokenA, deckID, "Q", "A")

	w := getWithToken(r, fmt.Sprintf("/api/v1/cards/%.0f", cardID), tokenB)
	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", w.Code)
	}
}

func TestGetCard_NotFound(t *testing.T) {
	r, _ := setupCardTestRouter(t)
	token := registerAndLoginCard(t, r, "alice@example.com", "pass")

	w := getWithToken(r, "/api/v1/cards/99999", token)
	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", w.Code)
	}
}

func TestGetCard_Unauthenticated(t *testing.T) {
	r, _ := setupCardTestRouter(t)
	w := getWithToken(r, "/api/v1/cards/1", "")
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", w.Code)
	}
}

// --- PUT /api/v1/cards/{id} ---

func TestUpdateCard_Success(t *testing.T) {
	r, _ := setupCardTestRouter(t)
	token := registerAndLoginCard(t, r, "alice@example.com", "pass")
	deckID := createDeckCard(t, r, token, "Deck")
	cardID := createCard(t, r, token, deckID, "Q", "A")

	w := putJSONWithToken(r, fmt.Sprintf("/api/v1/cards/%.0f", cardID), map[string]any{
		"front": "Updated Q",
		"back":  "Updated A",
	}, token)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", w.Code, w.Body)
	}
	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["front"] != "Updated Q" {
		t.Errorf("front = %v, want 'Updated Q'", resp["front"])
	}
}

func TestUpdateCard_OtherUserCard(t *testing.T) {
	r, _ := setupCardTestRouter(t)
	tokenA := registerAndLoginCard(t, r, "a@example.com", "pass")
	tokenB := registerAndLoginCard(t, r, "b@example.com", "pass")
	deckID := createDeckCard(t, r, tokenA, "A's Deck")
	cardID := createCard(t, r, tokenA, deckID, "Q", "A")

	w := putJSONWithToken(r, fmt.Sprintf("/api/v1/cards/%.0f", cardID), map[string]any{
		"front": "Hijacked",
		"back":  "X",
	}, tokenB)
	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", w.Code)
	}
}

func TestUpdateCard_MissingFields(t *testing.T) {
	r, _ := setupCardTestRouter(t)
	token := registerAndLoginCard(t, r, "alice@example.com", "pass")
	deckID := createDeckCard(t, r, token, "Deck")
	cardID := createCard(t, r, token, deckID, "Q", "A")

	w := putJSONWithToken(r, fmt.Sprintf("/api/v1/cards/%.0f", cardID), map[string]any{
		"front": "only front",
	}, token)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}

func TestUpdateCard_Unauthenticated(t *testing.T) {
	r, _ := setupCardTestRouter(t)
	w := putJSONWithToken(r, "/api/v1/cards/1", map[string]any{"front": "Q", "back": "A"}, "")
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", w.Code)
	}
}

// --- card_type in responses ---

func TestCardResponse_HasCardTypeField(t *testing.T) {
	r, _ := setupCardTestRouter(t)
	token := registerAndLoginCard(t, r, "alice@example.com", "pass")
	deckID := createDeckCard(t, r, token, "Deck")
	cardID := createCard(t, r, token, deckID, "Q", "A")

	w := getWithToken(r, fmt.Sprintf("/api/v1/cards/%.0f", cardID), token)
	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp)

	if resp["card_type"] != "basic" {
		t.Errorf("card_type = %v, want 'basic'", resp["card_type"])
	}
	if _, ok := resp["cloze_index"]; !ok {
		t.Error("expected cloze_index in response")
	}
	if _, ok := resp["extra"]; !ok {
		t.Error("expected extra in response")
	}
}

// --- cloze card creation ---

func TestCreateClozeCard_Success(t *testing.T) {
	r, _ := setupCardTestRouter(t)
	token := registerAndLoginCard(t, r, "alice@example.com", "pass")
	deckID := createDeckCard(t, r, token, "Deck")

	w := postJSONWithToken(r, fmt.Sprintf("/api/v1/decks/%.0f/cards", deckID), map[string]any{
		"front":       "{{c1::Paris}} is the capital of France",
		"card_type":   "cloze",
		"cloze_index": 1,
		"extra":       "Geography",
	}, token)

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201; body: %s", w.Code, w.Body)
	}
	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["card_type"] != "cloze" {
		t.Errorf("card_type = %v, want 'cloze'", resp["card_type"])
	}
	if resp["cloze_index"] != float64(1) {
		t.Errorf("cloze_index = %v, want 1", resp["cloze_index"])
	}
	if resp["extra"] != "Geography" {
		t.Errorf("extra = %v, want 'Geography'", resp["extra"])
	}
}

func TestCreateClozeCard_InvalidClozeText(t *testing.T) {
	r, _ := setupCardTestRouter(t)
	token := registerAndLoginCard(t, r, "alice@example.com", "pass")
	deckID := createDeckCard(t, r, token, "Deck")

	// Front has no cloze markers
	w := postJSONWithToken(r, fmt.Sprintf("/api/v1/decks/%.0f/cards", deckID), map[string]any{
		"front":       "plain text, no cloze",
		"card_type":   "cloze",
		"cloze_index": 1,
	}, token)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}

func TestCreateClozeCard_MismatchedIndex(t *testing.T) {
	r, _ := setupCardTestRouter(t)
	token := registerAndLoginCard(t, r, "alice@example.com", "pass")
	deckID := createDeckCard(t, r, token, "Deck")

	// Text has c1 but cloze_index says 2
	w := postJSONWithToken(r, fmt.Sprintf("/api/v1/decks/%.0f/cards", deckID), map[string]any{
		"front":       "{{c1::Paris}} is the capital",
		"card_type":   "cloze",
		"cloze_index": 2,
	}, token)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}

func TestCreateClozeCard_InvalidCardType(t *testing.T) {
	r, _ := setupCardTestRouter(t)
	token := registerAndLoginCard(t, r, "alice@example.com", "pass")
	deckID := createDeckCard(t, r, token, "Deck")

	w := postJSONWithToken(r, fmt.Sprintf("/api/v1/decks/%.0f/cards", deckID), map[string]any{
		"front":     "Q",
		"back":      "A",
		"card_type": "unknown_type",
	}, token)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}

// --- DELETE /api/v1/cards/{id} ---

func TestDeleteCard_Success(t *testing.T) {
	r, _ := setupCardTestRouter(t)
	token := registerAndLoginCard(t, r, "alice@example.com", "pass")
	deckID := createDeckCard(t, r, token, "Deck")
	cardID := createCard(t, r, token, deckID, "Q", "A")

	w := deleteWithToken(r, fmt.Sprintf("/api/v1/cards/%.0f", cardID), token)
	if w.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204; body: %s", w.Code, w.Body)
	}

	w = getWithToken(r, fmt.Sprintf("/api/v1/cards/%.0f", cardID), token)
	if w.Code != http.StatusNotFound {
		t.Errorf("after delete: status = %d, want 404", w.Code)
	}
}

func TestDeleteCard_OtherUserCard(t *testing.T) {
	r, _ := setupCardTestRouter(t)
	tokenA := registerAndLoginCard(t, r, "a@example.com", "pass")
	tokenB := registerAndLoginCard(t, r, "b@example.com", "pass")
	deckID := createDeckCard(t, r, tokenA, "A's Deck")
	cardID := createCard(t, r, tokenA, deckID, "Q", "A")

	w := deleteWithToken(r, fmt.Sprintf("/api/v1/cards/%.0f", cardID), tokenB)
	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", w.Code)
	}
}

func TestDeleteCard_NotFound(t *testing.T) {
	r, _ := setupCardTestRouter(t)
	token := registerAndLoginCard(t, r, "alice@example.com", "pass")

	w := deleteWithToken(r, "/api/v1/cards/99999", token)
	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", w.Code)
	}
}

func TestDeleteCard_Unauthenticated(t *testing.T) {
	r, _ := setupCardTestRouter(t)
	w := deleteWithToken(r, "/api/v1/cards/1", "")
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", w.Code)
	}
}
