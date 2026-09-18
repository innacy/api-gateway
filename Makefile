# Makefile
.PHONY: proto test build run lint clean loadtest loadtest-quick

PROTO_DIR := proto
GEN_DIR := gen
K6 := $(shell command -v k6 2>/dev/null || echo $(HOME)/go/bin/k6)

proto:
	cd $(PROTO_DIR) && buf generate

test:
	go test ./... -v -race -count=1

build:
	go build -o bin/gateway ./cmd/gateway
	go build -o bin/extractor-url ./cmd/extractor-url
	go build -o bin/extractor-data ./cmd/extractor-data
	go build -o bin/extractor-meta ./cmd/extractor-meta

run-gateway:
	go run ./cmd/gateway

lint:
	golangci-lint run ./...

clean:
	rm -rf bin/ gen/

loadtest:
	$(K6) run --env BASE_URL=http://localhost:8080 loadtest/gateway.js

loadtest-quick:
	$(K6) run --duration 30s --vus 50 --env BASE_URL=http://localhost:8080 loadtest/gateway.js
