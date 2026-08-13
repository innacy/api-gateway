package gateway

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/widasinnacy/api-gateway/internal/gateway/circuit"
	"github.com/widasinnacy/api-gateway/internal/gateway/config"
	"github.com/widasinnacy/api-gateway/internal/gateway/middleware"
	"github.com/widasinnacy/api-gateway/internal/gateway/router"
	"github.com/widasinnacy/api-gateway/internal/pkg/response"
	"github.com/widasinnacy/api-gateway/internal/pkg/telemetry"
)

type Server struct {
	cfg        *config.Config
	router     chi.Router
	circuitReg *circuit.Registry
	metrics    *telemetry.Metrics
}

type Deps struct {
	KeyStore       middleware.KeyStore
	RateLimitStore middleware.RateLimitStore
}

var (
	metricsOnce   sync.Once
	sharedMetrics *telemetry.Metrics
)

func initMetrics() *telemetry.Metrics {
	metricsOnce.Do(func() {
		sharedMetrics, _ = telemetry.InitMetrics("api-gateway")
	})
	return sharedMetrics
}

func New(cfg *config.Config) *Server {
	return NewWithDeps(cfg, Deps{})
}

func NewWithDeps(cfg *config.Config, deps Deps) *Server {
	s := &Server{
		cfg:        cfg,
		circuitReg: circuit.NewBreakerRegistry(cfg.CircuitBreakerThreshold, cfg.CircuitBreakerTimeout),
	}

	s.metrics = initMetrics()
	s.router = s.setupRouter(deps)
	return s
}

func (s *Server) setupRouter(deps Deps) chi.Router {
	r := chi.NewRouter()

	r.Use(chiMiddleware.Recoverer)
	r.Use(chiMiddleware.RealIP)
	r.Use(middleware.RequestID)

	if s.metrics != nil {
		r.Use(middleware.Telemetry(s.metrics))
	}

	// Public routes (no auth)
	r.Get("/api/v1/health", s.handleHealth)
	r.Handle("/metrics", promhttp.Handler())

	// Authenticated routes
	auth := middleware.NewAuthMiddleware(s.cfg.JWTSecret, deps.KeyStore)

	r.Group(func(r chi.Router) {
		r.Use(auth.Authenticate)

		if deps.RateLimitStore != nil {
			rl := middleware.NewRateLimiter(deps.RateLimitStore, s.cfg.RateLimits)
			r.Use(rl.Limit)
		}

		extRouter := router.New(s.cfg, s.circuitReg)
		extRouter.Register(r)
	})

	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		response.Error(w, http.StatusNotFound, "not_found", "endpoint not found")
	})

	return r
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	response.JSON(w, http.StatusOK, map[string]string{
		"status":  "healthy",
		"service": "api-gateway",
	})
}

func (s *Server) Handler() http.Handler {
	return s.router
}

func (s *Server) Start(ctx context.Context) error {
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", s.cfg.Port),
		Handler:      s.router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
		BaseContext:  func(_ net.Listener) context.Context { return ctx },
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		srv.Shutdown(shutdownCtx)
	}()

	return srv.ListenAndServe()
}
