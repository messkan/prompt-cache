---
layout: default
title: Providers
nav_order: 5
---

# Provider Guide

PromptCache supports multiple AI providers for embeddings and semantic verification.

## Supported Providers

| Provider | Embedding Model | Verification Model | Cost (per 1M tokens) |
|----------|----------------|-------------------|----------------------|
| OpenAI | text-embedding-3-small | gpt-4o-mini | $0.02 / $0.15 |
| Mistral AI | mistral-embed | mistral-small-latest | $0.10 / $0.20 |
| Claude | voyage-3 (Voyage AI) | claude-3-haiku | $0.10 / $0.25 |
| MiniMax | MiniMax-M3 | MiniMax-M3 | $0.60 / $2.40 |

---

## OpenAI (Default)

### Setup

```bash
export EMBEDDING_PROVIDER=openai
export OPENAI_API_KEY=your-openai-api-key
```

### Models Used

- **Embeddings**: `text-embedding-3-small` (1536 dimensions)
- **Verification**: `gpt-4o-mini`

### Characteristics

**Pros**:
- Fast response times
- High-quality embeddings
- Most cost-effective
- Excellent documentation

**Cons**:
- Requires OpenAI account
- API rate limits on free tier

### Best For

- General purpose use
- Cost-conscious deployments
- Production environments
- High-volume workloads

---

## Mistral AI

### Setup

```bash
export EMBEDDING_PROVIDER=mistral
export MISTRAL_API_KEY=your-mistral-api-key
```

