package meta_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	metaext "github.com/widasinnacy/api-gateway/internal/extractor/meta"
)

func TestService_GetMetadata_OpenGraph(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<html><head>
			<meta property="og:title" content="Test OG Title"/>
			<meta property="og:description" content="A test description"/>
			<meta property="og:image" content="https://example.com/img.png"/>
		</head><body></body></html>`))
	}))
	defer ts.Close()

	svc := metaext.NewService()
	result, err := svc.GetMetadata(context.Background(), ts.URL, []string{"opengraph"})
	if err != nil {
		t.Fatalf("GetMetadata() error = %v", err)
	}

	if result.OpenGraph["og:title"] != "Test OG Title" {
		t.Errorf("og:title = %q, want 'Test OG Title'", result.OpenGraph["og:title"])
	}
}

func TestService_GetMetadata_Headers(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Custom", "custom-value")
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<html></html>`))
	}))
	defer ts.Close()

	svc := metaext.NewService()
	result, err := svc.GetMetadata(context.Background(), ts.URL, []string{"headers"})
	if err != nil {
		t.Fatalf("GetMetadata() error = %v", err)
	}

	if result.HTTPHeaders["X-Custom"] != "custom-value" {
		t.Errorf("X-Custom header = %q, want custom-value", result.HTTPHeaders["X-Custom"])
	}
}

func TestService_GetMetadata_DNS(t *testing.T) {
	svc := metaext.NewService()
	result, err := svc.GetMetadata(context.Background(), "https://example.com", []string{"dns"})
	if err != nil {
		t.Fatalf("GetMetadata() error = %v", err)
	}

	if len(result.DNS.ARecords) == 0 {
		t.Error("DNS A records should not be empty for example.com")
	}
}
