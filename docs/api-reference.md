---
layout: default
title: API Reference
nav_order: 3
---

# API Reference

Complete reference for PromptCache REST API endpoints.

## Base URL

```
http://localhost:8080
```

---

## Health Checks

Kubernetes-ready health check endpoints.

### GET /health

General health status.

**Response (200 OK)**
```json
{
  "status": "healthy",
  "time": "2026-01-19T12:00:00Z"
}
```

### GET /health/ready

Readiness probe - verifies storage is accessible.

**Response (200 OK)**
```json
{
  "status": "ready"
}
```

**Response (503 Service Unavailable)**
```json
{
  "status": "not ready",
  "error": "storage not accessible"
}
```

### GET /health/live

Liveness probe - simple alive check.

**Response (200 OK)**
```json
{
  "status": "alive"
}
```

---

## Metrics & Statistics

Endpoints for monitoring and observability.

### GET /metrics

Prometheus-compatible metrics export.

**Response (200 OK)**
```
# HELP promptcache_cache_hits_total Total number of cache hits
# TYPE promptcache_cache_hits_total counter
promptcache_cache_hits_total 1234

# HELP promptcache_cache_misses_total Total number of cache misses
# TYPE promptcache_cache_misses_total counter
promptcache_cache_misses_total 567

# HELP promptcache_requests_total Total number of requests
# TYPE promptcache_requests_total counter
promptcache_requests_total 1801

# HELP promptcache_request_latency_seconds Request latency histogram
# TYPE promptcache_request_latency_seconds histogram
promptcache_request_latency_seconds_sum 45.2
promptcache_request_latency_seconds_count 1801
```

**Example - cURL**
```bash
curl http://localhost:8080/metrics
```

### GET /v1/stats

JSON statistics for dashboards.

**Response (200 OK)**
```json
{
  "cache_hits": 1234,
  "cache_misses": 567,
  "cache_hit_rate": 0.685,
  "gray_zone_checks": 89,
  "total_requests": 1801,
  "failed_requests": 2,
  "avg_latency_ms": 25.1,
  "stored_vectors": 892,
  "provider_calls": 567,
  "provider_errors": 1
}
```

**Example - cURL**
```bash
curl http://localhost:8080/v1/stats
```

---

## Cache Management

Endpoints for managing cached entries.

### GET /v1/cache/stats

Get cache statistics.

**Response (200 OK)**
```json
{
  "entry_count": 892,
  "max_entries": 100000,
  "ttl_hours": 24
}
```

### DELETE /v1/cache

Clear the entire cache.

**Response (200 OK)**
```json
{
  "message": "Cache cleared successfully",
  "deleted_count": 892
}
```

**Example - cURL**
```bash
curl -X DELETE http://localhost:8080/v1/cache
```

### DELETE /v1/cache/:key

Delete a specific cache entry.

**Parameters**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| key | string | Yes | The cache key to delete (URL path parameter) |

**Response (200 OK)**
```json
{
  "message": "Entry deleted successfully",
  "key": "abc123..."
}
```

**Response (404 Not Found)**
```json
{
  "error": "Entry not found"
}
```

---

## Chat Completions

OpenAI-compatible endpoint for chat completions with semantic caching.

### POST /v1/chat/completions

Create a chat completion with automatic caching.

**Request Headers**
```
Content-Type: application/json
```

**Request Body**
```json
{
  "model": "gpt-4",
  "messages": [
    {
      "role": "user",
      "content": "What is quantum computing?"
    }
  ]
}
```

**Parameters**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| model | string | Yes | Model name (passed to provider) |
| messages | array | Yes | Array of message objects |
| messages[].role | string | Yes | Message role (system, user, assistant) |
| messages[].content | string | Yes | Message content |

