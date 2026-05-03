package config_test

import (
	"testing"

	"kpp.dev/kpfc/internal/config"
)

func TestLoad_Defaults(t *testing.T) {
	t.Setenv("PORT", "")
	t.Setenv("DB_PATH", "")
	t.Setenv("JWT_SECRET", "")
	t.Setenv("CORS_ALLOW_ORIGINS", "")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Port != 8080 {
		t.Errorf("Port default: got %d, want 8080", cfg.Port)
	}
	if cfg.DBPath != "./data/kpfc.db" {
		t.Errorf("DBPath default: got %q, want ./data/kpfc.db", cfg.DBPath)
	}
	if cfg.JWTSecret != "" {
		t.Errorf("JWTSecret default: got %q, want empty", cfg.JWTSecret)
	}
	if cfg.CORSAllowOrigins != "http://localhost:5173" {
		t.Errorf("CORSAllowOrigins default: got %q, want http://localhost:5173", cfg.CORSAllowOrigins)
	}
}

func TestLoad_EnvOverrides(t *testing.T) {
	t.Setenv("PORT", "9090")
	t.Setenv("DB_PATH", "/tmp/test.db")
	t.Setenv("JWT_SECRET", "supersecret")
	t.Setenv("CORS_ALLOW_ORIGINS", "http://localhost:3000,http://localhost:5173")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Port != 9090 {
		t.Errorf("Port: got %d, want 9090", cfg.Port)
	}
	if cfg.DBPath != "/tmp/test.db" {
		t.Errorf("DBPath: got %q, want /tmp/test.db", cfg.DBPath)
	}
	if cfg.JWTSecret != "supersecret" {
		t.Errorf("JWTSecret: got %q, want supersecret", cfg.JWTSecret)
	}
	if cfg.CORSAllowOrigins != "http://localhost:3000,http://localhost:5173" {
		t.Errorf("CORSAllowOrigins: got %q, want http://localhost:3000,http://localhost:5173", cfg.CORSAllowOrigins)
	}
}

func TestLoad_InvalidPort(t *testing.T) {
	t.Setenv("PORT", "not-a-number")
	_, err := config.Load()
	if err == nil {
		t.Fatal("expected error for invalid PORT, got nil")
	}
}
