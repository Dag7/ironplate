BINARY_NAME := iron
MODULE := github.com/dag7/ironplate
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS := -s -w \
	-X $(MODULE)/internal/version.Version=$(VERSION) \
	-X $(MODULE)/internal/version.Commit=$(COMMIT) \
	-X $(MODULE)/internal/version.Date=$(DATE)

.PHONY: build install install-local completion test test-golden test-integration lint fmt vet clean help

## Build

build: ## Build the iron binary
	go build -ldflags "$(LDFLAGS)" -o bin/$(BINARY_NAME) ./cmd/iron

install: ## Install iron to GOPATH/bin
	go install -ldflags "$(LDFLAGS)" ./cmd/iron

install-local: build ## Install iron to /usr/local/bin (for devcontainers)
	sudo cp bin/$(BINARY_NAME) /usr/local/bin/$(BINARY_NAME)

## Completion

completion: install ## Generate shell completions to /usr/local/share
	@mkdir -p /usr/local/share/zsh/site-functions
	@iron completion zsh > /usr/local/share/zsh/site-functions/_iron
	@mkdir -p /etc/bash_completion.d
	@iron completion bash > /etc/bash_completion.d/iron
	@echo "Shell completions installed"

## Test

test: ## Run unit tests
	go test -race -count=1 ./...

test-golden: ## Run golden file tests
	go test -race -count=1 -run TestGolden ./...

test-integration: ## Run integration tests
	go test -race -count=1 -tags=integration ./...

test-all: test test-integration ## Run all tests

## Quality

lint: ## Run golangci-lint
	golangci-lint run ./...

fmt: ## Format Go source files
	gofmt -s -w .

vet: ## Run go vet
	go vet ./...

check: fmt vet lint test ## Run all checks

## Maintenance

clean: ## Remove build artifacts
	rm -rf bin/ dist/

tidy: ## Tidy Go modules
	go mod tidy

## Help

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

.DEFAULT_GOAL := help