**Response (200 OK)**
```json
{
  "id": "chatcmpl-...",
  "object": "chat.completion",
  "created": 1703721600,
  "model": "gpt-4",
  "choices": [
    {
      "index": 0,
      "message": {
        "role": "assistant",
        "content": "Quantum computing is..."
      },
      "finish_reason": "stop"
    }
  ],
  "usage": {
    "prompt_tokens": 10,
    "completion_tokens": 50,
    "total_tokens": 60
  }
}
```

**Cache Behavior**

1. **Cache Hit**: Returns cached response immediately (~300ms)
2. **Cache Miss**: Forwards to provider, caches response, returns result (~1.5s)
3. **Semantic Match**: Uses embeddings to detect similar prompts

**Example - Python**
```python
from openai import OpenAI

client = OpenAI(
    base_url="http://localhost:8080/v1",
    api_key="your-api-key"
)

response = client.chat.completions.create(
    model="gpt-4",
    messages=[{"role": "user", "content": "Explain AI"}]
)
```

**Example - cURL**
```bash
curl -X POST http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-4",
    "messages": [{"role": "user", "content": "Explain AI"}]
  }'
```

**Example - JavaScript**
```javascript
const response = await fetch('http://localhost:8080/v1/chat/completions', {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({
    model: 'gpt-4',
    messages: [{ role: 'user', content: 'Explain AI' }]
  })
});
```

---

## Provider Management

Endpoints for managing embedding providers at runtime.

### GET /v1/config/provider

Get the current provider and available options.

**Response (200 OK)**
```json
{
  "provider": "openai",
  "available_providers": ["openai", "mistral", "claude"]
}
```

**Example - cURL**
```bash
curl http://localhost:8080/v1/config/provider
```

**Example - Python**
```python
import requests

response = requests.get('http://localhost:8080/v1/config/provider')
print(response.json())
```

---

### POST /v1/config/provider

Switch the embedding provider at runtime.

**Request Headers**
```
Content-Type: application/json
```

**Request Body**
```json
{
  "provider": "mistral"
}
```

**Parameters**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| provider | string | Yes | Provider name (openai, mistral, claude) |

**Response (200 OK)**
```json
{
  "message": "Provider updated successfully",
  "provider": "mistral"
}
```

**Response (400 Bad Request)**
```json
{
  "error": "unsupported provider: invalid (supported: openai, mistral, claude)"
}
```

**Example - cURL**
```bash
curl -X POST http://localhost:8080/v1/config/provider \
  -H "Content-Type: application/json" \
  -d '{"provider": "mistral"}'
```

**Example - Python**
```python
import requests

response = requests.post(
    'http://localhost:8080/v1/config/provider',
    json={'provider': 'mistral'}
)
print(response.json())
```

**Example - JavaScript**
```javascript
const response = await fetch('http://localhost:8080/v1/config/provider', {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({ provider: 'mistral' })
});
```

**Use Cases**
- A/B testing different providers
- Failover during provider outages
- Cost optimization based on load
- Performance testing

---

## Error Responses

All endpoints may return these error responses:

### 400 Bad Request
```json
{
  "error": "Invalid JSON"
}
```

### 500 Internal Server Error
```json
{
  "error": "Failed to call OpenAI: connection timeout"
}
```

---

## Rate Limiting

PromptCache does not implement rate limiting. Rate limits are inherited from your provider's API.

---

## Authentication

PromptCache uses your provider's API key. Configure it via environment variables:

```bash
export OPENAI_API_KEY=your-key      # For OpenAI
export MISTRAL_API_KEY=your-key     # For Mistral
export ANTHROPIC_API_KEY=your-key   # For Claude
export VOYAGE_API_KEY=your-key      # For Claude embeddings
```

---

## SDK Support

PromptCache is compatible with any OpenAI SDK:

- **Python**: `openai` package
- **Node.js**: `openai` package
- **Go**: `go-openai` package
- **Ruby**: `ruby-openai` gem
- **Java**: OpenAI Java client

Just change the `base_url` to point to PromptCache.
