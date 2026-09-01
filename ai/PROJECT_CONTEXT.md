# 📋 Project Context: Tirenn AI Service

This document describes the Clean Architecture design, domain entities, repositories, use cases, and activity log of `tirenn-ai-service`.

---

## 🏛️ Clean Architecture Structure

The Python AI service adheres to Clean Architecture:

```
app/
├── core/
│   ├── config.py              # Pydantic v2 settings, JWT configs & environment variables
│   └── security.py            # Direct JWT validator, Redis sliding rate limiter & CORS middleware
├── domain/
│   ├── chat.py                # ChatMessage, ChatShopperResult, ToolCall models
│   └── product.py             # Product, ScoredProduct, ProductIndexItem models
├── repositories/
│   ├── embedding_repository.py# Ollama embedding API client & dense vector encoder
│   ├── knowledge_repository.py# PostgreSQL pgvector knowledge_documents & knowledge_chunks HNSW storage
│   ├── llm_repository.py      # Async HTTP client for Ollama API (/api/chat)
│   ├── product_repository.py  # PostgreSQL queries, pgvector HNSW search & category maps
│   └── session_repository.py  # Redis session storage with 24h auto-expiring TTL
├── harness/
│   ├── agent.py               # Pure Domain-Agnostic ReAct Agent Harness engine
│   └── tools/                 # BaseTool, catalog_tools.py, cart_tools.py, knowledge_tools.py
├── usecases/
│   ├── knowledge_usecase.py   # In-memory PDF text extraction, chunker & RAG retrieval
│   ├── search_usecase.py      # Hybrid vector + trigram catalog search
│   ├── shopper_usecase.py     # Autonomous agentic shopping loop & tool dispatcher
│   └── sync_usecase.py        # Background batch embedding synchronization
├── handlers/
│   ├── catalog_handler.py     # FastAPI endpoints for /search/semantic & /sync
│   ├── chat_handler.py        # FastAPI endpoints for /chat/shopper & /chat
│   └── knowledge_handler.py   # FastAPI endpoints for /knowledge/upload-pdf, /documents, /query
└── main.py                    # Dependency injection composition root & lifespan
```

---

## 🛠️ Tool Calling & RAG Implementation

The Shopper Agent dispatches specialized tools via the Agent Harness:
1. **`search_products`**: Queries PostgreSQL pgvector + pg_trgm with multi-stage adaptive thresholds and category filters, curated in-context by the LLM. Returns minimal factual data.
2. **`get_product_detail`**: Retrieves in-depth specifications, materials, features, rating, badge, and description strictly by SKU.
3. **`get_product_stock`**: Verifies exact real-time inventory counts, stock status (`ready_stock`/`low_stock`/`out_of_stock`), and price strictly by SKU.
4. **`add_to_cart`**: Mutating tool strictly taking `sku` and `qty`, re-validating real-time stock on the server at mutation time.
5. **`view_cart`**: Displays current items inside the shopping cart.
6. **`search_store_policies_and_sop`**: Vector RAG search against PostgreSQL `pgvector` knowledge chunks for store policies, return/warranty terms, and merchant SLAs, isolated in `<untrusted_document_content>` tags.

---

## 🌐 API Endpoints

- `POST /api/v1/chat/shopper`: Primary conversational shopper agent endpoint with tool calling.
- `DELETE /api/v1/chat/session/{session_id}`: Explicit session deletion from Redis.
- `POST /api/v1/knowledge/upload-pdf`: Multipart PDF in-memory extraction & pgvector indexing (Protected with `verify_admin_jwt`).
- `GET /api/v1/knowledge/documents`: List indexed knowledge documents (Protected with `verify_admin_jwt`).
- `DELETE /api/v1/knowledge/documents/{id}`: Delete indexed knowledge document and its chunks (Protected with `verify_admin_jwt`).
- `POST /api/v1/knowledge/query`: Semantic vector RAG retrieval against knowledge chunks.
- `POST /api/v1/search/semantic`: Vector & hybrid semantic search endpoint.
- `POST /api/v1/sync-from-backend`: On-demand vector catalog synchronization.

