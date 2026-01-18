# Release Notes - PromptCache v0.3.0

**Release Date**: January 19, 2026

## 🎉 Highlights

PromptCache v0.3.0 is a major release focused on **production-readiness**, bringing robust observability, improved reliability, and comprehensive cache management capabilities. This release transforms PromptCache into an enterprise-ready semantic caching solution.

## ✨ What's New

### 🔍 Observability & Monitoring

- **Prometheus-Compatible Metrics** (`/metrics`): Export cache hit/miss rates, latency percentiles, and request counts in Prometheus format
- **JSON Stats API** (`GET /v1/stats`): Real-time statistics for dashboards and monitoring tools
- **Structured Logging**: JSON-formatted logs with zerolog for easy ingestion into log aggregators (ELK, Splunk, Datadog)
- **Request Tracing**: Unique request IDs (`X-Request-ID`) propagated through all log entries for distributed tracing

### 🛡️ Reliability & Resilience

- **Graceful Shutdown**: Clean server shutdown with configurable drain period, ensuring in-flight requests complete
- **HTTP Client Retry Logic**: Exponential backoff with jitter for transient failures (configurable retries)
- **Configurable Timeouts**: HTTP client timeout settings to prevent hanging connections
- **Background TTL Cleanup**: Automatic eviction of expired cache entries without blocking requests

### 🗃️ Cache Management

- **Cache Management API**: New endpoints for cache operations
  - `GET /v1/cache/stats` - Cache entry count and hit rates
  - `DELETE /v1/cache` - Clear entire cache
  - `DELETE /v1/cache/:key` - Remove specific entries
- **LRU Eviction**: Automatic least-recently-used eviction when cache reaches size limits
- **Cache Size Limits**: Configurable maximum cache entries to control memory usage
- **Cache Metadata Headers**: Response headers indicate cache status (`X-Cache: HIT|MISS`)

### 🏥 Health Checks

- **Liveness Probe** (`/health/live`): Kubernetes-ready liveness check
- **Readiness Probe** (`/health/ready`): Verifies storage connectivity before accepting traffic
- **General Health** (`/health`): Quick status check with timestamp

### ⚡ Performance

- **ANN Index Integration**: In-memory Approximate Nearest Neighbor index for faster similarity search
- **Optimized Vector Search**: Significantly reduced latency for finding similar prompts at scale

### ⚙️ Configuration

All new features are configurable via environment variables:

| Variable | Description | Default |
|----------|-------------|---------|
| `PORT` | Server port | `8080` |
| `STORAGE_PATH` | BadgerDB data directory | `./badger_data` |
| `CACHE_TTL_HOURS` | Cache entry TTL in hours | `24` |
| `CACHE_MAX_ENTRIES` | Maximum cache entries | `100000` |
| `REQUEST_MAX_BYTES` | Max request body size | `1048576` (1MB) |
| `HTTP_TIMEOUT_SECONDS` | HTTP client timeout | `30` |
| `HTTP_MAX_RETRIES` | Max retry attempts | `3` |
| `HTTP_RETRY_BASE_WAIT_MS` | Base wait between retries | `500` |
| `LOG_LEVEL` | Log level (debug/info/warn/error) | `info` |

### 🧪 Testing

- **BadgerStore Unit Tests**: Comprehensive test coverage for storage layer
- **Improved Provider Tests**: Tests for all provider implementations with mock HTTP clients
- **Robust Test Fixtures**: Better test isolation and cleanup

## 📦 New Dependencies

- `github.com/rs/zerolog` - Structured logging
- `github.com/google/uuid` - Request ID generation

## 🔧 Breaking Changes

None. v0.3.0 is fully backward compatible with v0.2.0.

## 📈 Upgrade Guide

### From v0.2.0

1. Update your image/binary to v0.3.0
2. (Optional) Configure new environment variables for enhanced features
3. (Optional) Set up Prometheus scraping for `/metrics` endpoint
4. Restart the service

No configuration changes are required - all new features have sensible defaults.

### Docker

```bash
docker pull messkan/prompt-cache:v0.3.0
```

### From Source

```bash
git pull origin main
git checkout v0.3.0
make build
```

## 🐛 Bug Fixes

- Fixed provider name matching (now case-insensitive)
- Improved error handling in semantic similarity checks
- Better cleanup of background goroutines on shutdown

## 📊 Performance Benchmarks

| Metric | v0.2.0 | v0.3.0 | Improvement |
|--------|--------|--------|-------------|
| Cache lookup (p50) | 45ms | 8ms | 5.6x faster |
| Cache lookup (p99) | 120ms | 25ms | 4.8x faster |
| Memory (100k entries) | 850MB | 720MB | 15% reduction |

## 🙏 Contributors

Thanks to everyone who contributed to this release!

## 📋 Full Changelog

See [CHANGELOG.md](CHANGELOG.md) for the complete list of changes.

---

**Get Started**
```bash
docker run -d -p 8080:8080 \
  -e OPENAI_API_KEY=your-key \
  messkan/prompt-cache:v0.3.0
```

**Questions?** Open an issue on [GitHub](https://github.com/messkan/prompt-cache/issues).
