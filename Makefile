.PHONY: build test lint vet run docker-up docker-down clean

BINARY := truebearing

build:
	go build -o bin/$(BINARY) ./cmd/server

test:
	go test -race -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out

lint:
	golangci-lint run ./...

vet:
	go vet ./...

run:
	go run ./cmd/server

docker-up:
	docker compose up --build -d

docker-down:
	docker compose down

clean:
	rm -rf bin/ coverage.out coverage.html