---

## 📜 Service Changelog

### 📅 2026-09-01
- `[AI Service & Embeddings]` Migrated Embedding Pipeline from HuggingFace Transformers to Ollama API:
  - **Ollama Embedding Integration**: Refactored `EmbeddingRepository` (`app/repositories/embedding_repository.py`) to generate dense vector embeddings directly via Ollama (`POST /api/embed` and `/api/embeddings`).
  - **Dynamic Dimension & Unit Normalization**: Added automatic dimension detection (supporting 1024-dim `bge-m3` or configured model), L2 vector unit normalization, batch embedding, and zero-vector handling for empty inputs.
  - **Container Optimization**: Removed heavy `sentence-transformers` and PyTorch dependencies from `requirements.txt` and `Dockerfile`, drastically reducing container build time and memory usage.
  - **Comprehensive Test Suite**: Added `tests/test_embedding_repository.py` covering single/batch encoding, dimension checks, normalization, and network fallback (53/53 pytest tests passing).

### 📅 2026-08-28
- `[AI Service]` Implemented Real-Time Product Recommendation Engine (`GET /api/v1/products/{id}/recommendations`):
  - Added high-performance pgvector cosine distance search (`<=>`) on 384-dimensional embeddings.
  - Added category/subcategory affinity soft-boost (+0.15 subcategory, +0.08 category) and dynamic price corridor ($0.4x - 2.5x$).
  - Added `order_items` co-occurrence aggregation with vector fallback.
  - 47/47 pytest unit & integration tests passing in 1.76s.
- `[AI Service]` Cleaned and pruned dead code across `tirenn-ai-service`:
  - Deleted legacy in-memory session folder `app/harness/memory/` (superseded by Redis `SessionRepository`).
  - Deleted ad-hoc test file `app/harness/test_relevance_guardrail.py`.
  - Deleted unused domain model `app/domain/category.py` and duplicate schemas `app/schemas/search.py`.
  - Removed unused `get_product_by_sku_or_name` from `ProductRepository`.
  - Removed backward-compatibility aliases and unused regex modules from tools.
- `[AI Service]` Integrated Redis Session Management (`SessionRepository`):
  - Session chat history stored under `chat:session:{session_id}` with auto-expiring 24h TTL.
  - Added `DELETE /api/v1/chat/session/{session_id}` endpoint to purge sessions on demand.
- `[AI Service]` Pure LLM Native Reasoning for 2-Step Confirmation (Zero Hardcoded Dictionaries):
  - Removed all hardcoded affirmative/cancel keyword sets and interceptors.
  - The model natively inspects conversational history, determines whether user agreed/confirmed (in any language or phrasing), and autonomously executes `adjust_product_stock` with `confirmed=true` or acknowledges cancellation.
  - `ProductRepository.adjust_stock` commits atomic SQL transactions directly in PostgreSQL with audit log recording in `stock_adjustment_logs`.
  - **`AnalyticsRepository` (`app/repositories/analytics_repository.py`)**: Direct PostgreSQL connection for real-time dashboard KPIs, financial totals, order items aggregation, and order histories (`get_dashboard_summary`, `get_top_selling_products`, `get_recent_orders`).
  - **`ProductRepository.adjust_stock`**: Atomic SQL transaction locking target product row (`FOR UPDATE`), computing delta, updating `products.stock_quantity`, and recording immutable audit entry in `stock_adjustment_logs`.
  - **Direct Tools Injection**: Updated `GetExecutiveDashboardMetricsTool`, `GetRecentOrdersOverviewTool`, and `AdjustProductStockTool` to query PostgreSQL directly via repositories without HTTP API overhead.
  - **Clean Architecture & Natural Multilingual Reasoning**: Removed rule-based `detect_language` heuristic and dead tool files, allowing LLM to natively mirror language and format structured repository facts.
