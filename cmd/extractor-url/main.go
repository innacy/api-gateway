package main

import (
	"log"
	"net"
	"os"
	"strconv"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	extractorv1 "github.com/widasinnacy/api-gateway/gen/extractor/v1"
	urlext "github.com/widasinnacy/api-gateway/internal/extractor/url"
)

func main() {
	port := envStr("PORT", "50051")
	failureRate := envFloat("FAILURE_RATE", 0)

	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	svc := urlext.NewService()
	srv := urlext.NewServer(svc, failureRate)

	grpcServer := grpc.NewServer()
	extractorv1.RegisterURLExtractorServer(grpcServer, srv)
	reflection.Register(grpcServer)

	log.Printf("extractor-url listening on :%s (failure_rate=%.2f)", port, failureRate)
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

func envStr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envFloat(key string, fallback float64) float64 {
	if v := os.Getenv(key); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		}
	}
	return fallback
}
