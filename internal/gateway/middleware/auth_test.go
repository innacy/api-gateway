package middleware_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/widasinnacy/api-gateway/internal/gateway/middleware"
)

type mockKeyStore struct {
	keys map[string]*middleware.APIKeyData
}

func (m *mockKeyStore) GetAPIKey(ctx context.Context, key string) (*middleware.APIKeyData, error) {
	if data, ok := m.keys[key]; ok {
		return data, nil
	}
	return nil, nil
}

func newMockStore() *mockKeyStore {
	return &mockKeyStore{
		keys: map[string]*middleware.APIKeyData{
			"valid-key": {
				TenantID: "tenant-1",
				Tier:     "pro",
				Scopes:   []string{"url:read", "data:read"},
			},
		},
	}
}

func TestAuth_NoCredentials(t *testing.T) {
	auth := middleware.NewAuthMiddleware("secret", newMockStore())
	handler := auth.Authenticate(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called")
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", w.Code)
	}
}

func TestAuth_ValidAPIKey(t *testing.T) {
	auth := middleware.NewAuthMiddleware("secret", newMockStore())
	handler := auth.Authenticate(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tenant := middleware.GetTenantID(r.Context())
		if tenant != "tenant-1" {
			t.Errorf("tenant = %q, want tenant-1", tenant)
		}
		tier := middleware.GetTier(r.Context())
		if tier != "pro" {
			t.Errorf("tier = %q, want pro", tier)
		}
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-API-Key", "valid-key")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
}

func TestAuth_InvalidAPIKey(t *testing.T) {
	auth := middleware.NewAuthMiddleware("secret", newMockStore())
	handler := auth.Authenticate(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called")
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-API-Key", "bad-key")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", w.Code)
	}
}

func TestAuth_ValidJWT(t *testing.T) {
	secret := "test-secret"
	auth := middleware.NewAuthMiddleware(secret, newMockStore())

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":       "user-42",
		"tenant_id": "tenant-2",
		"scopes":    []interface{}{"url:read", "meta:read"},
		"exp":       time.Now().Add(time.Hour).Unix(),
		"iat":       time.Now().Unix(),
	})
	tokenStr, _ := token.SignedString([]byte(secret))

	handler := auth.Authenticate(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tenant := middleware.GetTenantID(r.Context())
		if tenant != "tenant-2" {
			t.Errorf("tenant = %q, want tenant-2", tenant)
		}
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
}

func TestAuth_ExpiredJWT(t *testing.T) {
	secret := "test-secret"
	auth := middleware.NewAuthMiddleware(secret, newMockStore())

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":       "user-42",
		"tenant_id": "tenant-2",
		"scopes":    []interface{}{"url:read"},
		"exp":       time.Now().Add(-time.Hour).Unix(),
		"iat":       time.Now().Add(-2 * time.Hour).Unix(),
	})
	tokenStr, _ := token.SignedString([]byte(secret))

	handler := auth.Authenticate(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called")
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", w.Code)
	}
}

func TestAuth_APIKeyTakesPrecedence(t *testing.T) {
	secret := "test-secret"
	auth := middleware.NewAuthMiddleware(secret, newMockStore())

	handler := auth.Authenticate(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tenant := middleware.GetTenantID(r.Context())
		if tenant != "tenant-1" {
			t.Errorf("tenant = %q, want tenant-1 (from API key)", tenant)
		}
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-API-Key", "valid-key")
	req.Header.Set("Authorization", "Bearer some-token")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
}
