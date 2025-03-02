#!/bin/bash

# Set colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[0;33m'
NC='\033[0m' # No Color

# Make sure scripts are executable
chmod +x scripts/test.sh
chmod +x scripts/lint.sh

# Run tests
echo -e "${YELLOW}Running all tests...${NC}"
./scripts/test.sh
TEST_RESULT=$?

echo ""

# Run linter
echo -e "${YELLOW}Running all linting checks...${NC}"
./scripts/lint.sh
LINT_RESULT=$?

echo ""

# Final results
if [ $TEST_RESULT -eq 0 ] && [ $LINT_RESULT -eq 0 ]; then
    echo -e "${GREEN}All checks passed successfully!${NC}"
    exit 0
else
    echo -e "${RED}Some checks failed!${NC}"
    exit 1
fi