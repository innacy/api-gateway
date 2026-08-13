# Cloud-Native Microservices API Gateway with Extractor

**Date:** 2026-08-13  
**Status:** Approved  
**Type:** Portfolio showcase project

## Purpose

An enterprise-grade API gateway that manages rate-limiting, authentication, and communication with internal scraper/extractor microservices. Designed as a portfolio piece demonstrating deep understanding of internal API design, gRPC performance, resilient error handling, and container orchestration.

## Architecture

Single monolithic Go gateway binary fronting three gRPC extractor microservices. REST/HTTP externally; gRPC internally (protocol translation at the gateway boundary).

```
┌─────────────────────────────────────────────────────────────────┐
│                        External Clients                          │
│                    (REST/HTTP + API Key or JWT)                  │
└──────────────────────────────┬──────────────────────────────────┘
                               │
                               ▼
┌──────────────────────────────────────────────────────────────────┐
│                     API Gateway (Go binary)                       │
│  ┌──────────┐ ┌──────────┐ ┌────────────┐ ┌──────────────────┐  │
│  │  Auth    │ │  Rate    │ │  Circuit   │ │  gRPC Client     │  │
│  │Middleware│→│  Limiter │→│  Breaker   │→│  Pool + Router   │  │
│  └──────────┘ └──────────┘ └────────────┘ └──────────────────┘  │
│        ↕              ↕                              ↕            │
│     Redis          Redis                    OpenTelemetry         │
└──────────────────────────────┬──────────────────────────────────┘
                               │ gRPC
              ┌────────────────┼────────────────┐
              ▼                ▼                ▼
┌──────────────────┐ ┌─────────────────┐ ┌──────────────────────┐
│  URL Extractor   │ │  Data Parser    │ │  Metadata Extractor  │
│  (Go + gRPC)     │ │  (Go + gRPC)    │ │  (Go + gRPC)         │
└──────────────────┘ └─────────────────┘ └──────────────────────┘
```

## Repository Structure

```
api-gateway/
├── cmd/
│   ├── gateway/          # Main gateway binary
│   ├── extractor-url/    # URL content extractor service
│   ├── extractor-data/   # Structured data parser service
│   └── extractor-meta/   # Metadata/image extractor service
├── internal/
│   ├── gateway/
│   │   ├── middleware/   # Auth, rate-limit, logging, tracing
│   │   ├── router/       # REST route definitions + gRPC dispatch
│   │   ├── circuit/      # Circuit breaker implementation
│   │   └── config/       # Configuration loading
│   ├── extractor/        # Shared extractor interfaces/types
│   └── pkg/              # Shared utilities (Redis client, telemetry)
├── proto/                # Protobuf definitions
├── deploy/
│   ├── docker/           # Dockerfiles per service
│   ├── helm/             # Helm charts
│   └── docker-compose.yml
├── loadtest/             # k6 load test scripts
├── grafana/              # Dashboard JSON exports
├── docs/
└── Makefile
```

## Request Lifecycle

Strict middleware chain with single-responsibility layers:

1. **Request ID** — Generate unique trace ID, inject into context
2. **Telemetry** — Start OpenTelemetry span, record method/path
3. **Auth** — Validate API key OR JWT; extract claims/tenant ID → 401/403 on failure
4. **Rate Limit** — Sliding window check against Redis (keyed by tenant) → 429 on exceeded
5. **Route Match** — Map HTTP path+method → target extractor service → 404 on no match
6. **Circuit Breaker** — Check circuit state for target service → 503 if open
7. **gRPC Call + Retry** — Translate HTTP body → protobuf, call extractor with retry
8. **Response Marshal** — Translate protobuf → JSON, set cache headers, record metrics

**Key ordering decisions:**
- Auth before rate-limit: unauthenticated requests don't consume quota
- Circuit breaker before gRPC call: saves network round-trip when service is known-down

**Anti-loop protection:** Outgoing gRPC calls carry `X-Gateway-Hop-Count` metadata. Extractors reject requests with hop count > 1, preventing circular calls back through the gateway.

## Authentication

Two mechanisms, selected by header format:

### API Key (`X-API-Key` header)

- Stored in Redis: `apikey:{key}` → `{tenant_id, tier, scopes[], created_at}`
- Tier determines rate-limit budget
- Scopes control extractor access (`url:read`, `data:read`, `meta:read`)
- Generated via management endpoint (JWT admin scope required)

