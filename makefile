.PHONY: build install run test fmt lint clean skill-install test-skill-install

build:
	mkdir -p bin
	go build -o bin/gha ./cmd/gha

install:
	$(MAKE) build
	go install ./cmd/gha

skill-install:
	$(MAKE) build
	bin/gha agent install --confirm

test-skill-install:
	go test ./internal/commands -run TestAgentInstall

run:
	go run ./cmd/gha

test:
	go test ./...

fmt:
	go fmt ./...
	go mod tidy

lint:
	golangci-lint run ./...

clean:
	rm -rf bin
