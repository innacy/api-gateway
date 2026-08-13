package middleware_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/widasinnacy/api-gateway/internal/gateway/config"
	"github.com/widasinnacy/api-gateway/internal/gateway/middleware"
)

type mockRateLimitStore struct {
	mu      sync.Mutex
	entries map[string][]time.Time
}

func newMockRateLimitStore() *mockRateLimitStore {
	return &mockRateLimitStore{entries: make(map[string][]time.Time)}
}

func (m *mockRateLimitStore) CheckRateLimit(ctx context.Context, key string, limit int, window time.Duration) (int, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-window)

	valid := make([]time.Time, 0)
	for _, t := range m.entries[key] {
		if t.After(cutoff) {
			valid = append(valid, t)
		}
	}

	if len(valid) >= limit {
		m.entries[key] = valid
		return limit - len(valid), false, nil
	}

	valid = append(valid, now)
	m.entries[key] = valid
	return limit - len(valid), true, nil
}

func authedRequest(tenantID, tier string) *http.Request {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	ctx := req.Context()
	ctx = context.WithValue(ctx, middleware.ExportedTenantIDKey, tenantID)
	ctx = context.WithValue(ctx, middleware.ExportedTierKey, tier)
	return req.WithContext(ctx)
}

func TestRateLimiter_AllowsUnderLimit(t *testing.T) {
	store := newMockRateLimitStore()
	limits := map[string]config.RateLimit{
		"free": {RequestsPerMin: 5, BurstPerSec: 2},
	}
	rl := middleware.NewRateLimiter(store, limits)

	handler := rl.Limit(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := authedRequest("tenant-1", "free")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}

	remaining := w.Header().Get("X-RateLimit-Remaining")
	if remaining == "" {
		t.Error("X-RateLimit-Remaining header should be set")
	}
}

func TestRateLimiter_BlocksOverLimit(t *testing.T) {
	store := newMockRateLimitStore()
	limits := map[string]config.RateLimit{
		"free": {RequestsPerMin: 2, BurstPerSec: 2},
	}
	rl := middleware.NewRateLimiter(store, limits)

	handler := rl.Limit(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	for i := 0; i < 2; i++ {
		req := authedRequest("tenant-1", "free")
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Errorf("request %d: status = %d, want 200", i, w.Code)
		}
	}

	req := authedRequest("tenant-1", "free")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusTooManyRequests {
		t.Errorf("status = %d, want 429", w.Code)
	}

	if ra := w.Header().Get("Retry-After"); ra == "" {
		t.Error("Retry-After header should be set")
	}
}

func TestRateLimiter_IsolatesPerTenant(t *testing.T) {
	store := newMockRateLimitStore()
	limits := map[string]config.RateLimit{
		"free": {RequestsPerMin: 1, BurstPerSec: 1},
	}
	rl := middleware.NewRateLimiter(store, limits)

	handler := rl.Limit(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := authedRequest("tenant-a", "free")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("tenant-a status = %d, want 200", w.Code)
	}

	req = authedRequest("tenant-b", "free")
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("tenant-b status = %d, want 200", w.Code)
	}
}
