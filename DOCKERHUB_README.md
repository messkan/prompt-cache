# 🚀 PromptCache

### **Reduce your LLM costs. Accelerate your application.**

**A smart semantic cache for high-scale GenAI workloads.**

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

## 🚀 Quick Start

PromptCache works as a **drop-in replacement** for the OpenAI API.

### Option 1: Run with Docker Compose (Recommended)

Create a `docker-compose.yml` file:

```yaml
version: '3.8'

services:
  prompt-cache:
    image: messkan/prompt-cache:latest
    ports:
      - "8080:8080"
    environment:
      - OPENAI_API_KEY=${OPENAI_API_KEY}
    volumes:
      - ./badger_data:/root/badger_data
    restart: unless-stopped
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
  messkan/prompt-cache:latest
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
