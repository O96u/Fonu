ARG VERSION=dev

FROM --platform=$BUILDPLATFORM node:22-alpine AS web-builder
ARG VERSION
WORKDIR /src/web
RUN apk add --no-cache bash && corepack enable
COPY web/package.json web/*lock* ./
COPY web/scripts/install-deps.sh ./scripts/install-deps.sh
RUN bash scripts/install-deps.sh
COPY web/ ./
ENV VITE_APP_VERSION=$VERSION
RUN pnpm run build

FROM --platform=$BUILDPLATFORM golang:1.27.4-bookworm AS go-builder
ARG VERSION
ARG TARGETARCH
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=web-builder /src/web/dist ./cmd/fonu/web/dist
RUN CGO_ENABLED=0 GOOS=linux GOARCH=${TARGETARCH} go build \
    -ldflags "-X github.com/fonu/fonu/internal/version.Version=${VERSION}" \
    -o /fonu ./cmd/fonu

FROM debian:bookworm-slim
ARG TARGETARCH
ARG FRP_VERSION=0.72.0
ARG FRP_SHA256
RUN apt-get update \
    && apt-get install -y --no-install-recommends nginx ca-certificates curl \
    && case "${TARGETARCH}" in \
         amd64) FRP_ARCH=amd64 ;; \
         arm64) FRP_ARCH=arm64 ;; \
         *) echo "unsupported arch: ${TARGETARCH}" && exit 1 ;; \
       esac \
    && if [ -z "${FRP_SHA256:-}" ]; then case "${FRP_VERSION}:${FRP_ARCH}" in \
         0.72.0:amd64) FRP_SHA256=daf719b32ca2778770bb92955e231a07414d754378b9af328aa2642862479804 ;; \
         0.72.0:arm64) FRP_SHA256=02180b857480302fd934e9ddbd50fe2f73229f99062f88756abc6de9df5e95cf ;; \
         *) echo "Set FRP_SHA256 for custom FRP version" >&2; exit 1 ;; esac; fi \
    && curl -fsSL --retry 3 "https://github.com/fatedier/frp/releases/download/v${FRP_VERSION}/frp_${FRP_VERSION}_linux_${FRP_ARCH}.tar.gz" -o /tmp/frp.tar.gz \
    && printf '%s  /tmp/frp.tar.gz\n' "${FRP_SHA256}" | sha256sum -c - \
    && tar -xzf /tmp/frp.tar.gz -C /tmp "frp_${FRP_VERSION}_linux_${FRP_ARCH}/frpc" \
    && rm -f /tmp/frp.tar.gz \
    && install -m 755 "/tmp/frp_${FRP_VERSION}_linux_${FRP_ARCH}/frpc" /usr/local/bin/frpc \
    && rm -rf "/tmp/frp_${FRP_VERSION}_linux_${FRP_ARCH}" \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app
COPY --from=go-builder /fonu /app/fonu
COPY --from=go-builder /src/web/assets/image /app/web/assets/image
COPY migrations /app/migrations

ENV FONU_DATA_DIR=/data \
    FONU_LISTEN=:6893 \
    FONU_NGINX_MIME_TYPES=/etc/nginx/mime.types \
    FONU_FRPC_BIN=/usr/local/bin/frpc

EXPOSE 6893 80 443

VOLUME ["/data"]

HEALTHCHECK --interval=30s --timeout=5s --start-period=60s --retries=3 CMD if [ -n "$FONU_TLS_CERT_FILE" ] || [ -n "$FONU_TLS_KEY_FILE" ]; then curl -fkSs --noproxy "*" --max-time 3 "https://127.0.0.1:${FONU_LISTEN##*:}/healthz"; else curl -fsS --noproxy "*" --max-time 3 "http://127.0.0.1:${FONU_LISTEN##*:}/healthz"; fi

CMD ["/app/fonu"]
