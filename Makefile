# Purple Hailo - Go Runtime for Hailo-8 AI Accelerator
# Makefile

# Build configuration
BINARY_NAME := hailort
MODULE := github.com/anthropics/purple-hailo
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME := $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')
GO_VERSION := $(shell go version | cut -d' ' -f3)

# Directories
BIN_DIR := bin
BUILD_DIR := build
DIST_DIR := dist
CMD_DIR := cmd/hailort
EXAMPLES_DIR := examples

# Go configuration
GOFLAGS := -trimpath
LDFLAGS := -s -w \
	-X main.Version=$(VERSION) \
	-X main.BuildTime=$(BUILD_TIME) \
	-X main.GoVersion=$(GO_VERSION)

# Target platforms
PLATFORMS := linux/amd64 linux/arm64

# Test configuration
TEST_TIMEOUT := 5m
COVERAGE_FILE := coverage.out
COVERAGE_HTML := coverage.html

# Tools
GOLINT := golangci-lint
GOFUMPT := gofumpt

.PHONY: all
all: build

# ============================================================================
# Build targets
# ============================================================================

.PHONY: build
build: ## Build the binary for current platform
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BIN_DIR)
	go build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(BINARY_NAME) ./$(CMD_DIR)

.PHONY: build-linux-amd64
build-linux-amd64: ## Build for Linux amd64
	@echo "Building for linux/amd64..."
	@mkdir -p $(DIST_DIR)
	GOOS=linux GOARCH=amd64 go build $(GOFLAGS) -ldflags "$(LDFLAGS)" \
		-o $(DIST_DIR)/$(BINARY_NAME)-linux-amd64 ./$(CMD_DIR)

.PHONY: build-linux-arm64
build-linux-arm64: ## Build for Linux arm64 (Raspberry Pi 5)
	@echo "Building for linux/arm64..."
	@mkdir -p $(DIST_DIR)
	GOOS=linux GOARCH=arm64 go build $(GOFLAGS) -ldflags "$(LDFLAGS)" \
		-o $(DIST_DIR)/$(BINARY_NAME)-linux-arm64 ./$(CMD_DIR)

.PHONY: build-all
build-all: build-linux-amd64 build-linux-arm64 ## Build for all platforms

