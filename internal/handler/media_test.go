package handler_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
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
	"kpp.dev/kpfc/internal/storage"
)

func setupMediaTestRouter(t *testing.T) (*chi.Mux, *service.AuthService) {
	t.Helper()
	dsn := "file:" + t.Name() + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: logger.Discard})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.Media{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	userRepo := repository.NewGORMUserRepository(db)
	mediaRepo := repository.NewGORMMediaRepository(db)
	authSvc := service.NewAuthService(userRepo, "test-secret")

	store := storage.NewLocalStorage(t.TempDir())
	n := 0
	idGen := func() string { n++; return fmt.Sprintf("file%d", n) }
	mediaSvc := service.NewMediaService(mediaRepo, store, idGen)

	authHandler := handler.NewAuthHandler(authSvc)
	mediaHandler := handler.NewMediaHandler(mediaSvc)

	r := chi.NewRouter()
	r.Post("/api/v1/auth/register", authHandler.Register)
	r.Post("/api/v1/auth/login", authHandler.Login)
	r.Get("/api/v1/media/{id}", mediaHandler.Serve)
	r.Group(func(r chi.Router) {
		r.Use(middleware.Auth(authSvc))
		r.Post("/api/v1/media", mediaHandler.Upload)
		r.Delete("/api/v1/media/{id}", mediaHandler.Delete)
	})
	return r, authSvc
}

// loginMedia registers + logs in a test user and returns a Bearer token.
func loginMedia(t *testing.T, r *chi.Mux, email string) string {
	t.Helper()
	body := fmt.Sprintf(`{"email":%q,"password":"pass1234","display_name":"Test"}`, email)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", strings.NewReader(body)))
	if rec.Code != http.StatusCreated {
		t.Fatalf("register: got %d", rec.Code)
	}
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(body)))
	if rec.Code != http.StatusOK {
		t.Fatalf("login: got %d", rec.Code)
	}
	var resp map[string]any
	json.NewDecoder(rec.Body).Decode(&resp)
	return resp["token"].(string)
}

// buildMultipart creates a multipart/form-data body with a single "file" field.
func buildMultipart(t *testing.T, filename, contentType string, content []byte) (io.Reader, string) {
	t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	h := make(map[string][]string)
	h["Content-Disposition"] = []string{fmt.Sprintf(`form-data; name="file"; filename="%s"`, filename)}
	h["Content-Type"] = []string{contentType}
	pw, err := mw.CreatePart(h)
	if err != nil {
		t.Fatalf("create part: %v", err)
	}
	pw.Write(content)
	mw.Close()
	return &buf, mw.FormDataContentType()
}

// minPNG is the smallest valid PNG (1×1 transparent pixel).
var minPNG = []byte{
	0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a, // PNG signature
	0x00, 0x00, 0x00, 0x0d, 0x49, 0x48, 0x44, 0x52, // IHDR chunk length + type
	0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01, // width=1, height=1
	0x08, 0x02, 0x00, 0x00, 0x00, 0x90, 0x77, 0x53, // bit depth, color type, ...
	0xde, 0x00, 0x00, 0x00, 0x0c, 0x49, 0x44, 0x41, // IDAT chunk
	0x54, 0x08, 0xd7, 0x63, 0xf8, 0xcf, 0xc0, 0x00, // IDAT data
	0x00, 0x00, 0x02, 0x00, 0x01, 0xe2, 0x21, 0xbc,
	0x33, 0x00, 0x00, 0x00, 0x00, 0x49, 0x45, 0x4e, // IEND chunk
	0x44, 0xae, 0x42, 0x60, 0x82,
}

func TestMedia_UploadAndServe(t *testing.T) {
	r, _ := setupMediaTestRouter(t)
	token := loginMedia(t, r, "media1@test.com")

	body, ct := buildMultipart(t, "test.png", "image/png", minPNG)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/media", body)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", ct)

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("upload: got %d body=%s", rec.Code, rec.Body)
	}

	var resp map[string]any
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp["id"] == nil {
		t.Fatal("upload response missing id")
	}
	if resp["url"] == nil {
		t.Fatal("upload response missing url")
	}
	if resp["content_type"] == nil {
		t.Fatal("upload response missing content_type")
	}

	// Serve the file (no auth)
	id := int(resp["id"].(float64))
	rec2 := httptest.NewRecorder()
	r.ServeHTTP(rec2, httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/media/%d", id), nil))
	if rec2.Code != http.StatusOK {
		t.Fatalf("serve: got %d", rec2.Code)
	}
	if got := rec2.Header().Get("Content-Type"); !strings.HasPrefix(got, "image/png") {
		t.Errorf("Content-Type: got %q", got)
	}
}

func TestMedia_Upload_InvalidType(t *testing.T) {
	r, _ := setupMediaTestRouter(t)
	token := loginMedia(t, r, "media2@test.com")

	body, ct := buildMultipart(t, "file.exe", "application/octet-stream", []byte("MZ fake exe"))
	req := httptest.NewRequest(http.MethodPost, "/api/v1/media", body)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", ct)

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", rec.Code, rec.Body)
	}
}

func TestMedia_Upload_Unauthenticated(t *testing.T) {
	r, _ := setupMediaTestRouter(t)
	body, ct := buildMultipart(t, "test.png", "image/png", minPNG)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/media", body)
	req.Header.Set("Content-Type", ct)

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestMedia_Serve_NotFound(t *testing.T) {
	r, _ := setupMediaTestRouter(t)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/media/999", nil))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestMedia_Delete(t *testing.T) {
	r, _ := setupMediaTestRouter(t)
	token := loginMedia(t, r, "media3@test.com")

	// upload
	body, ct := buildMultipart(t, "del.png", "image/png", minPNG)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/media", body)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", ct)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("upload: got %d", rec.Code)
	}
	var resp map[string]any
	json.NewDecoder(rec.Body).Decode(&resp)
	id := int(resp["id"].(float64))

	// delete
	req2 := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/v1/media/%d", id), nil)
	req2.Header.Set("Authorization", "Bearer "+token)
	rec2 := httptest.NewRecorder()
	r.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusNoContent {
		t.Fatalf("delete: got %d", rec2.Code)
	}

	// serve after delete should 404 (storage gone) or 500 (record gone)
	rec3 := httptest.NewRecorder()
	r.ServeHTTP(rec3, httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/media/%d", id), nil))
	if rec3.Code == http.StatusOK {
		t.Fatal("file should not be accessible after delete")
	}
}

func TestMedia_Delete_Forbidden(t *testing.T) {
	r, _ := setupMediaTestRouter(t)
	token1 := loginMedia(t, r, "media4a@test.com")
	token2 := loginMedia(t, r, "media4b@test.com")

	// user1 uploads
	body, ct := buildMultipart(t, "img.png", "image/png", minPNG)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/media", body)
	req.Header.Set("Authorization", "Bearer "+token1)
	req.Header.Set("Content-Type", ct)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	var resp map[string]any
	json.NewDecoder(rec.Body).Decode(&resp)
	id := int(resp["id"].(float64))

	// user2 tries to delete
	req2 := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/v1/media/%d", id), nil)
	req2.Header.Set("Authorization", "Bearer "+token2)
	rec2 := httptest.NewRecorder()
	r.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rec2.Code)
	}
}