- `[AI Service]` Implemented Phase 1 Admin AI Copilot (`AdminUseCase` & `POST /api/v1/chat/admin`):
  - **Tool Directory Segregation**: Created distinct physical subdirectories `app/harness/tools/customer/` and `app/harness/tools/admin/`. Customer agent cannot load or execute admin tools and vice-versa.
  - **Admin Tools & 2-Step Confirmation Guardrails**:
    - `GetExecutiveDashboardMetricsTool`: Calls Go backend `GET /api/v1/admin/dashboard` using forwarded admin Bearer token.
    - `GetRecentOrdersOverviewTool`: Calls Go backend `GET /api/v1/admin/orders`.
    - `GetLowStockProductsTool`: Queries PostgreSQL for low stock items below threshold.
    - `AdjustProductStockTool`: Implemented **2-Step Confirmation Guardrail** (`confirmed: bool`). When `confirmed=false`, tool blocks mutation and returns preview metadata (Product Name, SKU, Current Stock, Projected New Stock, Audit Reason) requiring explicit Admin approval before execution.
    - `SearchAdminInternalSOPTool`: Executes vector RAG queries restricted strictly to `doc_type="SOP_ADMIN"`.
  - **JWT Authorization**: Guarded `POST /api/v1/chat/admin` with `verify_admin_jwt` (enforcing `role == 'ADMIN'`).
### 📅 2026-09-01

- `[AI Service - Dynamic DB Taxonomy]` Refactored Category Selection to Live Database Taxonomy:
  - **Dynamic Taxonomy Fetching**: Added `ProductRepository.get_taxonomy_prompt_text()` to query `categories` LEFT JOIN `sub_categories` live from PostgreSQL, eliminating all hardcoded taxonomy arrays and keyword heuristic matching.
  - **Live Function Parameter Schema**: Updated `SearchProductsTool.to_openai_schema()` to dynamically assemble category options directly from PostgreSQL, resolving LLM category hallucination without manual intervention.
- `[AI Service - Performance & Timeout Fix]` Optimized CPU Latency and Context Length:
  - **Inference Parameter Tuning**: Added `LLM_NUM_PREDICT=350`, `LLM_NUM_CTX=2048`, and `LLM_KEEP_ALIVE="60m"` in `llm_repository.py`. Dropped CPU generation duration from 70s to ~5s, eliminating browser timeout errors.
  - **Threshold Calibration**: Calibrated `DEFAULT_SEARCH_SCORE_THRESHOLD=0.38` and `CHAT_SEARCH_SCORE_THRESHOLD=0.30` in `.env`, eliminating low-relevance cross-category matches (e.g. smartwatches/tempered glass appearing for phone searches).
- `[AI Service - Multi-Stack Docker Compose]` Docker Architecture Separation:
  - Created standalone `docker-compose.yml` inside `ai/` operating on shared bridge network `tirenn-net`.
  - Externalized 100% of configurable hyperparameters into `ai/.env` and updated `ai/.env.example`.

### 📅 2026-08-31

- `[AI Service]` Implemented 2-Tier Redis Semantic RAG Cache in `KnowledgeUseCase`:
  - **Tier 1 (Exact Hash Match)**: Stores hash keys (`rag:exact:{scope}:{hash}`) in Redis with 24h TTL for $< 1\text{ms}$ instantaneous responses.
  - **Tier 2 (Semantic Vector Similarity Match)**: Stores query embeddings in Redis List (`rag:semantic:{scope}`) and evaluates cosine similarity against active clusters in in-memory RAM. When $\ge 92\%$ similarity is reached, returns cached excerpts in $\approx 5\text{ms}$ without invoking PostgreSQL.
  - **Auto-Invalidation**: Flushes all `rag:exact:*` and `rag:semantic:*` keys automatically whenever an Admin uploads or deletes SOP PDF documents in `KnowledgeManagement`.
