package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"

	"github.com/widasinnacy/api-gateway/internal/gateway/middleware"
	"github.com/widasinnacy/api-gateway/internal/pkg/telemetry"
)

var testMetrics *telemetry.Metrics

func initTestTelemetry(t *testing.T) *telemetry.Metrics {
	t.Helper()
	if testMetrics == nil {
		_, err := telemetry.InitTracer("test-service", "")
		if err != nil {
			t.Fatalf("InitTracer() error = %v", err)
		}
		testMetrics, err = telemetry.InitMetrics("test-service")
		if err != nil {
			t.Fatalf("InitMetrics() error = %v", err)
		}
	}
	return testMetrics
}

func counterValue(c prometheus.Counter) float64 {
	var m dto.Metric
	if err := c.Write(&m); err != nil {
		return 0
	}
	return m.GetCounter().GetValue()
}

func TestTelemetry_RecordsResponseStatus(t *testing.T) {
	metrics := initTestTelemetry(t)

	handler := middleware.Telemetry(metrics)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", w.Code)
	}

	counter := metrics.RequestsTotal.WithLabelValues("GET", "/test", "404", "unknown")
	if got := counterValue(counter); got != 1 {
		t.Errorf("RequestsTotal = %v, want 1", got)
	}
}

func TestTelemetry_DefaultStatusOK(t *testing.T) {
	metrics := initTestTelemetry(t)

	handler := middleware.Telemetry(metrics)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	}))

	req := httptest.NewRequest(http.MethodPost, "/health", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}

	counter := metrics.RequestsTotal.WithLabelValues("POST", "/health", "200", "unknown")
	if got := counterValue(counter); got != 1 {
		t.Errorf("RequestsTotal = %v, want 1", got)
	}
}