.PHONY: build-examples
build-examples: ## Build all examples
	@echo "Building examples..."
	@mkdir -p $(BIN_DIR)
	@for dir in $(EXAMPLES_DIR)/*/; do \
		if [ -f "$$dir/main.go" ]; then \
			name=$$(basename $$dir); \
			echo "  Building example: $$name"; \
			go build $(GOFLAGS) -o $(BIN_DIR)/$$name ./$$dir; \
		fi \
	done

.PHONY: install
install: build ## Install binary to GOPATH/bin
	@echo "Installing $(BINARY_NAME)..."
	cp $(BIN_DIR)/$(BINARY_NAME) $(GOPATH)/bin/

# ============================================================================
# Test targets
# ============================================================================

.PHONY: test
test: ## Run unit tests
	@echo "Running unit tests..."
	go test -tags=unit -race -timeout $(TEST_TIMEOUT) ./...

.PHONY: test-verbose
test-verbose: ## Run unit tests with verbose output
	@echo "Running unit tests (verbose)..."
	go test -tags=unit -race -v -timeout $(TEST_TIMEOUT) ./...

.PHONY: test-integration
test-integration: ## Run integration tests (requires Hailo driver)
	@echo "Running integration tests..."
	go test -tags=integration -race -timeout $(TEST_TIMEOUT) ./...

.PHONY: test-hardware
test-hardware: ## Run hardware tests (requires Hailo device)
	@echo "Running hardware tests..."
	go test -tags=unit,integration,hardware -race -timeout $(TEST_TIMEOUT) ./...

.PHONY: test-all
test-all: ## Run all tests
	@echo "Running all tests..."
	go test -tags=unit,integration -race -timeout $(TEST_TIMEOUT) ./...

.PHONY: test-short
test-short: ## Run short tests only
	@echo "Running short tests..."
	go test -tags=unit -short -race -timeout $(TEST_TIMEOUT) ./...

.PHONY: coverage
coverage: ## Run tests with coverage
	@echo "Running tests with coverage..."
	go test -tags=unit -race -coverprofile=$(COVERAGE_FILE) -covermode=atomic -timeout $(TEST_TIMEOUT) ./...
	@echo "Coverage report: $(COVERAGE_FILE)"

.PHONY: coverage-html
coverage-html: coverage ## Generate HTML coverage report
	@echo "Generating HTML coverage report..."
	go tool cover -html=$(COVERAGE_FILE) -o $(COVERAGE_HTML)
	@echo "Coverage HTML: $(COVERAGE_HTML)"

.PHONY: coverage-func
coverage-func: coverage ## Show function coverage
	go tool cover -func=$(COVERAGE_FILE)

# ============================================================================
# Benchmark targets
# ============================================================================

.PHONY: bench
bench: ## Run benchmarks
	@echo "Running benchmarks..."
	go test -tags=benchmark -bench=. -benchmem -run=^$$ ./...

.PHONY: bench-cpu
bench-cpu: ## Run benchmarks with CPU profiling
	@echo "Running benchmarks with CPU profiling..."
	@mkdir -p $(BUILD_DIR)
	go test -tags=benchmark -bench=. -benchmem -cpuprofile=$(BUILD_DIR)/cpu.prof -run=^$$ ./...

.PHONY: bench-mem
bench-mem: ## Run benchmarks with memory profiling
	@echo "Running benchmarks with memory profiling..."
	@mkdir -p $(BUILD_DIR)
	go test -tags=benchmark -bench=. -benchmem -memprofile=$(BUILD_DIR)/mem.prof -run=^$$ ./...

# ============================================================================
# Code quality targets
# ============================================================================

.PHONY: lint
lint: ## Run linter
	@echo "Running linter..."
	$(GOLINT) run ./...

.PHONY: lint-fix
lint-fix: ## Run linter with auto-fix
	@echo "Running linter with auto-fix..."
	$(GOLINT) run --fix ./...

.PHONY: fmt
fmt: ## Format code
	@echo "Formatting code..."
	go fmt ./...

.PHONY: fumpt
fumpt: ## Format code with gofumpt (stricter)
	@echo "Formatting code with gofumpt..."
	$(GOFUMPT) -l -w .

.PHONY: vet
vet: ## Run go vet
	@echo "Running go vet..."
	go vet ./...

.PHONY: check
check: fmt vet lint ## Run all checks (fmt, vet, lint)

# ============================================================================
# Dependency targets
# ============================================================================

.PHONY: deps
deps: ## Download dependencies
	@echo "Downloading dependencies..."
	go mod download

.PHONY: deps-tidy
deps-tidy: ## Tidy dependencies
	@echo "Tidying dependencies..."
	go mod tidy

.PHONY: deps-verify
deps-verify: ## Verify dependencies
	@echo "Verifying dependencies..."
	go mod verify

.PHONY: deps-update
deps-update: ## Update dependencies
	@echo "Updating dependencies..."
	go get -u ./...
	go mod tidy

# ============================================================================
# Generation targets
# ============================================================================

.PHONY: generate
generate: ## Run go generate
	@echo "Running go generate..."
	go generate ./...

.PHONY: proto
proto: ## Generate protobuf code
	@echo "Generating protobuf code..."
	protoc --go_out=. --go_opt=paths=source_relative pkg/hef/proto/*.proto

# ============================================================================
# Documentation targets
# ============================================================================

.PHONY: docs
docs: ## Generate documentation
	@echo "Generating documentation..."
	go doc -all ./... > $(BUILD_DIR)/docs.txt

.PHONY: godoc
godoc: ## Start godoc server
	@echo "Starting godoc server at http://localhost:6060..."
	godoc -http=:6060

# ============================================================================
# Clean targets
# ============================================================================

.PHONY: clean
clean: ## Clean build artifacts
	@echo "Cleaning..."
	rm -rf $(BIN_DIR)
	rm -rf $(BUILD_DIR)
	rm -rf $(DIST_DIR)
	rm -f $(COVERAGE_FILE)
	rm -f $(COVERAGE_HTML)

.PHONY: clean-all
clean-all: clean ## Clean everything including cache
	@echo "Cleaning all..."
	go clean -cache -testcache -modcache

# ============================================================================
# Development targets
# ============================================================================

.PHONY: dev
dev: deps build test ## Development build (deps, build, test)

.PHONY: ci
ci: deps check test-all coverage-func ## CI pipeline (deps, check, test, coverage)

.PHONY: release
release: clean check test-all build-all ## Release build

# ============================================================================
# Tool installation
# ============================================================================

.PHONY: tools
tools: ## Install development tools
	@echo "Installing development tools..."
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install mvdan.cc/gofumpt@latest
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest

# ============================================================================
# Docker targets
# ============================================================================

.PHONY: docker-build
docker-build: ## Build Docker image
	@echo "Building Docker image..."
	docker build -t purple-hailo:$(VERSION) .

.PHONY: docker-build-arm64
docker-build-arm64: ## Build Docker image for arm64
	@echo "Building Docker image for arm64..."
	docker buildx build --platform linux/arm64 -t purple-hailo:$(VERSION)-arm64 .

# ============================================================================
# Raspberry Pi targets
# ============================================================================

.PHONY: deploy-rpi
deploy-rpi: build-linux-arm64 ## Deploy to Raspberry Pi (requires RPI_HOST env var)
ifndef RPI_HOST
	$(error RPI_HOST is not set. Usage: RPI_HOST=pi@raspberrypi.local make deploy-rpi)
endif
	@echo "Deploying to $(RPI_HOST)..."
	scp $(DIST_DIR)/$(BINARY_NAME)-linux-arm64 $(RPI_HOST):~/$(BINARY_NAME)
	ssh $(RPI_HOST) "chmod +x ~/$(BINARY_NAME)"

.PHONY: test-rpi
test-rpi: ## Run tests on Raspberry Pi (requires RPI_HOST env var)
ifndef RPI_HOST
	$(error RPI_HOST is not set. Usage: RPI_HOST=pi@raspberrypi.local make test-rpi)
endif
	@echo "Running tests on $(RPI_HOST)..."
	ssh $(RPI_HOST) "cd ~/purple-hailo && go test -tags=hardware ./..."

# ============================================================================
# Help
# ============================================================================

.PHONY: help
help: ## Show this help
	@echo "Purple Hailo - Go Runtime for Hailo-8 AI Accelerator"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'
	@echo ""
	@echo "Examples:"
	@echo "  make build              Build for current platform"
	@echo "  make test               Run unit tests"
	@echo "  make coverage-html      Generate coverage report"
	@echo "  make bench              Run benchmarks"
	@echo "  make build-linux-arm64  Build for Raspberry Pi 5"
	@echo "  make deploy-rpi RPI_HOST=pi@raspberrypi.local"

.DEFAULT_GOAL := help
