package config_test

import (
	"testing"

	"kpp.dev/kpfc/internal/config"
)

func TestLoad_Defaults(t *testing.T) {
	t.Setenv("PORT", "")
	t.Setenv("DB_PATH", "")
	t.Setenv("JWT_SECRET", "")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Port != 8080 {
		t.Errorf("Port default: got %d, want 8080", cfg.Port)
	}
	if cfg.DBPath != "./kpfc.db" {
		t.Errorf("DBPath default: got %q, want ./kpfc.db", cfg.DBPath)
	}
	if cfg.JWTSecret != "" {
		t.Errorf("JWTSecret default: got %q, want empty", cfg.JWTSecret)
	}
}

func TestLoad_EnvOverrides(t *testing.T) {
	t.Setenv("PORT", "9090")
	t.Setenv("DB_PATH", "/tmp/test.db")
	t.Setenv("JWT_SECRET", "supersecret")

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
}

func TestLoad_InvalidPort(t *testing.T) {
	t.Setenv("PORT", "not-a-number")
	_, err := config.Load()
	if err == nil {
		t.Fatal("expected error for invalid PORT, got nil")
	}
}
