package config_test

import (
	"testing"

	"kpp.dev/kpfc/internal/config"
)

func TestLoad_Defaults(t *testing.T) {
	t.Setenv("PORT", "")
	t.Setenv("DATABASE_URL", "")
	t.Setenv("JWT_SECRET", "")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Port != 8080 {
		t.Errorf("Port default: got %d, want 8080", cfg.Port)
	}
	if cfg.DatabaseURL != "" {
		t.Errorf("DatabaseURL default: got %q, want empty", cfg.DatabaseURL)
	}
	if cfg.JWTSecret != "" {
		t.Errorf("JWTSecret default: got %q, want empty", cfg.JWTSecret)
	}
}

func TestLoad_EnvOverrides(t *testing.T) {
	t.Setenv("PORT", "9090")
	t.Setenv("DATABASE_URL", "postgres://u:p@localhost/testdb")
	t.Setenv("JWT_SECRET", "supersecret")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Port != 9090 {
		t.Errorf("Port: got %d, want 9090", cfg.Port)
	}
	if cfg.DatabaseURL != "postgres://u:p@localhost/testdb" {
		t.Errorf("DatabaseURL: got %q, want postgres://u:p@localhost/testdb", cfg.DatabaseURL)
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
