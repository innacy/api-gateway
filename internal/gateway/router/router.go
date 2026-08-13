package router

import (
	"context"
	"encoding/json"
	"math"
	"math/rand"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	extractorv1 "github.com/widasinnacy/api-gateway/gen/extractor/v1"
	"github.com/widasinnacy/api-gateway/internal/gateway/circuit"
	"github.com/widasinnacy/api-gateway/internal/gateway/config"
	"github.com/widasinnacy/api-gateway/internal/gateway/middleware"
	"github.com/widasinnacy/api-gateway/internal/pkg/response"
)

type Router struct {
	cfg        *config.Config
	circuitReg *circuit.Registry
	conns      map[string]*grpc.ClientConn
}

func New(cfg *config.Config, circuitReg *circuit.Registry) *Router {
	return &Router{
		cfg:        cfg,
		circuitReg: circuitReg,
		conns:      make(map[string]*grpc.ClientConn),
	}
}

func (rt *Router) Routes() chi.Router {
	r := chi.NewRouter()
	rt.Register(r)
	return r
}

func (rt *Router) Register(r chi.Router) {
	r.Post("/api/v1/extract/url", rt.handleExtractURL)
	r.Post("/api/v1/extract/data", rt.handleExtractData)
	r.Post("/api/v1/extract/metadata", rt.handleExtractMetadata)
}

func (rt *Router) Close() {
	for _, conn := range rt.conns {
		conn.Close()
	}
}

func (rt *Router) handleExtractURL(w http.ResponseWriter, r *http.Request) {
	breaker := rt.circuitReg.Get("extractor-url")
	if !breaker.Allow() {
		response.Error(w, http.StatusServiceUnavailable, "service_unavailable",
			"extractor-url service is temporarily unavailable")
		return
	}

	var req struct {
		URL           string `json:"url"`
		IncludeLinks  bool   `json:"include_links"`
		IncludeImages bool   `json:"include_images"`
		MaxDepth      int32  `json:"max_depth"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid_request", "invalid JSON body")
		return
	}

	if req.URL == "" {
		response.Error(w, http.StatusBadRequest, "invalid_request", "url field is required")
		return
	}

	conn, err := rt.getConn(rt.cfg.ExtractorURLAddr)
	if err != nil {
		breaker.RecordFailure()
		response.Error(w, http.StatusServiceUnavailable, "service_unavailable", "failed to connect to extractor")
		return
	}

	client := extractorv1.NewURLExtractorClient(conn)
	grpcReq := &extractorv1.URLRequest{
		Url:           req.URL,
		IncludeLinks:  req.IncludeLinks,
		IncludeImages: req.IncludeImages,
		MaxDepth:      req.MaxDepth,
		Metadata:      map[string]string{"X-Gateway-Hop-Count": "1"},
	}

	ctx := rt.outgoingContext(r.Context())
	resp, err := rt.callWithRetry(ctx, func(ctx context.Context) (interface{}, error) {
		return client.Extract(ctx, grpcReq)
	})

	if err != nil {
		breaker.RecordFailure()
		st, _ := status.FromError(err)
		response.Error(w, grpcToHTTPStatus(st.Code()), "extractor_error", st.Message())
		return
	}

	breaker.RecordSuccess()
	response.JSON(w, http.StatusOK, resp)
}

func (rt *Router) handleExtractData(w http.ResponseWriter, r *http.Request) {
	breaker := rt.circuitReg.Get("extractor-data")
	if !breaker.Allow() {
		response.Error(w, http.StatusServiceUnavailable, "service_unavailable",
			"extractor-data service is temporarily unavailable")
		return
	}

	var req struct {
		Content    string `json:"content"`
		SchemaHint string `json:"schema_hint"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid_request", "invalid JSON body")
		return
	}

	conn, err := rt.getConn(rt.cfg.ExtractorDataAddr)
	if err != nil {
		breaker.RecordFailure()
		response.Error(w, http.StatusServiceUnavailable, "service_unavailable", "failed to connect to extractor")
		return
	}

	client := extractorv1.NewDataParserClient(conn)
	grpcReq := &extractorv1.ParseRequest{
		Content:    req.Content,
		SchemaHint: req.SchemaHint,
		Metadata:   map[string]string{"X-Gateway-Hop-Count": "1"},
	}

	ctx := rt.outgoingContext(r.Context())
	resp, err := rt.callWithRetry(ctx, func(ctx context.Context) (interface{}, error) {
		return client.Parse(ctx, grpcReq)
	})

	if err != nil {
		breaker.RecordFailure()
		st, _ := status.FromError(err)
		response.Error(w, grpcToHTTPStatus(st.Code()), "extractor_error", st.Message())
		return
	}

	breaker.RecordSuccess()
	response.JSON(w, http.StatusOK, resp)
}

func (rt *Router) handleExtractMetadata(w http.ResponseWriter, r *http.Request) {
	breaker := rt.circuitReg.Get("extractor-meta")
	if !breaker.Allow() {
		response.Error(w, http.StatusServiceUnavailable, "service_unavailable",
			"extractor-meta service is temporarily unavailable")
		return
	}

	var req struct {
		URL           string   `json:"url"`
		RawContent    string   `json:"raw_content"`
		MetadataTypes []string `json:"metadata_types"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid_request", "invalid JSON body")
		return
	}

	conn, err := rt.getConn(rt.cfg.ExtractorMetaAddr)
	if err != nil {
		breaker.RecordFailure()
		response.Error(w, http.StatusServiceUnavailable, "service_unavailable", "failed to connect to extractor")
		return
	}

	client := extractorv1.NewMetadataExtractorClient(conn)
	grpcReq := &extractorv1.MetaRequest{
		Url:           req.URL,
		RawContent:    req.RawContent,
		MetadataTypes: req.MetadataTypes,
		Metadata:      map[string]string{"X-Gateway-Hop-Count": "1"},
	}

	ctx := rt.outgoingContext(r.Context())
	resp, err := rt.callWithRetry(ctx, func(ctx context.Context) (interface{}, error) {
		return client.GetMetadata(ctx, grpcReq)
	})

	if err != nil {
		breaker.RecordFailure()
		st, _ := status.FromError(err)
		response.Error(w, grpcToHTTPStatus(st.Code()), "extractor_error", st.Message())
		return
	}

	breaker.RecordSuccess()
	response.JSON(w, http.StatusOK, resp)
}

func (rt *Router) getConn(addr string) (*grpc.ClientConn, error) {
	if conn, ok := rt.conns[addr]; ok {
		return conn, nil
	}
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	rt.conns[addr] = conn
	return conn, nil
}

func (rt *Router) outgoingContext(ctx context.Context) context.Context {
	requestID := middleware.GetRequestID(ctx)
	md := metadata.Pairs(
		"x-request-id", requestID,
		"x-gateway-hop-count", "1",
	)
	return metadata.NewOutgoingContext(ctx, md)
}

func (rt *Router) callWithRetry(ctx context.Context, fn func(ctx context.Context) (interface{}, error)) (interface{}, error) {
	var lastErr error

	for attempt := 0; attempt < rt.cfg.RetryMaxAttempts; attempt++ {
		callCtx, cancel := context.WithTimeout(ctx, rt.cfg.RequestTimeout)
		resp, err := fn(callCtx)
		cancel()

		if err == nil {
			return resp, nil
		}

		lastErr = err
		st, _ := status.FromError(err)
		if !isRetryable(st.Code()) {
			return nil, err
		}

		if attempt < rt.cfg.RetryMaxAttempts-1 {
			delay := rt.cfg.RetryBaseDelay * time.Duration(math.Pow(2, float64(attempt)))
			jitter := time.Duration(rand.Int63n(int64(delay / 4)))
			time.Sleep(delay + jitter)
		}
	}

	return nil, lastErr
}

func isRetryable(code codes.Code) bool {
	return code == codes.Unavailable || code == codes.DeadlineExceeded
}

func grpcToHTTPStatus(code codes.Code) int {
	switch code {
	case codes.OK:
		return http.StatusOK
	case codes.InvalidArgument:
		return http.StatusBadRequest
	case codes.NotFound:
		return http.StatusNotFound
	case codes.PermissionDenied:
		return http.StatusForbidden
	case codes.Unauthenticated:
		return http.StatusUnauthorized
	case codes.Unavailable:
		return http.StatusServiceUnavailable
	case codes.DeadlineExceeded:
		return http.StatusGatewayTimeout
	default:
		return http.StatusInternalServerError
	}
}
