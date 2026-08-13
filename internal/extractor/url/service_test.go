package url_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	urlext "github.com/widasinnacy/api-gateway/internal/extractor/url"
)

func TestService_Extract_BasicHTML(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<html><head><title>Test Page</title></head>
			<body><p>Hello world</p>
			<a href="https://example.com/link1">Link 1</a>
			<img src="https://example.com/image.png"/>
			</body></html>`))
	}))
	defer ts.Close()

	svc := urlext.NewService()
	result, err := svc.Extract(context.Background(), ts.URL, urlext.ExtractOptions{
		IncludeLinks:  true,
		IncludeImages: true,
	})

	if err != nil {
		t.Fatalf("Extract() error = %v", err)
	}
	if result.Title != "Test Page" {
		t.Errorf("Title = %q, want 'Test Page'", result.Title)
	}
	if !strings.Contains(result.Content, "Hello world") {
		t.Errorf("Content should contain 'Hello world', got %q", result.Content)
	}
	if len(result.Links) == 0 {
		t.Error("Links should not be empty")
	}
	if len(result.Images) == 0 {
		t.Error("Images should not be empty")
	}
	if result.WordCount == 0 {
		t.Error("WordCount should be > 0")
	}
}

func TestService_Extract_InvalidURL(t *testing.T) {
	svc := urlext.NewService()
	_, err := svc.Extract(context.Background(), "not-a-url", urlext.ExtractOptions{})

	if err == nil {
		t.Error("Expected error for invalid URL")
	}
}

func TestService_Extract_Timeout(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done():
		case <-time.After(10 * time.Second):
		}
	}))
	defer ts.Close()

	svc := urlext.NewService()
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := svc.Extract(ctx, ts.URL, urlext.ExtractOptions{})
	if err == nil {
		t.Error("Expected timeout error")
	}
}
