package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCORS_AllowsConfiguredOrigin(t *testing.T) {
	mw := CORS("http://localhost:5173")
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	rr := httptest.NewRecorder()

	mw(next).ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status: got %d, want %d", rr.Code, http.StatusOK)
	}
	if got := rr.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:5173" {
		t.Fatalf("allow-origin: got %q, want %q", got, "http://localhost:5173")
	}
}

func TestCORS_HandlesPreflight(t *testing.T) {
	mw := CORS("http://localhost:5173")
	nextCalled := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
	})

	req := httptest.NewRequest(http.MethodOptions, "/api/v1/auth/register", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	req.Header.Set("Access-Control-Request-Method", http.MethodPost)
	rr := httptest.NewRecorder()

	mw(next).ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("status: got %d, want %d", rr.Code, http.StatusNoContent)
	}
	if nextCalled {
		t.Fatal("expected preflight to short-circuit before next handler")
	}
}
