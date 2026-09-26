.PHONY: build install run test check fmt lint clean skill-install test-skill-install

build:
	mkdir -p bin
	go build -o bin/gha ./cmd/gha

install:
	$(MAKE) build
	go install ./cmd/gha

skill-install:
	$(MAKE) build
	bin/gha agent install --agent codex

test-skill-install:
	go test ./internal/commands -run TestAgentInstall

run:
	go run ./cmd/gha

test:
	go test ./...

check:
	go mod download
	go mod tidy
	git diff --exit-code go.mod go.sum
	golangci-lint run
	go test -v -race -covermode=atomic -coverprofile=coverage.out ./...
	@go tool cover -func=coverage.out | awk '/^total:/ { gsub("%", "", $$3); if ($$3 + 0 < 95) { printf "coverage %.1f%% is below the required 95%%\\n", $$3; exit 1 } printf "coverage %.1f%% meets the required 95%%\\n", $$3 }'

fmt:
	go fmt ./...
	go mod tidy

lint:
	golangci-lint run ./...

clean:
	rm -rf bin
