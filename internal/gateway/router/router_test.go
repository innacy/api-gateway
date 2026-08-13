package router_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/widasinnacy/api-gateway/internal/gateway/circuit"
	"github.com/widasinnacy/api-gateway/internal/gateway/config"
	"github.com/widasinnacy/api-gateway/internal/gateway/router"
)

func TestRouter_ExtractURL_CircuitOpen(t *testing.T) {
	reg := circuit.NewBreakerRegistry(1, 30*time.Second)
	breaker := reg.Get("extractor-url")
	breaker.RecordFailure() // trip circuit

	cfg := &config.Config{
		ExtractorURLAddr:  "localhost:50051",
		ExtractorDataAddr: "localhost:50052",
		ExtractorMetaAddr: "localhost:50053",
		RetryMaxAttempts:  3,
		RetryBaseDelay:    100 * time.Millisecond,
		RequestTimeout:    5 * time.Second,
	}

	r := router.New(cfg, reg)
	handler := r.Routes()

	body, _ := json.Marshal(map[string]interface{}{
		"url":            "https://example.com",
		"include_links":  true,
		"include_images": false,
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/extract/url", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want 503 when circuit is open", w.Code)
	}
}

func TestRouter_UnknownRoute(t *testing.T) {
	reg := circuit.NewBreakerRegistry(5, 30*time.Second)
	cfg := &config.Config{
		ExtractorURLAddr:  "localhost:50051",
		ExtractorDataAddr: "localhost:50052",
		ExtractorMetaAddr: "localhost:50053",
		RetryMaxAttempts:  3,
		RetryBaseDelay:    100 * time.Millisecond,
		RequestTimeout:    5 * time.Second,
	}

	r := router.New(cfg, reg)
	handler := r.Routes()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/extract/unknown", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", w.Code)
	}
}
