package url_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"google.golang.org/grpc/metadata"

	extractorv1 "github.com/widasinnacy/api-gateway/gen/extractor/v1"
	urlext "github.com/widasinnacy/api-gateway/internal/extractor/url"
)

func TestServer_Extract(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html><head><title>Hello</title></head><body><p>Content</p></body></html>`))
	}))
	defer ts.Close()

	svc := urlext.NewService()
	srv := urlext.NewServer(svc, 0)

	resp, err := srv.Extract(context.Background(), &extractorv1.URLRequest{
		Url:          ts.URL,
		IncludeLinks: true,
	})
	if err != nil {
		t.Fatalf("Extract() error = %v", err)
	}
	if resp.Title != "Hello" {
		t.Errorf("Title = %q, want Hello", resp.Title)
	}
}

func TestServer_Extract_RejectsHighHopCount(t *testing.T) {
	svc := urlext.NewService()
	srv := urlext.NewServer(svc, 0)

	md := metadata.Pairs("x-gateway-hop-count", "2")
	ctx := metadata.NewIncomingContext(context.Background(), md)

	_, err := srv.Extract(ctx, &extractorv1.URLRequest{Url: "http://example.com"})
	if err == nil {
		t.Error("Expected error for hop count > 1")
	}
}

func TestServer_Health(t *testing.T) {
	svc := urlext.NewService()
	srv := urlext.NewServer(svc, 0)

	resp, err := srv.Health(context.Background(), &extractorv1.HealthRequest{})
	if err != nil {
		t.Fatalf("Health() error = %v", err)
	}
	if resp.Status != "healthy" {
		t.Errorf("Status = %q, want healthy", resp.Status)
	}
}
