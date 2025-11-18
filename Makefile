.PHONY: build run test clean install lint fmt help

# Binary name
BINARY_NAME=tfpipboy
BINARY_PATH=./bin/$(BINARY_NAME)

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOFMT=$(GOCMD) fmt

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'

build: ## Build the application
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p bin
	$(GOBUILD) -o $(BINARY_PATH) ./cmd/tfpipboy

run: ## Run the application
	@echo "Running $(BINARY_NAME)..."
	$(GOCMD) run ./cmd/tfpipboy/main.go

test: ## Run all tests (unit + race + lint + security)
	@$(MAKE) test-unit
	@$(MAKE) test-race
	@$(MAKE) lint
	@$(MAKE) test-security

test-unit: ## Run unit tests
	@echo "Running unit tests..."
	$(GOTEST) -v ./...

test-race: ## Run tests with race detector
	@echo "Running tests with race detector..."
	$(GOTEST) -v -race ./...

test-coverage: ## Run tests with coverage
	@echo "Running tests with coverage..."
	$(GOTEST) -v -coverprofile=coverage.out -covermode=atomic ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	$(GOCMD) tool cover -func=coverage.out
	@echo "Coverage report generated: coverage.html"

test-security: ## Run security scans (gosec)
	@echo "Running security scans..."
	@if command -v gosec >/dev/null 2>&1; then \
		gosec -fmt=json -out=gosec-report.json ./...; \
		gosec ./...; \
	else \
		echo "gosec not installed. Installing..."; \
		go install github.com/securego/gosec/v2/cmd/gosec@latest; \
		gosec -fmt=json -out=gosec-report.json ./...; \
		gosec ./...; \
	fi

test-bench: ## Run benchmark tests
	@echo "Running benchmark tests..."
	$(GOTEST) -bench=. -benchmem ./...

clean: ## Clean build artifacts
	@echo "Cleaning..."
	$(GOCLEAN)
	rm -rf bin/
	rm -f coverage.out coverage.html

install: build ## Install the binary to $GOPATH/bin
	@echo "Installing $(BINARY_NAME)..."
	@cp $(BINARY_PATH) $(GOPATH)/bin/$(BINARY_NAME)
	@echo "Installed to $(GOPATH)/bin/$(BINARY_NAME)"

lint: ## Run linter (strict mode - fails on errors)
	@echo "Running linter..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run --timeout=5m ./...; \
	else \
		echo "golangci-lint not installed. Installing..."; \
		go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest; \
		golangci-lint run --timeout=5m ./...; \
	fi

lint-fix: ## Run linter and auto-fix issues
	@echo "Running linter with auto-fix..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run --fix --timeout=5m ./...; \
	else \
		echo "golangci-lint not installed. Installing..."; \
		go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest; \
		golangci-lint run --fix --timeout=5m ./...; \
	fi

fmt: ## Format code
	@echo "Formatting code..."
	$(GOFMT) ./...

tidy: ## Tidy go modules
	@echo "Tidying go modules..."
	$(GOMOD) tidy

deps: ## Download dependencies
	@echo "Downloading dependencies..."
	$(GOMOD) download

dev-setup: deps ## Setup development environment
	@echo "Setting up development environment..."
	@if ! command -v golangci-lint >/dev/null 2>&1; then \
		echo "Installing golangci-lint..."; \
		brew install golangci-lint; \
	fi
	@echo "Development environment ready!"

.DEFAULT_GOAL := help
