package gateway_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	gateway "github.com/widasinnacy/api-gateway/internal/gateway"
	"github.com/widasinnacy/api-gateway/internal/gateway/config"
)

func TestServer_HealthEndpoint(t *testing.T) {
	cfg := &config.Config{Port: 8080}
	srv := gateway.New(cfg)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	w := httptest.NewRecorder()

	srv.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
}

func TestServer_NotFound(t *testing.T) {
	cfg := &config.Config{Port: 8080}
	srv := gateway.New(cfg)

	req := httptest.NewRequest(http.MethodGet, "/nonexistent", nil)
	w := httptest.NewRecorder()

	srv.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", w.Code)
	}
}

func TestServer_StartAndShutdown(t *testing.T) {
	cfg := &config.Config{Port: 0}
	srv := gateway.New(cfg)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := srv.Start(ctx)
	if err != nil && err != http.ErrServerClosed {
		t.Errorf("Start() unexpected error: %v", err)
	}
}
