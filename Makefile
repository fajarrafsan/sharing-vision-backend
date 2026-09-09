.PHONY: tidy run build migrate-up migrate-down migrate-version fmt vet test clean up down db-up db-down logs

BINARY := bin/article-service

tidy:
	go mod tidy

run:
	go run ./cmd/api

build:
	go build -o $(BINARY) ./cmd/api

migrate-up:
	go run ./cmd/migrate up

migrate-down:
	go run ./cmd/migrate down

migrate-version:
	go run ./cmd/migrate version

fmt:
	go fmt ./...

vet:
	go vet ./...

test:
	go test ./...

clean:
	rm -rf bin

up:
	docker compose up -d --build

down:
	docker compose down

db-up:
	docker compose up -d mysql

db-down:
	docker compose stop mysql

logs:
	docker compose logs -f api
