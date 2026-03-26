package handler_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
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

func setupDeckTestRouter(t *testing.T) (*chi.Mux, *service.AuthService) {
	t.Helper()
	dsn := "file:" + t.Name() + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Discard,
	})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.Deck{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	userRepo := repository.NewGORMUserRepository(db)
	deckRepo := repository.NewGORMDeckRepository(db)
	authSvc := service.NewAuthService(userRepo, "test-secret")
	deckSvc := service.NewDeckService(deckRepo)
	authHandler := handler.NewAuthHandler(authSvc)
	deckHandler := handler.NewDeckHandler(deckSvc)

	r := chi.NewRouter()
	r.Post("/api/v1/auth/register", authHandler.Register)
	r.Post("/api/v1/auth/login", authHandler.Login)
	r.Group(func(r chi.Router) {
		r.Use(middleware.Auth(authSvc))
		r.Get("/api/v1/decks", deckHandler.List)
		r.Post("/api/v1/decks", deckHandler.Create)
		r.Get("/api/v1/decks/{id}", deckHandler.Get)
		r.Put("/api/v1/decks/{id}", deckHandler.Update)
		r.Delete("/api/v1/decks/{id}", deckHandler.Delete)
	})
	return r, authSvc
}

func deleteWithToken(r *chi.Mux, path, token string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodDelete, path, nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// registerAndLoginDeck is a local version that works with any router.
func registerAndLoginDeck(t *testing.T, r *chi.Mux, email, password string) string {
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
	token, _ := resp["token"].(string)
	return token
}

// createDeck is a helper that POSTs a new deck and returns its ID.
func createDeck(t *testing.T, r *chi.Mux, token, title string) float64 {
	t.Helper()
	w := postJSONWithToken(r, "/api/v1/decks", map[string]any{
		"title":    title,
		"is_public": false,
	}, token)
	if w.Code != http.StatusCreated {
		t.Fatalf("createDeck: status = %d, want 201; body: %s", w.Code, w.Body)
	}
	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp)
	return resp["id"].(float64)
}

func postJSONWithToken(r *chi.Mux, path string, body any, token string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	r.ServeHTTP(w, req)
	return w
}

// --- GET /api/v1/decks ---

func TestListDecks_Authenticated(t *testing.T) {
	r, _ := setupDeckTestRouter(t)
	token := registerAndLoginDeck(t, r, "alice@example.com", "pass")

	postJSONWithToken(r, "/api/v1/decks", map[string]any{"title": "Deck A"}, token)
	postJSONWithToken(r, "/api/v1/decks", map[string]any{"title": "Deck B"}, token)

	w := getWithToken(r, "/api/v1/decks", token)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", w.Code, w.Body)
	}

	var resp []any
	json.NewDecoder(w.Body).Decode(&resp)
	if len(resp) != 2 {
		t.Errorf("len(decks) = %d, want 2", len(resp))
	}
}

func TestListDecks_Unauthenticated(t *testing.T) {
	r, _ := setupDeckTestRouter(t)
	w := getWithToken(r, "/api/v1/decks", "")
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", w.Code)
	}
}

func TestListDecks_OnlyOwnDecks(t *testing.T) {
	r, _ := setupDeckTestRouter(t)
	tokenA := registerAndLoginDeck(t, r, "a@example.com", "pass")
	tokenB := registerAndLoginDeck(t, r, "b@example.com", "pass")

	postJSONWithToken(r, "/api/v1/decks", map[string]any{"title": "A's Deck"}, tokenA)
	postJSONWithToken(r, "/api/v1/decks", map[string]any{"title": "B's Deck"}, tokenB)

	w := getWithToken(r, "/api/v1/decks", tokenA)
	var resp []any
	json.NewDecoder(w.Body).Decode(&resp)
	if len(resp) != 1 {
		t.Errorf("user A should only see 1 deck, got %d", len(resp))
	}
}

// --- POST /api/v1/decks ---

func TestCreateDeck_Success(t *testing.T) {
	r, _ := setupDeckTestRouter(t)
	token := registerAndLoginDeck(t, r, "alice@example.com", "pass")

	w := postJSONWithToken(r, "/api/v1/decks", map[string]any{
		"title":       "Go Fundamentals",
		"description": "Basic Go concepts",
		"is_public":   true,
	}, token)

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201; body: %s", w.Code, w.Body)
	}

	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["title"] != "Go Fundamentals" {
		t.Errorf("title = %v, want 'Go Fundamentals'", resp["title"])
	}
	if resp["is_public"] != true {
		t.Errorf("is_public = %v, want true", resp["is_public"])
	}
	if resp["id"] == nil {
		t.Error("expected id in response")
	}
}

func TestCreateDeck_MissingTitle(t *testing.T) {
	r, _ := setupDeckTestRouter(t)
	token := registerAndLoginDeck(t, r, "alice@example.com", "pass")

	w := postJSONWithToken(r, "/api/v1/decks", map[string]any{"description": "no title"}, token)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}

