.PHONY: web build test lint fmt docker install-hooks compress-images

compress-images:
	python scripts/compress-images.py

install-hooks:
	chmod +x githooks/pre-commit githooks/commit-msg 2>/dev/null || true
	git config core.hooksPath githooks
	@echo "Git hooks installed from githooks/ (pre-commit + commit-msg)"

web:
	cd web && npm ci && npm run build
	rm -rf cmd/fonu/web/dist
	mkdir -p cmd/fonu/web/dist
	cp -r web/dist/* cmd/fonu/web/dist/

VERSION ?= $(shell tr -d '\r\n' < VERSION 2>/dev/null || echo dev)

build: web
	go build -ldflags "-X github.com/fonu/fonu/internal/version.Version=$(VERSION)" -o bin/fonu ./cmd/fonu

test:
	go test ./...

fmt:
	gofmt -w .

lint: test
	cd web && npm run typecheck

docker:
	docker compose build

seed:
	FONU_DATA_DIR=./.data FONU_SESSION_SECRET=dev-secret-change-me go run ./cmd/seed
