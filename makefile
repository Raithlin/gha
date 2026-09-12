.PHONY: build install run test fmt lint clean

build:
	mkdir -p bin
	go build -o bin/gha ./cmd/gha

install:
	$(MAKE) build
	go install ./cmd/gha

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
