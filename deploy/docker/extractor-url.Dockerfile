# deploy/docker/extractor-url.Dockerfile
FROM golang:1.25-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /extractor-url ./cmd/extractor-url

FROM alpine:3.21
RUN apk --no-cache add ca-certificates
COPY --from=builder /extractor-url /extractor-url
EXPOSE 50051
ENTRYPOINT ["/extractor-url"]
