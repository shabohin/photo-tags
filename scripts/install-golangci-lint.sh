#!/bin/bash

# Script to install golangci-lint v2 (latest)
# Can be called directly or via make install-tools

# Set colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[0;33m'
NC='\033[0m' # No Color

GOLANGCI_LINT_VERSION="v2.1.6"

echo -e "${YELLOW}Installing golangci-lint ${GOLANGCI_LINT_VERSION}...${NC}"

# Check if golangci-lint is already installed
if command -v golangci-lint &> /dev/null; then
    CURRENT_VERSION=$(golangci-lint version --format short 2>/dev/null || echo "unknown")
    echo -e "${GREEN}golangci-lint is already installed (version: ${CURRENT_VERSION})${NC}"
    
    read -p "Do you want to reinstall/update golangci-lint? (y/n): " REINSTALL
    if [[ ! $REINSTALL =~ ^[Yy]$ ]]; then
        echo -e "${GREEN}Skipping installation.${NC}"
        exit 0
    fi
fi

# Detect OS
OS=$(uname -s | tr '[:upper:]' '[:lower:]')

if [[ "$OS" == "darwin" ]]; then
    # macOS - try Homebrew first
    if command -v brew &> /dev/null; then
        echo -e "${YELLOW}Installing via Homebrew...${NC}"
        brew install golangci-lint
        if [ $? -eq 0 ]; then
            echo -e "${GREEN}golangci-lint installed successfully via Homebrew!${NC}"
            golangci-lint version
            exit 0
        fi
    fi
elif [[ "$OS" == "linux" ]]; then
    # Linux - try package managers
    if command -v apt-get &> /dev/null; then
        sudo apt-get update && sudo apt-get install -y golangci-lint
        if [ $? -eq 0 ]; then
            echo -e "${GREEN}golangci-lint installed successfully via apt!${NC}"
            golangci-lint version
            exit 0
        fi
    fi
fi

# Fallback to manual installation
echo -e "${YELLOW}Installing golangci-lint ${GOLANGCI_LINT_VERSION} manually...${NC}"

GOPATH=$(go env GOPATH 2>/dev/null)
if [ -z "$GOPATH" ]; then
    GOPATH="$HOME/go"
fi

BIN_DIR="$GOPATH/bin"
mkdir -p "$BIN_DIR"

curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b "$BIN_DIR" "$GOLANGCI_LINT_VERSION"

if [ $? -eq 0 ]; then
    echo -e "${GREEN}golangci-lint installed successfully!${NC}"
    
    # Check if GOPATH/bin is in PATH
    if [[ ":$PATH:" != *":$BIN_DIR:"* ]]; then
        echo -e "${YELLOW}Warning: $BIN_DIR is not in your PATH${NC}"
        echo -e "${YELLOW}Add the following line to your shell profile:${NC}"
        echo -e "${GREEN}export PATH=\$PATH:$BIN_DIR${NC}"
    fi
    
    "$BIN_DIR/golangci-lint" version
else
    echo -e "${RED}Failed to install golangci-lint${NC}"
    exit 1
fi
