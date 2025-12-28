#!/bin/bash

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}   PromptCache Benchmark Suite${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

# Configuration
PORT=${PORT:-8080}
BENCHMARK_REQUESTS=${BENCHMARK_REQUESTS:-100}
CONCURRENT_REQUESTS=${CONCURRENT_REQUESTS:-10}
SERVER_PID=""

# Check if server is already running
check_server() {
    if lsof -Pi :$PORT -sTCP:LISTEN -t >/dev/null 2>&1; then
        return 0
    else
        return 1
    fi
}

# Start the server
start_server() {
    echo -e "${YELLOW}Starting PromptCache server...${NC}"
    
    if check_server; then
        echo -e "${GREEN}Server already running on port $PORT${NC}"
        return
    fi
    
    # Build the project
    echo -e "${YELLOW}Building project...${NC}"
    go build -o prompt-cache cmd/api/main.go
    
    # Start server in background
    ./prompt-cache > server.log 2>&1 &
    SERVER_PID=$!
    
    # Wait for server to be ready
    echo -e "${YELLOW}Waiting for server to start...${NC}"
    for i in {1..30}; do
        if check_server; then
            echo -e "${GREEN}Server started successfully (PID: $SERVER_PID)${NC}"
            sleep 2  # Give it a moment to fully initialize
            return
        fi
        sleep 1
    done
    
    echo -e "${RED}Failed to start server${NC}"
    exit 1
}

# Stop the server
stop_server() {
    if [ ! -z "$SERVER_PID" ]; then
        echo -e "${YELLOW}Stopping server (PID: $SERVER_PID)...${NC}"
        kill $SERVER_PID 2>/dev/null || true
        wait $SERVER_PID 2>/dev/null || true
    fi
}

# Cleanup on exit
trap stop_server EXIT

# Benchmark function
run_benchmark() {
    local test_name=$1
    local prompt=$2
    local num_requests=$3
    
    echo -e "\n${BLUE}Testing: $test_name${NC}"
    echo -e "${YELLOW}Sending $num_requests requests...${NC}"
    
    local start_time=$(date +%s.%N)
    local success_count=0
    local cache_hit_count=0
    
    for ((i=1; i<=$num_requests; i++)); do
        response=$(curl -s -w "\n%{time_total}" -X POST http://localhost:$PORT/v1/chat/completions \
            -H "Content-Type: application/json" \
            -d "{
                \"model\": \"gpt-4\",
                \"messages\": [{\"role\": \"user\", \"content\": \"$prompt\"}]
            }")
        
        # Check if request was successful
        if [ $? -eq 0 ]; then
            ((success_count++))
        fi
        
        # Simple progress indicator
        if [ $((i % 10)) -eq 0 ]; then
            echo -n "."
        fi
    done
    
    echo ""
    local end_time=$(date +%s.%N)
    local duration=$(echo "$end_time - $start_time" | bc)
    local avg_time=$(echo "scale=3; $duration / $num_requests" | bc)
    local requests_per_sec=$(echo "scale=2; $num_requests / $duration" | bc)
    
    echo -e "${GREEN}Results:${NC}"
    echo -e "  Total Requests: $num_requests"
    echo -e "  Successful: $success_count"
    echo -e "  Total Time: ${duration}s"
    echo -e "  Avg Time/Request: ${avg_time}s"
    echo -e "  Requests/Second: $requests_per_sec"
}

# Main benchmarking
main() {
    start_server
    
    echo -e "\n${BLUE}========================================${NC}"
    echo -e "${BLUE}   Running Benchmarks${NC}"
    echo -e "${BLUE}========================================${NC}"
    
    # Test 1: Cache miss (first request)
    run_benchmark "Cache Miss (First Request)" \
        "What is quantum computing in simple terms?" 10
    
    # Test 2: Cache hit (exact same prompt)
    run_benchmark "Cache Hit (Exact Match)" \
        "What is quantum computing in simple terms?" 20
    
    # Test 3: Semantic similarity (similar prompts)
    echo -e "\n${BLUE}Testing semantic similarity...${NC}"
    run_benchmark "Semantic Match 1" \
        "Explain quantum computing simply" 10
    run_benchmark "Semantic Match 2" \
        "How does quantum computing work in basic terms?" 10
    
    # Test 4: Different prompts (cache miss)
    run_benchmark "Cache Miss (Different Prompt)" \
        "What is machine learning?" 10
    
    echo -e "\n${BLUE}========================================${NC}"
    echo -e "${GREEN}   Benchmark Complete!${NC}"
    echo -e "${BLUE}========================================${NC}"
    
    # Show server logs if there were errors
    if [ -f server.log ]; then
        echo -e "\n${YELLOW}Server logs:${NC}"
        tail -20 server.log
    fi
}

# Run the benchmarks
main
