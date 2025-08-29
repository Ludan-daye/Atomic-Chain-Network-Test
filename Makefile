# NetCrate Makefile
# Provides common build and release tasks

.PHONY: help build test clean release snapshot install deps lint format tidy version

# Default target
help: ## Show this help message
	@echo "NetCrate Build System"
	@echo "===================="
	@echo ""
	@echo "Available targets:"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-12s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# Build information
BUILD_TIME := $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
GIT_VERSION := $(shell git describe --tags --abbrev=0 2>/dev/null || echo "dev")
LDFLAGS := -ldflags="-s -w -X github.com/netcrate/netcrate/internal/version.Version=$(GIT_VERSION) -X github.com/netcrate/netcrate/internal/version.Commit=$(GIT_COMMIT) -X github.com/netcrate/netcrate/internal/version.Date=$(BUILD_TIME) -X github.com/netcrate/netcrate/internal/version.BuiltBy=make"

# Go build settings
GOOS := $(shell go env GOOS)
GOARCH := $(shell go env GOARCH)
CGO_ENABLED := 0

deps: ## Install dependencies
	@echo "Installing dependencies..."
	go mod download
	go mod tidy

build: deps ## Build the main netcrate binary
	@echo "Building netcrate for $(GOOS)/$(GOARCH)..."
	CGO_ENABLED=$(CGO_ENABLED) go build $(LDFLAGS) -o netcrate ./cmd/netcrate

build-simple: deps ## Build the simple netcrate binary
	@echo "Building netcrate-simple for $(GOOS)/$(GOARCH)..."
	CGO_ENABLED=$(CGO_ENABLED) go build $(LDFLAGS) -o netcrate-simple ./cmd/netcrate-simple

test: ## Run tests
	@echo "Running tests..."
	go test -v ./...

test-integration: build ## Run integration tests
	@echo "Running integration tests..."
	@if [ -f "./test_rate_profiles.go" ]; then \
		echo "Testing rate profiles..."; \
		go run test_rate_profiles.go; \
	fi
	@if [ -f "./test_privilege_fallback.go" ]; then \
		echo "Testing privilege fallback..."; \
		go run test_privilege_fallback.go; \
	fi
	@if [ -f "./test_compliance.go" ]; then \
		echo "Testing compliance system..."; \
		go run test_compliance.go --auto-yes; \
	fi

lint: ## Run linter
	@echo "Running linter..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not found, running go vet..."; \
		go vet ./...; \
	fi

format: ## Format code
	@echo "Formatting code..."
	go fmt ./...

tidy: ## Tidy modules
	@echo "Tidying modules..."
	go mod tidy

clean: ## Clean build artifacts
	@echo "Cleaning build artifacts..."
	rm -f netcrate netcrate-simple
	rm -rf dist/

version: ## Show version information
	@echo "Version: $(GIT_VERSION)"
	@echo "Commit: $(GIT_COMMIT)"
	@echo "Build time: $(BUILD_TIME)"
	@echo "Platform: $(GOOS)/$(GOARCH)"

# GoReleaser targets
install-goreleaser: ## Install GoReleaser
	@echo "Installing GoReleaser..."
	@if command -v brew >/dev/null 2>&1; then \
		brew install goreleaser/tap/goreleaser; \
	else \
		echo "Please install GoReleaser manually: https://goreleaser.com/install/"; \
	fi

snapshot: clean ## Build snapshot release (no git tags required)
	@echo "Building snapshot release..."
	@if command -v goreleaser >/dev/null 2>&1; then \
		goreleaser release --snapshot --rm-dist; \
	else \
		echo "GoReleaser not found. Run 'make install-goreleaser' first."; \
		exit 1; \
	fi

release: clean ## Build production release (requires git tag)
	@echo "Building production release..."
	@if [ -z "$$(git tag --points-at HEAD)" ]; then \
		echo "Error: No git tag found at HEAD. Create a tag first:"; \
		echo "  git tag -a v0.1.0 -m 'Release v0.1.0'"; \
		exit 1; \
	fi
	@if command -v goreleaser >/dev/null 2>&1; then \
		goreleaser release --rm-dist; \
	else \
		echo "GoReleaser not found. Run 'make install-goreleaser' first."; \
		exit 1; \
	fi

install: build ## Install netcrate to GOPATH/bin
	@echo "Installing netcrate..."
	go install $(LDFLAGS) ./cmd/netcrate

# Docker targets
docker-build: ## Build Docker image
	@echo "Building Docker image..."
	docker build -t netcrate:latest .

docker-run: docker-build ## Run netcrate in Docker
	@echo "Running netcrate in Docker..."
	docker run --rm -it netcrate:latest

# Development targets
dev-setup: deps ## Set up development environment
	@echo "Setting up development environment..."
	@if ! command -v golangci-lint >/dev/null 2>&1; then \
		echo "Installing golangci-lint..."; \
		curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $$(go env GOPATH)/bin; \
	fi
	@echo "Development environment ready!"

check: format lint test ## Run all checks (format, lint, test)

all: clean format lint test build ## Run full build pipeline

# Platform-specific builds
build-linux: deps ## Build for Linux
	@echo "Building for Linux..."
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build $(LDFLAGS) -o netcrate-linux-amd64 ./cmd/netcrate

build-windows: deps ## Build for Windows
	@echo "Building for Windows..."
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build $(LDFLAGS) -o netcrate-windows-amd64.exe ./cmd/netcrate

build-macos: deps ## Build for macOS
	@echo "Building for macOS..."
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build $(LDFLAGS) -o netcrate-darwin-amd64 ./cmd/netcrate
	GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build $(LDFLAGS) -o netcrate-darwin-arm64 ./cmd/netcrate

build-all: build-linux build-windows build-macos ## Build for all platforms