func TestCreateDeck_Unauthenticated(t *testing.T) {
	r, _ := setupDeckTestRouter(t)
	w := postJSONWithToken(r, "/api/v1/decks", map[string]any{"title": "X"}, "")
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", w.Code)
	}
}

// --- GET /api/v1/decks/{id} ---

func TestGetDeck_OwnDeck(t *testing.T) {
	r, _ := setupDeckTestRouter(t)
	token := registerAndLoginDeck(t, r, "alice@example.com", "pass")
	id := createDeck(t, r, token, "My Deck")

	w := getWithToken(r, fmt.Sprintf("/api/v1/decks/%.0f", id), token)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", w.Code, w.Body)
	}

	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["title"] != "My Deck" {
		t.Errorf("title = %v, want 'My Deck'", resp["title"])
	}
}

func TestGetDeck_OtherUserDeck(t *testing.T) {
	r, _ := setupDeckTestRouter(t)
	tokenA := registerAndLoginDeck(t, r, "a@example.com", "pass")
	tokenB := registerAndLoginDeck(t, r, "b@example.com", "pass")
	id := createDeck(t, r, tokenA, "A's Deck")

	w := getWithToken(r, fmt.Sprintf("/api/v1/decks/%.0f", id), tokenB)
	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", w.Code)
	}
}

func TestGetDeck_NotFound(t *testing.T) {
	r, _ := setupDeckTestRouter(t)
	token := registerAndLoginDeck(t, r, "alice@example.com", "pass")

	w := getWithToken(r, "/api/v1/decks/99999", token)
	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", w.Code)
	}
}

func TestGetDeck_Unauthenticated(t *testing.T) {
	r, _ := setupDeckTestRouter(t)
	w := getWithToken(r, "/api/v1/decks/1", "")
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", w.Code)
	}
}

// --- PUT /api/v1/decks/{id} ---

func TestUpdateDeck_OwnDeck(t *testing.T) {
	r, _ := setupDeckTestRouter(t)
	token := registerAndLoginDeck(t, r, "alice@example.com", "pass")
	id := createDeck(t, r, token, "Original")

	w := putJSONWithToken(r, fmt.Sprintf("/api/v1/decks/%.0f", id), map[string]any{
		"title":       "Updated",
		"description": "New desc",
		"is_public":   true,
	}, token)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", w.Code, w.Body)
	}

	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["title"] != "Updated" {
		t.Errorf("title = %v, want 'Updated'", resp["title"])
	}
	if resp["is_public"] != true {
		t.Errorf("is_public = %v, want true", resp["is_public"])
	}
}

func TestUpdateDeck_OtherUserDeck(t *testing.T) {
	r, _ := setupDeckTestRouter(t)
	tokenA := registerAndLoginDeck(t, r, "a@example.com", "pass")
	tokenB := registerAndLoginDeck(t, r, "b@example.com", "pass")
	id := createDeck(t, r, tokenA, "A's Deck")

	w := putJSONWithToken(r, fmt.Sprintf("/api/v1/decks/%.0f", id), map[string]any{
		"title": "Hijacked",
	}, tokenB)
	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", w.Code)
	}
}

func TestUpdateDeck_Unauthenticated(t *testing.T) {
	r, _ := setupDeckTestRouter(t)
	w := putJSONWithToken(r, "/api/v1/decks/1", map[string]any{"title": "X"}, "")
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", w.Code)
	}
}

// --- DELETE /api/v1/decks/{id} ---

func TestDeleteDeck_OwnDeck(t *testing.T) {
	r, _ := setupDeckTestRouter(t)
	token := registerAndLoginDeck(t, r, "alice@example.com", "pass")
	id := createDeck(t, r, token, "To Delete")

	w := deleteWithToken(r, fmt.Sprintf("/api/v1/decks/%.0f", id), token)
	if w.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204; body: %s", w.Code, w.Body)
	}

	// Confirm it's gone
	w = getWithToken(r, fmt.Sprintf("/api/v1/decks/%.0f", id), token)
	if w.Code != http.StatusNotFound {
		t.Errorf("after delete: status = %d, want 404", w.Code)
	}
}

func TestDeleteDeck_OtherUserDeck(t *testing.T) {
	r, _ := setupDeckTestRouter(t)
	tokenA := registerAndLoginDeck(t, r, "a@example.com", "pass")
	tokenB := registerAndLoginDeck(t, r, "b@example.com", "pass")
	id := createDeck(t, r, tokenA, "A's Deck")

	w := deleteWithToken(r, fmt.Sprintf("/api/v1/decks/%.0f", id), tokenB)
	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", w.Code)
	}
}

func TestDeleteDeck_NotFound(t *testing.T) {
	r, _ := setupDeckTestRouter(t)
	token := registerAndLoginDeck(t, r, "alice@example.com", "pass")

	w := deleteWithToken(r, "/api/v1/decks/99999", token)
	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", w.Code)
	}
}

func TestDeleteDeck_Unauthenticated(t *testing.T) {
	r, _ := setupDeckTestRouter(t)
	w := deleteWithToken(r, "/api/v1/decks/1", "")
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", w.Code)
	}
}
