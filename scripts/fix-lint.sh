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

echo -e "${BLUE}Auto-fixing linting issues for all modules...${NC}"
echo -e "${YELLOW}This will automatically fix issues where possible${NC}"

# Function to run auto-fix
auto_fix_service() {
    local service_name=$1
    local service_path=$2
    
    echo -e "\n${BLUE}Auto-fixing ${service_name}...${NC}"
    cd "$service_path"
    
    # Show issues before fix
    echo -e "${YELLOW}Issues before fix:${NC}"
    golangci-lint run --timeout=5m | head -10
    
    # Apply fixes
    echo -e "\n${BLUE}Applying automatic fixes...${NC}"
    golangci-lint run --fix --timeout=5m
    local result=$?
    
    # Show remaining issues
    echo -e "\n${YELLOW}Remaining issues after fix:${NC}"
    golangci-lint run --timeout=5m | head -10
    
    cd - > /dev/null
    return $result
}

# Process each service
auto_fix_service "Gateway service" "services/gateway"
GATEWAY_RESULT=$?

auto_fix_service "Analyzer service" "services/analyzer"  
ANALYZER_RESULT=$?

auto_fix_service "Processor service" "services/processor"
PROCESSOR_RESULT=$?

auto_fix_service "Shared packages" "pkg"
PKG_RESULT=$?

# Summary
echo ""
echo -e "${BLUE}=== AUTO-FIX SUMMARY ===${NC}"

total_fixed=0
if [ $GATEWAY_RESULT -eq 0 ]; then
    echo -e "${GREEN}✓ Gateway - all issues fixed${NC}"
    ((total_fixed++))
else
    echo -e "${YELLOW}⚠ Gateway - some issues remain${NC}"
fi

if [ $ANALYZER_RESULT -eq 0 ]; then
    echo -e "${GREEN}✓ Analyzer - all issues fixed${NC}"
    ((total_fixed++))
else
    echo -e "${YELLOW}⚠ Analyzer - some issues remain${NC}"
fi

if [ $PROCESSOR_RESULT -eq 0 ]; then
    echo -e "${GREEN}✓ Processor - all issues fixed${NC}"
    ((total_fixed++))
else
    echo -e "${YELLOW}⚠ Processor - some issues remain${NC}"
fi

if [ $PKG_RESULT -eq 0 ]; then
    echo -e "${GREEN}✓ Shared packages - all issues fixed${NC}"
    ((total_fixed++))
else
    echo -e "${YELLOW}⚠ Shared packages - some issues remain${NC}"
fi

echo ""
if [ $total_fixed -eq 4 ]; then
    echo -e "${GREEN}🎉 All linting issues have been automatically fixed!${NC}"
else
    echo -e "${BLUE}Auto-fix completed. ${total_fixed}/4 modules fully fixed.${NC}"
    echo -e "${YELLOW}For remaining issues, manual fixes may be required.${NC}"
    echo -e "${BLUE}Run './scripts/lint.sh' to see what's left to fix.${NC}"
fi

exit 0