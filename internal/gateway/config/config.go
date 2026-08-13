package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	Port     int
	RedisURL string

	JWTSecret string

	ExtractorURLAddr  string
	ExtractorDataAddr string
	ExtractorMetaAddr string

	CircuitBreakerTimeout   time.Duration
	CircuitBreakerThreshold int

	RetryMaxAttempts int
	RetryBaseDelay   time.Duration
	RequestTimeout   time.Duration

	JaegerEndpoint string
	OTelEnabled    bool

	RateLimits map[string]RateLimit
}

type RateLimit struct {
	RequestsPerMin int
	BurstPerSec    int
}

func Load() (*Config, error) {
	cfg := &Config{
		Port:     envInt("PORT", 8080),
		RedisURL: envStr("REDIS_URL", "redis://localhost:6379"),

		JWTSecret: envStr("JWT_SECRET", "dev-secret-change-in-production"),

		ExtractorURLAddr:  envStr("EXTRACTOR_URL_ADDR", "localhost:50051"),
		ExtractorDataAddr: envStr("EXTRACTOR_DATA_ADDR", "localhost:50052"),
		ExtractorMetaAddr: envStr("EXTRACTOR_META_ADDR", "localhost:50053"),

		CircuitBreakerTimeout:   time.Duration(envInt("CIRCUIT_BREAKER_TIMEOUT_SEC", 30)) * time.Second,
		CircuitBreakerThreshold: envInt("CIRCUIT_BREAKER_THRESHOLD", 5),

		RetryMaxAttempts: envInt("RETRY_MAX_ATTEMPTS", 3),
		RetryBaseDelay:   time.Duration(envInt("RETRY_BASE_DELAY_MS", 100)) * time.Millisecond,
		RequestTimeout:   time.Duration(envInt("REQUEST_TIMEOUT_SEC", 5)) * time.Second,

		JaegerEndpoint: envStr("JAEGER_ENDPOINT", "http://localhost:4318"),
		OTelEnabled:    envBool("OTEL_ENABLED", true),

		RateLimits: map[string]RateLimit{
			"free":       {RequestsPerMin: 100, BurstPerSec: 10},
			"pro":        {RequestsPerMin: 1000, BurstPerSec: 50},
			"enterprise": {RequestsPerMin: 10000, BurstPerSec: 500},
		},
	}
	return cfg, nil
}

func envStr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return fallback
}

func envBool(key string, fallback bool) bool {
	if v := os.Getenv(key); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}
	return fallback
}
