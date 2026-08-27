# 📋 Project Context: Tirenn AI Service

This document describes the Clean Architecture design, domain entities, repositories, and use cases of `tirenn-ai-service`.

---

## 🏛️ Clean Architecture Structure

The Python AI service adheres to Clean Architecture:

```
app/
├── core/
│   ├── config.py              # Pydantic v2 settings & environment variables
│   └── security.py            # Redis sliding window rate limiter & CORS middleware
├── domain/
│   ├── category.py            # Category domain model
│   ├── chat.py                # ChatMessage, ChatShopperResult, ToolCall models
│   └── product.py             # Product, ScoredProduct, ProductIndexItem models
├── repositories/
│   ├── embedding_repository.py# SentenceTransformers embedding model loader & encoder
│   ├── llm_repository.py      # Async HTTP client for Ollama API (/api/chat)
│   └── product_repository.py  # PostgreSQL queries, pgvector HNSW search & category maps
├── usecases/
│   ├── search_usecase.py      # Hybrid vector + trigram catalog search
│   ├── shopper_usecase.py     # Autonomous agentic shopping loop & tool dispatcher
│   └── sync_usecase.py        # Background batch embedding synchronization
├── handlers/
│   ├── catalog_handler.py     # FastAPI endpoints for /search/semantic & /sync
│   └── chat_handler.py        # FastAPI endpoints for /chat/shopper & /chat
└── main.py                    # Dependency injection composition root & lifespan
```

---

## 🛠️ Tool Calling Implementation

The Shopper Agent dispatches three specialized tools:
1. **`search_products`**: Queries pgvector with multi-stage adaptive thresholds (`CHAT_SEARCH_SCORE_THRESHOLD: 0.20`, fallback `0.10`) and category filters.
2. **`check_product_stock`**: Verifies exact real-time inventory counts and prices in PostgreSQL.
3. **`add_to_cart`**: Authentication-aware tool. If user is guest (`is_authenticated=false`), intercepts and returns `action: auth_required`. If authenticated, returns `action: cart_added`.

---

## 🌐 API Endpoints

- `POST /api/v1/chat/shopper`: Primary conversational shopper agent endpoint.
- `POST /api/v1/chat`: Alias for shopper chat.
- `POST /api/v1/search/semantic`: Vector & hybrid semantic search endpoint.
- `POST /api/v1/sync`: Batch catalog indexing endpoint.
- `GET /health`: Microservice liveness & readiness check.
