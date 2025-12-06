.PHONY: build build-all clean install test

VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-X main.version=$(VERSION)"

# Build for current platform
build:
	go build $(LDFLAGS) -o opencore .

# Build for all platforms
build-all: clean
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o build/opencore-windows-amd64.exe .
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o build/opencore-darwin-amd64 .
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o build/opencore-darwin-arm64 .
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o build/opencore-linux-amd64 .

# Clean build artifacts
clean:
	rm -rf build/
	rm -f opencore opencore.exe

# Install locally
install:
	go install $(LDFLAGS) .

# Run tests
test:
	go test -v ./...

# Run linter
lint:
	golangci-lint run

# Download dependencies
deps:
	go mod download
	go mod tidy

