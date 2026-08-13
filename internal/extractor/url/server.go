package url

import (
	"context"
	"math/rand"
	"strconv"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	extractorv1 "github.com/widasinnacy/api-gateway/gen/extractor/v1"
)

var startTime = time.Now()

type Server struct {
	extractorv1.UnimplementedURLExtractorServer
	svc         *Service
	failureRate float64
}

func NewServer(svc *Service, failureRate float64) *Server {
	return &Server{svc: svc, failureRate: failureRate}
}

func (s *Server) Extract(ctx context.Context, req *extractorv1.URLRequest) (*extractorv1.URLResponse, error) {
	if err := s.checkHopCount(ctx); err != nil {
		return nil, err
	}

	if s.shouldFail() {
		return nil, status.Error(codes.Unavailable, "simulated failure")
	}

	opts := ExtractOptions{
		IncludeLinks:  req.IncludeLinks,
		IncludeImages: req.IncludeImages,
		MaxDepth:      req.MaxDepth,
	}

	result, err := s.svc.Extract(ctx, req.Url, opts)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "extraction failed: %v", err)
	}

	return &extractorv1.URLResponse{
		Title:     result.Title,
		Content:   result.Content,
		Links:     result.Links,
		Images:    result.Images,
		WordCount: int32(result.WordCount),
		Language:  result.Language,
	}, nil
}

func (s *Server) Health(ctx context.Context, req *extractorv1.HealthRequest) (*extractorv1.HealthResponse, error) {
	return &extractorv1.HealthResponse{
		Status:        "healthy",
		Service:       "extractor-url",
		UptimeSeconds: int64(time.Since(startTime).Seconds()),
	}, nil
}

func (s *Server) checkHopCount(ctx context.Context) error {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil
	}
	hops := md.Get("x-gateway-hop-count")
	if len(hops) > 0 {
		count, _ := strconv.Atoi(hops[0])
		if count > 1 {
			return status.Error(codes.FailedPrecondition, "request loop detected: hop count exceeded")
		}
	}
	return nil
}

func (s *Server) shouldFail() bool {
	if s.failureRate <= 0 {
		return false
	}
	return rand.Float64() < s.failureRate
}
