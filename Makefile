.PHONY: all build test clean install linux windows darwin help

# Variables
BINARY_NAME=ti
VERSION=0.1.0
BUILD_DIR=build
GO=go
BUILD_NUMBER=$(shell echo $$(($(shell git rev-list --count HEAD 2>/dev/null || echo "0") + 1)))
GOFLAGS=-ldflags="-s -w -X main.version=$(VERSION) -X main.buildNumber=$(BUILD_NUMBER)"

# Default target
all: test build

# Build for current platform
build:
	@echo "Building $(BINARY_NAME) for current platform..."
	@mkdir -p $(BUILD_DIR)
	$(GO) build $(GOFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) .
	@echo "Build complete: $(BUILD_DIR)/$(BINARY_NAME)"

# Build for Linux
linux:
	@echo "Building $(BINARY_NAME) for Linux..."
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 $(GO) build $(GOFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 .
	GOOS=linux GOARCH=arm64 $(GO) build $(GOFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-aarch64 .
	@echo "Build complete: $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 and $(BUILD_DIR)/$(BINARY_NAME)-linux-aarch64"

# Build for Windows
windows:
	@echo "Building $(BINARY_NAME) for Windows..."
	@mkdir -p $(BUILD_DIR)
	GOOS=windows GOARCH=amd64 $(GO) build $(GOFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe .
	@echo "Build complete: $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe"

# Build for macOS
darwin:
	@echo "Building $(BINARY_NAME) for macOS..."
	@mkdir -p $(BUILD_DIR)
	GOOS=darwin GOARCH=amd64 $(GO) build $(GOFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 .
	GOOS=darwin GOARCH=arm64 $(GO) build $(GOFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 .
	@echo "Build complete: $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 and $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64"

# Build for all platforms
all-platforms: linux windows darwin
	@echo "All platform builds complete!"

# Run tests
test:
	@echo "Running tests..."
	$(GO) test ./... -v

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	$(GO) test ./... -cover -coverprofile=coverage.out
	$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Run property-based tests only
test-property:
	@echo "Running property-based tests..."
	$(GO) test ./tests/property/... -v

# Run unit tests only
test-unit:
	@echo "Running unit tests..."
	$(GO) test ./tests/unit/... -v

# Run integration tests only
test-integration:
	@echo "Running integration tests..."
	$(GO) test ./tests/integration/... -v

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf $(BUILD_DIR)
	@rm -f coverage.out coverage.html
	@echo "Clean complete"

# Install to local system
install: build
	@echo "Installing $(BINARY_NAME)..."
	@cp $(BUILD_DIR)/$(BINARY_NAME) $(GOPATH)/bin/$(BINARY_NAME)
	@echo "Installation complete: $(GOPATH)/bin/$(BINARY_NAME)"

# Format code
fmt:
	@echo "Formatting code..."
	$(GO) fmt ./...
	@echo "Format complete"

# Run linter
lint:
	@echo "Running linter..."
	@which golangci-lint > /dev/null || (echo "golangci-lint not installed. Install with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest" && exit 1)
	golangci-lint run ./...
	@echo "Lint complete"

# Run the application
run: build
	@echo "Running $(BINARY_NAME)..."
	$(BUILD_DIR)/$(BINARY_NAME)

# Show help
help:
	@echo "Terminal Intelligence (TI) Makefile"
	@echo ""
	@echo "Usage:"
	@echo "  make [target]"
	@echo ""
	@echo "Targets:"
	@echo "  all              - Run tests and build for current platform (default)"
	@echo "  build            - Build for current platform"
	@echo "  linux            - Build for Linux (amd64)"
	@echo "  windows          - Build for Windows (amd64)"
	@echo "  darwin           - Build for macOS (amd64 and arm64)"
	@echo "  all-platforms    - Build for all platforms"
	@echo "  test             - Run all tests"
	@echo "  test-coverage    - Run tests with coverage report"
	@echo "  test-property    - Run property-based tests only"
	@echo "  test-unit        - Run unit tests only"
	@echo "  test-integration - Run integration tests only"
	@echo "  clean            - Remove build artifacts"
	@echo "  install          - Install to local system"
	@echo "  fmt              - Format code"
	@echo "  lint             - Run linter"
	@echo "  run              - Build and run the application"
	@echo "  help             - Show this help message"
