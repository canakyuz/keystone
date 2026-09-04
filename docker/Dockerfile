# syntax=docker/dockerfile:1.7

# --- build ---------------------------------------------------------------
# Sürüm sabitlenmiştir: :latest ile build'ler yeniden üretilebilir olmaz.
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Bağımlılıklar önce kopyalanır: kaynak değiştiğinde bu katman cache'ten gelir.
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download

COPY . .

# CGO kapalı: statik binary, distroless/scratch tabanlı runner'da libc gerekmez.
# -trimpath: build yollarını binary'den siler (bilgi sızıntısı + reproducibility).
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

# Yalnızca çalışmak için gereken dosyalar. .env KOPYALANMAZ:
# yapılandırma ortam değişkeni veya secret mount ile gelir, imaja gömülmez.
COPY --from=builder --chown=65532:65532 /out/keystone ./keystone
COPY --from=builder --chown=65532:65532 /app/migrations ./migrations
COPY --from=builder --chown=65532:65532 /app/api ./api
COPY --from=builder --chown=65532:65532 /app/web ./web

USER 65532:65532

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 \
    CMD wget -qO- http://127.0.0.1:8080/health || exit 1

ENTRYPOINT ["./keystone"]
