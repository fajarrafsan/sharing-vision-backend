.PHONY: tidy run build migrate-up migrate-down migrate-version fmt fmt-check vet test test-integration check clean up down db-up db-down logs

BINARY := bin/warta-api

# Kredensial MySQL untuk test integrasi, cocok dengan docker-compose.yml.
TEST_DB_HOST ?= 127.0.0.1
TEST_DB_PASSWORD ?= root

tidy:
	go mod tidy

run:
	go run ./cmd/api

build:
	go build -o $(BINARY) ./cmd/api
	go build -o bin/warta-migrate ./cmd/migrate

migrate-up:
	go run ./cmd/migrate up

migrate-down:
	go run ./cmd/migrate down

migrate-version:
	go run ./cmd/migrate version

fmt:
	go fmt ./...

fmt-check:
	@test -z "$$(gofmt -l .)" || (gofmt -l . && echo "jalankan: make fmt" && exit 1)

vet:
	go vet ./...

# Unit test saja; test yang butuh MySQL otomatis dilewati.
test:
	go test ./...

# Semua test, termasuk end-to-end terhadap MySQL (make db-up lebih dulu).
test-integration:
	TEST_DB_HOST=$(TEST_DB_HOST) TEST_DB_PASSWORD=$(TEST_DB_PASSWORD) go test -race -count=1 ./...

check: fmt-check vet test

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
