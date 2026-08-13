package url

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"golang.org/x/net/html"
)

type ExtractOptions struct {
	IncludeLinks  bool
	IncludeImages bool
	MaxDepth      int32
}

type ExtractResult struct {
	Title     string
	Content   string
	Links     []string
	Images    []string
	WordCount int
	Language  string
}

type Service struct {
	client *http.Client
}

func NewService() *Service {
	return &Service{
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

func (s *Service) Extract(ctx context.Context, targetURL string, opts ExtractOptions) (*ExtractResult, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, targetURL, nil)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 5*1024*1024)) // 5MB limit
	if err != nil {
		return nil, fmt.Errorf("read body failed: %w", err)
	}

	doc, err := html.Parse(strings.NewReader(string(body)))
	if err != nil {
		return nil, fmt.Errorf("parse HTML failed: %w", err)
	}

	result := &ExtractResult{}
	var textBuilder strings.Builder

	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode {
			switch n.Data {
			case "title":
				if n.FirstChild != nil {
					result.Title = n.FirstChild.Data
				}
			case "a":
				if opts.IncludeLinks {
					for _, attr := range n.Attr {
						if attr.Key == "href" {
							result.Links = append(result.Links, attr.Val)
						}
					}
				}
			case "img":
				if opts.IncludeImages {
					for _, attr := range n.Attr {
						if attr.Key == "src" {
							result.Images = append(result.Images, attr.Val)
						}
					}
				}
			case "script", "style":
				return
			}
		}

		if n.Type == html.TextNode {
			text := strings.TrimSpace(n.Data)
			if text != "" {
				textBuilder.WriteString(text)
				textBuilder.WriteString(" ")
			}
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)

	result.Content = strings.TrimSpace(textBuilder.String())
	result.WordCount = len(strings.Fields(result.Content))
	result.Language = detectLanguage(result.Content)

	return result, nil
}

func detectLanguage(text string) string {
	if len(text) == 0 {
		return "unknown"
	}
	return "en"
}
