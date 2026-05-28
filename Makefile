# Makefile for Persephone (Purr) - Git-like VCS in Go

BINARY_NAME=purr
MAIN_PATH=./cmd/purr
GO=go

.PHONY: all build install dev clean test fmt help

# Default target
all: build

## help: Show available commands
help:
	@echo "Available targets:"
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' | sed -e 's/^/ /'

## build: Build the binary
build:
	$(GO) build -o $(BINARY_NAME) $(MAIN_PATH)

## install: Install to GOPATH/bin
install:
	$(GO) install $(MAIN_PATH)

## dev: Run without building (e.g., make dev ARGS="init")
dev:
	$(GO) run $(MAIN_PATH) $(ARGS)

## clean: Remove build artifacts
clean:
	$(GO) clean
	rm -f $(BINARY_NAME) $(BINARY_NAME).exe

## test: Run tests
test:
	$(GO) test -v ./...

## fmt: Format code
fmt:
	$(GO) fmt ./...
