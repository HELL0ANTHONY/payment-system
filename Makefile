.PHONY: all build test lint fmt clean deploy help

# ============================================================================
# VARIABLES
# ============================================================================
LAMBDAS_DIR := lambdas
SHARED_DIR := shared
LAMBDAS := $(shell ls -d $(LAMBDAS_DIR)/*/ 2>/dev/null | xargs -n1 basename)

# Colors for output
GREEN := \033[0;32m
YELLOW := \033[0;33m
RED := \033[0;31m
NC := \033[0m

# ============================================================================
# DEFAULT
# ============================================================================
all: lint test build

# ============================================================================
# WORKSPACE
# ============================================================================
.PHONY: workspace-sync
workspace-sync:
	@echo "$(GREEN)==> Syncing Go workspace...$(NC)"
	@go work sync

.PHONY: deps
deps:
	@echo "$(GREEN)==> Tidying all modules...$(NC)"
	@cd $(SHARED_DIR) && go mod tidy
	@for lambda in $(LAMBDAS); do \
		echo "  -> $$lambda"; \
		cd $(LAMBDAS_DIR)/$$lambda && go mod tidy && cd ../..; \
	done

# ============================================================================
# DEVELOPMENT (ALL LAMBDAS)
# ============================================================================
.PHONY: fmt
fmt:
	@echo "$(GREEN)==> Formatting all code...$(NC)"
	@for lambda in $(LAMBDAS); do \
		echo "  -> $$lambda"; \
		$(MAKE) -C $(LAMBDAS_DIR)/$$lambda fmt; \
	done

.PHONY: lint
lint:
	@echo "$(GREEN)==> Linting shared...$(NC)"
	@cd $(SHARED_DIR) && golangci-lint run ./...
	@echo "$(GREEN)==> Linting all lambdas...$(NC)"
	@for lambda in $(LAMBDAS); do \
		echo "  -> $$lambda"; \
		$(MAKE) -C $(LAMBDAS_DIR)/$$lambda lint; \
	done

.PHONY: lint-fix
lint-fix:
	@echo "$(GREEN)==> Fixing lint issues...$(NC)"
	@cd $(SHARED_DIR) && golangci-lint run --fix ./...
	@for lambda in $(LAMBDAS); do \
		$(MAKE) -C $(LAMBDAS_DIR)/$$lambda lint-fix; \
	done

# ============================================================================
# TESTING
# ============================================================================
.PHONY: test
test:
	@echo "$(GREEN)==> Testing shared...$(NC)"
	@cd $(SHARED_DIR) && go test -race -shuffle=on ./...
	@echo "$(GREEN)==> Testing all lambdas...$(NC)"
	@for lambda in $(LAMBDAS); do \
		echo "  -> $$lambda"; \
		$(MAKE) -C $(LAMBDAS_DIR)/$$lambda test; \
	done

.PHONY: test-v
test-v:
	@echo "$(GREEN)==> Testing all (verbose)...$(NC)"
	@cd $(SHARED_DIR) && go test -race -shuffle=on -v ./...
	@for lambda in $(LAMBDAS); do \
		$(MAKE) -C $(LAMBDAS_DIR)/$$lambda test-v; \
	done

.PHONY: coverage
coverage:
	@echo "$(GREEN)==> Running coverage for all...$(NC)"
	@for lambda in $(LAMBDAS); do \
		echo "  -> $$lambda"; \
		$(MAKE) -C $(LAMBDAS_DIR)/$$lambda coverage-report; \
	done

# ============================================================================
# BUILD
# ============================================================================
.PHONY: build
build:
	@echo "$(GREEN)==> Building all lambdas...$(NC)"
	@for lambda in $(LAMBDAS); do \
		echo "  -> $$lambda"; \
		$(MAKE) -C $(LAMBDAS_DIR)/$$lambda build; \
	done

.PHONY: zip
zip:
	@echo "$(GREEN)==> Packaging all lambdas...$(NC)"
	@for lambda in $(LAMBDAS); do \
		echo "  -> $$lambda"; \
		$(MAKE) -C $(LAMBDAS_DIR)/$$lambda zip; \
	done

# ============================================================================
# SINGLE LAMBDA OPERATIONS
# ============================================================================
.PHONY: build-%
build-%:
	@if [ -d "$(LAMBDAS_DIR)/$*" ]; then \
		echo "$(GREEN)==> Building $*...$(NC)"; \
		$(MAKE) -C $(LAMBDAS_DIR)/$* build; \
	else \
		echo "$(RED)==> Lambda '$*' not found$(NC)"; \
		exit 1; \
	fi

.PHONY: test-%
test-%:
	@if [ -d "$(LAMBDAS_DIR)/$*" ]; then \
		echo "$(GREEN)==> Testing $*...$(NC)"; \
		$(MAKE) -C $(LAMBDAS_DIR)/$* test; \
	else \
		echo "$(RED)==> Lambda '$*' not found$(NC)"; \
		exit 1; \
	fi

.PHONY: lint-%
lint-%:
	@if [ -d "$(LAMBDAS_DIR)/$*" ]; then \
		echo "$(GREEN)==> Linting $*...$(NC)"; \
		$(MAKE) -C $(LAMBDAS_DIR)/$* lint; \
	else \
		echo "$(RED)==> Lambda '$*' not found$(NC)"; \
		exit 1; \
	fi

.PHONY: deploy-%
deploy-%:
	@if [ -d "$(LAMBDAS_DIR)/$*" ]; then \
		echo "$(GREEN)==> Deploying $*...$(NC)"; \
		$(MAKE) -C $(LAMBDAS_DIR)/$* deploy profile=$(profile); \
	else \
		echo "$(RED)==> Lambda '$*' not found$(NC)"; \
		exit 1; \
	fi

.PHONY: coverage-%
coverage-%:
	@if [ -d "$(LAMBDAS_DIR)/$*" ]; then \
		echo "$(GREEN)==> Coverage for $*...$(NC)"; \
		$(MAKE) -C $(LAMBDAS_DIR)/$* coverage-html; \
	else \
		echo "$(RED)==> Lambda '$*' not found$(NC)"; \
		exit 1; \
	fi

# ============================================================================
# DEPLOYMENT
# ============================================================================
.PHONY: deploy
deploy:
	@echo "$(YELLOW)==> Deploying ALL lambdas...$(NC)"
	@echo "$(YELLOW)    Are you sure? Use 'deploy-<lambda>' for single deploy$(NC)"
	@for lambda in $(LAMBDAS); do \
		echo "  -> $$lambda"; \
		$(MAKE) -C $(LAMBDAS_DIR)/$$lambda deploy profile=$(profile); \
	done

# ============================================================================
# CLEANUP
# ============================================================================
.PHONY: clean
clean:
	@echo "$(GREEN)==> Cleaning all build artifacts...$(NC)"
	@for lambda in $(LAMBDAS); do \
		$(MAKE) -C $(LAMBDAS_DIR)/$$lambda clean; \
	done

.PHONY: clean-cache
clean-cache:
	@echo "$(GREEN)==> Cleaning Go caches...$(NC)"
	@go clean -cache -testcache -modcache

# ============================================================================
# UTILITIES
# ============================================================================
.PHONY: list
list:
	@echo "$(GREEN)Available lambdas:$(NC)"
	@for lambda in $(LAMBDAS); do \
		echo "  - $$lambda"; \
	done

.PHONY: check
check:
	@echo "$(GREEN)==> Checking project structure...$(NC)"
	@echo "Go version: $$(go version)"
	@echo "Workspace:"
	@go work edit -json | head -20
	@echo ""
	@echo "Lambdas found: $(LAMBDAS)"

# ============================================================================
# HELP
# ============================================================================
.PHONY: help
help:
	@echo "Payment System - Makefile"
	@echo ""
	@echo "$(GREEN)All Lambdas:$(NC)"
	@echo "  make all              Run lint, test, and build for all"
	@echo "  make deps             Tidy all Go modules"
	@echo "  make fmt              Format all code"
	@echo "  make lint             Lint shared + all lambdas"
	@echo "  make lint-fix         Auto-fix lint issues"
	@echo "  make test             Test shared + all lambdas"
	@echo "  make test-v           Test all (verbose)"
	@echo "  make coverage         Coverage report for all"
	@echo "  make build            Build all lambdas"
	@echo "  make zip              Package all lambdas"
	@echo "  make deploy           Deploy all (requires profile=<name>)"
	@echo "  make clean            Clean all build artifacts"
	@echo ""
	@echo "$(GREEN)Single Lambda:$(NC)"
	@echo "  make build-<name>     Build specific lambda"
	@echo "  make test-<name>      Test specific lambda"
	@echo "  make lint-<name>      Lint specific lambda"
	@echo "  make coverage-<name>  Coverage for specific lambda"
	@echo "  make deploy-<name>    Deploy specific lambda (requires profile=<name>)"
	@echo ""
	@echo "$(GREEN)Examples:$(NC)"
	@echo "  make build-payment-orchestrator"
	@echo "  make test-wallet-service"
	@echo "  make deploy-gateway-processor profile=dev"
	@echo ""
	@echo "$(GREEN)Utilities:$(NC)"
	@echo "  make list             List all lambdas"
	@echo "  make check            Check project structure"
	@echo "  make workspace-sync   Sync Go workspace"
	@echo "  make clean-cache      Clean Go caches"
