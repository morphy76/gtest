COMPONENT  := gtest
VERSION    ?= $(shell cat VERSION.$(COMPONENT) 2>/dev/null || echo "0.0.0")
COMMIT     ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME ?= $(shell date -u +'%Y-%m-%dT%H:%M:%SZ')
GOPATH        ?= $(shell go env GOPATH)
GOLANGCI_LINT ?= $(shell which golangci-lint 2>/dev/null || echo $(GOPATH)/bin/golangci-lint)

LDFLAGS    := -s -w \
  -X 'github.com/morphy76/gtest/internal/version.Version=$(VERSION)' \
  -X 'github.com/morphy76/gtest/internal/version.Commit=$(COMMIT)' \
  -X 'github.com/morphy76/gtest/internal/version.BuildTime=$(BUILD_TIME)'

## test: Run all library unit tests
.PHONY: test
test:
	go test -count=1 ./...

## test-integration: Run integration tests (requires external services)
.PHONY: test-integration
test-integration:
	go test -count=1 -tags=integration ./...

## test-examples: Build example binaries to verify they compile
.PHONY: test-examples
test-examples:
	go build -tags=gtest_example ./examples/...

## test-race: Run unit tests with race detector
.PHONY: test-race
test-race:
	go test -count=1 -race ./...

## test-bench: Run benchmark tests
.PHONY: test-bench
test-bench:
	go test -count=1 -bench=. -benchmem ./...

## lint: Run static analysis
.PHONY: lint
lint:
	$(GOLANGCI_LINT) run ./...

## generate: Run code generators
.PHONY: generate
generate:
	go generate ./...

## help: Print this help
.PHONY: help
help:
	@grep -E '^## ' Makefile | sed 's/## //'
