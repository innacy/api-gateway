package telemetry

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type Metrics struct {
	RequestsTotal   *prometheus.CounterVec
	RequestDuration *prometheus.HistogramVec
	GRPCCallsTotal  *prometheus.CounterVec
	GRPCDuration    *prometheus.HistogramVec
	CircuitState    *prometheus.GaugeVec
	RateLimitHits   *prometheus.CounterVec
	RateLimitErrors prometheus.Counter
}

func InitMetrics(serviceName string) (*Metrics, error) {
	m := &Metrics{
		RequestsTotal: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "gateway_requests_total",
			Help: "Total HTTP requests",
		}, []string{"method", "path", "status", "tenant_tier"}),

		RequestDuration: promauto.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "gateway_request_duration_seconds",
			Help:    "HTTP request duration",
			Buckets: prometheus.DefBuckets,
		}, []string{"method", "path", "service"}),

		GRPCCallsTotal: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "gateway_grpc_calls_total",
			Help: "Total gRPC calls to extractors",
		}, []string{"service", "method", "grpc_code"}),

		GRPCDuration: promauto.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "gateway_grpc_duration_seconds",
			Help:    "gRPC call duration",
			Buckets: prometheus.DefBuckets,
		}, []string{"service"}),

		CircuitState: promauto.NewGaugeVec(prometheus.GaugeOpts{
			Name: "gateway_circuit_state",
			Help: "Circuit breaker state (0=closed, 1=half-open, 2=open)",
		}, []string{"service"}),

		RateLimitHits: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "gateway_ratelimit_hits_total",
			Help: "Rate limit check results",
		}, []string{"tenant_tier", "result"}),

		RateLimitErrors: promauto.NewCounter(prometheus.CounterOpts{
			Name: "gateway_ratelimit_redis_errors_total",
			Help: "Redis errors during rate limiting",
		}),
	}
	return m, nil
}
