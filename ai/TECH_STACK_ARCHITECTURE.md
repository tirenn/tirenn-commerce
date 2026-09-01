# 🏗️ Tech Stack & Architecture: Tirenn AI Service

This document outlines the AI models, vector math, rate limiting, agent architecture, and containerization design of `tirenn-ai-service`.

---

## 💻 Technical Stack

| Layer | Technology | Details |
| :--- | :--- | :--- |
| **Language** | Python 3.11 | Modern typing, async/await runtime |
| **Framework** | FastAPI 0.115+ | High-throughput asynchronous REST API |
| **Embedding Model** | `bge-m3` / `paraphrase-multilingual` | Multilingual dense embeddings generated via Ollama API |
| **LLM Model** | Qwen 2.5:3b (via Ollama) | Fast local model with native function calling and ReAct tool support |
| **Vector Database** | PostgreSQL 16 + `pgvector` | HNSW cosine distance indexing (`<=>`) for vector catalog and RAG chunks |
| **Full-Text Lexical** | PostgreSQL 16 + `pg_trgm` | Trigram similarity for exact word and SKU match boosting |
| **Session & Cache** | Redis 7 | 24-hour auto-expiring chat session history & sliding token-bucket rate limiter |
| **PDF Extraction** | `pypdf` | Zero-disk in-memory byte buffer extraction (`io.BytesIO`) |

---

## 🏛️ 4-Layer Single Agent Harness Architecture

The AI Service implements a strict 4-Layer Single Agent Harness:

```
┌────────────────────────────────────────────────────────┐
│ 1. Client Layer (React Frontend & UI)                  │
│    - AIChatModal renders chat, cart badges, toast      │
│    - Talks strictly to Agent API (/api/v1/chat/shopper)│
└──────────────────────────┬─────────────────────────────┘
                           │ HTTP POST / JSON
┌──────────────────────────▼─────────────────────────────┐
│ 2. Agent Orchestrator (AgentHarness & ShopperUseCase)  │
│    - Domain-Agnostic ReAct Loop (max_iterations = 5)   │
│    - In-Context Curation & Product Card Synchronizer   │
│    - Security & Prompt Injection Immunity Directive    │
└──────────────────────────┬─────────────────────────────┘
                           │ Structured Function Calls
┌──────────────────────────▼─────────────────────────────┐
│ 3. Tool Layer (Self-Describing Tool Schemas)           │
│    - search_products: Hybrid vector + trigram query    │
│    - get_product_detail: Direct SKU specs lookup       │
│    - get_product_stock: Direct SKU inventory query     │
│    - add_to_cart: SKU + qty mutation with stock check  │
│    - view_cart: Active cart item inspection            │
│    - search_store_policies_and_sop: RAG Vector Search  │
│    * Returns 100% factual JSON facts, zero opinions    │
└──────────────────────────┬─────────────────────────────┘
                           │ Pure SQL / pgvector / HTTP
┌──────────────────────────▼─────────────────────────────┐
│ 4. Storage & Commerce Backend (Source of Truth)        │
│    - PostgreSQL + pgvector (products, knowledge_chunks)│
│    - Redis (chat sessions & rate limits)               │
│    - Go Backend (Order placement & auth validation)    │
└────────────────────────────────────────────────────────┘
```

---

## 🔢 Hybrid Vector & Text Ranking Formula

$$\text{Score} = w_{\text{vec}} \cdot (1 - \text{CosineDistance}(\vec{q}, \vec{p})) + w_{\text{text}} \cdot \text{TrigramSimilarity}(q, p_{\text{text}})$$

Where $w_{\text{vec}} = 0.70$ (Semantic Understanding) and $w_{\text{text}} = 0.30$ (Exact Keyword & SKU match).

---

## 🐳 Container Architecture (`docker-compose.yml`)

Configuration values are strictly sourced from the root `.env` file without hardcoded credentials:

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
      - "${AI_SERVICE_PORT:-8000}:${PORT:-8000}"
    env_file:
      - ../.env
    networks:
      - tirenn-net

networks:
  tirenn-net:
    external: true
```
