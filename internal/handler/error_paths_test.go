package handler_test

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCreateDeck_InvalidJSON(t *testing.T) {
	r, _ := setupDeckTestRouter(t)
	token := registerAndLoginDeck(t, r, "json-deck@example.com", "pass")

	req := httptest.NewRequest(http.MethodPost, "/api/v1/decks", strings.NewReader("{"))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestGetDeck_InvalidID(t *testing.T) {
	r, _ := setupDeckTestRouter(t)
	token := registerAndLoginDeck(t, r, "bad-id-deck@example.com", "pass")

	w := getWithToken(r, "/api/v1/decks/not-a-number", token)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}

func TestCreateCard_InvalidJSON(t *testing.T) {
	r, _ := setupCardTestRouter(t)
	token := registerAndLoginCard(t, r, "json-card@example.com", "pass")
	deckID := createDeckCard(t, r, token, "Deck")

	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/decks/%.0f/cards", deckID), strings.NewReader("{"))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestStudyErrorPaths(t *testing.T) {
	r, _ := setupStudyTestRouter(t)
	token := registerAndLoginStudy(t, r, "study-errors@example.com", "pass")
	deckID := createDeckStudy(t, r, token, "Deck")
	cardID := createCardStudy(t, r, token, deckID, "Q", "A")

	tests := []struct {
		name   string
		method string
		path   string
		body   string
		want   int
	}{
		{name: "start session invalid deck id", method: http.MethodPost, path: "/api/v1/decks/nope/study", body: `{"mode":"random"}`, want: http.StatusBadRequest},
		{name: "start session invalid json", method: http.MethodPost, path: "/api/v1/decks/1/study", body: `{`, want: http.StatusBadRequest},
		{name: "submit review invalid card id", method: http.MethodPost, path: "/api/v1/cards/nope/review", body: `{"quality":4}`, want: http.StatusBadRequest},
		{name: "submit review invalid json", method: http.MethodPost, path: "/api/v1/cards/1/review", body: `{`, want: http.StatusBadRequest},
	}

	_ = cardID

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			path := tc.path
			path = strings.ReplaceAll(path, "/decks/1/", fmt.Sprintf("/decks/%.0f/", deckID))
			path = strings.ReplaceAll(path, "/cards/1", fmt.Sprintf("/cards/%.0f", cardID))
			req := httptest.NewRequest(tc.method, path, strings.NewReader(tc.body))
			req.Header.Set("Authorization", "Bearer "+token)
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, req)
			if rec.Code != tc.want {
				t.Fatalf("status = %d, want %d; body=%s", rec.Code, tc.want, rec.Body.String())
			}
		})
	}
}

func TestCardUpdateErrorPaths(t *testing.T) {
	r, _ := setupCardTestRouter(t)
	token := registerAndLoginCard(t, r, "card-update-errors@example.com", "pass")
	deckID := createDeckCard(t, r, token, "Deck")
	cardID := createCard(t, r, token, deckID, "Q", "A")

	tests := []struct {
		name   string
		path   string
		body   string
		want   int
	}{
		{name: "invalid id", path: "/api/v1/cards/nope", body: `{"front":"Q","back":"A"}`, want: http.StatusBadRequest},
		{name: "invalid json", path: "/api/v1/cards/1", body: `{`, want: http.StatusBadRequest},
		{name: "update cloze success", path: "/api/v1/cards/1", body: `{"front":"{{c1::Paris}}","card_type":"cloze","cloze_index":1,"extra":"geo"}`, want: http.StatusOK},
		{name: "update cloze invalid", path: "/api/v1/cards/1", body: `{"front":"plain","card_type":"cloze","cloze_index":1}`, want: http.StatusBadRequest},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			path := strings.ReplaceAll(tc.path, "/cards/1", fmt.Sprintf("/cards/%.0f", cardID))
			req := httptest.NewRequest(http.MethodPut, path, strings.NewReader(tc.body))
			req.Header.Set("Authorization", "Bearer "+token)
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, req)
			if rec.Code != tc.want {
				t.Fatalf("status = %d, want %d; body=%s", rec.Code, tc.want, rec.Body.String())
			}
		})
	}
}

func TestUpdateUser_InvalidJSON(t *testing.T) {
	r, _ := setupUserTestRouter(t)
	token := registerAndLogin(t, r, "json-user@example.com", "pass")

	req := httptest.NewRequest(http.MethodPut, "/api/v1/users/me", bytes.NewBufferString("{"))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestMediaUpload_InvalidMultipart(t *testing.T) {
	r, _ := setupMediaTestRouter(t)
	token := loginMedia(t, r, "bad-multipart@example.com")

	req := httptest.NewRequest(http.MethodPost, "/api/v1/media", strings.NewReader("not multipart"))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "multipart/form-data; boundary=broken")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}
