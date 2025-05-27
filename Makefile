# Photo Tags Service Makefile

# Variables
GOCMD = go
GOLANGCI_LINT = golangci-lint
DOCKER_COMPOSE = docker compose

# Colors for output
GREEN := $(shell tput -Txterm setaf 2)
YELLOW := $(shell tput -Txterm setaf 3)
RED := $(shell tput -Txterm setaf 1)
RESET := $(shell tput -Txterm sgr0)

.PHONY: help
help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  $(YELLOW)%-20s$(RESET) %s\n", $$1, $$2}' $(MAKEFILE_LIST)

##@ Code Quality

.PHONY: lint
lint: ## Run linter on all modules
	@echo "$(YELLOW)Running linter...$(RESET)"
	@./scripts/lint.sh

.PHONY: lint-fix
lint-fix: ## Run linter with auto-fix
	@echo "$(YELLOW)Running linter with auto-fix...$(RESET)"
	@cd services/gateway && $(GOLANGCI_LINT) run --fix --timeout=5m
	@cd services/analyzer && $(GOLANGCI_LINT) run --fix --timeout=5m
	@cd services/processor && $(GOLANGCI_LINT) run --fix --timeout=5m
	@cd pkg && $(GOLANGCI_LINT) run --fix --timeout=5m
	@echo "$(GREEN)Linting completed with auto-fix!$(RESET)"

.PHONY: fmt
fmt: ## Format all Go files using golangci-lint v2
	@echo "$(YELLOW)Formatting Go files with golangci-lint...$(RESET)"
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint fmt; \
	else \
		echo "$(YELLOW)golangci-lint not found, using fallback...$(RESET)"; \
		find . -name "*.go" -not -path "./vendor/*" -exec gofmt -w {} \;; \
		find . -name "*.go" -not -path "./vendor/*" -exec goimports -w -local github.com/shabohin/photo-tags {} \;; \
	fi
	@echo "$(GREEN)Go files formatted!$(RESET)"

.PHONY: test
test: ## Run all tests
	@echo "$(YELLOW)Running tests...$(RESET)"
	@./scripts/test.sh

.PHONY: check
check: ## Run all quality checks (tests + linting)
	@echo "$(YELLOW)Running all quality checks...$(RESET)"
	@./scripts/check.sh

.PHONY: pre-commit
pre-commit: fmt lint test ## Run pre-commit checks (format, lint, test)
	@echo "$(GREEN)Pre-commit checks completed successfully!$(RESET)"

.PHONY: build
build: ## Build all services
	@echo "$(YELLOW)Building services...$(RESET)"
	@./scripts/build.sh

.PHONY: start
start: ## Start all services
	@echo "$(YELLOW)Starting services...$(RESET)"
	@./scripts/start.sh

.PHONY: stop
stop: ## Stop all services
	@echo "$(YELLOW)Stopping services...$(RESET)"
	@./scripts/stop.sh

.PHONY: install-hooks
install-hooks: ## Install Git pre-commit hooks
	@echo "$(YELLOW)Installing Git pre-commit hooks...$(RESET)"
	@mkdir -p .git/hooks
	@cp scripts/pre-commit .git/hooks/pre-commit
	@chmod +x .git/hooks/pre-commit
	@echo "$(GREEN)Git pre-commit hooks installed!$(RESET)"

.PHONY: version
version: ## Show Go and tool versions
	@echo "$(YELLOW)Tool Versions:$(RESET)"
	@$(GOCMD) version
	@if command -v $(GOLANGCI_LINT) >/dev/null 2>&1; then \
		$(GOLANGCI_LINT) version; \
	else \
		echo "golangci-lint: $(RED)not installed$(RESET)"; \
	fi
##@ Development

.PHONY: install-tools
install-tools: ## Install development tools
	@echo "$(YELLOW)Installing development tools...$(RESET)"
	@./scripts/install-golangci-lint.sh
	@echo "$(GREEN)Development tools installed!$(RESET)"

.PHONY: migrate-config
migrate-config: ## Migrate golangci-lint config from v1 to v2
	@echo "$(YELLOW)Migrating golangci-lint configuration...$(RESET)"
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint migrate --format yaml; \
		echo "$(GREEN)Configuration migrated to v2!$(RESET)"; \
	else \
		echo "$(RED)golangci-lint not found. Run 'make install-tools' first.$(RESET)"; \
	fi

.PHONY: deps
deps: ## Download and tidy dependencies
	@echo "$(YELLOW)Downloading dependencies...$(RESET)"
	@cd services/gateway && $(GOCMD) mod download && $(GOCMD) mod tidy
	@cd services/analyzer && $(GOCMD) mod download && $(GOCMD) mod tidy
	@cd services/processor && $(GOCMD) mod download && $(GOCMD) mod tidy
	@cd pkg && $(GOCMD) mod download && $(GOCMD) mod tidy
	@echo "$(GREEN)Dependencies updated successfully!$(RESET)"

.PHONY: deps-clean
deps-clean: ## Clean and reinstall dependencies
	@echo "$(YELLOW)Cleaning dependencies...$(RESET)"
	@cd services/gateway && $(GOCMD) clean -modcache && $(GOCMD) mod download && $(GOCMD) mod tidy
	@cd services/analyzer && $(GOCMD) clean -modcache && $(GOCMD) mod download && $(GOCMD) mod tidy
	@cd services/processor && $(GOCMD) clean -modcache && $(GOCMD) mod download && $(GOCMD) mod tidy
	@cd pkg && $(GOCMD) clean -modcache && $(GOCMD) mod download && $(GOCMD) mod tidy
	@echo "$(GREEN)Dependencies cleaned and reinstalled!$(RESET)"
