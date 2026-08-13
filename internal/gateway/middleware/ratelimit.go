package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/widasinnacy/api-gateway/internal/gateway/config"
	"github.com/widasinnacy/api-gateway/internal/pkg/response"
)

// Exported for test injection
var (
	ExportedTenantIDKey = tenantIDKey
	ExportedTierKey     = tierKey
)

type RateLimitStore interface {
	CheckRateLimit(ctx context.Context, key string, limit int, window time.Duration) (remaining int, allowed bool, err error)
}

type RateLimiter struct {
	store  RateLimitStore
	limits map[string]config.RateLimit
}

func NewRateLimiter(store RateLimitStore, limits map[string]config.RateLimit) *RateLimiter {
	return &RateLimiter{store: store, limits: limits}
}

func (rl *RateLimiter) Limit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		tenantID := GetTenantID(ctx)
		tier := GetTier(ctx)

		limit, ok := rl.limits[tier]
		if !ok {
			limit = rl.limits["free"]
		}

		key := fmt.Sprintf("ratelimit:%s:%s", tenantID, "min")
		remaining, allowed, err := rl.store.CheckRateLimit(ctx, key, limit.RequestsPerMin, time.Minute)

		if err != nil {
			next.ServeHTTP(w, r)
			return
		}

		w.Header().Set("X-RateLimit-Limit", strconv.Itoa(limit.RequestsPerMin))
		w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(remaining))
		w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(time.Now().Add(time.Minute).Unix(), 10))

		if !allowed {
			w.Header().Set("Retry-After", "60")
			response.Error(w, http.StatusTooManyRequests, "rate_limited", "rate limit exceeded")
			return
		}

		next.ServeHTTP(w, r)
	})
}
