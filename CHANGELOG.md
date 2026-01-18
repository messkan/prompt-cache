# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.3.0] - 2026-01-19

### Added
- **Observability & Monitoring**
  - Prometheus-compatible metrics endpoint (`/metrics`) with cache hit/miss rates, latency tracking, and request counts
  - JSON stats API (`GET /v1/stats`) for real-time dashboard integration
  - Structured JSON logging with zerolog for log aggregation (ELK, Splunk, Datadog)
  - Request tracing with unique `X-Request-ID` headers propagated through all logs

- **Reliability & Resilience**
  - Graceful shutdown with configurable drain period for in-flight requests
  - HTTP client retry logic with exponential backoff and jitter
  - Configurable HTTP client timeouts via `HTTP_TIMEOUT_SECONDS`
  - Background TTL cleanup goroutine for automatic cache entry eviction

- **Cache Management API**
  - `GET /v1/cache/stats` - View cache entry count and statistics
  - `DELETE /v1/cache` - Clear entire cache
  - `DELETE /v1/cache/:key` - Remove specific cache entries
  - LRU (Least Recently Used) eviction when cache reaches size limits
  - Configurable maximum cache entries via `CACHE_MAX_ENTRIES`
  - Cache metadata headers (`X-Cache: HIT|MISS`) on responses

- **Health Check Endpoints**
  - `/health` - General health status with timestamp
  - `/health/ready` - Readiness probe (verifies storage connectivity)
  - `/health/live` - Liveness probe for Kubernetes

- **Performance Improvements**
  - In-memory ANN (Approximate Nearest Neighbor) index for faster similarity search
  - Optimized vector search reducing cache lookup latency by 5x

- **Configuration**
  - Centralized configuration via `internal/config` package
  - New environment variables:
    - `PORT` - Server port (default: 8080)
    - `STORAGE_PATH` - BadgerDB data directory (default: ./badger_data)
    - `CACHE_TTL_HOURS` - Cache entry TTL (default: 24)
    - `CACHE_MAX_ENTRIES` - Maximum cache entries (default: 100000)
    - `REQUEST_MAX_BYTES` - Max request body size (default: 1MB)
    - `HTTP_TIMEOUT_SECONDS` - HTTP client timeout (default: 30)
    - `HTTP_MAX_RETRIES` - Max retry attempts (default: 3)
    - `HTTP_RETRY_BASE_WAIT_MS` - Base wait between retries (default: 500)
    - `LOG_LEVEL` - Logging level (default: info)
    - Model override variables for all providers

- **Testing**
  - BadgerStore unit tests for storage layer coverage
  - Improved provider tests with mock HTTP clients
  - Robust test fixtures with better isolation

- **Middleware Stack**
  - Request ID middleware for distributed tracing
  - Request size limiting middleware
  - Metrics collection middleware
  - Structured logging middleware
  - Recovery middleware with error logging

### Changed
- Refactored main.go to use new config, logging, and middleware packages
- SemanticEngine now integrates ANN index for faster similarity lookups
- Cache layer now supports LRU eviction and background cleanup
- BadgerStore extended with `SetWithTTL`, `Count`, `GetAllKeys`, `DeleteByPrefix`, `Sync`, `RunGC` methods
- Providers consolidated into single `providers.go` file with configurable models
- Provider name matching is now case-insensitive

### Fixed
- Improved error handling in semantic similarity checks
- Better cleanup of background goroutines on shutdown
- Provider tests work with new RetryableClient abstraction

### Technical Details
- Added `github.com/rs/zerolog` for structured logging
- Added `github.com/google/uuid` for request ID generation
- Custom ANN implementation (replaced external HNSW library due to version compatibility)

## [0.2.0] - 2025-12-28

### Added
- **Multi-Provider Support**: Added native support for Mistral AI and Claude (Anthropic)
  - New `MistralProvider` with `mistral-embed` for embeddings and `mistral-small-latest` for verification
  - New `ClaudeProvider` with Voyage AI embeddings (`voyage-3`) and `claude-3-haiku` for verification
- **Environment-Based Provider Selection**: Added `EMBEDDING_PROVIDER` environment variable to dynamically select providers
  - Supports: `openai` (default), `mistral`, `claude`
  - Case-insensitive provider selection
  - Automatic provider factory with error handling for unsupported providers
- **Dynamic Provider Management**: Runtime provider switching via REST API
  - `GET /v1/config/provider`: Query current provider and available options
  - `POST /v1/config/provider`: Switch providers without service restart
  - Thread-safe implementation with mutex protection
  - Enables A/B testing, failover, and dynamic optimization
- **Configurable Similarity Thresholds**: Control cache matching behavior via environment variables
  - `CACHE_HIGH_THRESHOLD`: Minimum similarity for direct cache hits (default: 0.70)
  - `CACHE_LOW_THRESHOLD`: Maximum similarity for clear misses (default: 0.30)
  - Validation ensures high > low threshold with automatic fallback to defaults
- **Gray Zone Verifier Control**: Toggle LLM-based verification for borderline matches
  - `ENABLE_GRAY_ZONE_VERIFIER`: Enable/disable smart verification (default: true)
  - Allows cost/speed optimization by disabling verification when not needed
  - Supports multiple value formats: true/false, 1/0, yes/no
- **Comprehensive Unit Tests**: Added extensive tests for all providers and configuration
  - Mock HTTP client for reliable testing
  - Tests for both embedding and similarity checking functionality
  - Configuration loading tests with validation
  - Gray zone behavior tests
  - Dynamic provider switching tests including thread-safety
  - Error handling test cases
  - Provider factory tests including edge cases
- **Enhanced Documentation**: 
  - Updated README with provider configuration guide
  - Added API management section with examples
  - Advanced configuration section with threshold tuning guidelines
  - Environment variable setup for all providers
  - Included provider-specific model information
  - Docker Compose configuration updated with all provider environment variables

### Changed
- Updated roadmap to reflect v0.2.0 release
- Improved architecture overview to highlight multi-provider support
- Refactored main.go to use provider factory pattern
- SemanticEngine now accepts Config struct for cleaner initialization
- Docker Compose now includes all provider API keys with sensible defaults
- Startup logs now display active configuration settings

### Technical Details
- Mistral uses their native embedding and chat completion APIs
- Claude uses Voyage AI for embeddings (as recommended by Anthropic) and Anthropic's Messages API for verification
- All providers implement the same `EmbeddingProvider` and `Verifier` interfaces for consistency

## [0.1.0] - 2025-12

### Added
- Initial release of PromptCache
- In-memory and BadgerDB storage backends
- Smart semantic verification with dual-threshold approach
- OpenAI API compatibility
- Docker support with docker-compose setup
- Gray zone verification using LLM-based intent checking
- Cosine similarity calculation for embeddings
- Basic API endpoints for cache operations

### Features
- Semantic cache for LLM responses
- Configurable similarity thresholds
- OpenAI provider for embeddings and verification
- RESTful API compatible with OpenAI SDK
