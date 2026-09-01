# 🤖 Tirenn Commerce - AI Service Microservice

Autonomous Conversational AI Shopper and Vector Semantic Search engine built with **Python 3.11**, **FastAPI**, **Clean Architecture**, **Ollama Embeddings & LLM**, **PostgreSQL pgvector**, and **Redis**.

---

## 🌟 Key Features

1. **🛍️ Autonomous AI Shopper Agent**:
   - Multi-turn conversational loop powered by **Qwen 2.5:3b** on local Ollama.
   - Dynamic Tool Calling:
     - `search_products`: Semantic vector + trigram hybrid search with live category schema.
     - `check_product_stock`: Real-time stock and price queries from PostgreSQL.
     - `add_to_cart`: Authentication-aware cart dispatching.
   - Grounding Enforcement: Strict system prompts preventing LLM hallucinations.
2. **🔎 High-Precision Semantic & Hybrid Search**:
   - Dense vector embeddings generated directly via **Ollama Embeddings API** (`bge-m3` / `paraphrase-multilingual`).
   - Hybrid ranking combining Vector Cosine Similarity (70%) and Trigram Text Matching (30%).
3. **🛡️ Production Security & Performance**:
   - Redis-backed Sliding Window Rate Limiter (`X-RateLimit-*` headers).
   - Strict CORS configuration.
   - Real-time parameter and latency logging.

---

## 🛠️ Tech Stack

- **Language**: Python 3.11
- **Framework**: FastAPI + Uvicorn + Pydantic v2
- **Embeddings**: Ollama (`bge-m3` / `paraphrase-multilingual`)
- **LLM Engine**: Ollama (`qwen2.5:3b`)
- **Database**: PostgreSQL 16 (`pgvector` HNSW indexes)
- **Cache & Rate Limiting**: Redis 7

---

## 🚀 Getting Started

### 1. Environment Configuration
Copy `.env.example` to `.env`:
```bash
PORT=8000
ENVIRONMENT=development

DB_HOST=localhost
DB_PORT=5432
DB_USER=gouser
DB_PASSWORD=gopassword
DB_NAME=gocommerce_db

REDIS_HOST=localhost
REDIS_PORT=6379

OLLAMA_BASE_URL=http://localhost:11434
LLM_MODEL=qwen2.5:3b
BACKEND_API_URL=http://localhost:8080/api/v1
```

### 2. Local Development
```bash
# Create and activate virtualenv
python -m venv venv
source venv/bin/activate  # Or on Windows: .\venv\Scripts\activate

# Install dependencies
pip install -r requirements.txt

# Run development server
make dev
# -> Accessible on http://localhost:8000
```

### 3. Docker Containerization
```bash
# Build and run standalone AI container
make docker-up

# Stop container
make docker-down
```
