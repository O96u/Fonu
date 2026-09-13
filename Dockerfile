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
RUN apt-get update \
    && apt-get install -y --no-install-recommends nginx ca-certificates \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app
COPY --from=go-builder /fonu /app/fonu
COPY migrations /app/migrations

ENV FONU_DATA_DIR=/data \
    FONU_LISTEN=:6893 \
    FONU_NGINX_MIME_TYPES=/etc/nginx/mime.types

EXPOSE 6893 80 443

VOLUME ["/data"]

CMD ["/app/fonu"]
