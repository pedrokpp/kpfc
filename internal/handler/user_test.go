package handler_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
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

func setupUserTestRouter(t *testing.T) (*chi.Mux, *service.AuthService) {
	t.Helper()
	dsn := "file:" + t.Name() + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Discard,
	})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	userRepo := repository.NewGORMUserRepository(db)
	authSvc := service.NewAuthService(userRepo, "test-secret")
	userSvc := service.NewUserService(userRepo)
	authHandler := handler.NewAuthHandler(authSvc)
	userHandler := handler.NewUserHandler(userSvc)

	r := chi.NewRouter()
	r.Post("/api/v1/auth/register", authHandler.Register)
	r.Post("/api/v1/auth/login", authHandler.Login)
	r.Group(func(r chi.Router) {
		r.Use(middleware.Auth(authSvc))
		r.Get("/api/v1/users/me", userHandler.GetMe)
		r.Put("/api/v1/users/me", userHandler.UpdateMe)
	})
	return r, authSvc
}

// registerAndLogin is a helper that registers a user and returns the JWT token.
func registerAndLogin(t *testing.T, r *chi.Mux, email, password string) string {
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

func getWithToken(r *chi.Mux, path, token string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, path, nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func putJSONWithToken(r *chi.Mux, path string, body any, token string) *httptest.ResponseRecorder {
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPut, path, strings.NewReader(string(b)))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// --- GET /api/v1/users/me ---

func TestGetMe_Authenticated(t *testing.T) {
	r, _ := setupUserTestRouter(t)
	token := registerAndLogin(t, r, "alice@example.com", "pass123")

	w := getWithToken(r, "/api/v1/users/me", token)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", w.Code, w.Body)
	}

	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp)

	if resp["email"] != "alice@example.com" {
		t.Errorf("email = %v, want alice@example.com", resp["email"])
	}
	if _, ok := resp["password_hash"]; ok {
		t.Error("response must not include password_hash")
	}
	if _, ok := resp["login_streak"]; !ok {
		t.Error("expected login_streak in response")
	}
	if _, ok := resp["total_points"]; !ok {
		t.Error("expected total_points in response")
	}
}

func TestGetMe_Unauthenticated(t *testing.T) {
	r, _ := setupUserTestRouter(t)
	w := getWithToken(r, "/api/v1/users/me", "")
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", w.Code)
	}
}

func TestGetMe_InvalidToken(t *testing.T) {
	r, _ := setupUserTestRouter(t)
	w := getWithToken(r, "/api/v1/users/me", "not.a.valid.token")
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", w.Code)
	}
}

// --- PUT /api/v1/users/me ---

func TestUpdateMe_Authenticated(t *testing.T) {
	r, _ := setupUserTestRouter(t)
	token := registerAndLogin(t, r, "bob@example.com", "pass123")

	w := putJSONWithToken(r, "/api/v1/users/me", map[string]string{
		"display_name": "Bobby",
	}, token)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", w.Code, w.Body)
	}

	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["display_name"] != "Bobby" {
		t.Errorf("display_name = %v, want Bobby", resp["display_name"])
	}
}

func TestUpdateMe_Unauthenticated(t *testing.T) {
	r, _ := setupUserTestRouter(t)
	w := putJSONWithToken(r, "/api/v1/users/me", map[string]string{"display_name": "X"}, "")
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", w.Code)
	}
}

func TestUpdateMe_PersistedAcrossGet(t *testing.T) {
	r, _ := setupUserTestRouter(t)
	token := registerAndLogin(t, r, "carol@example.com", "pass123")

	putJSONWithToken(r, "/api/v1/users/me", map[string]string{"display_name": "Carol Updated"}, token)

	w := getWithToken(r, "/api/v1/users/me", token)
	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["display_name"] != "Carol Updated" {
		t.Errorf("display_name = %v, want Carol Updated", resp["display_name"])
	}
}
