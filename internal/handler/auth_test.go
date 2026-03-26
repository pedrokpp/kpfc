package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"kpp.dev/kpfc/internal/handler"
	"kpp.dev/kpfc/internal/model"
	"kpp.dev/kpfc/internal/repository"
	"kpp.dev/kpfc/internal/service"
)

func setupTestRouter(t *testing.T) *chi.Mux {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{
		Logger: logger.Discard,
	})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	userRepo := repository.NewGORMUserRepository(db)
	authSvc := service.NewAuthService(userRepo, "test-jwt-secret")
	authHandler := handler.NewAuthHandler(authSvc)

	r := chi.NewRouter()
	r.Post("/api/v1/auth/register", authHandler.Register)
	r.Post("/api/v1/auth/login", authHandler.Login)
	return r
}

func postJSON(r *chi.Mux, path string, body any) *httptest.ResponseRecorder {
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// --- Register ---

func TestRegister_Success(t *testing.T) {
	r := setupTestRouter(t)
	w := postJSON(r, "/api/v1/auth/register", map[string]string{
		"email":        "alice@example.com",
		"password":     "secret123",
		"display_name": "Alice",
	})

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201; body: %s", w.Code, w.Body)
	}

	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["email"] != "alice@example.com" {
		t.Errorf("email = %v, want alice@example.com", resp["email"])
	}
	if resp["display_name"] != "Alice" {
		t.Errorf("display_name = %v, want Alice", resp["display_name"])
	}
	if _, ok := resp["password_hash"]; ok {
		t.Error("response must not include password_hash")
	}
}

func TestRegister_DuplicateEmail(t *testing.T) {
	r := setupTestRouter(t)
	payload := map[string]string{
		"email":    "bob@example.com",
		"password": "pass",
	}
	postJSON(r, "/api/v1/auth/register", payload)
	w := postJSON(r, "/api/v1/auth/register", payload)

	if w.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409", w.Code)
	}
}

func TestRegister_MissingFields(t *testing.T) {
	r := setupTestRouter(t)
	w := postJSON(r, "/api/v1/auth/register", map[string]string{"email": "x@x.com"})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}

// --- Login ---

func TestLogin_Success(t *testing.T) {
	r := setupTestRouter(t)
	postJSON(r, "/api/v1/auth/register", map[string]string{
		"email":        "carol@example.com",
		"password":     "mypass",
		"display_name": "Carol",
	})

	w := postJSON(r, "/api/v1/auth/login", map[string]string{
		"email":    "carol@example.com",
		"password": "mypass",
	})

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", w.Code, w.Body)
	}

	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp)

	if _, ok := resp["token"].(string); !ok || resp["token"] == "" {
		t.Error("expected non-empty token in response")
	}

	user, ok := resp["user"].(map[string]any)
	if !ok {
		t.Fatalf("expected user object in response")
	}
	if user["email"] != "carol@example.com" {
		t.Errorf("user.email = %v, want carol@example.com", user["email"])
	}
	if _, ok := user["login_streak"]; !ok {
		t.Error("expected login_streak in user object")
	}
	if _, ok := user["total_points"]; !ok {
		t.Error("expected total_points in user object")
	}
}

func TestLogin_WrongPassword(t *testing.T) {
	r := setupTestRouter(t)
	postJSON(r, "/api/v1/auth/register", map[string]string{
		"email":    "dave@example.com",
		"password": "correct",
	})

	w := postJSON(r, "/api/v1/auth/login", map[string]string{
		"email":    "dave@example.com",
		"password": "wrong",
	})

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", w.Code)
	}
}

func TestLogin_UnknownEmail(t *testing.T) {
	r := setupTestRouter(t)
	w := postJSON(r, "/api/v1/auth/login", map[string]string{
		"email":    "nobody@example.com",
		"password": "pass",
	})
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", w.Code)
	}
}

func TestLogin_StreakIncrement(t *testing.T) {
	r := setupTestRouter(t)
	postJSON(r, "/api/v1/auth/register", map[string]string{
		"email":    "eve@example.com",
		"password": "pass",
	})

	w := postJSON(r, "/api/v1/auth/login", map[string]string{
		"email":    "eve@example.com",
		"password": "pass",
	})

	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp)
	user := resp["user"].(map[string]any)

	// First login — streak should be 1
	if user["login_streak"].(float64) != 1 {
		t.Errorf("login_streak = %v, want 1 on first login", user["login_streak"])
	}
}
