#!/bin/bash

# Set colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[0;33m'
NC='\033[0m' # No Color

# Script to install golangci-lint
GOLANGCI_LINT_VERSION="v1.55.2"

echo -e "${YELLOW}Installing golangci-lint...${NC}"

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

# Detect OS and architecture
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case $ARCH in
    x86_64)
        ARCH="amd64"
        ;;
    arm64|aarch64)
        ARCH="arm64"
        ;;
    *)
        echo -e "${RED}Unsupported architecture: $ARCH${NC}"
        exit 1
        ;;
esac

# Set installation method based on OS
if [[ "$OS" == "darwin" ]]; then
    # macOS
    echo -e "${YELLOW}Detected macOS${NC}"
    
    # Check if Homebrew is available
    if command -v brew &> /dev/null; then
        echo -e "${YELLOW}Installing via Homebrew...${NC}"
        brew install golangci-lint
        if [ $? -eq 0 ]; then
            echo -e "${GREEN}golangci-lint installed successfully via Homebrew!${NC}"
            exit 0
        else
            echo -e "${RED}Failed to install via Homebrew, trying manual installation...${NC}"
        fi
    fi
    
elif [[ "$OS" == "linux" ]]; then
    echo -e "${YELLOW}Detected Linux${NC}"
    
    # Check if package manager is available
    if command -v apt-get &> /dev/null; then
        echo -e "${YELLOW}Trying to install via apt...${NC}"
        sudo apt-get update && sudo apt-get install -y golangci-lint
        if [ $? -eq 0 ]; then
            echo -e "${GREEN}golangci-lint installed successfully via apt!${NC}"
            exit 0
        else
            echo -e "${RED}Failed to install via apt, trying manual installation...${NC}"
        fi
    elif command -v yum &> /dev/null; then
        echo -e "${YELLOW}Trying to install via yum...${NC}"
        sudo yum install -y golangci-lint
        if [ $? -eq 0 ]; then
            echo -e "${GREEN}golangci-lint installed successfully via yum!${NC}"
            exit 0
        else
            echo -e "${RED}Failed to install via yum, trying manual installation...${NC}"
        fi
    fi
fi

# Manual installation via curl
echo -e "${YELLOW}Installing golangci-lint ${GOLANGCI_LINT_VERSION} manually...${NC}"

# Get GOPATH or use default
GOPATH=$(go env GOPATH 2>/dev/null)
if [ -z "$GOPATH" ]; then
    GOPATH="$HOME/go"
fi

# Create bin directory if it doesn't exist
BIN_DIR="$GOPATH/bin"
mkdir -p "$BIN_DIR"

# Download and install
curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b "$BIN_DIR" "$GOLANGCI_LINT_VERSION"

if [ $? -eq 0 ]; then
    echo -e "${GREEN}golangci-lint installed successfully!${NC}"
    
    # Check if GOPATH/bin is in PATH
    if [[ ":$PATH:" != *":$BIN_DIR:"* ]]; then
        echo -e "${YELLOW}Warning: $BIN_DIR is not in your PATH${NC}"
        echo -e "${YELLOW}Add the following line to your shell profile (~/.bashrc, ~/.zshrc, etc.):${NC}"
        echo -e "${GREEN}export PATH=\$PATH:$BIN_DIR${NC}"
        echo ""
        echo -e "${YELLOW}Or run: export PATH=\$PATH:$BIN_DIR${NC}"
    fi
    
    # Verify installation
    echo -e "${YELLOW}Verifying installation...${NC}"
    "$BIN_DIR/golangci-lint" version
    
    if [ $? -eq 0 ]; then
        echo -e "${GREEN}Installation verified successfully!${NC}"
        echo -e "${GREEN}You can now run: golangci-lint run${NC}"
    else
        echo -e "${RED}Installation verification failed${NC}"
        exit 1
    fi
else
    echo -e "${RED}Failed to install golangci-lint${NC}"
    echo -e "${YELLOW}Please try installing manually from: https://golangci-lint.run/usage/install/${NC}"
    exit 1
fi
