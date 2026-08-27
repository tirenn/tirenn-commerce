# 🏗️ Tech Stack & Architecture: Tirenn AI Service

This document outlines the AI models, vector math, rate limiting, and containerization design of `tirenn-ai-service`.

---

## 💻 Technical Stack

| Layer | Technology | Details |
| :--- | :--- | :--- |
| **Language** | Python 3.11 | Modern typing, async/await runtime |
| **Framework** | FastAPI 0.115+ | High-throughput asynchronous REST API |
| **Embedding Model** | `paraphrase-multilingual-MiniLM-L12-v2` | 384 dimensions, pre-cached in container |
| **LLM Model** | Qwen 2.5:3b (via Ollama) | Local lightweight model with strong tool calling |
| **Vector DB** | PostgreSQL 16 `pgvector` | HNSW cosine distance indexing (`<=>`) |
| **Cache & Limiter** | Redis 7 | Sliding window rate limiting per IP/client |

---

## 🔢 Hybrid Vector & Text Ranking Formula

$$\text{Score} = w_{\text{vec}} \cdot (1 - \text{CosineDistance}(\vec{q}, \vec{p})) + w_{\text{text}} \cdot \text{TrigramSimilarity}(q, p_{\text{text}})$$

Where $w_{\text{vec}} = 0.70$ and $w_{\text{text}} = 0.30$.

---

## 🐳 Container Architecture (`docker-compose.yml`)

```yaml
version: '3.8'

services:
  ai-service:
    build:
      context: .
      dockerfile: Dockerfile
    container_name: tirenn-ai-service
    restart: always
    ports:
      - "8000:8000"
    environment:
      - PORT=8000
      - ENVIRONMENT=production
      - DB_HOST=postgres
      - DB_PORT=5432
      - DB_USER=gouser
      - DB_PASSWORD=gopassword
      - DB_NAME=gocommerce_db
      - REDIS_HOST=redis
      - REDIS_PORT=6379
      - OLLAMA_BASE_URL=http://ollama:11434
      - LLM_MODEL=qwen2.5:3b
      - BACKEND_API_URL=http://backend:8080/api/v1
    networks:
      - tirenn-net

networks:
  tirenn-net:
    external: true
```
