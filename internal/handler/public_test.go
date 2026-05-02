package handler_test

import (
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

func setupPublicTestRouter(t *testing.T) *chi.Mux {
	t.Helper()
	dsn := "file:" + t.Name() + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Discard,
	})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.Deck{}, &model.UserDeckUpvote{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	userRepo := repository.NewGORMUserRepository(db)
	deckRepo := repository.NewGORMDeckRepository(db)
	accessSvc := service.NewAccessService(deckRepo, nil, nil)
	authSvc := service.NewAuthService(userRepo, "test-secret")
	deckSvc := service.NewDeckService(deckRepo, accessSvc)
	authHandler := handler.NewAuthHandler(authSvc)
	deckHandler := handler.NewDeckHandler(deckSvc)
	publicHandler := handler.NewPublicHandler(deckSvc)

	r := chi.NewRouter()
	r.Post("/api/v1/auth/register", authHandler.Register)
	r.Post("/api/v1/auth/login", authHandler.Login)
	r.Get("/api/v1/public/decks", publicHandler.ListDecks)
	r.Group(func(r chi.Router) {
		r.Use(middleware.Auth(authSvc))
		r.Post("/api/v1/decks", deckHandler.Create)
		r.Post("/api/v1/decks/{id}/upvote", deckHandler.Upvote)
	})
	return r
}

func registerAndLoginPublic(t *testing.T, r *chi.Mux, email, password string) string {
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

func createPublicDeck(t *testing.T, r *chi.Mux, token, title string) float64 {
	t.Helper()
	w := postJSONWithToken(r, "/api/v1/decks", map[string]any{
		"title":     title,
		"is_public": true,
	}, token)
	if w.Code != http.StatusCreated {
		t.Fatalf("createPublicDeck: status = %d, want 201; body: %s", w.Code, w.Body)
	}
	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp)
	return resp["id"].(float64)
}

func upvoteDeck(r *chi.Mux, deckID float64, token string) *httptest.ResponseRecorder {
	path := fmt.Sprintf("/api/v1/decks/%d/upvote", int(deckID))
	return postJSONWithToken(r, path, nil, token)
}

// --- GET /api/v1/public/decks ---

func TestListPublicDecks_Empty(t *testing.T) {
	r := setupPublicTestRouter(t)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/public/decks", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	var resp []any
	json.NewDecoder(w.Body).Decode(&resp)
	if len(resp) != 0 {
		t.Errorf("expected 0 decks, got %d", len(resp))
	}
}

func TestListPublicDecks_NoAuthRequired(t *testing.T) {
	r := setupPublicTestRouter(t)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/public/decks", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("unauthenticated listing: status = %d, want 200", w.Code)
	}
}

func TestListPublicDecks_OnlyPublicVisible(t *testing.T) {
	r := setupPublicTestRouter(t)
	token := registerAndLoginPublic(t, r, "alice@example.com", "pass")

	createPublicDeck(t, r, token, "Public Deck")
	postJSONWithToken(r, "/api/v1/decks", map[string]any{
		"title":     "Private Deck",
		"is_public": false,
	}, token)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/public/decks", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	var resp []any
	json.NewDecoder(w.Body).Decode(&resp)
	if len(resp) != 1 {
		t.Errorf("expected 1 public deck, got %d", len(resp))
	}
}

func TestListPublicDecks_SortedByUpvotes(t *testing.T) {
	r := setupPublicTestRouter(t)
	alice := registerAndLoginPublic(t, r, "alice@example.com", "pass")
	bob := registerAndLoginPublic(t, r, "bob@example.com", "pass")

	lowID := createPublicDeck(t, r, bob, "Low Votes")
	highID := createPublicDeck(t, r, bob, "High Votes")

	// Alice upvotes only "High Votes"
	upvoteDeck(r, highID, alice)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/public/decks?sort=upvotes", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	var resp []map[string]any
	json.NewDecoder(w.Body).Decode(&resp)
	if len(resp) != 2 {
		t.Fatalf("expected 2 decks, got %d", len(resp))
	}
	if resp[0]["id"].(float64) != highID {
		t.Errorf("first deck (id=%v) should be highID=%v (lowID=%v)", resp[0]["id"], highID, lowID)
	}
}

// --- POST /api/v1/decks/{id}/upvote ---

func TestUpvote_AddAndRemove(t *testing.T) {
	r := setupPublicTestRouter(t)
	alice := registerAndLoginPublic(t, r, "alice@example.com", "pass")
	bob := registerAndLoginPublic(t, r, "bob@example.com", "pass")

	deckID := createPublicDeck(t, r, bob, "Bob's Deck")

	// Alice adds upvote
	w := upvoteDeck(r, deckID, alice)
	if w.Code != http.StatusNoContent {
		t.Fatalf("add upvote: status = %d, want 204; body: %s", w.Code, w.Body)
	}

	// Alice removes upvote
	w = upvoteDeck(r, deckID, alice)
	if w.Code != http.StatusNoContent {
		t.Fatalf("remove upvote: status = %d, want 204; body: %s", w.Code, w.Body)
	}
}

func TestUpvote_CannotUpvoteOwnDeck(t *testing.T) {
	r := setupPublicTestRouter(t)
	alice := registerAndLoginPublic(t, r, "alice@example.com", "pass")

	deckID := createPublicDeck(t, r, alice, "Alice's Deck")

	w := upvoteDeck(r, deckID, alice)
	if w.Code != http.StatusForbidden {
		t.Fatalf("own deck upvote: status = %d, want 403; body: %s", w.Code, w.Body)
	}
}

func TestUpvote_CannotUpvotePrivateDeck(t *testing.T) {
	r := setupPublicTestRouter(t)
	alice := registerAndLoginPublic(t, r, "alice@example.com", "pass")
	bob := registerAndLoginPublic(t, r, "bob@example.com", "pass")

	w := postJSONWithToken(r, "/api/v1/decks", map[string]any{
		"title":     "Private",
		"is_public": false,
	}, bob)
	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp)
	deckID := resp["id"].(float64)

	w = upvoteDeck(r, deckID, alice)
	if w.Code != http.StatusForbidden {
		t.Fatalf("private deck upvote: status = %d, want 403; body: %s", w.Code, w.Body)
	}
}

func TestUpvote_Unauthenticated(t *testing.T) {
	r := setupPublicTestRouter(t)
	token := registerAndLoginPublic(t, r, "alice@example.com", "pass")
	deckID := createPublicDeck(t, r, token, "Public Deck")

	w := upvoteDeck(r, deckID, "")
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated upvote: status = %d, want 401", w.Code)
	}
}

func TestUpvote_CountAccuracy(t *testing.T) {
	r := setupPublicTestRouter(t)
	alice := registerAndLoginPublic(t, r, "alice@example.com", "pass")
	bob := registerAndLoginPublic(t, r, "bob@example.com", "pass")
	carol := registerAndLoginPublic(t, r, "carol@example.com", "pass")

	deckID := createPublicDeck(t, r, alice, "Shared Deck")

	upvoteDeck(r, deckID, bob)   // count → 1
	upvoteDeck(r, deckID, carol) // count → 2
	upvoteDeck(r, deckID, bob)   // bob removes → count → 1

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/public/decks", nil)
	r.ServeHTTP(w, req)

	var resp []map[string]any
	json.NewDecoder(w.Body).Decode(&resp)
	if len(resp) == 0 {
		t.Fatal("expected at least 1 public deck")
	}
	upvoteCount := resp[0]["upvote_count"].(float64)
	if upvoteCount != 1 {
		t.Errorf("upvote_count = %v, want 1", upvoteCount)
	}
}
