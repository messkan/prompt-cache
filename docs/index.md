---
layout: default
title: Home
nav_order: 1
---

# PromptCache Documentation

![PromptCache](https://img.shields.io/badge/Go-1.24+-00ADD8?style=flat&logo=go)
![License](https://img.shields.io/badge/license-MIT-green)
![Version](https://img.shields.io/badge/version-0.4.0-blue)

**A smart semantic cache for high-scale GenAI workloads.**

## What is PromptCache?

PromptCache is a lightweight middleware that sits between your application and your LLM provider. It uses **semantic understanding** to detect when a new prompt has the same intent as a previous one — and returns the cached result instantly.

## Key Benefits

- **Reduce Costs**: Save up to 80% on LLM API costs
- **Improve Latency**: ~300ms vs ~1.5s average response time
- **Better Scale**: Unlimited throughput without API rate limits
- **Smart Matching**: Semantic understanding prevents incorrect cache hits

## What's New in v0.4.0

- 🔐 **API Authentication** - Bearer-token middleware gating all management endpoints (`API_AUTH_TOKEN`)
- 🌊 **Streaming (SSE)** - Full `stream: true` support across OpenAI, Mistral, and Claude — including streamed cache hits
- ⚙️ **Runtime Config API** - `GET`/`PATCH /v1/config` for live threshold and gray-zone updates
- 🔥 **Cache Warming** - `POST /v1/cache/warm` to bulk pre-populate from historical prompt/response pairs

### Previously in v0.3.0

- 📊 Prometheus metrics, structured logging, health checks
- 🗃️ Cache management API, LRU eviction
- ⚡ ANN index for 5x faster similarity search
- 🔄 Graceful shutdown, retry logic

## Quick Links

- [Getting Started](getting-started.md)
- [API Reference](api-reference.md)
- [Configuration Guide](configuration.md)
- [Provider Setup](providers.md)

## Architecture

PromptCache uses a **three-stage verification strategy**:

1. **High similarity (≥70%)** → Direct cache hit
2. **Low similarity (<30%)** → Skip cache directly  
3. **Gray zone (30-70%)** → LLM verification for accuracy

This ensures cached responses are semantically correct, not just "close enough".

## Supported Providers

- **OpenAI**: text-embedding-3-small + gpt-4o-mini
- **Mistral AI**: mistral-embed + mistral-small-latest
- **Claude (Anthropic)**: voyage-3 + claude-3-haiku

## Features

- ✅ Multiple provider support (OpenAI, Mistral, Claude)
- ✅ Dynamic provider switching via API
- ✅ Configurable similarity thresholds (env vars + runtime PATCH)
- ✅ Gray zone verification control
- ✅ OpenAI-compatible API (including SSE streaming)
- ✅ Bearer-token authentication for management endpoints
- ✅ Cache warming from historical data
- ✅ Docker support
- ✅ Thread-safe operations
- ✅ BadgerDB persistence
- ✅ Prometheus metrics export
- ✅ Health check endpoints
- ✅ Cache management API
- ✅ Structured JSON logging
- ✅ LRU cache eviction
- ✅ Request tracing (X-Request-ID)

## Community

- [GitHub Repository](https://github.com/messkan/prompt-cache)
- [Report Issues](https://github.com/messkan/prompt-cache/issues)
- [Contribute](https://github.com/messkan/prompt-cache/blob/main/CONTRIBUTING.md)

## License

MIT License - see [LICENSE](https://github.com/messkan/prompt-cache/blob/main/LICENSE) for details.
