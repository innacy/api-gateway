package gateway_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	gateway "github.com/widasinnacy/api-gateway/internal/gateway"
	"github.com/widasinnacy/api-gateway/internal/gateway/config"
)

func TestServer_FullMiddlewareChain_Unauthenticated(t *testing.T) {
	cfg := &config.Config{
		Port:      8080,
		JWTSecret: "test-secret",
		RateLimits: map[string]config.RateLimit{
			"free": {RequestsPerMin: 100, BurstPerSec: 10},
		},
	}
	srv := gateway.New(cfg)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/extract/url", nil)
	w := httptest.NewRecorder()

	srv.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401 for unauthenticated request", w.Code)
	}

	var body map[string]string
	json.NewDecoder(w.Body).Decode(&body)
	if body["error"] != "unauthorized" {
		t.Errorf("error = %q, want 'unauthorized'", body["error"])
	}
}

func TestServer_HealthBypassesAuth(t *testing.T) {
	cfg := &config.Config{
		Port:      8080,
		JWTSecret: "test-secret",
		RateLimits: map[string]config.RateLimit{
			"free": {RequestsPerMin: 100, BurstPerSec: 10},
		},
	}
	srv := gateway.New(cfg)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	w := httptest.NewRecorder()

	srv.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200 for health endpoint", w.Code)
	}
}

func TestServer_MetricsEndpoint(t *testing.T) {
	cfg := &config.Config{
		Port:      8080,
		JWTSecret: "test-secret",
		RateLimits: map[string]config.RateLimit{
			"free": {RequestsPerMin: 100, BurstPerSec: 10},
		},
	}
	srv := gateway.New(cfg)

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	w := httptest.NewRecorder()

	srv.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200 for metrics", w.Code)
	}
}
