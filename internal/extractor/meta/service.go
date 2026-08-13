package meta

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/net/html"
)

type DNSResult struct {
	ARecords     []string
	CNAMERecords []string
	MXRecords    []string
	TTL          string
}

type MetaResult struct {
	OpenGraph   map[string]string
	SchemaOrg   map[string]string
	HTTPHeaders map[string]string
	DNS         DNSResult
}

type Service struct {
	client *http.Client
}

func NewService() *Service {
	return &Service{
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

func (s *Service) GetMetadata(ctx context.Context, targetURL string, types []string) (*MetaResult, error) {
	result := &MetaResult{
		OpenGraph:   make(map[string]string),
		SchemaOrg:   make(map[string]string),
		HTTPHeaders: make(map[string]string),
	}

	typeSet := make(map[string]bool)
	for _, t := range types {
		typeSet[t] = true
	}

	if typeSet["headers"] || typeSet["opengraph"] || typeSet["schema_org"] {
		if err := s.fetchHTTP(ctx, targetURL, typeSet, result); err != nil {
			return nil, err
		}
	}

	if typeSet["dns"] {
		if err := s.lookupDNS(ctx, targetURL, result); err != nil {
			return nil, err
		}
	}

	return result, nil
}

func (s *Service) fetchHTTP(ctx context.Context, targetURL string, types map[string]bool, result *MetaResult) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, targetURL, nil)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("fetch failed: %w", err)
	}
	defer resp.Body.Close()

	if types["headers"] {
		for key, values := range resp.Header {
			result.HTTPHeaders[key] = strings.Join(values, ", ")
		}
	}

	if types["opengraph"] || types["schema_org"] {
		body, err := io.ReadAll(io.LimitReader(resp.Body, 1*1024*1024))
		if err != nil {
			return fmt.Errorf("read body: %w", err)
		}
		s.parseMetaTags(string(body), types, result)
	}

	return nil
}

func (s *Service) parseMetaTags(body string, types map[string]bool, result *MetaResult) {
	doc, err := html.Parse(strings.NewReader(body))
	if err != nil {
		return
	}

	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "meta" {
			var property, name, content string
			for _, attr := range n.Attr {
				switch attr.Key {
				case "property":
					property = attr.Val
				case "name":
					name = attr.Val
				case "content":
					content = attr.Val
				}
			}

			if types["opengraph"] && strings.HasPrefix(property, "og:") {
				result.OpenGraph[property] = content
			}
			if types["schema_org"] && (strings.HasPrefix(name, "schema") || strings.HasPrefix(property, "schema")) {
				key := name
				if key == "" {
					key = property
				}
				result.SchemaOrg[key] = content
			}
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)
}

func (s *Service) lookupDNS(ctx context.Context, targetURL string, result *MetaResult) error {
	u, err := url.Parse(targetURL)
	if err != nil {
		return fmt.Errorf("parse URL for DNS: %w", err)
	}

	host := u.Hostname()
	resolver := &net.Resolver{}

	addrs, err := resolver.LookupHost(ctx, host)
	if err == nil {
		result.DNS.ARecords = addrs
	}

	cname, err := resolver.LookupCNAME(ctx, host)
	if err == nil && cname != "" {
		result.DNS.CNAMERecords = []string{cname}
	}

	mxRecords, err := resolver.LookupMX(ctx, host)
	if err == nil {
		for _, mx := range mxRecords {
			result.DNS.MXRecords = append(result.DNS.MXRecords, mx.Host)
		}
	}

	return nil
}
