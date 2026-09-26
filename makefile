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
	@set -eu; \
	step="dependency download"; \
	coverage="not measured"; \
	trap 'status=$$?; if [ "$$status" -eq 0 ]; then printf "\nCHECK PASSED | lint: pass | tests: pass | coverage: %s%% (required: 95%%)\n" "$$coverage"; else printf "\nCHECK FAILED | step: %s | exit: %s | see preceding output for details\n" "$$step" "$$status"; fi' EXIT; \
	go mod download; \
	step="module tidy"; \
	go mod tidy; \
	step="module diff"; \
	git diff --exit-code go.mod go.sum; \
	step="lint"; \
	golangci-lint run; \
	step="tests"; \
	go test -v -race -covermode=atomic -coverprofile=coverage.out ./...; \
	step="coverage gate"; \
	coverage=$$(go tool cover -func=coverage.out | awk '/^total:/ { gsub("%", "", $$3); print $$3 }'); \
	awk -v coverage="$$coverage" 'BEGIN { if (coverage + 0 < 95) { printf "coverage %.1f%% is below the required 95%%\n", coverage; exit 1 } printf "coverage %.1f%% meets the required 95%%\n", coverage }'; \
	step="complete"

fmt:
	go fmt ./...
	go mod tidy

lint:
	golangci-lint run ./...

clean:
	rm -rf bin
