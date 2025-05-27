#!/bin/bash

# Set colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Check if golangci-lint is installed
if ! command -v golangci-lint &> /dev/null; then
    echo -e "${RED}golangci-lint is not installed.${NC}"
    echo -e "Please install it first: ${YELLOW}https://golangci-lint.run/usage/install/${NC}"
    exit 1
fi

# Parse command line arguments
FIX_MODE=false
if [[ "$1" == "--fix" || "$1" == "-f" ]]; then
    FIX_MODE=true
    echo -e "${BLUE}Running linter in FIX mode - will automatically fix issues where possible...${NC}"
else
    echo -e "${YELLOW}Running linter in CHECK mode...${NC}"
    echo -e "${BLUE}Tip: Use ${YELLOW}./scripts/lint.sh --fix${BLUE} to automatically fix issues${NC}"
fi

echo -e "${YELLOW}Processing all modules...${NC}"

# Function to run linter with or without fix
run_linter() {
    local service_name=$1
    local service_path=$2
    
    echo -e "\n${YELLOW}Processing ${service_name}...${NC}"
    cd "$service_path"
    
    if [ "$FIX_MODE" = true ]; then
        echo -e "${BLUE}Applying automatic fixes...${NC}"
        golangci-lint run --fix --timeout=5m
        local result=$?
        if [ $result -eq 0 ]; then
            echo -e "${GREEN}✓ ${service_name} - fixes applied successfully${NC}"
        else
            echo -e "${YELLOW}⚠ ${service_name} - some issues remain after auto-fix${NC}"
        fi
    else
        golangci-lint run --timeout=5m
        local result=$?
    fi
    
    cd - > /dev/null
    return $result
}

# Process each service
run_linter "Gateway service" "services/gateway"
GATEWAY_RESULT=$?

run_linter "Analyzer service" "services/analyzer"  
ANALYZER_RESULT=$?

run_linter "Processor service" "services/processor"
PROCESSOR_RESULT=$?

run_linter "Shared packages" "pkg"
PKG_RESULT=$?

# Summary
echo ""
echo -e "${BLUE}=== SUMMARY ===${NC}"

if [ $GATEWAY_RESULT -eq 0 ]; then
    echo -e "${GREEN}✓ Gateway${NC}"
else
    echo -e "${RED}✗ Gateway${NC}"
fi

if [ $ANALYZER_RESULT -eq 0 ]; then
    echo -e "${GREEN}✓ Analyzer${NC}"
else
    echo -e "${RED}✗ Analyzer${NC}"
fi

if [ $PROCESSOR_RESULT -eq 0 ]; then
    echo -e "${GREEN}✓ Processor${NC}"
else
    echo -e "${RED}✗ Processor${NC}"
fi

if [ $PKG_RESULT -eq 0 ]; then
    echo -e "${GREEN}✓ Shared packages${NC}"
else
    echo -e "${RED}✗ Shared packages${NC}"
fi

# Final result
if [ $GATEWAY_RESULT -eq 0 ] && [ $ANALYZER_RESULT -eq 0 ] && [ $PROCESSOR_RESULT -eq 0 ] && [ $PKG_RESULT -eq 0 ]; then
    if [ "$FIX_MODE" = true ]; then
        echo -e "\n${GREEN}All linting issues fixed successfully!${NC}"
    else
        echo -e "\n${GREEN}All linting checks passed!${NC}"
    fi
    exit 0
else
    if [ "$FIX_MODE" = true ]; then
        echo -e "\n${YELLOW}Some issues could not be auto-fixed. Manual review required.${NC}"
        echo -e "${BLUE}Run without --fix to see remaining issues.${NC}"
    else
        echo -e "\n${RED}Some linting checks failed!${NC}"
        echo -e "${BLUE}Run with --fix to automatically fix what's possible.${NC}"
    fi
    exit 1
fi