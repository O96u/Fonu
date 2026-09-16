ARG VERSION=dev

FROM --platform=$BUILDPLATFORM node:22-alpine AS web-builder
ARG VERSION
WORKDIR /src/web
COPY web/package.json web/package-lock.json* ./
RUN npm ci
COPY web/ ./
ENV VITE_APP_VERSION=$VERSION
RUN npm run build

FROM --platform=$BUILDPLATFORM golang:1.23-bookworm AS go-builder
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
ARG FRP_VERSION=0.71.0
RUN apt-get update \
    && apt-get install -y --no-install-recommends nginx ca-certificates curl \
    && case "${TARGETARCH}" in \
         amd64) FRP_ARCH=amd64 ;; \
         arm64) FRP_ARCH=arm64 ;; \
         *) echo "unsupported arch: ${TARGETARCH}" && exit 1 ;; \
       esac \
    && curl -fsSL "https://github.com/fatedier/frp/releases/download/v${FRP_VERSION}/frp_${FRP_VERSION}_linux_${FRP_ARCH}.tar.gz" \
         | tar -xz -C /tmp \
    && install -m 755 "/tmp/frp_${FRP_VERSION}_linux_${FRP_ARCH}/frpc" /usr/local/bin/frpc \
    && rm -rf "/tmp/frp_${FRP_VERSION}_linux_${FRP_ARCH}" \
    && apt-get purge -y curl \
    && apt-get autoremove -y \
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

CMD ["/app/fonu"]
