#!/bin/bash

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}   Starting PromptCache${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

# Check for required environment variables
check_env() {
    local provider=${EMBEDDING_PROVIDER:-openai}
    
    echo -e "${YELLOW}Checking environment configuration...${NC}"
    echo -e "Provider: ${GREEN}$provider${NC}"
    
    case "$provider" in
        openai)
            if [ -z "$OPENAI_API_KEY" ]; then
                echo -e "${RED}Error: OPENAI_API_KEY is not set${NC}"
                exit 1
            fi
            echo -e "${GREEN}✓ OpenAI API key configured${NC}"
            ;;
        mistral)
            if [ -z "$MISTRAL_API_KEY" ]; then
                echo -e "${RED}Error: MISTRAL_API_KEY is not set${NC}"
                exit 1
            fi
            echo -e "${GREEN}✓ Mistral API key configured${NC}"
            ;;
        claude)
            if [ -z "$ANTHROPIC_API_KEY" ]; then
                echo -e "${RED}Error: ANTHROPIC_API_KEY is not set${NC}"
                exit 1
            fi
            if [ -z "$VOYAGE_API_KEY" ]; then
                echo -e "${RED}Error: VOYAGE_API_KEY is not set (required for Claude embeddings)${NC}"
                exit 1
            fi
            echo -e "${GREEN}✓ Claude API keys configured${NC}"
            ;;
    esac
    
    echo ""
}

# Build the project
build() {
    echo -e "${YELLOW}Building PromptCache...${NC}"
    go build -o prompt-cache cmd/api/main.go
    echo -e "${GREEN}✓ Build successful${NC}"
    echo ""
}

# Run the server
run() {
    echo -e "${BLUE}========================================${NC}"
    echo -e "${GREEN}   PromptCache is running!${NC}"
    echo -e "${BLUE}========================================${NC}"
    echo ""
    echo -e "Server: ${GREEN}http://localhost:8080${NC}"
    echo -e "Provider: ${GREEN}${EMBEDDING_PROVIDER:-openai}${NC}"
    echo -e ""
    echo -e "${YELLOW}Management Endpoints:${NC}"
    echo -e "  GET  /v1/config/provider - Check current provider"
    echo -e "  POST /v1/config/provider - Switch provider"
    echo -e ""
    echo -e "${YELLOW}Cache Endpoint:${NC}"
    echo -e "  POST /v1/chat/completions - OpenAI-compatible API"
    echo -e ""
    echo -e "Press ${RED}Ctrl+C${NC} to stop"
    echo ""
    
    ./prompt-cache
}

# Main execution
main() {
    check_env
    build
    run
}

main
