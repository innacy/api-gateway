package middleware

import (
	"net/http"
	"strconv"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/widasinnacy/api-gateway/internal/pkg/telemetry"
)

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func Telemetry(metrics *telemetry.Metrics) func(http.Handler) http.Handler {
	tracer := otel.Tracer("api-gateway")

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			ctx, span := tracer.Start(r.Context(), r.Method+" "+r.URL.Path,
				trace.WithAttributes(
					attribute.String("http.method", r.Method),
					attribute.String("http.url", r.URL.String()),
				),
			)
			defer span.End()

			rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
			next.ServeHTTP(rw, r.WithContext(ctx))

			duration := time.Since(start).Seconds()
			status := strconv.Itoa(rw.statusCode)
			tier := GetTier(r.Context())
			if tier == "" {
				tier = "unknown"
			}

			if metrics != nil {
				metrics.RequestsTotal.WithLabelValues(r.Method, r.URL.Path, status, tier).Inc()
				metrics.RequestDuration.WithLabelValues(r.Method, r.URL.Path, "").Observe(duration)
			}

			span.SetAttributes(attribute.Int("http.status_code", rw.statusCode))
		})
	}
}
