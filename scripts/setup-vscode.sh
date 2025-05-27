#!/bin/bash

# VS Code integration script for golangci-lint
# This script provides real-time linting feedback in VS Code

# Set colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to setup Go tools for VS Code
setup_go_tools() {
    echo -e "${BLUE}Setting up Go tools for VS Code integration...${NC}"
    
    # Install/update required Go tools
    echo -e "${YELLOW}Installing Go tools...${NC}"
    
    # Language server
    go install golang.org/x/tools/gopls@latest
    
    # Debugging
    go install github.com/go-delve/delve/cmd/dlv@latest
    
    # Code generation
    go install golang.org/x/tools/cmd/goimports@latest
    go install mvdan.cc/gofumpt@latest
    
    # Testing
    go install github.com/cweill/gotests/gotests@latest
    go install github.com/fatih/gomodifytags@latest
    go install github.com/josharian/impl@latest
    
    # Install golangci-lint if not present
    if ! command -v golangci-lint &> /dev/null; then
        echo -e "${YELLOW}Installing golangci-lint...${NC}"
        curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin latest
    fi
    
    echo -e "${GREEN}Go tools installed successfully!${NC}"
}

# Function to check VS Code Go extension
check_vscode_extension() {
    echo -e "${BLUE}Checking VS Code Go extension...${NC}"
    
    if command -v code &> /dev/null; then
        # Install Go extension if not present
        code --install-extension golang.go 2>/dev/null || true
        echo -e "${GREEN}VS Code Go extension is ready!${NC}"
    else
        echo -e "${YELLOW}VS Code CLI not found. Please install Go extension manually.${NC}"
    fi
}

# Function to test linting integration
test_linting() {
    echo -e "${BLUE}Testing linting integration...${NC}"
    
    # Test on Gateway service
    cd services/gateway
    echo -e "${YELLOW}Running test lint on Gateway service...${NC}"
    
    # Run golangci-lint with JSON output for VS Code
    golangci-lint run --out-format=json > /tmp/lint-test.json 2>&1 || true
    
    if [ -s /tmp/lint-test.json ]; then
        echo -e "${GREEN}Linting integration working! Issues found and formatted for VS Code.${NC}"
    else
        echo -e "${GREEN}No linting issues found in Gateway service.${NC}"
    fi
    
    cd ../..
    rm -f /tmp/lint-test.json
}

# Function to create VS Code launch configuration
create_launch_config() {
    echo -e "${BLUE}Creating VS Code launch configuration...${NC}"
    
    cat > .vscode/launch.json << EOF
{
    "version": "0.2.0",
    "configurations": [
        {
            "name": "Launch Gateway Service",
            "type": "go",
            "request": "launch",
            "mode": "auto",
            "program": "./services/gateway/cmd/main.go",
            "env": {
                "TELEGRAM_TOKEN": "test-token",
                "RABBITMQ_URL": "amqp://user:password@localhost:5672/",
                "MINIO_ENDPOINT": "localhost:9000",
                "MINIO_ACCESS_KEY": "minioadmin",
                "MINIO_SECRET_KEY": "minioadmin"
            },
            "args": [],
            "cwd": "\${workspaceFolder}",
            "console": "integratedTerminal"
        },
        {
            "name": "Launch Analyzer Service", 
            "type": "go",
            "request": "launch",
            "mode": "auto",
            "program": "./services/analyzer/cmd/main.go",
            "env": {
                "RABBITMQ_URL": "amqp://user:password@localhost:5672/",
                "MINIO_ENDPOINT": "localhost:9000",
                "MINIO_ACCESS_KEY": "minioadmin",
                "MINIO_SECRET_KEY": "minioadmin",
                "OPENAI_API_KEY": "test-key"
            },
            "args": [],
            "cwd": "\${workspaceFolder}",
            "console": "integratedTerminal"
        },
        {
            "name": "Launch Processor Service",
            "type": "go", 
            "request": "launch",
            "mode": "auto",
            "program": "./services/processor/cmd/main.go",
            "env": {
                "RABBITMQ_URL": "amqp://user:password@localhost:5672/",
                "MINIO_ENDPOINT": "localhost:9000",
                "MINIO_ACCESS_KEY": "minioadmin", 
                "MINIO_SECRET_KEY": "minioadmin"
            },
            "args": [],
            "cwd": "\${workspaceFolder}",
            "console": "integratedTerminal"
        },
        {
            "name": "Debug Current File",
            "type": "go",
            "request": "launch",
            "mode": "auto",
            "program": "\${file}",
            "cwd": "\${workspaceFolder}",
            "console": "integratedTerminal"
        },
        {
            "name": "Debug Test",
            "type": "go",
            "request": "launch",
            "mode": "test",
            "program": "\${workspaceFolder}",
            "cwd": "\${workspaceFolder}",
            "console": "integratedTerminal"
        }
    ]
}
EOF

    echo -e "${GREEN}VS Code launch configuration created!${NC}"
}

# Main execution
main() {
    echo -e "${BLUE}=== VS Code + golangci-lint Integration Setup ===${NC}"
    
    # Check if we're in the right directory
    if [ ! -f ".golangci.yml" ]; then
        echo -e "${RED}Error: .golangci.yml not found. Run this script from the project root.${NC}"
        exit 1
    fi
    
    setup_go_tools
    check_vscode_extension  
    create_launch_config
    test_linting
    
    echo ""
    echo -e "${GREEN}=== Setup Complete! ===${NC}"
    echo -e "${BLUE}Next steps:${NC}"
    echo -e "1. Restart VS Code"
    echo -e "2. Open a Go file in services/gateway/"
    echo -e "3. You should see linting errors highlighted in red squiggly lines"
    echo -e "4. Use ${YELLOW}Ctrl+Shift+M${NC} to open the Problems panel"
    echo -e "5. Use ${YELLOW}F8${NC} to navigate between errors"
    echo ""
    echo -e "${YELLOW}Tips:${NC}"
    echo -e "- Save files to trigger linting (Ctrl+S)"
    echo -e "- Use 'Go: Restart Language Server' command if needed"
    echo -e "- Check Output panel > Go for detailed logs"
}

# Parse command line arguments
case "${1:-setup}" in
    "setup")
        main
        ;;
    "tools")
        setup_go_tools
        ;;
    "test")
        test_linting
        ;;
    "help")
        echo "Usage: $0 [setup|tools|test|help]"
        echo "  setup - Full setup (default)"
        echo "  tools - Install Go tools only"
        echo "  test  - Test linting integration"
        echo "  help  - Show this help"
        ;;
    *)
        echo "Unknown command: $1"
        echo "Use '$0 help' for usage information"
        exit 1
        ;;
esac