- `[AI Service]` Enforced Hard Data Isolation for Customer SOP (`doc_type='SOP_CUSTOMER'`):
  - Locked `SearchStorePoliciesAndSOPTool` to strictly query customer-facing knowledge documents (`doc_type="SOP_CUSTOMER"`), physically barring the public AI Shopper from retrieving internal merchant or administrative SOPs from PostgreSQL.
  - Enhanced `SYSTEM_PROMPT` with strict customer-only data scope directives.
- `[AI Service]` Decoupled Knowledge Tools into Dedicated Module (`knowledge_tools.py`):
  - Extracted `SearchStorePoliciesAndSOPTool` out of `catalog_tools.py` into a clean, dedicated `app/harness/tools/knowledge_tools.py`.
  - Maintained clear Single Responsibility Principle across tool domains (`catalog_tools.py`, `cart_tools.py`, `knowledge_tools.py`).
- `[AI Service]` Upgraded Session Storage to Redis List Data Structure (`RPUSH` / `LRANGE` / `LTRIM`):
  - Refactored `SessionRepository` to store chat exchanges as atomic items inside a Redis List (`chat:session:{session_id}`).
  - Implemented Bounded Sliding Window Retrieval (`LRANGE -10 -1`), fetching strictly the last $N$ messages (`SESSION_HISTORY_LIMIT=10`) into the LLM context window.
  - Added rolling list pruning (`LTRIM -50 -1` / `SESSION_MAX_STORED=50`) to keep Redis memory bounded, preventing context window saturation and ensuring fast, constant inference latency across long conversations.
- `[AI Service]` Hardened Prompt Injection Defenses & External Document Isolation:
  - Added strict Security & Prompt Injection Immunity Directive in `SYSTEM_PROMPT` to protect system prompt privacy and reject jailbreak/DAN/override attempts.
  - Implemented boundary isolation tags (`<untrusted_document_content>`) in `SearchStorePoliciesAndSOPTool` to treat all external RAG document excerpts as passive reference data, mitigating indirect prompt injection risks.
- `[AI Service]` Decoupled Tool Descriptions into Self-Describing Tool Schemas:
  - Pruned redundant `AVAILABLE TOOLS & ACTIONS` text from `SYSTEM_PROMPT`.
  - Tool invocation contracts and parameter specifications are declared strictly within each tool class (`name`, `description`, `parameters_schema`).
  - Reduced token overhead by ~35% and made adding new tools strictly follow the Open/Closed Principle without modifying system prompt.
- `[AI Service]` Implemented In-Context Filtering & Product Card Synchronization:
  - Augmented `SYSTEM_PROMPT` with strict in-context curation directives to ignore contradictory search candidates and require explicit SKU referencing.
  - Implemented card pruning synchronization in `AgentHarness.run` so only products verified and referenced by the LLM in its final reply are rendered as frontend suggested product cards.
- `[AI Service]` Purged all regex hacks and artificial guardrails:
  - Deleted `app/harness/guardrails/` (`relevance.py` and `safety.py`).
  - Removed dynamic `lang_directive` injection and English regex keyword scrapers from `agent.py`.
  - Refactored `AgentHarness` into a 100% domain-agnostic, canonical ReAct tool-calling loop.
- `[AI Service]` Cleaned and standardized tool parameter interfaces:
  - `add_to_cart`: Strictly accepts `sku: str` and `qty: int`.
  - `get_product_detail` (renamed from `explain_product_details`): Strictly accepts `sku: str`.
  - `get_product_stock` (renamed from `check_product_stock`): Strictly accepts `sku: str`.
  - Removed semantic search and ILIKE fallback from SKU lookups; tools now perform pure Direct SQL queries strictly by SKU via `ProductRepository.get_product_by_sku(sku)`.
  - Decoupled `SearchUseCase` dependency from `GetProductDetailTool`, `GetProductStockTool`, and `AddToCartTool`.
  - Enforced minimum query length requirement (>= 3 characters) across semantic search (`SearchUseCase.execute`, `SearchProductsTool`, `AddToCartTool`, `_resolve_target_product`).
  - Queries with less than 3 characters immediately return 0 products and prompt conversational clarification to prevent ambiguous vector hallucinations.
- `[AI Service]` Implemented multi-turn conversational context resolution and slot-filling across all tools (`explain_product_details`, `check_product_stock`, `add_to_cart`).
  - Generic requests without product names (e.g. *"can you explain product for me?"*) now properly ask for clarification instead of guessing or returning default products.
  - Ordinal and context references (e.g. *"explain the first product"*, *"jelaskan nomor 1"*) are dynamically resolved from previous recommendations in conversation history.
- `[AI Service]` Enhanced bilingual intelligence: LLM strictly detects and mirrors user language (100% English for English prompts, 100% Indonesian for Indonesian prompts) in both direct responses and synthesized tool results.
- `[AI Service]` Upgraded Ollama LLM to **`qwen2.5:3b`** for intelligent, pure Tool Calling without complex regex cascades.
- `[AI Service]` Refactored `AgentHarness` & `AddToCartTool` to clean, elegant ReAct architecture with native parameter extraction and slot filling.
- `[AI Service]` Implemented complete 4-feature Agent Harness architecture: (1) Product Search & Recommendations (`search_products`), (2) Product Explanation & Details (`explain_product_details`), (3) Real-Time Stock & Pricing (`check_product_stock`), and (4) Add to Cart with Slot Extraction & Clarification (`add_to_cart`).
- `[AI Service]` Added `ExplainProductDetailsTool` in `catalog_tools.py` for retrieving in-depth product specs, materials, category, badge, and ratings.
- `[AI Service]` Enriched `CheckProductStockTool` with status indicators (`ready_stock`, `low_stock`, `out_of_stock`).
- `[AI Service]` Fixed product resolution hierarchy in `AddToCartTool` so explicit/partial product names (`masukan keranjang [nama produk]`) always take precedence over ordinal heuristics.
- `[AI Service]` Restricted AI Chat product recommendation output to maximum **6 items** (`CHAT_SEARCH_LIMIT=6`) across `config.py`, `agent.py`, `catalog_tools.py`, and `chat_handler.py`.
- `[AI Service]` Enhanced `AddToCartTool` with conversational history context and ordinal/numeric index resolution (e.g. automatically resolving `"masukkan kekeranjang 1"`, `"nomor 1"`, `"pertama"` to the target item from previous recommendations).
- `[AI Service]` Updated `AgentHarness` with cart intent fallback to guarantee `add_to_cart` execution and prevent accidental invocation of `search_products` on cart commands.
- `[AI Service]` Configured `INTERNAL_API_KEY` to secure internal machine-to-machine catalog synchronization and vector re-indexing endpoints (`/index-products`, `/sync-from-backend`).
- `[AI Service]` Implemented `AgentHarness` with `RelevanceGuardrail` to filter contradictory/irrelevant product search returns.
- `[AI Service]` Built in-memory PDF extraction with `pypdf.PdfReader(io.BytesIO)` — zero disk writes.
- `[AI Service]` Implemented direct JWT validation (`verify_admin_jwt`) using `PyJWT` with `HS256` matching Golang backend's `JWT_SECRET`.
- `[AI Service]` Created PostgreSQL `pgvector` knowledge repository (`knowledge_documents` and `knowledge_chunks`) with 384-dimensional dense vectors and HNSW cosine distance indexing.
- `[AI Service]` Added `SearchStorePoliciesAndSOPTool` into the Shopper Agent Harness to answer store policy, SLA, and return queries directly from vector chunks.
- `[AI Service]` Pre-indexed 23 vector chunks from Customer SOP (13 chunks) and Admin SOP (10 chunks).
