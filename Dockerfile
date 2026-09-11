FROM node:22-alpine AS web-builder
WORKDIR /src/web
COPY web/package.json web/package-lock.json* ./
RUN npm ci
COPY web/ ./
RUN npm run build

FROM golang:1.23-bookworm AS go-builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=web-builder /src/web/dist ./cmd/fonu/web/dist
RUN CGO_ENABLED=0 GOOS=linux go build -o /fonu ./cmd/fonu

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
