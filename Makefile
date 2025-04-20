# Build variables
BINARY_NAME=mimic
BUILD_DIR=build
MAIN_PACKAGE=./cmd/mimic
LDFLAGS=-ldflags "-s -w"
GO=go
GOBUILD=$(GO) build
GOCLEAN=$(GO) clean
GOTEST=$(GO) test
GOGET=$(GO) get
GOMOD=$(GO) mod
GOLINT=golangci-lint
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

# Default target
.PHONY: all
all: lint test build

# Build the application
.PHONY: build
build: clean
	@echo "Building $(BINARY_NAME)..."
	mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PACKAGE)
	chmod +x $(BUILD_DIR)/$(BINARY_NAME)
	@echo "Done! Binary available at $(BUILD_DIR)/$(BINARY_NAME)"
	@echo "Run with: ./$(BUILD_DIR)/$(BINARY_NAME) -source [source_dir] -dest [destination_dir]"

# Install dependencies
.PHONY: deps
deps:
	@echo "Installing dependencies..."
	$(GOMOD) download
	$(GOGET) -u golang.org/x/lint/golint
	@echo "Installing golangci-lint..."
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(shell go env GOPATH)/bin v1.55.2

# Test all packages
.PHONY: test
test:
	@echo "Running tests..."
	$(GOTEST) -v ./...

# Run tests with coverage
.PHONY: test-coverage
test-coverage:
	@echo "Running tests with coverage..."
	mkdir -p $(BUILD_DIR)
	$(GOTEST) -coverprofile=$(BUILD_DIR)/coverage.out -covermode=atomic ./...
	$(GO) tool cover -html=$(BUILD_DIR)/coverage.out -o $(BUILD_DIR)/coverage.html
	@echo "Coverage report available at $(BUILD_DIR)/coverage.html"

# Run linter
.PHONY: lint
lint:
	@echo "Running linter..."
	$(GOLINT) run ./...

# Run the application
.PHONY: run
run:
	@echo "Running $(BINARY_NAME)..."
	$(GO) run $(MAIN_PACKAGE) $(ARGS)

# Clean build artifacts
.PHONY: clean
clean:
	@echo "Cleaning build artifacts..."
	$(GOCLEAN)
	rm -rf $(BUILD_DIR)

# Format code
.PHONY: fmt
fmt:
	@echo "Formatting code..."
	$(GO) fmt ./...

# Check for race conditions
.PHONY: race
race:
	@echo "Testing for race conditions..."
	$(GOTEST) -race ./...

# Display help information
.PHONY: help
help:
	@echo "Available targets:"
	@echo "  all             - Run lint, test, and build"
	@echo "  build           - Build the application for the current platform"
	@echo "  clean           - Remove build artifacts"
	@echo "  deps            - Install dependencies"
	@echo "  fmt             - Format Go code"
	@echo "  help            - Display this help message"
	@echo "  lint            - Run golangci-lint"
	@echo "  race            - Run tests with race detector"
	@echo "  run             - Run the application (use ARGS=\"-source dir1 -dest dir2\")"
	@echo "  test            - Run tests"
	@echo "  test-coverage   - Run tests with coverage reporting"
