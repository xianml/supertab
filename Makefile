.PHONY: build install clean test fmt vet deps

# Build variables
BINARY_NAME=sug
BUILD_DIR=build
INSTALL_DIR=/usr/local/bin

# Go build flags
LDFLAGS=-ldflags "-s -w"

# Build the binary
build:
	mkdir -p $(BUILD_DIR)
	go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) .

# Install the binary to system path
install: build
	sudo cp $(BUILD_DIR)/$(BINARY_NAME) $(INSTALL_DIR)/

# Clean build artifacts
clean:
	rm -rf $(BUILD_DIR)

# Run tests
test:
	go test -v ./...

# Format Go code
fmt:
	go fmt ./...

# Run go vet
vet:
	go vet ./...

# Download dependencies
deps:
	go mod download
	go mod tidy

# Development build with debug info
build-dev:
	mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/$(BINARY_NAME) .

# Run linting checks
lint: fmt vet

# Full development workflow
dev: deps lint test build-dev

# Create release build for multiple platforms
release:
	mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 .
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 .
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 .
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe . 