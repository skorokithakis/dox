.PHONY: build clean test install dev run-example

# Build variables
BINARY_NAME=dox
BUILD_DIR=build
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS=-ldflags "-X github.com/skorokithakis/dox/internal/cli.version=$(VERSION)"

# Default target
all: build

# Build the binary
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) main.go

# Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR)
	@go clean

# Run tests
test:
	@echo "Running tests..."
	go test -v ./...

# Install binary to GOPATH/bin
install: build
	@echo "Installing $(BINARY_NAME)..."
	@cp $(BUILD_DIR)/$(BINARY_NAME) $(GOPATH)/bin/$(BINARY_NAME)

# Development build (with race detector)
dev:
	@echo "Building $(BINARY_NAME) with race detector..."
	@mkdir -p $(BUILD_DIR)
	go build -race $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) cmd/dox/main.go

# Format code
fmt:
	@echo "Formatting code..."
	go fmt ./...

# Run linters
lint:
	@echo "Running linters..."
	golangci-lint run

# Download dependencies
deps:
	@echo "Downloading dependencies..."
	go mod download
	go mod tidy

# Create example configuration
example-config:
	@echo "Creating example configuration..."
	@mkdir -p ~/.config/dox/commands
	@echo "image: python:3.11-slim" > ~/.config/dox/commands/python.yaml
	@echo "volumes:" >> ~/.config/dox/commands/python.yaml
	@echo "  - .:/workspace" >> ~/.config/dox/commands/python.yaml
	@echo "environment:" >> ~/.config/dox/commands/python.yaml
	@echo "  - PYTHONPATH" >> ~/.config/dox/commands/python.yaml
	@echo "Created example Python configuration at ~/.config/dox/commands/python.yaml"

# Help target
help:
	@echo "Available targets:"
	@echo "  build         - Build the binary"
	@echo "  clean         - Remove build artifacts"
	@echo "  test          - Run tests"
	@echo "  install       - Install binary to GOPATH/bin"
	@echo "  dev           - Build with race detector"
	@echo "  fmt           - Format code"
	@echo "  lint          - Run linters"
	@echo "  deps          - Download and tidy dependencies"
	@echo "  example-config - Create example configuration"
	@echo "  help          - Show this help message"