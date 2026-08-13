package data_test

import (
	"context"
	"testing"

	dataext "github.com/widasinnacy/api-gateway/internal/extractor/data"
)

func TestService_Parse_JSON(t *testing.T) {
	svc := dataext.NewService()
	input := `Some text before {"name": "John", "age": 30} and after`

	result, err := svc.Parse(context.Background(), input, "json")
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if result.DetectedSchema != "json" {
		t.Errorf("DetectedSchema = %q, want json", result.DetectedSchema)
	}
	if result.StructuredData == "" {
		t.Error("StructuredData should not be empty")
	}
	if result.Confidence < 0.5 {
		t.Errorf("Confidence = %f, want >= 0.5", result.Confidence)
	}
}

func TestService_Parse_KeyValue(t *testing.T) {
	svc := dataext.NewService()
	input := `Name: John Doe
Email: john@example.com
Phone: 555-1234
City: New York`

	result, err := svc.Parse(context.Background(), input, "key-value")
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if result.DetectedSchema != "key-value" {
		t.Errorf("DetectedSchema = %q, want key-value", result.DetectedSchema)
	}
	if result.StructuredData == "" {
		t.Error("StructuredData should not be empty")
	}
}

func TestService_Parse_Table(t *testing.T) {
	svc := dataext.NewService()
	input := `<table><tr><th>Name</th><th>Age</th></tr><tr><td>Alice</td><td>25</td></tr><tr><td>Bob</td><td>30</td></tr></table>`

	result, err := svc.Parse(context.Background(), input, "table")
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if result.DetectedSchema != "table" {
		t.Errorf("DetectedSchema = %q, want table", result.DetectedSchema)
	}
}

func TestService_Parse_AutoDetect(t *testing.T) {
	svc := dataext.NewService()
	input := `{"items": [1, 2, 3]}`

	result, err := svc.Parse(context.Background(), input, "")
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if result.DetectedSchema != "json" {
		t.Errorf("DetectedSchema = %q, want json (auto-detected)", result.DetectedSchema)
	}
}