### JWT (`Authorization: Bearer <token>`)

- HMAC-SHA256 validation with configurable secret
- Claims: `sub`, `tenant_id`, `scopes[]`, `exp`, `iat`
- Required for user-scoped operations (usage stats, key management)

### Resolution Logic

1. Check `X-API-Key` → validate → use
2. If absent, check `Authorization: Bearer` → validate JWT → use
3. Neither present → 401
4. Extract `tenant_id` from either path → inject into request context

## Rate Limiting

**Algorithm:** Sliding window log using Redis sorted sets.

**Redis operations per request (single pipeline):**
1. `ZREMRANGEBYSCORE` — remove entries outside window
2. `ZADD` — add current timestamp
3. `ZCARD` — count entries in window
4. `EXPIRE` — set key TTL

**Tier limits:**

| Tier       | Requests/min | Burst/sec |
|------------|-------------|-----------|
| Free       | 100         | 10        |
| Pro        | 1,000       | 50        |
| Enterprise | 10,000      | 500       |

Burst limit is enforced via a secondary 1-second sliding window on the same sorted set (no additional Redis key needed — same ZRANGEBYSCORE with a 1-second lookback).

**Response headers (every response):**
- `X-RateLimit-Limit`
- `X-RateLimit-Remaining`
- `X-RateLimit-Reset` (Unix timestamp)

**On exceeded:** HTTP 429 with `Retry-After` header.

**Redis failure mode:** Fail open (allow request), log warning, increment `ratelimit_redis_errors_total` metric.

## Circuit Breaker

Per-service, in-memory state machine:

**States:** Closed → Open → Half-Open

- **Closed:** Requests pass through. Track failures in rolling 10-second window. Trip to Open on 5 consecutive failures OR >50% error rate.
- **Open:** Immediate 503 response. Duration: 30 seconds (configurable).
- **Half-Open:** Allow 1 probe request. Success → Closed. Failure → Open.

**Implementation:** Go struct with mutex-protected state. Resets to Closed on gateway restart.

## Retry Strategy

- Max 3 attempts (1 original + 2 retries)
- Exponential backoff: 100ms, 400ms (base × 2^attempt with jitter)
- Retryable codes: `UNAVAILABLE`, `DEADLINE_EXCEEDED`
- Non-retryable: `INVALID_ARGUMENT`, `NOT_FOUND`, `PERMISSION_DENIED`, `UNAUTHENTICATED`
- Per-request timeout: 5 seconds
- Total deadline: 15 seconds

**Circuit breaker interaction:**
- Open circuit → no retry, immediate 503
- Half-Open → single attempt, no retry
- Closed → normal retry behavior

## Extractor Microservices

### 1. URL Extractor (`extractor-url`)

- **Input:** URL + extraction options (include_links, include_images, max_depth)
- **Output:** Text content, title, links[], images[], word_count, language
- **Behavior:** Fetches URL via `net/http`, parses with `golang.org/x/net/html`
- **Failure simulation:** `FAILURE_RATE` env var (default 0, set to 0.1 for demo)

### 2. Data Parser (`extractor-data`)

- **Input:** Raw text/HTML + schema hint ("json", "table", "key-value")
- **Output:** Structured JSON, confidence score, detected schema
- **Behavior:** Pattern matching — JSON extraction, HTML table parsing, key-value detection

### 3. Metadata Extractor (`extractor-meta`)

- **Input:** URL or raw content + metadata types (opengraph, schema_org, headers, dns)
- **Output:** OpenGraph tags, schema.org data, HTTP headers, DNS info, SSL cert details
- **Behavior:** Fetches headers, parses meta tags, DNS lookup

**Shared characteristics:**
- Own Dockerfile (multi-stage, ~15MB final image)
- Health check RPC for Kubernetes liveness probes
- OpenTelemetry spans linked to gateway parent span
- Configurable failure rate for resilience testing

## Protobuf Definitions

```protobuf
service URLExtractor {
  rpc Extract(URLRequest) returns (URLResponse);
  rpc Health(HealthRequest) returns (HealthResponse);
}

service DataParser {
  rpc Parse(ParseRequest) returns (ParseResponse);
  rpc Health(HealthRequest) returns (HealthResponse);
}

service MetadataExtractor {
  rpc GetMetadata(MetaRequest) returns (MetaResponse);
  rpc Health(HealthRequest) returns (HealthResponse);
}
```

