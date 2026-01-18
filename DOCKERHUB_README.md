# 🚀 PromptCache

### **Reduce your LLM costs. Accelerate your application.**

**A smart semantic cache for high-scale GenAI workloads.**

![Version](https://img.shields.io/badge/version-0.3.0-blue)

---

## 💰 The Problem

In production, **a large percentage of LLM requests are repetitive**:

* **RAG applications**: Variations of the same employee questions
* **AI Agents**: Repeated reasoning steps or tool calls
* **Support Bots**: Thousands of similar customer queries

Every redundant request means **extra token cost** and **extra latency**.

---

## 💡 The Solution

PromptCache is a lightweight middleware that sits between your application and your LLM provider.
It uses **semantic understanding** to detect when a new prompt has *the same intent* as a previous one — and returns the cached result instantly.

---

## ✨ What's New in v0.3.0

- 📊 **Prometheus Metrics** - `/metrics` endpoint for observability
- 🏥 **Health Checks** - Kubernetes-ready liveness/readiness probes
- 🗃️ **Cache Management** - Clear cache, view stats via API
- 📝 **Structured Logging** - JSON logs for log aggregation
- ⚡ **5x Faster** - ANN index for similarity search
- 🔄 **Graceful Shutdown** - Clean request draining

---

## 🚀 Quick Start

PromptCache works as a **drop-in replacement** for the OpenAI API.

### Option 1: Run with Docker Compose (Recommended)

Create a `docker-compose.yml` file:

```yaml
version: '3.8'

services:
  prompt-cache:
    image: messkan/prompt-cache:0.3.0
    ports:
      - "8080:8080"
    environment:
      - OPENAI_API_KEY=${OPENAI_API_KEY}
      - EMBEDDING_PROVIDER=openai
      - LOG_LEVEL=info
    volumes:
      - ./badger_data:/root/badger_data
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8080/health/ready"]
      interval: 30s
      timeout: 10s
      retries: 3
```

Then run:

```bash
export OPENAI_API_KEY=your_key_here
docker-compose up -d
```

### Option 2: Run with Docker CLI

```bash
# Set your OpenAI Key
export OPENAI_API_KEY=your_key_here

# Run the container
docker run -d -p 8080:8080 \
  -e OPENAI_API_KEY=$OPENAI_API_KEY \
  -v prompt_cache_data:/root/badger_data \
  messkan/prompt-cache:0.3.0
```

### Verify Installation

```bash
# Health check
curl http://localhost:8080/health

# View metrics
curl http://localhost:8080/metrics

# Check stats
curl http://localhost:8080/v1/stats
```

### Update your Client

Simply change the `base_url` in your SDK:

```python
from openai import OpenAI

client = OpenAI(
    base_url="http://localhost:8080/v1",  # Point to PromptCache
    api_key="sk-..."
)

# First request → goes to the LLM provider
client.chat.completions.create(
    model="gpt-4",
    messages=[{"role": "user", "content": "Explain quantum physics"}]
)

# Semantically similar request → served from PromptCache (Instant & Free)
client.chat.completions.create(
    model="gpt-4",
    messages=[{"role": "user", "content": "How does quantum physics work?"}]
)
```

---

## 📊 Key Benefits

| Metric                      | Without Cache | With PromptCache | Benefit      |
| --------------------------- | ------------- | ---------------- | ------------ |
| **Cost per 1,000 Requests** | ≈ $30         | **≈ $6**         | Lower cost   |
| **Avg Latency**             | ~1.5s         | **~300ms**       | Faster UX    |
| **Throughput**              | API-limited   | **Unlimited**    | Better scale |

---

## 🧠 Smart Semantic Matching

PromptCache uses a **two-stage verification strategy** to ensure accuracy:

1. **High similarity (>0.95)** → direct cache hit
2. **Low similarity (<0.70)** → skip cache directly
3. **Gray zone (0.70 - 0.95)** → intent check using a small, cheap verification model

This ensures cached responses are **semantically correct**, not just “close enough”.
