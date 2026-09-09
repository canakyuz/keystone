# syntax=docker/dockerfile:1.7

# --- build ---------------------------------------------------------------
# The version is pinned: :latest makes builds non-reproducible.
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Dependencies are copied first, so this layer comes from cache when only the
# source changes.
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download

COPY . .

# CGO off: a static binary needs no libc on a distroless or scratch runner.
# -trimpath strips build paths from the binary, for both reproducibility and to
# avoid leaking local paths.
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build \
      -trimpath \
      -ldflags="-s -w" \
      -o /out/keystone ./cmd/server

# --- runtime -------------------------------------------------------------
FROM alpine:3.20

RUN apk add --no-cache ca-certificates wget \
 && addgroup -g 65532 -S app \
 && adduser  -u 65532 -S app -G app

WORKDIR /app

# Only what is needed at runtime. .env is NOT copied: configuration arrives through
# environment variables or a secret mount, never baked into the image.
COPY --from=builder --chown=65532:65532 /out/keystone ./keystone
COPY --from=builder --chown=65532:65532 /app/migrations ./migrations
COPY --from=builder --chown=65532:65532 /app/api ./api
COPY --from=builder --chown=65532:65532 /app/web ./web

USER 65532:65532

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 \
    CMD wget -qO- http://127.0.0.1:8080/health || exit 1

ENTRYPOINT ["./keystone"]
