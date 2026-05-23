.PHONY: run build test tidy docker-up

run:
	go run ./cmd/server

build:
	go build -o bin/server ./cmd/server

test:
	go test ./...

tidy:
	go mod tidy

docker-up:
	cd .. && docker compose up -d postgres
