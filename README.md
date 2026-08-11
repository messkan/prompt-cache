# 🚀 PromptCache

### Reduce LLM cost and latency with semantic caching

PromptCache is a lightweight, self-hosted semantic cache for GenAI workloads. It sits between your application and an LLM provider, detects similar prompts, and can return a previously cached response instead of making another model call.

![Go](https://img.shields.io/badge/Go-1.24+-00ADD8?style=flat&logo=go)
![License](https://img.shields.io/badge/license-MIT-green)

![PromptCache Demo](assets/demo.png)

> [!NOTE]
> **v0.4.0 is available.** It adds Bearer-token authentication for management endpoints, SSE streaming support, runtime threshold configuration, and cache warming.

---

## Why PromptCache?

Many production LLM workloads contain repeated or near-repeated requests:

- RAG applications with recurring internal questions
- AI agents with repeated reasoning or tool-use patterns
- support bots receiving similar customer questions

A cache hit can avoid an upstream generation request, reducing provider usage and latency for that request.

## How it works

PromptCache uses a two-threshold semantic matching strategy:

1. **High similarity** → direct cache hit
2. **Low similarity** → cache miss
3. **Gray zone** → optional intent verification using a smaller model

This approach is designed to reduce incorrect semantic matches, but semantic matching remains probabilistic. It does **not** guarantee that two prompts are interchangeable.

> [!IMPORTANT]
> **Semantic similarity is not an authorization mechanism.** Do not use cache matching as a security boundary between users, tenants, or authorization scopes. See [Responsible Use and Data Handling](RESPONSIBLE_USE.md) before using PromptCache with sensitive or multi-user data.

---

## Example performance characteristics

Actual results depend on the provider, model, prompt distribution, cache hit rate, hardware, and configuration.

| Metric | Without Cache | Cache Hit | Typical effect |
| --- | --- | --- | --- |
| Provider generation cost | Paid upstream request | No new generation request | Lower provider usage |
| Latency | Provider-dependent | Served locally | Lower latency |
| Upstream rate-limit usage | Consumed | Avoided for the cached generation | More headroom |

Example micro-benchmark results from the repository:

```text
BenchmarkCosineSimilarity-12      2593046    441.0 ns/op    0 B/op    0 allocs/op
BenchmarkFindSimilar-12             50000    32000 ns/op  2048 B/op   45 allocs/op
```

Treat benchmark numbers as examples, not guarantees. Run the included benchmark suite against your own workload before relying on performance or cost estimates.

---

## Quick start

### Docker

```bash
git clone https://github.com/messkan/prompt-cache.git
cd prompt-cache

export EMBEDDING_PROVIDER=openai
export OPENAI_API_KEY=your_key_here

# Strongly recommended for every non-local deployment
export API_AUTH_TOKEN=your-secret-token

docker-compose up -d
```

### Run from source

```bash
git clone https://github.com/messkan/prompt-cache.git
cd prompt-cache

export EMBEDDING_PROVIDER=openai
export OPENAI_API_KEY=your-openai-api-key
export API_AUTH_TOKEN=your-secret-token

./scripts/run.sh
```

You can also use:

```bash
make run
# or
go build -o prompt-cache cmd/api/main.go
./prompt-cache
```

### OpenAI-compatible client

Point an OpenAI-compatible SDK at PromptCache:

```python
from openai import OpenAI

client = OpenAI(
    base_url="http://localhost:8080/v1",
    api_key="your-openai-api-key",
)

client.chat.completions.create(
    model="gpt-4o-mini",
    messages=[{"role": "user", "content": "Explain quantum physics"}],
)
```

The first request is forwarded to the configured provider. A later request that PromptCache determines to be sufficiently similar may be served from cache.

---

## Providers

Select the embedding/verification provider with `EMBEDDING_PROVIDER`.

### OpenAI

```bash
export EMBEDDING_PROVIDER=openai
export OPENAI_API_KEY=your-openai-api-key
```

Defaults:

- Embedding: `text-embedding-3-small`
- Verification: `gpt-4o-mini`

### Mistral AI

```bash
export EMBEDDING_PROVIDER=mistral
export MISTRAL_API_KEY=your-mistral-key
```

Defaults:

- Embedding: `mistral-embed`
- Verification: `mistral-small-latest`

### Anthropic / Claude

```bash
export EMBEDDING_PROVIDER=claude
export ANTHROPIC_API_KEY=your-anthropic-key
export VOYAGE_API_KEY=your-voyage-key
```

Defaults:

- Embedding: `voyage-3` through Voyage AI
- Verification: `claude-3-haiku-20240307`

Provider names identify compatibility only. PromptCache is an independent project and is not affiliated with, endorsed by, or sponsored by OpenAI, Anthropic, Mistral AI, or Voyage AI. Trademarks belong to their respective owners.

---

## Semantic matching configuration

```bash
# Direct hit at or above this score
export CACHE_HIGH_THRESHOLD=0.70

# Clear miss below this score
export CACHE_LOW_THRESHOLD=0.30

# Verify prompts in the gray zone
export ENABLE_GRAY_ZONE_VERIFIER=true
```

Higher hit thresholds are stricter. Always keep `CACHE_HIGH_THRESHOLD > CACHE_LOW_THRESHOLD`.

Disabling gray-zone verification reduces provider calls and latency, but may also reduce matching accuracy.

---

## Authentication

Management endpoints are protected by Bearer-token authentication when `API_AUTH_TOKEN` is set:

```bash
export API_AUTH_TOKEN=your-secret-token
```

```bash
curl http://localhost:8080/v1/stats \
  -H "Authorization: Bearer your-secret-token"
```

Protected management endpoints include `/metrics`, `/v1/stats`, `/v1/config`, `/v1/config/provider`, `/v1/cache`, and `/v1/cache/warm`.

If `API_AUTH_TOKEN` is unset, management authentication is disabled and PromptCache logs a warning. **Set it for every non-local deployment.**

The inference endpoint `/v1/chat/completions` is not an application-level authorization system. Put PromptCache behind the authentication and authorization controls required by your application.

---

## Data handling and retention

PromptCache can persist prompts, responses, embeddings, and cache metadata in BadgerDB. The cache TTL is configurable and defaults to 24 hours.

A cache TTL does not replace a complete data-retention or privacy policy. If your prompts can contain personal, confidential, regulated, user-specific, or tenant-specific information, review [RESPONSIBLE_USE.md](RESPONSIBLE_USE.md).

Protect the BadgerDB directory and backups like any other storage containing application data.

---

## Streaming

`/v1/chat/completions` supports `"stream": true`.

- On a cache miss, PromptCache forwards the provider stream and buffers the assembled response for caching.
- On a cache hit, it synthesizes OpenAI-compatible SSE chunks from the cached response.

```python
client.chat.completions.create(
    model="gpt-4o-mini",
    messages=[{"role": "user", "content": "Stream me a poem"}],
    stream=True,
)
```

---

## Runtime configuration

Read the current configuration:

```bash
curl http://localhost:8080/v1/config \
  -H "Authorization: Bearer $API_AUTH_TOKEN"
```

Update thresholds at runtime:

```bash
curl -X PATCH http://localhost:8080/v1/config \
  -H "Authorization: Bearer $API_AUTH_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"high_threshold": 0.85, "low_threshold": 0.40, "enable_gray_zone_verifier": true}'
```

Validation requires `0 <= low < high <= 1.0`.

### Cache warming

```bash
curl -X POST http://localhost:8080/v1/cache/warm \
  -H "Authorization: Bearer $API_AUTH_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "entries": [
      {
        "prompt": "What is Go?",
        "response": {
          "id": "...",
          "choices": [{"message": {"role": "assistant", "content": "Go is..."}}]
        }
      }
    ]
  }'
```

Cache warming stores the supplied response and computes an embedding for the prompt. Do not import historical data that you are not permitted to retain or reuse.

### Provider switching

```bash
curl -X POST http://localhost:8080/v1/config/provider \
  -H "Authorization: Bearer $API_AUTH_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"provider": "mistral"}'
```

---

## Architecture

- Go implementation
- BadgerDB persistent storage
- in-memory LRU tracking
- ANN index for similarity search
- OpenAI-compatible chat-completions endpoint
- OpenAI, Mistral AI, and Anthropic/Claude provider support
- Prometheus metrics and structured logging
- Docker support and health checks

---

## Benchmarks and tests

```bash
# Full benchmark suite
./scripts/benchmark.sh
# or
make benchmark

# Go micro-benchmarks
go test ./internal/semantic/... -bench=. -benchmem

# Tests
go test ./...
```

Useful make targets:

```text
make help
make build
make test
make benchmark
make clean
make docker-build
make docker-run
```

---

## Security

For security reporting, supported-version expectations, and disclosure guidance, see [SECURITY.md](SECURITY.md).

For deployment, privacy, retention, provider, semantic-matching, and multi-user considerations, see [RESPONSIBLE_USE.md](RESPONSIBLE_USE.md).

---

## Roadmap

Released features include persistent caching, multiple providers, semantic verification, observability, cache management, authentication, streaming, runtime configuration, and cache warming.

Potential future work includes clustered operation, additional embedding backends, rate limiting/request shaping, and a web dashboard.

---

## License

PromptCache is licensed under the [MIT License](LICENSE) and is provided on an **"AS IS"** basis, without warranty of any kind. See `LICENSE` for the complete terms.

---

## Documentation

Project documentation is available at `messkan.github.io/prompt-cache`, including getting-started, API, configuration, provider, and deployment guides.
