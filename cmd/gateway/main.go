package main

import (
	"context"
	"log"
	"os/signal"
	"strings"
	"syscall"

	gateway "github.com/widasinnacy/api-gateway/internal/gateway"
	"github.com/widasinnacy/api-gateway/internal/gateway/config"
	"github.com/widasinnacy/api-gateway/internal/gateway/middleware"
	"github.com/widasinnacy/api-gateway/internal/pkg/redis"
)

type redisKeyStore struct {
	client *redis.Client
}

func (s *redisKeyStore) GetAPIKey(ctx context.Context, key string) (*middleware.APIKeyData, error) {
	result, err := s.client.GetAPIKey(ctx, key)
	if err != nil {
		return nil, err
	}
	if len(result) == 0 {
		return nil, nil
	}

	var scopes []string
	if raw, ok := result["scopes"]; ok && raw != "" {
		scopes = strings.Split(raw, ",")
	}

	return &middleware.APIKeyData{
		TenantID: result["tenant_id"],
		Tier:     result["tier"],
		Scopes:   scopes,
	}, nil
}

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	rdb, err := redis.NewClient(cfg.RedisURL)
	if err != nil {
		log.Fatalf("failed to connect to redis: %v", err)
	}
	defer rdb.Close()

	deps := gateway.Deps{
		KeyStore:       &redisKeyStore{client: rdb},
		RateLimitStore: rdb,
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	srv := gateway.NewWithDeps(cfg, deps)
	log.Printf("starting gateway on :%d", cfg.Port)

	if err := srv.Start(ctx); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
