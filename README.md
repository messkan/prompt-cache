# 🚀 PromptCache

### **Reduce your LLM costs. Accelerate your application.**

**A smart semantic cache for high-scale GenAI workloads.**

![Go](https://img.shields.io/badge/Go-1.25+-00ADD8?style=flat\&logo=go)
![License](https://img.shields.io/badge/license-MIT-green)

![PromptCache Demo](assets/demo.png)

> [!WARNING]
> **v0.1.0 is currently in Alpha.** It is not yet production-ready.
> Significant improvements in stability, performance, and configuration are coming in v0.2.0.
---

## 💰 The Problem

In production, **a large percentage of LLM requests are repetitive**:

* **RAG applications**: Variations of the same employee questions
* **AI Agents**: Repeated reasoning steps or tool calls
* **Support Bots**: Thousands of similar customer queries

Every redundant request means **extra token cost** and **extra latency**.

Why pay your LLM provider multiple times for the *same answer*?

---

## 💡 The Solution: PromptCache

PromptCache is a lightweight middleware that sits between your application and your LLM provider.
It uses **semantic understanding** to detect when a new prompt has *the same intent* as a previous one — and returns the cached result instantly.

---

## 📊 Key Benefits

| Metric                      | Without Cache | With PromptCache | Benefit      |
| --------------------------- | ------------- | ---------------- | ------------ |
| **Cost per 1,000 Requests** | ≈ $30         | **≈ $6**         | Lower cost   |
| **Avg Latency**         | ~1.5s         | **~300ms**       | Faster UX    |
| **Throughput**              | API-limited   | **Unlimited**    | Better scale |

Numbers vary per model, but the pattern holds across real workloads:
**semantic caching dramatically reduces cost and latency**.

\* Results may vary depending on model, usage patterns, and configuration.

---

## 🧠 Smart Semantic Matching (Safer by Design)

Naive semantic caches can be risky — they may return incorrect answers when prompts look similar but differ in intent.

PromptCache uses a **two-stage verification strategy** to ensure accuracy:

1. **High similarity → direct cache hit**
2. **Low similarity → skip cache directly**
3. **Gray zone → intent check using a small, cheap verification model**

This ensures cached responses are **semantically correct**, not just “close enough”.

---

## 🚀 Quick Start

PromptCache works as a **drop-in replacement** for the OpenAI API.

### 1. Run with Docker (Recommended)

```bash
# Clone the repo
git clone https://github.com/messkan/prompt-cache.git
cd prompt-cache

# Run with Docker Compose
export OPENAI_API_KEY=your_key_here
docker-compose up -d
```

### 2. Run from Source

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

# Semantically similar request → served from PromptCache
client.chat.completions.create(
    model="gpt-4",
    messages=[{"role": "user", "content": "How does quantum physics work?"}]
)
```

No code changes. Just point your client to PromptCache.

---

## ⚙️ Configuration

PromptCache can be configured using environment variables to fine-tune its behavior.

### Environment Variables

| Variable | Description | Default Value | Example |
|----------|-------------|---------------|---------|
| `OPENAI_API_KEY` | Your OpenAI API key (required) | - | `sk-...` |
| `GRAY_ZONE_FALLBACK_MODEL` | Model used for semantic verification in the "gray zone" | `gpt-4o-mini` | `gpt-4`, `gpt-4o-mini` |
| `HIGH_SIMILARITY_THRESHOLD` | Threshold above which cache hit is guaranteed | `0.95` | `0.90`, `0.98` |
| `LOW_SIMILARITY_THRESHOLD` | Threshold below which cache miss is guaranteed | `0.80` | `0.70`, `0.85` |

### Example Usage

```bash
# Run with custom configuration
export OPENAI_API_KEY=your_key_here
export GRAY_ZONE_FALLBACK_MODEL=gpt-4
export HIGH_SIMILARITY_THRESHOLD=0.98
export LOW_SIMILARITY_THRESHOLD=0.85
./prompt-cache
```

**Understanding Thresholds:**

- **High Threshold (0.95 default)**: Similarity scores **above** this value result in an immediate cache hit without verification.
- **Low Threshold (0.80 default)**: Similarity scores **below** this value result in an immediate cache miss.
- **Gray Zone (between thresholds)**: Scores in this range trigger semantic verification using the `GRAY_ZONE_FALLBACK_MODEL` to ensure intent matches.

**Tuning Tips:**

- **Higher thresholds** (e.g., 0.98/0.90): More conservative, fewer false positives, but may miss valid cache hits.
- **Lower thresholds** (e.g., 0.90/0.75): More aggressive caching, better hit rates, but slightly higher risk of semantic mismatch.
- **Fallback model**: Use `gpt-4o-mini` (default) for cost-effectiveness, or `gpt-4` for higher accuracy in verification.

---

## 🏗 Architecture Overview

Built for speed, safety, and reliability:

* **Pure Go implementation** (high concurrency, minimal overhead)
* **BadgerDB** for fast embedded persistent storage
* **In-memory caching** for ultra-fast responses
* **OpenAI-compatible API** for seamless integration
* **Docker Setup**
---

## 🛣️ Roadmap

### ✔️ v0.1.0 (Released)

* In-memory & BadgerDB storage
* Smart semantic verification (dual-threshold + intent check)
* OpenAI API compatibility

### 🚧 v0.2.0 (Planned)

* **Core Improvements**: Bug fixes and performance optimizations.
* **Public API**: Improve cache create/delete operations.
* **Enhanced Configuration**:
    * Configurable "gray zone" fallback model (enable/disable env var).
    * User-definable similarity thresholds with sensible defaults.

### 🚧 v0.3.0 (Planned)
* Built-in support for Claude & Mistral APIs

### 🚀 v1.0.0

* Clustered mode (Raft or gossip-based replication)
* Custom embedding backends (Ollama, local models)
* Rate-limiting & request shaping
* Web dashboard (hit rate, latency, cost metrics)

### ❤️ Support the Project

We are working hard to reach **v1.0.0**! If you find this project useful, please give it a ⭐️ on GitHub and consider contributing. Your support helps us ship v0.2.0 and v1.0.0 faster!

---

## 📄 License

MIT License.
