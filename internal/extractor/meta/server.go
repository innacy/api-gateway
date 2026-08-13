package meta

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

var metaStartTime = time.Now()

type Server struct {
	extractorv1.UnimplementedMetadataExtractorServer
	svc         *Service
	failureRate float64
}

func NewServer(svc *Service, failureRate float64) *Server {
	return &Server{svc: svc, failureRate: failureRate}
}

func (s *Server) GetMetadata(ctx context.Context, req *extractorv1.MetaRequest) (*extractorv1.MetaResponse, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if ok {
		hops := md.Get("x-gateway-hop-count")
		if len(hops) > 0 {
			count, _ := strconv.Atoi(hops[0])
			if count > 1 {
				return nil, status.Error(codes.FailedPrecondition, "request loop detected")
			}
		}
	}

	if s.failureRate > 0 && rand.Float64() < s.failureRate {
		return nil, status.Error(codes.Unavailable, "simulated failure")
	}

	result, err := s.svc.GetMetadata(ctx, req.Url, req.MetadataTypes)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "metadata extraction failed: %v", err)
	}

	resp := &extractorv1.MetaResponse{
		Opengraph:   result.OpenGraph,
		SchemaOrg:   result.SchemaOrg,
		HttpHeaders: result.HTTPHeaders,
		Dns: &extractorv1.DNSInfo{
			ARecords:     result.DNS.ARecords,
			CnameRecords: result.DNS.CNAMERecords,
			MxRecords:    result.DNS.MXRecords,
			Ttl:          result.DNS.TTL,
		},
	}

	return resp, nil
}

func (s *Server) Health(ctx context.Context, req *extractorv1.HealthRequest) (*extractorv1.HealthResponse, error) {
	return &extractorv1.HealthResponse{
		Status:        "healthy",
		Service:       "extractor-meta",
		UptimeSeconds: int64(time.Since(metaStartTime).Seconds()),
	}, nil
}
