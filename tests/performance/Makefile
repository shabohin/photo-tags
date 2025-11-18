.PHONY: help all k6 benchmarks clean install-k6 install-tools test

# Default target
.DEFAULT_GOAL := help

# Colors for output
GREEN  := $(shell tput -Txterm setaf 2)
YELLOW := $(shell tput -Txterm setaf 3)
WHITE  := $(shell tput -Txterm setaf 7)
RESET  := $(shell tput -Txterm sgr0)

# Help target
help: ## Show this help message
	@echo ''
	@echo 'Usage:'
	@echo '  ${YELLOW}make${RESET} ${GREEN}<target>${RESET}'
	@echo ''
	@echo 'Targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  ${YELLOW}%-20s${GREEN}%s${RESET}\n", $$1, $$2}' $(MAKEFILE_LIST)
	@echo ''

all: ## Run all performance tests
	@echo "${GREEN}Running all performance tests...${RESET}"
	./run-all-tests.sh

k6: ## Run k6 load tests only
	@echo "${GREEN}Running k6 load tests...${RESET}"
	./run-k6-tests.sh all

k6-gateway: ## Run Gateway load test only
	@echo "${GREEN}Running Gateway load test...${RESET}"
	./run-k6-tests.sh gateway

k6-latency: ## Run latency test only
	@echo "${GREEN}Running latency test...${RESET}"
	./run-k6-tests.sh latency

k6-throughput: ## Run throughput test only
	@echo "${GREEN}Running throughput test...${RESET}"
	./run-k6-tests.sh throughput

k6-resource: ## Run resource monitor test only
	@echo "${GREEN}Running resource monitor test...${RESET}"
	./run-k6-tests.sh resource

benchmarks: ## Run Go benchmarks only
	@echo "${GREEN}Running Go benchmarks...${RESET}"
	./run-benchmarks.sh all

bench-exiftool: ## Run ExifTool benchmarks only
	@echo "${GREEN}Running ExifTool benchmarks...${RESET}"
	./run-benchmarks.sh exiftool

bench-minio: ## Run MinIO benchmarks only
	@echo "${GREEN}Running MinIO benchmarks...${RESET}"
	./run-benchmarks.sh minio

test: all ## Alias for 'all' target

quick: ## Run quick tests (reduced duration)
	@echo "${GREEN}Running quick performance tests...${RESET}"
	BENCH_TIME=3s BENCH_COUNT=1 ./run-all-tests.sh

clean: ## Clean test reports and artifacts
	@echo "${GREEN}Cleaning test reports...${RESET}"
	rm -rf reports/
	rm -rf benchmarks/*.test
	rm -rf benchmarks/*.prof
	@echo "${GREEN}✓ Cleaned${RESET}"

install-k6: ## Install k6 (macOS only)
	@echo "${GREEN}Installing k6...${RESET}"
	@if command -v brew >/dev/null 2>&1; then \
		brew install k6; \
		echo "${GREEN}✓ k6 installed${RESET}"; \
	else \
		echo "${YELLOW}Warning: Homebrew not found. Please install k6 manually.${RESET}"; \
		echo "Visit: https://k6.io/docs/getting-started/installation/"; \
		exit 1; \
	fi

install-exiftool: ## Install ExifTool (macOS only)
	@echo "${GREEN}Installing ExifTool...${RESET}"
	@if command -v brew >/dev/null 2>&1; then \
		brew install exiftool; \
		echo "${GREEN}✓ ExifTool installed${RESET}"; \
	else \
		echo "${YELLOW}Warning: Homebrew not found. Please install ExifTool manually.${RESET}"; \
		echo "macOS: brew install exiftool"; \
		echo "Linux: sudo apt-get install libimage-exiftool-perl"; \
		exit 1; \
	fi

install-tools: install-k6 install-exiftool ## Install all testing tools (macOS only)
	@echo "${GREEN}✓ All tools installed${RESET}"

deps: ## Download Go dependencies
	@echo "${GREEN}Downloading Go dependencies...${RESET}"
	cd benchmarks && go mod download
	@echo "${GREEN}✓ Dependencies downloaded${RESET}"

check: ## Check if all required tools are installed
	@echo "${GREEN}Checking required tools...${RESET}"
	@command -v go >/dev/null 2>&1 || { echo "${YELLOW}✗ Go not found${RESET}"; exit 1; }
	@echo "${GREEN}✓ Go found${RESET}"
	@command -v k6 >/dev/null 2>&1 || echo "${YELLOW}✗ k6 not found (optional for Go benchmarks)${RESET}"
	@command -v k6 >/dev/null 2>&1 && echo "${GREEN}✓ k6 found${RESET}" || true
	@command -v exiftool >/dev/null 2>&1 || echo "${YELLOW}✗ exiftool not found (optional)${RESET}"
	@command -v exiftool >/dev/null 2>&1 && echo "${GREEN}✓ exiftool found${RESET}" || true
	@echo "${GREEN}Tool check complete${RESET}"

view: ## Open the consolidated report in browser
	@echo "${GREEN}Opening test reports...${RESET}"
	@if [ -f reports/index.html ]; then \
		if command -v open >/dev/null 2>&1; then \
			open reports/index.html; \
		elif command -v xdg-open >/dev/null 2>&1; then \
			xdg-open reports/index.html; \
		else \
			echo "${YELLOW}Please open reports/index.html manually${RESET}"; \
		fi \
	else \
		echo "${YELLOW}No reports found. Run 'make all' first.${RESET}"; \
		exit 1; \
	fi

stats: ## Show statistics about test reports
	@echo "${GREEN}Test Report Statistics${RESET}"
	@echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
	@if [ -d reports ]; then \
		echo "Total reports: $$(find reports -name '*.html' | wc -l)"; \
		echo "k6 reports:    $$(find reports -name '*-summary.html' | wc -l)"; \
		echo "Go reports:    $$(find reports -name '*-benchmark.html' | wc -l)"; \
		echo "Report size:   $$(du -sh reports 2>/dev/null | cut -f1)"; \
		echo "Latest test:   $$(ls -lt reports/*.html 2>/dev/null | head -1 | awk '{print $$6, $$7, $$8}')"; \
	else \
		echo "${YELLOW}No reports directory found${RESET}"; \
	fi
	@echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

init: ## Initialize performance test environment
	@echo "${GREEN}Initializing performance test environment...${RESET}"
	@chmod +x *.sh
	@mkdir -p reports data/images
	@echo "${GREEN}✓ Environment initialized${RESET}"
	@$(MAKE) check
