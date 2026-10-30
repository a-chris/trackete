# syntax=docker/dockerfile:1

# ---- frontend ---------------------------------------------------------------
FROM oven/bun:1.4.4-alpine AS web
WORKDIR /web
COPY frontend/package.json frontend/bun.lock ./
RUN bun install --frozen-lockfile
COPY frontend/ ./
RUN bun --bun run build

# ---- backend ----------------------------------------------------------------
FROM golang:1.27-alpine AS api
WORKDIR /src
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
ARG VERSION=dev
RUN CGO_ENABLED=0 GOOS=linux go build \
    -trimpath \
    -ldflags "-s -w -X main.version=${VERSION}" \
    -o /out/trackete ./cmd/server

# ---- runtime ----------------------------------------------------------------
FROM alpine:3.24
RUN apk add --no-cache ca-certificates tzdata \
    && addgroup -S trackete \
    && adduser -S -G trackete -h /app trackete \
    && mkdir -p /data \
    && chown trackete:trackete /data

WORKDIR /app
COPY --from=api /out/trackete /usr/local/bin/trackete
COPY --from=web /web/dist ./dist

ENV PORT=8080 \
    DB_PATH=/data/trackete.db \
    STATIC_DIR=/app/dist \
    GIN_MODE=release

VOLUME ["/data"]
EXPOSE 8080
USER trackete

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD wget -q -O /dev/null http://127.0.0.1:8080/api/health || exit 1

ENTRYPOINT ["/usr/local/bin/trackete"]
