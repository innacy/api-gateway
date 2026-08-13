package redis_test

import (
	"testing"

	"github.com/widasinnacy/api-gateway/internal/pkg/redis"
)

func TestNewClient_InvalidURL(t *testing.T) {
	_, err := redis.NewClient("not-a-valid-redis-url")
	if err == nil {
		t.Fatal("expected error for invalid URL, got nil")
	}
}
