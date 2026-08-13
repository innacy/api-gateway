package config_test

import (
	"os"
	"testing"

	"github.com/widasinnacy/api-gateway/internal/gateway/config"
)

func TestLoad_Defaults(t *testing.T) {
	os.Unsetenv("PORT")
	os.Unsetenv("REDIS_URL")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Port != 8080 {
		t.Errorf("Port = %d, want 8080", cfg.Port)
	}
	if cfg.RedisURL != "redis://localhost:6379" {
		t.Errorf("RedisURL = %q, want redis://localhost:6379", cfg.RedisURL)
	}
	if cfg.JWTSecret == "" {
		t.Error("JWTSecret should have a default for dev")
	}
}

func TestLoad_FromEnv(t *testing.T) {
	t.Setenv("PORT", "9090")
	t.Setenv("REDIS_URL", "redis://custom:6380")
	t.Setenv("JWT_SECRET", "my-secret")
	t.Setenv("EXTRACTOR_URL_ADDR", "localhost:50051")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Port != 9090 {
		t.Errorf("Port = %d, want 9090", cfg.Port)
	}
	if cfg.RedisURL != "redis://custom:6380" {
		t.Errorf("RedisURL = %q", cfg.RedisURL)
	}
	if cfg.JWTSecret != "my-secret" {
		t.Errorf("JWTSecret = %q", cfg.JWTSecret)
	}
	if cfg.ExtractorURLAddr != "localhost:50051" {
		t.Errorf("ExtractorURLAddr = %q", cfg.ExtractorURLAddr)
	}
}
