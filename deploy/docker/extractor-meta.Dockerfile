# deploy/docker/extractor-meta.Dockerfile
FROM golang:1.25-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /extractor-meta ./cmd/extractor-meta

FROM alpine:3.21
RUN apk --no-cache add ca-certificates
COPY --from=builder /extractor-meta /extractor-meta
EXPOSE 50053
ENTRYPOINT ["/extractor-meta"]
