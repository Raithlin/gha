.PHONY: build run test fmt lint clean

build:
	go build -o bin/gha ./cmd/gha

run:
	go run ./cmd/gha

test:
	go test ./...

fmt:
	go fmt ./...
	go mod tidy

lint:
	staticcheck ./...

clean:
	rm -rf bin