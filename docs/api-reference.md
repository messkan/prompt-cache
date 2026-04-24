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

## Authentication

As of v0.4.0, all management endpoints (`/metrics`, `/v1/stats`, `/v1/config`, `/v1/config/provider`, `/v1/cache`, `/v1/cache/:key`, `/v1/cache/warm`) require a Bearer token when `API_AUTH_TOKEN` is set. The inference endpoint (`/v1/chat/completions`) and `/health*` endpoints are always public.

```bash
export API_AUTH_TOKEN=your-secret-token

curl http://localhost:8080/v1/stats \
  -H "Authorization: Bearer your-secret-token"
```

If `API_AUTH_TOKEN` is unset, auth is disabled and a startup warning is logged. Always set it for non-local deployments.

**Provider API keys** for upstream LLMs are still configured via environment variables:

```bash
export OPENAI_API_KEY=your-key      # For OpenAI
export MISTRAL_API_KEY=your-key     # For Mistral
export ANTHROPIC_API_KEY=your-key   # For Claude
export VOYAGE_API_KEY=your-key      # For Claude embeddings
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

### POST /v1/cache/warm

Bulk pre-populate the cache from historical prompt/response pairs. For each entry, PromptCache stores the response, computes an embedding, and registers the entry in the ANN index. If embedding fails for an entry, that entry is rolled back; other entries are unaffected.

**Request Body**
```json
{
  "entries": [
    {
      "prompt": "What is Go?",
      "response": {
        "id": "chatcmpl-...",
        "object": "chat.completion",
        "model": "gpt-4o-mini",
        "choices": [
          {"index": 0, "message": {"role": "assistant", "content": "Go is..."}, "finish_reason": "stop"}
        ]
      }
    }
  ]
}
```

**Parameters**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| entries | array | Yes | Non-empty array of prompt/response pairs |
| entries[].prompt | string | Yes | The prompt text to embed and key on |
| entries[].response | object | Yes | The full chat-completion response payload to cache |

**Response (200 OK)**
```json
{
  "processed": 1,
  "failed": 0,
  "entries": [
    {"key": "abc123...", "status": "ok"}
  ]
}
```

**Example - cURL**
```bash
curl -X POST http://localhost:8080/v1/cache/warm \
  -H "Authorization: Bearer $API_AUTH_TOKEN" \
  -H "Content-Type: application/json" \
  -d @warm.json
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

#### Streaming (SSE)

Set `"stream": true` in the request body to receive Server-Sent Events:

```json
{
  "model": "gpt-4o-mini",
  "stream": true,
  "messages": [{"role": "user", "content": "Stream me a poem"}]
}
```

- **Cache miss**: PromptCache forwards a streaming request to the provider, pipes SSE chunks through to the client, and buffers the assembled response for future caching.
- **Cache hit**: The cached non-streaming response is synthesized into OpenAI-compatible SSE chunks (role delta → content delta → stop) so streaming clients work transparently.

Works across OpenAI, Mistral, and Claude — Claude's native event stream is translated to OpenAI SSE format.

```python
for chunk in client.chat.completions.create(
    model="gpt-4o-mini",
    messages=[{"role": "user", "content": "Hello"}],
    stream=True,
):
    print(chunk.choices[0].delta.content or "", end="")
```

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

## Configuration

Endpoints for inspecting and updating runtime configuration. All require `Authorization: Bearer $API_AUTH_TOKEN` when auth is enabled.

### GET /v1/config

Return the current provider, available providers, similarity thresholds, gray-zone verifier flag, and cache settings.

**Response (200 OK)**
```json
{
  "provider": "openai",
  "available_providers": ["openai", "mistral", "claude"],
  "high_threshold": 0.70,
  "low_threshold": 0.30,
  "enable_gray_zone_verifier": true,
  "cache_ttl_hours": 24,
  "cache_max_entries": 100000
}
```

**Example - cURL**
```bash
curl http://localhost:8080/v1/config \
  -H "Authorization: Bearer $API_AUTH_TOKEN"
```

---

### PATCH /v1/config

Update similarity thresholds and the gray-zone verifier flag at runtime. Any subset of fields may be supplied; omitted fields are unchanged.

**Request Body**
```json
{
  "high_threshold": 0.85,
  "low_threshold": 0.40,
  "enable_gray_zone_verifier": true
}
```

**Validation**: `0 <= low_threshold < high_threshold <= 1.0`. Invalid values return `400`.

**Response (200 OK)** — same shape as `GET /v1/config`, reflecting the updated values.

**Response (400 Bad Request)**
```json
{
  "error": "high_threshold (0.4000) must be greater than low_threshold (0.5000)"
}
```

**Example - cURL**
```bash
curl -X PATCH http://localhost:8080/v1/config \
  -H "Authorization: Bearer $API_AUTH_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"high_threshold": 0.85, "low_threshold": 0.40}'
```

---

## Provider Management

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

## SDK Support

PromptCache is compatible with any OpenAI SDK:

- **Python**: `openai` package
- **Node.js**: `openai` package
- **Go**: `go-openai` package
- **Ruby**: `ruby-openai` gem
- **Java**: OpenAI Java client

Just change the `base_url` to point to PromptCache.