Get your API key from: [console.mistral.ai](https://console.mistral.ai/)

### Models Used

- **Embeddings**: `mistral-embed` (1024 dimensions)
- **Verification**: `mistral-small-latest`

### Characteristics

**Pros**:
- European data residency
- Strong multilingual support
- Privacy-focused
- Open source models available

**Cons**:
- Slightly higher cost than OpenAI
- Smaller ecosystem
- Newer provider

### Best For

- European deployments (GDPR compliance)
- Multilingual applications
- Privacy-sensitive workloads
- Supporting open source AI

---

## Claude (Anthropic)

### Setup

```bash
export EMBEDDING_PROVIDER=claude
export ANTHROPIC_API_KEY=your-anthropic-api-key
export VOYAGE_API_KEY=your-voyage-api-key
```

Get your API keys:
- Anthropic: [console.anthropic.com](https://console.anthropic.com/)
- Voyage AI: [dashboard.voyageai.com](https://dashboard.voyageai.com/)

{: .note }
> Claude uses Voyage AI for embeddings as recommended by Anthropic. You need both API keys.

### Models Used

- **Embeddings**: `voyage-3` via Voyage AI (1024 dimensions)
- **Verification**: `claude-3-haiku-20240307`

### Characteristics

**Pros**:
- Constitutional AI (safer outputs)
- Excellent reasoning capabilities
- Strong context understanding
- Longer context windows

**Cons**:
- Requires two API keys
- Highest cost
- More complex setup

### Best For

- High-quality reasoning requirements
- Safety-critical applications
- Complex prompt understanding
- Long-context scenarios

---

## MiniMax

### Setup

```bash
export EMBEDDING_PROVIDER=minimax
export MINIMAX_API_KEY=your-minimax-api-key
```

Get your API key from: [platform.minimax.io](https://platform.minimax.io/)

### Regional Endpoints

MiniMax provides two regional endpoints. Set `MINIMAX_BASE_URL` to choose the region:

| Region | Base URL |
|--------|----------|
| Global (English) | `https://api.minimax.io/v1` (default) |
| China | `https://api.minimaxi.com/v1` |

```bash
# Use the China endpoint
export MINIMAX_BASE_URL=https://api.minimaxi.com/v1
```

### Models Used

- **Embeddings**: `MiniMax-M3`
- **Verification**: `MiniMax-M3`

### Characteristics

**Pros**:
- OpenAI-compatible chat completions API
- Supports global and China regional endpoints
- Multimodal model support (text, image, video)
- Large context window (up to 1,000,000 tokens)

**Cons**:
- Higher cost than OpenAI
- Newer provider ecosystem

### Best For

- Deployments requiring China regional endpoint
- Multimodal workloads
- Long-context applications

---

## Comparing Providers

### Performance

Based on benchmarks with 1000 requests:

| Provider | Avg Latency (Cache Miss) | Avg Latency (Cache Hit) | Embedding Quality |
|----------|--------------------------|-------------------------|-------------------|
| OpenAI | ~1.2s | ~280ms | Excellent |
| Mistral | ~1.4s | ~290ms | Very Good |
| Claude | ~1.6s | ~300ms | Excellent |
| MiniMax | ~1.5s | ~290ms | Very Good |

### Cost Comparison

For 1M cached requests (assuming 50% cache hit rate):

| Provider | Embedding Cost | Verification Cost | Total Cost |
|----------|---------------|------------------|------------|
| OpenAI | $10 | $7.50 | **$17.50** |
| Mistral | $50 | $10 | **$60** |
| Claude | $50 + $50 | $12.50 | **$112.50** |

{: .tip }
> OpenAI is the most cost-effective for high-volume workloads

---

## Switching Providers

### At Startup

Set the `EMBEDDING_PROVIDER` environment variable before starting:

```bash
export EMBEDDING_PROVIDER=mistral
export MISTRAL_API_KEY=your-key
./prompt-cache
```

### At Runtime (Dynamic Switching)

Use the provider management API:

```bash
curl -X POST http://localhost:8080/v1/config/provider \
  -H "Content-Type: application/json" \
  -d '{"provider": "mistral"}'
```

**Response**:
```json
{
  "message": "Provider updated successfully",
  "provider": "mistral"
}
```

### Check Current Provider

```bash
curl http://localhost:8080/v1/config/provider
```

**Response**:
```json
{
  "provider": "openai",
  "available_providers": ["openai", "mistral", "claude", "minimax"]
}
```

---

## Multi-Provider Strategy

### Failover Strategy

Start with OpenAI, fallback to Mistral:

```python
import requests

def switch_provider(provider_name):
    response = requests.post(
        'http://localhost:8080/v1/config/provider',
        json={'provider': provider_name}
    )
    return response.status_code == 200

# Monitor OpenAI health
if openai_api_down:
    switch_provider('mistral')
```

### A/B Testing

Test multiple providers simultaneously:

```python
providers = ['openai', 'mistral', 'claude']
results = {}

for provider in providers:
    switch_provider(provider)
    start = time.time()
    # Run test queries
    results[provider] = time.time() - start

print(f"Fastest: {min(results, key=results.get)}")
```

### Cost Optimization

Use cheaper providers during high load:

```python
import psutil

cpu_usage = psutil.cpu_percent()

if cpu_usage > 80:
    # Switch to most cost-effective provider
    switch_provider('openai')
else:
    # Use preferred provider
    switch_provider('claude')
```

---

## Provider-Specific Notes

### OpenAI

- **Rate Limits**: 3,500 RPM (free tier), 10,000+ RPM (paid)
- **Region**: US-based
- **Compliance**: SOC 2, GDPR-compliant

### Mistral AI

- **Rate Limits**: Varies by plan
- **Region**: Europe-based (Paris)
- **Compliance**: GDPR-native, ISO 27001

### Claude

- **Rate Limits**: 4,000 RPM (default)
- **Region**: US-based
- **Compliance**: SOC 2, HIPAA-eligible

---

## Troubleshooting

### Provider Not Found

```
Error: unsupported provider: xxx
```

**Solution**: Use `openai`, `mistral`, `claude`, or `minimax` (case-insensitive)

### API Key Missing

```
Error: OPENAI_API_KEY is not set
```

**Solution**: Export the required API key before starting

### Rate Limit Exceeded

**Solution**: 
1. Upgrade your provider plan
2. Switch to another provider
3. Implement request throttling

### Poor Cache Hit Rate

**Solution**:
1. Try different providers - embedding quality varies
2. Adjust similarity thresholds
3. Enable gray zone verification

---

## Best Practices

1. **Development**: Use OpenAI (fastest, cheapest)
2. **Production**: Use OpenAI or Mistral depending on requirements
3. **Compliance**: Use Mistral for EU/GDPR, Claude for HIPAA
4. **Experimentation**: Use dynamic switching to compare providers
5. **Failover**: Have API keys for multiple providers ready

---

## Future Providers

Planned support for:
- Local models (Ollama)
- Azure OpenAI
- Google PaLM
- Cohere

Want to add a provider? [Open an issue](https://github.com/messkan/prompt-cache/issues) or submit a PR!
