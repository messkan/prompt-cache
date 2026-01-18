#!/bin/bash

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

API_URL="http://localhost:8080"

echo -e "${YELLOW}=== PromptCache API Tests ===${NC}\n"

# Test 1: First request - will be a CACHE MISS (stored for future)
echo -e "${YELLOW}Test 1: First request about Go embeddings (CACHE MISS expected)${NC}"
curl -X POST "$API_URL/v1/chat/completions" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-4o-mini",
    "messages": [
      {
        "role": "system",
        "content": "You are a helpful coding assistant."
      },
      {
        "role": "user",
        "content": "How do I implement vector embeddings in Go?"
      }
    ]
  }'
echo -e "\n\n"

# Test 2: Similar prompt - should be a CACHE HIT
echo -e "${YELLOW}Test 2: Similar prompt (CACHE HIT expected)${NC}"
curl -X POST "$API_URL/v1/chat/completions" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-4o-mini",
    "messages": [
      {
        "role": "user",
        "content": "How to implement embeddings in Golang?"
      }
    ]
  }'
echo -e "\n\n"

# # Test 3: Exact same prompt - should be a CACHE HIT
# echo -e "${YELLOW}Test 3: Exact same prompt (CACHE HIT expected)${NC}"
# curl -X POST "$API_URL/v1/chat/completions" \
#   -H "Content-Type: application/json" \
#   -d '{
#     "model": "gpt-4o-mini",
#     "messages": [
#       {
#         "role": "user",
#         "content": "How do I implement vector embeddings in Go?"
#       }
#     ]
#   }'
# echo -e "\n\n"

# # Test 4: Different topic - will be a CACHE MISS
# echo -e "${YELLOW}Test 4: Different topic (CACHE MISS expected)${NC}"
# curl -X POST "$API_URL/v1/chat/completions" \
#   -H "Content-Type: application/json" \
#   -d '{
#     "model": "gpt-4o-mini",
#     "messages": [
#       {
#         "role": "user",
#         "content": "How do I make a pizza at home?"
#       }
#     ]
#   }'
# echo -e "\n\n"

# # Test 5: Similar to pizza topic - might be CACHE HIT
# echo -e "${YELLOW}Test 5: Similar to pizza topic (CACHE HIT expected)${NC}"
# curl -X POST "$API_URL/v1/chat/completions" \
#   -H "Content-Type: application/json" \
#   -d '{
#     "model": "gpt-4o-mini",
#     "messages": [
#       {
#         "role": "user",
#         "content": "What is the best way to make homemade pizza?"
#       }
#     ]
#   }'
# echo -e "\n\n"

# # Test 6: Get metrics to see cache performance
# echo -e "${YELLOW}Test 6: Get cache metrics${NC}"
# curl -X GET "$API_URL/metrics"
# echo -e "\n\n"

# echo -e "${GREEN}=== Tests Complete ===${NC}"
# echo -e "${YELLOW}Check the server logs to see CACHE HIT 🔥 and CACHE MISS 💨 indicators${NC}"
