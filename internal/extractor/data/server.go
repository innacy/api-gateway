package data

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

var dataStartTime = time.Now()

type Server struct {
	extractorv1.UnimplementedDataParserServer
	svc         *Service
	failureRate float64
}

func NewServer(svc *Service, failureRate float64) *Server {
	return &Server{svc: svc, failureRate: failureRate}
}

func (s *Server) Parse(ctx context.Context, req *extractorv1.ParseRequest) (*extractorv1.ParseResponse, error) {
	if err := checkHopCount(ctx); err != nil {
		return nil, err
	}

	if s.failureRate > 0 && rand.Float64() < s.failureRate {
		return nil, status.Error(codes.Unavailable, "simulated failure")
	}

	result, err := s.svc.Parse(ctx, req.Content, req.SchemaHint)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "parse failed: %v", err)
	}

	return &extractorv1.ParseResponse{
		StructuredData: result.StructuredData,
		Confidence:     result.Confidence,
		DetectedSchema: result.DetectedSchema,
	}, nil
}

func (s *Server) Health(ctx context.Context, req *extractorv1.HealthRequest) (*extractorv1.HealthResponse, error) {
	return &extractorv1.HealthResponse{
		Status:        "healthy",
		Service:       "extractor-data",
		UptimeSeconds: int64(time.Since(dataStartTime).Seconds()),
	}, nil
}

func checkHopCount(ctx context.Context) error {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil
	}
	hops := md.Get("x-gateway-hop-count")
	if len(hops) > 0 {
		count, _ := strconv.Atoi(hops[0])
		if count > 1 {
			return status.Error(codes.FailedPrecondition, "request loop detected")
		}
	}
	return nil
}
