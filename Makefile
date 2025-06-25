.PHONY: all build test clean install run help verify fix install-tools

# Variables
BINARY_NAME=gemini-cli-ntfy
BUILD_DIR=build
CMD_DIR=cmd/gemini-cli-ntfy
VERSION?=dev
LDFLAGS=-ldflags "-X main.version=$(VERSION)"

# Default target
all: test build

# Build the binary
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./$(CMD_DIR)

# Run tests
test:
	@echo "Running tests..."
	go test -race -v ./...

# Clean build artifacts
clean:
	@echo "Cleaning..."
	rm -rf $(BUILD_DIR)
	go clean

# Install the binary to GOPATH/bin
install:
	@echo "Installing $(BINARY_NAME)..."
	go install $(LDFLAGS) ./$(CMD_DIR)

# Run the application
run: build
	@echo "Running $(BINARY_NAME)..."
	./$(BUILD_DIR)/$(BINARY_NAME)

# Install development tools
install-tools:
	@echo "Installing development tools..."
	go install golang.org/x/tools/cmd/goimports@latest
	go install honnef.co/go/tools/cmd/staticcheck@latest
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Fix code formatting and imports
fix:
	@echo "Fixing code..."
	go fmt ./...
	goimports -w .
	go mod tidy

# Verify code quality
verify: test
	@echo "Running static analysis..."
	go vet ./...
	staticcheck ./...
	golangci-lint run

# Cross-compile for multiple platforms
build-all:
	@echo "Building for multiple platforms..."
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 ./$(CMD_DIR)
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 ./$(CMD_DIR)
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 ./$(CMD_DIR)

# Help
help:
	@echo "Available targets:"
	@echo "  build      - Build the binary"
	@echo "  test       - Run tests"
	@echo "  clean      - Clean build artifacts"
	@echo "  install    - Install to GOPATH/bin"
	@echo "  run        - Build and run the application"
	@echo "  fix        - Fix code formatting and imports"
	@echo "  verify     - Run tests and static analysis"
	@echo "  build-all  - Cross-compile for multiple platforms"
	@echo "  help       - Show this help message"