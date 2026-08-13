package data

import (
	"context"
	"encoding/json"
	"regexp"
	"strings"

	"golang.org/x/net/html"
)

type ParseResult struct {
	StructuredData string
	Confidence     float64
	DetectedSchema string
}

type Service struct{}

func NewService() *Service {
	return &Service{}
}

func (s *Service) Parse(_ context.Context, content, schemaHint string) (*ParseResult, error) {
	if schemaHint == "" {
		schemaHint = s.detectSchema(content)
	}

	switch schemaHint {
	case "json":
		return s.parseJSON(content)
	case "key-value":
		return s.parseKeyValue(content)
	case "table":
		return s.parseTable(content)
	default:
		return s.parseJSON(content)
	}
}

func (s *Service) detectSchema(content string) string {
	trimmed := strings.TrimSpace(content)
	if (strings.HasPrefix(trimmed, "{") && strings.HasSuffix(trimmed, "}")) ||
		(strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]")) {
		return "json"
	}
	if strings.Contains(content, "<table") {
		return "table"
	}
	if kvPattern.MatchString(content) {
		return "key-value"
	}
	return "json"
}

var kvPattern = regexp.MustCompile(`(?m)^\s*[\w\s]+:\s*.+$`)

func (s *Service) parseJSON(content string) (*ParseResult, error) {
	jsonPattern := regexp.MustCompile(`\{[^{}]*\}|\[[^\[\]]*\]`)
	matches := jsonPattern.FindAllString(content, -1)

	for _, match := range matches {
		var parsed interface{}
		if err := json.Unmarshal([]byte(match), &parsed); err == nil {
			formatted, _ := json.MarshalIndent(parsed, "", "  ")
			return &ParseResult{
				StructuredData: string(formatted),
				Confidence:     0.9,
				DetectedSchema: "json",
			}, nil
		}
	}

	var parsed interface{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(content)), &parsed); err == nil {
		formatted, _ := json.MarshalIndent(parsed, "", "  ")
		return &ParseResult{
			StructuredData: string(formatted),
			Confidence:     0.95,
			DetectedSchema: "json",
		}, nil
	}

	return &ParseResult{
		StructuredData: "{}",
		Confidence:     0.1,
		DetectedSchema: "json",
	}, nil
}

func (s *Service) parseKeyValue(content string) (*ParseResult, error) {
	lines := strings.Split(content, "\n")
	result := make(map[string]string)

	for _, line := range lines {
		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			if key != "" && value != "" {
				result[key] = value
			}
		}
	}

	formatted, _ := json.MarshalIndent(result, "", "  ")
	confidence := 0.8
	if len(result) == 0 {
		confidence = 0.1
	}

	return &ParseResult{
		StructuredData: string(formatted),
		Confidence:     confidence,
		DetectedSchema: "key-value",
	}, nil
}

func (s *Service) parseTable(content string) (*ParseResult, error) {
	doc, err := html.Parse(strings.NewReader(content))
	if err != nil {
		return &ParseResult{StructuredData: "[]", Confidence: 0.1, DetectedSchema: "table"}, nil
	}

	var headers []string
	var rows []map[string]string

	var walk func(*html.Node)
	var currentRow []string
	inHeader := false

	walk = func(n *html.Node) {
		if n.Type == html.ElementNode {
			switch n.Data {
			case "th":
				inHeader = true
			case "td":
				inHeader = false
			case "tr":
				currentRow = nil
			}
		}

		if n.Type == html.TextNode && n.Parent != nil {
			text := strings.TrimSpace(n.Data)
			if text != "" {
				if n.Parent.Data == "th" {
					headers = append(headers, text)
				} else if n.Parent.Data == "td" {
					currentRow = append(currentRow, text)
				}
			}
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}

		if n.Type == html.ElementNode && n.Data == "tr" && len(currentRow) > 0 && !inHeader {
			row := make(map[string]string)
			for i, val := range currentRow {
				if i < len(headers) {
					row[headers[i]] = val
				}
			}
			rows = append(rows, row)
		}
	}
	walk(doc)

	formatted, _ := json.MarshalIndent(rows, "", "  ")
	return &ParseResult{
		StructuredData: string(formatted),
		Confidence:     0.85,
		DetectedSchema: "table",
	}, nil
}
