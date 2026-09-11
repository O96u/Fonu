.PHONY: web build test lint fmt docker

web:
	cd web && npm ci && npm run build
	rm -rf cmd/fonu/web/dist
	mkdir -p cmd/fonu/web/dist
	cp -r web/dist/* cmd/fonu/web/dist/

build: web
	go build -o bin/fonu ./cmd/fonu

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
