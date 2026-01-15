.PHONY: all build test lint fmt clean deploy help changelog

# ============================================================================
# VARIABLES
# ============================================================================
LAMBDA_NAME := $(shell grep -A3 'artifactId:' lambda.yaml | tail -n1 | awk '{ print $$2}')
BINARY_NAME := bootstrap
BUILD_DIR := target
ARCH := arm64
REGION := us-east-1
COVERAGE_FILE := coverage.out
COVERAGE_HTML := coverage.html

# Build flags
LDFLAGS := -s -w
BUILD_FLAGS := -trimpath -ldflags "$(LDFLAGS)"

# Tools
GOLANGCI_LINT := golangci-lint
GOLANGCI_LINT_VERSION := v2.7.2

# ============================================================================
# DEFAULT
# ============================================================================
all: lint test build

# ============================================================================
# DEVELOPMENT
# ============================================================================
.PHONY: deps
deps:
	@echo "==> Tidying modules..."
	@go mod tidy
	@go mod verify

.PHONY: fmt
fmt:
	@echo "==> Formatting code..."
	@$(GOLANGCI_LINT) fmt ./...

.PHONY: lint
lint: lint-install
	@echo "==> Running linters..."
	@$(GOLANGCI_LINT) run ./...

.PHONY: lint-fix
lint-fix: lint-install
	@echo "==> Running linters with auto-fix..."
	@$(GOLANGCI_LINT) run --fix ./...

.PHONY: lint-install
lint-install:
	@if ! command -v $(GOLANGCI_LINT) &> /dev/null || \
		[ "$($(GOLANGCI_LINT) --version | grep -oE 'v[0-9]+\.[0-9]+\.[0-9]+')" != "$(GOLANGCI_LINT_VERSION)" ]; then \
		echo "==> Installing golangci-lint $(GOLANGCI_LINT_VERSION)..."; \
		go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION); \
	fi

# ============================================================================
# TESTING
# ============================================================================
.PHONY: test
test:
	@echo "==> Running tests..."
	@go test -race -shuffle=on ./...

.PHONY: test-v
test-v:
	@echo "==> Running tests (verbose)..."
	@go test -race -shuffle=on -v ./...

.PHONY: coverage
coverage:
	@echo "==> Running tests with coverage..."
	@go test -race -shuffle=on -coverprofile=$(COVERAGE_FILE) -covermode=atomic -coverpkg=./... ./...
	@grep -v "_mock.go\|_test.go" $(COVERAGE_FILE) > $(COVERAGE_FILE).tmp && mv $(COVERAGE_FILE).tmp $(COVERAGE_FILE)

.PHONY: coverage-report
coverage-report: coverage
	@echo "==> Coverage summary:"
	@go tool cover -func=$(COVERAGE_FILE) | tail -1

.PHONY: coverage-html
coverage-html: coverage
	@echo "==> Generating HTML coverage report..."
	@go tool cover -html=$(COVERAGE_FILE) -o $(COVERAGE_HTML)
	@echo "==> Report generated: $(COVERAGE_HTML)"

# ============================================================================
# BUILD
# ============================================================================
.PHONY: build
build: deps
	@echo "==> Building $(BINARY_NAME) for linux/$(ARCH)..."
	@mkdir -p $(BUILD_DIR)
	@CGO_ENABLED=0 GOOS=linux GOARCH=$(ARCH) go build $(BUILD_FLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/main.go
	@echo "==> Binary size: $$(du -h $(BUILD_DIR)/$(BINARY_NAME) | cut -f1)"

.PHONY: zip
zip: build
	@echo "==> Creating deployment package..."
	@cd $(BUILD_DIR) && zip -q $(LAMBDA_NAME).zip $(BINARY_NAME)
	@echo "==> Package size: $$(du -h $(BUILD_DIR)/$(LAMBDA_NAME).zip | cut -f1)"

.PHONY: hash
hash: zip
	@echo "==> Generating SHA256 hash..."
	@cd $(BUILD_DIR) && openssl dgst -sha256 -binary $(LAMBDA_NAME).zip | openssl enc -base64 > $(LAMBDA_NAME).hash
	@echo "==> Hash: $$(cat $(BUILD_DIR)/$(LAMBDA_NAME).hash)"

# ============================================================================
# DEPLOYMENT
# ============================================================================
.PHONY: deploy
deploy: lint test zip
	@echo "==> Deploying $(LAMBDA_NAME) to AWS..."
	@aws lambda update-function-code \
		--function-name $(LAMBDA_NAME) \
		--zip-file fileb://$(BUILD_DIR)/$(LAMBDA_NAME).zip \
		--region $(REGION) \
		--profile $(profile) \
		--output table

# ============================================================================
# CLEANUP
# ============================================================================
.PHONY: clean
clean:
	@echo "==> Cleaning build artifacts..."
	@rm -rf $(BUILD_DIR)
	@rm -f $(COVERAGE_FILE) $(COVERAGE_HTML)

.PHONY: clean-cache
clean-cache:
	@echo "==> Cleaning Go caches..."
	@go clean -cache -testcache -modcache

.PHONY: changelog

changelog:
	@chmod +x ../../scripts/generate-changelog.sh
	@../../scripts/generate-changelog.sh

# ============================================================================
# HELP
# ============================================================================
.PHONY: help
help:
	@echo "Usage: make [target]"
	@echo ""
	@echo "Development:"
	@echo "  deps            Tidy and verify modules"
	@echo "  fmt             Format code using golangci-lint"
	@echo "  lint            Run linters"
	@echo "  lint-fix        Run linters with auto-fix"
	@echo ""
	@echo "Testing:"
	@echo "  test            Run tests with race detection"
	@echo "  test-v          Run tests verbose"
	@echo "  coverage        Run tests with coverage"
	@echo "  coverage-report Show coverage summary"
	@echo "  coverage-html   Generate HTML coverage report"
	@echo ""
	@echo "Build:"
	@echo "  build           Build lambda binary"
	@echo "  zip             Create deployment package"
	@echo "  hash            Generate package hash"
	@echo ""
	@echo "Deployment:"
	@echo "  deploy          Deploy to AWS (requires profile=<name>)"
	@echo ""
	@echo "Cleanup:"
	@echo "  clean           Remove build artifacts"
	@echo "  clean-cache     Clean Go caches"
