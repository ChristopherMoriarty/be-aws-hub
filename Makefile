.PHONY: up down sqlc migrate run lint fmt

LINT = go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.15.0

up:
	docker compose up -d --build

down:
	docker compose down

sqlc:
	sqlc generate

migrate:
	go run ./cmd/migrate up

run:
	go run ./cmd/api

lint:
	$(LINT) run ./...

fmt:
	$(LINT) fmt ./...
