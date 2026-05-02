package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewUUID_ReturnsHex16Bytes(t *testing.T) {
	a := newUUID()
	b := newUUID()
	if len(a) != 32 {
		t.Fatalf("len = %d, want 32", len(a))
	}
	if a == b {
		t.Fatal("expected distinct UUID values")
	}
	for _, ch := range a {
		if !(ch >= '0' && ch <= '9' || ch >= 'a' && ch <= 'f') {
			t.Fatalf("non-hex character %q in %q", ch, a)
		}
	}
}

func TestHealthHandler(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	healthHandler("v1.2.3").ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if rec.Header().Get("Content-Type") != "application/json" {
		t.Fatalf("content-type = %q, want application/json", rec.Header().Get("Content-Type"))
	}

	var body map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if body["status"] != "ok" || body["version"] != "v1.2.3" {
		t.Fatalf("body = %#v", body)
	}
}

func TestVersionHandler(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/version", nil)
	versionHandler("build-9").ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	var body map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if body["version"] != "build-9" {
		t.Fatalf("version = %q, want build-9", body["version"])
	}
}
