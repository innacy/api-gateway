package telemetry_test

import (
	"context"
	"testing"

	"github.com/widasinnacy/api-gateway/internal/pkg/telemetry"
)

func TestInitTracer_ReturnsShutdown(t *testing.T) {
	shutdown, err := telemetry.InitTracer("test-service", "")
	if err != nil {
		t.Fatalf("InitTracer() error = %v", err)
	}
	if shutdown == nil {
		t.Fatal("shutdown function should not be nil")
	}
	if err := shutdown(context.Background()); err != nil {
		t.Errorf("shutdown error = %v", err)
	}
}

func TestInitMetrics(t *testing.T) {
	m, err := telemetry.InitMetrics("test-service")
	if err != nil {
		t.Fatalf("InitMetrics() error = %v", err)
	}
	if m == nil {
		t.Fatal("metrics should not be nil")
	}
	if m.RequestsTotal == nil {
		t.Error("RequestsTotal counter should be initialized")
	}
	if m.RequestDuration == nil {
		t.Error("RequestDuration histogram should be initialized")
	}
}
