#!/bin/bash

# Set colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[0;33m'
NC='\033[0m' # No Color

# Check if golangci-lint is installed
if ! command -v golangci-lint &> /dev/null; then
    echo -e "${RED}golangci-lint is not installed.${NC}"
    echo -e "Please install it first: ${YELLOW}https://golangci-lint.run/usage/install/${NC}"
    exit 1
fi

echo -e "${YELLOW}Running linter for all modules...${NC}"

# Lint gateway
echo -e "\n${YELLOW}Linting Gateway service...${NC}"
cd services/gateway
golangci-lint run --timeout=5m
GATEWAY_RESULT=$?
cd ../..

# Lint analyzer
echo -e "\n${YELLOW}Linting Analyzer service...${NC}"
cd services/analyzer
golangci-lint run --timeout=5m
ANALYZER_RESULT=$?
cd ../..

# Lint processor
echo -e "\n${YELLOW}Linting Processor service...${NC}"
cd services/processor
golangci-lint run --timeout=5m
PROCESSOR_RESULT=$?
cd ../..

# Lint shared packages
echo -e "\n${YELLOW}Linting Shared packages...${NC}"
cd pkg
golangci-lint run --timeout=5m
PKG_RESULT=$?
cd ..

# Check lint results
echo ""
if [ $GATEWAY_RESULT -eq 0 ]; then
    echo -e "${GREEN}✓ Gateway linting passed${NC}"
else
    echo -e "${RED}✗ Gateway linting failed${NC}"
fi

if [ $ANALYZER_RESULT -eq 0 ]; then
    echo -e "${GREEN}✓ Analyzer linting passed${NC}"
else
    echo -e "${RED}✗ Analyzer linting failed${NC}"
fi

if [ $PROCESSOR_RESULT -eq 0 ]; then
    echo -e "${GREEN}✓ Processor linting passed${NC}"
else
    echo -e "${RED}✗ Processor linting failed${NC}"
fi

if [ $PKG_RESULT -eq 0 ]; then
    echo -e "${GREEN}✓ Shared packages linting passed${NC}"
else
    echo -e "${RED}✗ Shared packages linting failed${NC}"
fi

# Return overall result
if [ $GATEWAY_RESULT -eq 0 ] && [ $ANALYZER_RESULT -eq 0 ] && [ $PROCESSOR_RESULT -eq 0 ] && [ $PKG_RESULT -eq 0 ]; then
    echo -e "\n${GREEN}All linting checks passed successfully!${NC}"
    exit 0
else
    echo -e "\n${RED}Some linting checks failed!${NC}"
    exit 1
fi