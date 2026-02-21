BINARY := r-cli
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-s -w -X main.version=$(VERSION)"

.PHONY: build test test-integration test-all lint clean install

build: lint
	go build $(LDFLAGS) -o $(BINARY) ./cmd/r-cli

test:
	go test -v -race -coverprofile=coverage.out ./...

test-integration:
	go test -v -race -tags integration ./...

test-all: test test-integration

lint:
	golangci-lint run

clean:
	rm -f $(BINARY) coverage.out

install:
	go install $(LDFLAGS) ./cmd/r-cli
