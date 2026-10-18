FROM golang:1.27-alpine AS builder
WORKDIR /src
RUN apk add --no-cache git ca-certificates
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/online-judge-api ./cmd/api

FROM alpine:3.22
RUN apk add --no-cache ca-certificates tzdata docker-cli
WORKDIR /app
COPY --from=builder /out/online-judge-api /app/online-judge-api
COPY migrations /app/migrations
RUN mkdir -p /app/tmp/submissions && adduser -D -u 10001 appuser && chown -R appuser:appuser /app
EXPOSE 8080
# Docker socket access is required only when using the built-in Docker execution worker.
USER root
ENTRYPOINT ["/app/online-judge-api"]