## Observability

### Metrics (Prometheus)

Gateway (`/metrics`):
- `gateway_requests_total` — labels: method, path, status, tenant_tier
- `gateway_request_duration_seconds` — histogram: method, path, service
- `gateway_grpc_calls_total` — labels: service, method, grpc_code
- `gateway_grpc_duration_seconds` — histogram: service
- `gateway_circuit_state` — gauge: service (0=closed, 1=half-open, 2=open)
- `gateway_ratelimit_hits_total` — labels: tenant_tier, result (allowed/denied)
- `gateway_ratelimit_redis_errors_total` — counter

Each extractor: `extractor_requests_total`, `extractor_duration_seconds`, `extractor_errors_total`

### Distributed Tracing (OpenTelemetry → Jaeger)

- Root span per HTTP request at gateway
- Child spans for each middleware stage
- Trace context propagated via gRPC metadata to extractors
- Full trace: HTTP → Auth → RateLimit → CircuitCheck → gRPC → Extractor
- Jaeger UI at `localhost:16686`

### Grafana Dashboard (pre-provisioned)

- Request throughput (req/sec) over time
- P50/P95/P99 latency histograms
- Error rate by service
- Circuit breaker state timeline
- Rate-limit rejection rate
- Per-tenant usage breakdown

Dashboard JSON committed to repo, auto-provisioned via docker-compose volume.

## Deployment

### Docker Compose (local dev)

`docker-compose up` starts 8 containers:
- API Gateway (port 8080)
- 3 Extractor services (internal gRPC, not externally exposed)
- Redis (port 6379)
- Prometheus (port 9090)
- Grafana (port 3000, no login, pre-provisioned)
- Jaeger (port 16686)

First build: ~2 minutes. Subsequent: <10 seconds.

### Helm Charts (production K8s)

```
deploy/helm/
├── api-gateway/        # Gateway deployment, service, HPA, ingress
├── extractors/         # One deployment per extractor
└── infrastructure/     # Redis, monitoring stack
```

HPA: scales gateway pods on CPU + custom metric (requests-per-second).

### Load Testing

`loadtest/` directory with k6 scripts:
- Realistic traffic patterns (mixed endpoints, varying auth)
- `make loadtest` runs against docker-compose stack
- Results formatted for README showcase

## API Surface

### Gateway Endpoints

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| POST | `/api/v1/extract/url` | API Key or JWT | Extract content from URL |
| POST | `/api/v1/extract/data` | API Key or JWT | Parse structured data |
| POST | `/api/v1/extract/metadata` | API Key or JWT | Extract metadata |
| GET | `/api/v1/health` | None | Gateway health check |
| GET | `/api/v1/usage` | JWT | Current tenant usage stats |
| POST | `/api/v1/keys` | JWT (admin) | Generate new API key |
| DELETE | `/api/v1/keys/{id}` | JWT (admin) | Revoke API key |
| GET | `/metrics` | None | Prometheus metrics |

## Tech Stack

- **Language:** Go 1.22+
- **HTTP Framework:** `net/http` with chi router (lightweight, stdlib-compatible)
- **gRPC:** google.golang.org/grpc + protobuf
- **Redis:** go-redis/redis/v9
- **Telemetry:** OpenTelemetry SDK → Jaeger exporter + Prometheus exporter
- **Containerization:** Multi-stage Docker builds
- **Orchestration:** Helm 3 charts, K8s manifests
- **Load Testing:** k6
- **Dashboards:** Grafana with JSON provisioning

## Success Criteria (README Showcase)

- Single `docker-compose up` brings up entire system
- Architectural block diagram (Mermaid) in README
- Working curl examples in README that produce real results
- Load test results: target 10k req/sec at <20ms P99 (gateway overhead only)
- Circuit breaker demo: configurable failure injection → visible state transitions in Grafana
- Full request trace visible in Jaeger from HTTP ingress through extractor processing

## Non-Goals

- Production-grade secret management (demo uses env vars)
- Token refresh/rotation (JWT issuer is out of scope)
- Persistent storage (extractors are stateless)
- Multi-region deployment
- WebSocket/streaming support
- Admin UI
