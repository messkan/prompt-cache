# Release Notes - PromptCache v0.4.0

**Release Date**: April 25, 2026

## 🎉 Highlights

PromptCache v0.4.0 closes out the feature roadmap from the v0.3.0 release with four focused additions: **Bearer-token authentication** for management endpoints, **end-to-end SSE streaming** (including streamed cache hits across all providers), a **runtime configuration API** for similarity thresholds, and **bulk cache warming** for pre-populating from historical traffic.

## ✨ What's New

### 🔐 API Authentication

All management endpoints are now gated behind a Bearer-token middleware that uses constant-time comparison:

- Endpoints affected: `/metrics`, `/v1/stats`, `/v1/config`, `/v1/config/provider`, `/v1/cache`, `/v1/cache/:key`, `/v1/cache/warm`
- Public endpoints unchanged: `/health*` and `/v1/chat/completions`
- Set `API_AUTH_TOKEN` to enable; when unset, auth is disabled and a startup warning is logged

```bash
export API_AUTH_TOKEN=your-secret-token
curl http://localhost:8080/v1/stats -H "Authorization: Bearer $API_AUTH_TOKEN"
```

### 🌊 Streaming Support (SSE)

`/v1/chat/completions` now honors `"stream": true` end-to-end:

- **Cache miss**: streams provider SSE events to the client while buffering the assembled response for caching
- **Cache hit**: synthesizes OpenAI-compatible SSE chunks from the cached non-streaming response (role delta → content delta → stop)
- **Cross-provider**: works with OpenAI, Mistral, and Claude — Claude's native event stream is translated to OpenAI SSE format

Existing OpenAI SDKs work unchanged.

### ⚙️ Runtime Configuration API

```bash
# Read current configuration
curl http://localhost:8080/v1/config -H "Authorization: Bearer $API_AUTH_TOKEN"

# Update similarity thresholds and the gray-zone verifier flag
curl -X PATCH http://localhost:8080/v1/config \
  -H "Authorization: Bearer $API_AUTH_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"high_threshold": 0.85, "low_threshold": 0.40, "enable_gray_zone_verifier": true}'
```

Updates are atomic (single write lock) and validated: `0 <= low < high <= 1.0`.

### 🔥 Cache Warming

Pre-populate the cache from historical prompt/response pairs in a single call:

```bash
curl -X POST http://localhost:8080/v1/cache/warm \
  -H "Authorization: Bearer $API_AUTH_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"entries":[{"prompt":"...","response":{...}}]}'
```

Each entry is embedded, stored, and registered in the ANN index. Failures roll back the entry. The endpoint returns a per-entry status report with `processed` / `failed` counts.

## 🔧 Breaking Changes

- `Provider` interface gained a new method: `ForwardStreamingChatCompletion(ctx, body, w) ([]byte, int, error)`. Custom provider implementations must implement it.
- `GET /v1/config/provider` removed — provider information is now part of `GET /v1/config`. The `POST /v1/config/provider` switch endpoint is unchanged.

## 🐛 Fixes

- Bounds-checked short cache keys when synthesizing SSE on cache hit (would have panicked on keys < 8 chars)
- SSE fallback no longer drops content when cached payload fails to parse — emits the raw cached bytes as a single content chunk
- Streaming provider errors now emit an SSE error event rather than returning silently after headers were flushed
- Threshold validation accepts `0` correctly

## 🧪 Tests

- `internal/middleware/auth_test.go` — disabled mode, missing / malformed / wrong / valid token
- `internal/semantic/streaming_test.go` — SSE assembly + malformed-chunk resilience
- `internal/semantic/config_test.go` — `UpdateThresholds` validation paths and `GetConfig`

Run the full suite with `go test ./...`.

## ⬆️ Upgrading

1. Set `API_AUTH_TOKEN` in your environment if you expose management endpoints.
2. Update any callers of `GET /v1/config/provider` to use `GET /v1/config`.
3. If you maintain a custom `Provider` implementation, add `ForwardStreamingChatCompletion`.

No storage migrations required.
