package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/widasinnacy/api-gateway/internal/pkg/response"
)

const (
	tenantIDKey contextKey = "tenant_id"
	tierKey     contextKey = "tier"
	scopesKey   contextKey = "scopes"
)

type APIKeyData struct {
	TenantID string
	Tier     string
	Scopes   []string
}

type KeyStore interface {
	GetAPIKey(ctx context.Context, key string) (*APIKeyData, error)
}

type AuthMiddleware struct {
	jwtSecret []byte
	keyStore  KeyStore
}

func NewAuthMiddleware(jwtSecret string, keyStore KeyStore) *AuthMiddleware {
	return &AuthMiddleware{
		jwtSecret: []byte(jwtSecret),
		keyStore:  keyStore,
	}
}

func (a *AuthMiddleware) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// API key takes precedence
		if apiKey := r.Header.Get("X-API-Key"); apiKey != "" {
			data, err := a.keyStore.GetAPIKey(ctx, apiKey)
			if err != nil || data == nil {
				response.Error(w, http.StatusUnauthorized, "unauthorized", "invalid API key")
				return
			}
			ctx = withAuthContext(ctx, data.TenantID, data.Tier, data.Scopes)
			SetRequestMeta(ctx, data.Tier)
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		// Fall back to JWT
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			response.Error(w, http.StatusUnauthorized, "unauthorized", "missing credentials")
			return
		}

		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
		token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return a.jwtSecret, nil
		})

		if err != nil || !token.Valid {
			response.Error(w, http.StatusUnauthorized, "unauthorized", "invalid or expired token")
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			response.Error(w, http.StatusUnauthorized, "unauthorized", "invalid token claims")
			return
		}

		tenantID, _ := claims["tenant_id"].(string)
		scopes := extractScopes(claims["scopes"])

		ctx = withAuthContext(ctx, tenantID, "jwt", scopes)
		SetRequestMeta(ctx, "jwt")
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func withAuthContext(ctx context.Context, tenantID, tier string, scopes []string) context.Context {
	ctx = context.WithValue(ctx, tenantIDKey, tenantID)
	ctx = context.WithValue(ctx, tierKey, tier)
	ctx = context.WithValue(ctx, scopesKey, scopes)
	return ctx
}

func extractScopes(raw interface{}) []string {
	switch v := raw.(type) {
	case []interface{}:
		scopes := make([]string, 0, len(v))
		for _, s := range v {
			if str, ok := s.(string); ok {
				scopes = append(scopes, str)
			}
		}
		return scopes
	case []string:
		return v
	default:
		return nil
	}
}

func GetTenantID(ctx context.Context) string {
	if id, ok := ctx.Value(tenantIDKey).(string); ok {
		return id
	}
	return ""
}

func GetTier(ctx context.Context) string {
	if tier, ok := ctx.Value(tierKey).(string); ok {
		return tier
	}
	return ""
}

func GetScopes(ctx context.Context) []string {
	if scopes, ok := ctx.Value(scopesKey).([]string); ok {
		return scopes
	}
	return nil
}
