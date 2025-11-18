#!/bin/bash

# Set colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[0;33m'
NC='\033[0m' # No Color

echo -e "${YELLOW}Running tests for all modules...${NC}"

# Test gateway
echo -e "\n${YELLOW}Testing Gateway service...${NC}"
cd services/gateway
go test -race -v ./...
GATEWAY_RESULT=$?
cd ../..

# Test analyzer
echo -e "\n${YELLOW}Testing Analyzer service...${NC}"
cd services/analyzer
go test -race -v ./...
ANALYZER_RESULT=$?
cd ../..

# Test processor
echo -e "\n${YELLOW}Testing Processor service...${NC}"
cd services/processor
go test -race -v ./...
PROCESSOR_RESULT=$?
cd ../..

# Test shared packages
echo -e "\n${YELLOW}Testing Shared packages...${NC}"
cd pkg
go test -race -v ./...
PKG_RESULT=$?
cd ..

# Test contract tests
echo -e "\n${YELLOW}Testing RabbitMQ Contract tests...${NC}"
cd tests/contracts
go test -race -v ./...
CONTRACTS_RESULT=$?
cd ../..

# Check test results
echo ""
if [ $GATEWAY_RESULT -eq 0 ]; then
    echo -e "${GREEN}✓ Gateway tests passed${NC}"
else
    echo -e "${RED}✗ Gateway tests failed${NC}"
fi

if [ $ANALYZER_RESULT -eq 0 ]; then
    echo -e "${GREEN}✓ Analyzer tests passed${NC}"
else
    echo -e "${RED}✗ Analyzer tests failed${NC}"
fi

if [ $PROCESSOR_RESULT -eq 0 ]; then
    echo -e "${GREEN}✓ Processor tests passed${NC}"
else
    echo -e "${RED}✗ Processor tests failed${NC}"
fi

if [ $PKG_RESULT -eq 0 ]; then
    echo -e "${GREEN}✓ Shared packages tests passed${NC}"
else
    echo -e "${RED}✗ Shared packages tests failed${NC}"
fi

if [ $CONTRACTS_RESULT -eq 0 ]; then
    echo -e "${GREEN}✓ Contract tests passed${NC}"
else
    echo -e "${RED}✗ Contract tests failed${NC}"
fi

# Return overall result
if [ $GATEWAY_RESULT -eq 0 ] && [ $ANALYZER_RESULT -eq 0 ] && [ $PROCESSOR_RESULT -eq 0 ] && [ $PKG_RESULT -eq 0 ] && [ $CONTRACTS_RESULT -eq 0 ]; then
    echo -e "\n${GREEN}All tests passed successfully!${NC}"
    exit 0
else
    echo -e "\n${RED}Some tests failed!${NC}"
    exit 1
fi