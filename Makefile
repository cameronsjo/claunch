.DEFAULT_GOAL := help

BINARY  := claunch
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -s -w -X github.com/cameronsjo/claunch/cmd.version=$(VERSION)

## Development
.PHONY: build
# Build the binary into ./bin
build:
	go build -ldflags "$(LDFLAGS)" -o bin/$(BINARY) .

.PHONY: install
# Install the binary into $GOBIN / $GOPATH/bin
install:
	go install -ldflags "$(LDFLAGS)" .

.PHONY: run
# Run claunch from source (pass ARGS="...")
run:
	go run . $(ARGS)

## Quality
.PHONY: check
# Run linting
check:
	golangci-lint run

.PHONY: fix
# Auto-fix lint + format issues
fix:
	golangci-lint run --fix

.PHONY: fmt
# Format the code
fmt:
	gofmt -w .

.PHONY: test
# Run the test suite with the race detector
test:
	go test -race ./...

.PHONY: cover
# Run tests with coverage report
cover:
	go test -coverprofile=coverage.txt ./... && go tool cover -func=coverage.txt | tail -1

.PHONY: snapshot
# Cross-compile a snapshot release (no publish)
snapshot:
	go run github.com/goreleaser/goreleaser/v2@latest release --snapshot --clean

## Help
.PHONY: help
# Show available targets
help:
	@grep -E '^[a-zA-Z_-]+:.*?#' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?# "}; {printf "  \033[36m%-12s\033[0m %s\n", $$1, $$2}'
