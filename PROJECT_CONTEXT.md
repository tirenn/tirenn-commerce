# 📋 Project Context & Activity Changelog: Tirenn Commerce

This document serves as the single source of truth for architectural guidelines, system components, and the historical activity log across all services in Tirenn Commerce.

---

## 📌 Engineering Guidelines & Protocols

1. **Activity History Log**: Always record and update recent activities in this `PROJECT_CONTEXT.md` using domain-specific tags:
   - `[PM]` Project planning, requirement breakdown, PRD alignment, acceptance criteria.
   - `[Golang]` Backend API changes, GORM models, repository/service/handler updates, migrations.
   - `[AI Service]` Python FastAPI, sentence-transformers, Ollama LLM, Agent Harness, RAG knowledge pipeline.
   - `[Frontend]` React/Vite UI, Tailwind CSS, i18n, modals, admin dashboard, cart drawer.
   - `[QA]` Playwright E2E tests, Vitest/Jest, test automation, quality gates.
   - `[Infra]` Docker compose, PostgreSQL pgvector, Redis, Prometheus/Loki/Grafana.
2. **Database Schema Migrations**:
   - Any database changes (new tables, columns, constraints, or vector indexes) **MUST ALWAYS** be created using Goose SQL migration files via `make migrate-create name=<migration_name>` in `tirenn-backend/` (and mirrored in GORM domain models under `tirenn-backend/internal/domain/`). Never create DDL tables directly in other services.
3. **Targeted Service Deployment**:
   - If changes only affect a specific service, run/rebuild Docker **only for that specific service** (e.g. `docker compose restart ai-service` or `docker compose up -d --build backend`), preserving container cache and reducing turnaround time.
4. **Environment Variables Isolation**:
   - Environment values **MUST ONLY** reside inside `.env` files.
   - Never hardcode environment values in `docker-compose.yml`, `Makefile`, or `.md` files. Use `env_file: .env` in docker compose and maintain clean `.env.example` templates with empty values for secrets.

---

## 🏛️ System Architecture Overview

```
tirenn-commerce/
├── tirenn-backend/        # Go (Golang 1.23) REST API + GORM + PostgreSQL + JWT Auth
├── tirenn-ai-service/      # Python 3.11 FastAPI + sentence-transformers + pgvector RAG + Agent Harness
├── tirenn-frontend/        # React 19 + TypeScript + Vite + Tailwind CSS + i18n (IDR/USD, ID/EN)
├── tirenn-infra/           # Docker Compose, PostgreSQL (pgvector), Redis, Grafana/Loki/Promtail, QA (Playwright)
└── docs/                   # PRD, API Docs, Official SOPs (PDF & Markdown)
```

---

## 📜 Activity Changelog & History

### 📅 2026-09-01

- `[Architecture & Multi-Stack Docker Compose]` Separated Docker Stacks:
  - **Modular Directory Compose**: Separated monolithic compose into 4 independent, modular `docker-compose.yml` stacks (`infra/`, `ai/`, `backend/`, and `frontend/`) sharing a unified external bridge network `tirenn-net`.
  - **Independent Life-cycle**: Each stack can be built, started, stopped, or scaled independently with zero side effects on other containers.
- `[Mobile Responsive UI & UX]` Comprehensive Mobile Optimization:
  - **Mobile Bottom Navigation**: Added sticky bottom navigation bar for mobile viewports (`sm:hidden fixed bottom-0 left-0 right-0 z-40`) providing one-thumb access to Store, AI Copilot, Cart (with live badge), and Orders/Profile.
  - **2-Column Smartphone Grid**: Upgraded catalog layout to a 2-column mobile grid (`grid-cols-2 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4`).
  - **Mobile Bottom Sheets**: Converted product detail modals, cart drawer, and AI chat to fluid mobile bottom sheets / full viewports.
  - **Horizontal Touch Carousels**: Made category and subcategory filter tabs swipeable horizontally on mobile devices.
- `[AI Search & Dynamic Database Taxonomy]` Elimination of Hardcoded Categories:
  - **Live Dynamic Taxonomy**: Refactored `ProductRepository.get_taxonomy_prompt_text()` and `SearchProductsTool.to_openai_schema()` to query `categories` LEFT JOIN `sub_categories` live from PostgreSQL, injecting the exact schema into the LLM system prompt and tool definitions dynamically.
  - **Information Asymmetry Resolution**: Resolved ambiguous category ID picking in `qwen2.5:3b` by exposing full taxonomy tree at runtime, eliminating hardcoded heuristics.
- `[AI Performance & Threshold Tuning]` Inference Speedup & Threshold Calibration:
  - **Threshold Calibration**: Calibrated `DEFAULT_SEARCH_SCORE_THRESHOLD=0.38` and `CHAT_SEARCH_SCORE_THRESHOLD=0.30`, eliminating low-relevance false positives (e.g. smartwatch/tempered glass appearing for phone queries).
  - **Latency Optimization**: Added `LLM_NUM_PREDICT=350`, `LLM_NUM_CTX=2048`, and `LLM_KEEP_ALIVE="60m"` in `llm_repository.py`, reducing inference time from 70s to ~5s on CPU.
- `[Config & Environment Standardization]` Complete Centralized .env Configuration:
  - **100% Config via .env**: Externalized all configurable thresholds, weights, limits, LLM hyperparameters, and database/redis connection settings to `.env`.
  - **Synchronized .env.example**: Updated comprehensive `.env.example` templates across `ai/`, `backend/`, `frontend/`, and `infra/`.
- `[AI Service & Embeddings]` Migrated Embedding Generation to Ollama API:
  - **Ollama Direct Embeddings**: Refactored `EmbeddingRepository` in Python AI service to call Ollama (`/api/embed` and `/api/embeddings`) directly with dynamic dimension discovery, L2 unit normalization, and fallback resilience.
  - **Lightweight Image**: Removed `sentence-transformers` and PyTorch dependencies from `requirements.txt` and `Dockerfile`.
- `[Golang Backend & Clean Architecture]` Decoupled Backend from Ollama & Relocated AI Client:
  - **Total Ollama Decoupling**: Removed all Ollama HTTP client adapters, configs, and calls from the Go backend. All AI operations are now dispatched through the dedicated Python AI microservice (`AI_SERVICE_URL=http://localhost:8000`).
  - **Dedicated Adapter Package (`internal/client/ai`)**: Moved AI client out of `domain/product` to `internal/client/ai/client.go` implementing `SearchSemantic`, `SyncProducts`, and `GetRecommendations`.
- `[Database Schema & Goose Migrations]` 1024-Dimension Vector Migrations:
  - **Products Embedding (`20260828000002_add_embedding_to_products.sql`)**: Created Goose migration adding `embedding vector(1024)` column and HNSW cosine distance index `idx_products_embedding_hnsw` on `products`.
  - **Knowledge Documents & Chunks (`20260828000003_create_knowledge_tables.sql`)**: Created Goose migration for `knowledge_documents` and `knowledge_chunks` with `embedding vector(1024)` and HNSW index `idx_knowledge_chunks_embedding_hnsw`.

### 📅 2026-08-31

- `[Golang & Clean Architecture]` AI Tool Invocation Refactor to Domain Repositories:
  - **Clean Architecture & Separation of Concerns**: Refactored all AI tools (`SearchProductsTool`, `GetProductDetailTool`, `CheckProductStockTool`, `AddToCartTool`, `GetExecutiveDashboardMetricsTool`, `GetRecentOrdersOverviewTool`, `GetLowStockProductsTool`, `AdjustProductStockTool`) to strictly inject and invoke Domain Repositories (`product.Repository`, `dashboard.Repository`, `order.Repository`) instead of issuing raw SQL or holding direct `*gorm.DB` references.
  - **Dynamic Similarity Scoring**: Eliminated all hardcoded similarity/score values. `SearchProductsTool` now computes real-time mathematical Cosine Similarity using the 1024-dim `bge-m3` vectors combined with keyword matching weights ($0.70 \times \text{vector} + 0.30 \times \text{text}$).
  - **Centralized Database Operations**: All atomic transactions (including warehouse stock adjustment logging in `stock_adjustment_logs`) and hybrid search queries now reside exclusively in repository implementations.
  - **Constructor & DI Wiring**: Updated `ShopperUseCase`, `AdminUseCase`, `ai.Handler`, and `cmd/server/main.go` to wire repositories through Clean Architecture dependency injection.
- `[Golang & Hybrid Search Tuning]` Environment-Driven Search Ratio & Threshold:
  - **Environment-Driven Configuration**: Externalized hybrid search weights and thresholds into `.env` (`DEFAULT_SEARCH_SCORE_THRESHOLD=0.45`, `CHAT_SEARCH_SCORE_THRESHOLD=0.40`, `CHAT_SEARCH_FALLBACK_THRESHOLD=0.25`, `HYBRID_VECTOR_WEIGHT=0.40`, `HYBRID_TEXT_WEIGHT=0.60`).
  - **Dynamic SQL Query Evaluation**: Updated `product.Repository.List` and `product.NewRepository(db, cfg)` to evaluate hybrid score ranking and cutoff thresholds dynamically from config without hardcoded values.
  - **Relevance Precision**: Verified search results for queries like `"celana panjang"` and `"sepatu lari"` strictly return exact and relevant category matches, eliminating loose semantic false positives.
- `[Frontend & Storefront Search]` Always Hybrid Storefront Search & Min 3-Char Policy:
  - **Always Hybrid Search**: Transformed product search to be 100% Always Hybrid by default ($0.70 \times \text{vector} + 0.30 \times \text{text}$). Removed the manual `🧠 AI Semantic ON/OFF` toggle from [`FilterBar.tsx`](file:///c:/Users/Ryzen/Documents/Projects/ai-commerce/tirenn-ai-commerce-frontend/src/components/FilterBar.tsx) and cleaned up redundant `semantic` query params across [`App.tsx`](file:///c:/Users/Ryzen/Documents/Projects/ai-commerce/tirenn-ai-commerce-frontend/src/App.tsx) and [`product.dto.go`](file:///c:/Users/Ryzen/Documents/Projects/ai-commerce/tirenn-ai-commerce-backend/internal/domain/product/dto.go).
  - **Min-3 Char Guardrail**: Enforced minimum 3-character threshold in both React frontend (`App.tsx`) and Go backend (`product.UseCase.ListProducts` & `product.Repository.List`), preventing premature or resource-heavy searches on 1-2 character inputs.
- `[Infra & Production Readiness]` Automated Ollama Model Provisioning Service (`ollama-init`):
  - **Auto-Pull on Deploy**: Added `ollama-init` helper container in [`tirenn-ai-commerce-infra/docker-compose.yml`](file:///c:/Users/Ryzen/Documents/Projects/ai-commerce/tirenn-ai-commerce-infra/docker-compose.yml) (`curlimages/curl:latest`) with `restart: "no"`.
  - **Zero-Manual-Intervention**: On `docker compose up -d`, it waits for Ollama to become healthy and automatically pulls `OLLAMA_CHAT_MODEL` (`qwen2.5:3b`) and `OLLAMA_EMBED_MODEL` (`bge-m3`) from environment variables in `.env`.
- `[AI & Embedding Pipeline]` Upgraded Embedding Model to `bge-m3` (1024-dim dense vectors) via Ollama:
  - **Ollama Model Integration**: Pulled `bge-m3:latest` into `tirenn-ollama` container (dim: 1024, context: 8192 tokens). Updated `.env` to `OLLAMA_EMBED_MODEL=bge-m3`.
  - **Vector Dimension Migration**: Migrated database schema (`products` and `knowledge_chunks`) from `vector(768)` to `vector(1024)`.
  - **Complete Re-Embedding**: Re-embedded 100% of 560 catalog products with 1024-dimensional multilingual embeddings in PostgreSQL pgvector.
- `[Golang - RAG & UTF-8 Encoding Bug Fix]` Fixed PostgreSQL `invalid byte sequence for encoding "UTF8": 0x80` (`SQLSTATE 22021`):
  - **UTF-8 Sanitizer**: Replaced flawed `charmap.Windows1252` decoder (which corrupted valid UTF-8 and generated illegal byte sequences) with `utf8.ValidString` and `strings.ToValidUTF8` stripping along with null-byte sanitization.
  - **In-Memory PDF Parsing**: Integrated `github.com/ledongthuc/pdf` in Go backend `POST /api/v1/knowledge/upload-pdf` to parse multi-page PDFs in-memory without disk writes.
  - **Full Endpoint Parity with AI Service**: Aligned `/api/v1/knowledge/upload-pdf`, `/api/v1/knowledge/query`, `/api/v1/knowledge/documents`, and `/api/v1/knowledge/documents/:id`. Verified RAG query similarity matching with score > 0.63 on policy search.

### 📅 2026-08-28

- `[AI & Embedding Pipeline]` Pure Ollama `paraphrase-multilingual` (768-dim) Integration & Direct Pipeline:
  - **Direct Ollama Embedding**: Pulled `paraphrase-multilingual:latest` into Ollama (`ba13c2e06707`). Go backend connects 100% directly to Ollama API.
  - **Vector Dimension Upgrade**: Migrated database schema and structs from `vector(384)` to `vector(768)`.
  - **Complete Catalog Embeddings**: 100% of all 560 catalog products populated with 768-dimensional multilingual dense embeddings in PostgreSQL pgvector.
- `[QA & E2E Validation]` Complete Playwright & API Integration Test Suite Pass (24/24):
  - **UI E2E**: 18/18 Playwright test scenarios passed across Storefront, PDP, Cart Drawer, AI Shopper, and Admin Panel.
  - **API E2E**: 6/6 Go API test suites passed (Health, Auth/RBAC, Catalog, Product/Stock, Atomic Concurrency, Order Lifecycle & Restock).
- `[Project Architecture & Standardization]` Unified Project Naming `tirenn-ai-commerce`:
  - **Go Module Rename (`go.mod`)**: Migrated module from `gocommerce-backend` to `module tirenn-ai-commerce`.
  - **Source Import Refactor**: Standardized all 37 Go packages to `tirenn-ai-commerce/internal/...`.
  - **Fullstack Alignment**: Updated Frontend (`package.json` -> `tirenn-ai-commerce-frontend`), AI Service (`SERVICE_NAME`), and QA module (`tirenn-ai-commerce-qa`).
- `[Database & Goose Migrations]` PostgreSQL pgvector DDL Conversion & Clean Schema Automation:
  - **DDL Syntax Standardization**: Converted legacy MySQL syntax across all 10 Goose migration files to idiomatic PostgreSQL (`BIGSERIAL`, `TIMESTAMPTZ`, `pg_trgm` GIN trigram indexes, `vector(384)`).
  - **Schema/Data Separation**: Removed redundant GORM `AutoMigrate` in seeder to grant Goose 100% authority over DDL schema.
- `[Golang & AI Product Embedding]` Automated pgvector Dense Embedding for Create, Update, and Seeder:
  - **Create & Update Pipeline (`product.UseCase`)**: Injected `ollamaClient` into `product.UseCase`. `CreateProduct` automatically calculates 384-dimensional dense vector embeddings. `UpdateProduct` detects changes to name/description and recomputes embeddings dynamically.
  - **Concurrent Seeding Worker Pool (`database.ForceSeed`)**: Implemented an 8-worker goroutine pool in `seeder.go` and `cmd/seed` to embed all 560 products in ~25s.
- `[Golang & Tooling]` Database Seeder CLI & Automation Make Commands:
  - **Seeder CLI (`cmd/seed/main.go`)**: Created dedicated seeder tool supporting `-force` reset and seeding all 560 products, users, and categories.
  - **Host Agnostic Connection (`database.go`)**: Implemented automatic `127.0.0.1` fallback when executed outside Docker network.
  - **Make Automation (`Makefile`)**: Added `make seed`, `make db-seed`, and `make db-reset`.
- `[Golang & AI Fullstack]` LLM Temperature, Similarity Thresholds & Hybrid Search Config:
  - **Dynamic Environment Binding (`.env` & `internal/config`)**: Mapped `LLM_TOOL_TEMPERATURE`, `LLM_CHAT_TEMPERATURE`, `DEFAULT_SEARCH_SCORE_THRESHOLD`, `CHAT_SEARCH_SCORE_THRESHOLD`, `CHAT_SEARCH_FALLBACK_THRESHOLD`, `ENABLE_HYBRID_SEARCH`, `HYBRID_VECTOR_WEIGHT`, `HYBRID_TEXT_WEIGHT`, `SEARCH_LIMIT`, `CHAT_SEARCH_LIMIT`.
  - **Hybrid Dense-Vector + Keyword SQL Ranking**: Implemented weighted combination score formula in `SearchProductsTool` with automatic fallback.
  - **Dynamic ReAct Temperatures**: Configured `toolTemperature` (0.0) for deterministic tool parameter invocation and `chatTemperature` (0.3) for natural conversations in `AgentHarness`.
- `[Fullstack Observability & Distributed Tracing]` Distributed Tracing, AI/RAG Telemetry, and Grafana Loki Dashboard:
  - **High-Precision Structured Logger (`internal/logger`)**: Standardized telemetry payload with `func`, `duration_ms`, `event`, `is_slow`, `metadata`. Added `Track(ctx, layer, funcName)` for zero-overhead function timing and bottleneck identification ($> 200\text{ms}$).
  - **AI & LLM Model Telemetry**: Integrated prompt tracking, token/message count, model inference latency in ms (`LLM_RESPONSE`), and ReAct tool execution time (`TOOL_EXECUTION`).
  - **RAG Stage Breakdown**: Created `logger.NewRAGTracker` capturing exact latency across Stage 1 (Vector Retrieval), Stage 2 (Context Augmentation), and Stage 3 (LLM Generation).
  - **Grafana Observability Dashboard**: Pre-provisioned Grafana dashboard (`tirenn-observability` at `:3001`) with P95 latency charts, slow operation alerts, and live log stream.
- `[Golang & Hexagonal Architecture]` AI Subsystem Modularization & External Client Extraction:
  - **External Infrastructure Client (`internal/client/ollama`)**: Extracted all HTTP communication with Ollama LLM and Embeddings to dedicated package in the outer client layer.
  - **Modular ReAct Tools Subpackage (`internal/domain/ai/tools`)**: Organized tools (`catalog.go`, `cart.go`, `analytics.go`, `inventory.go`, `knowledge.go`, `models.go`) into dedicated subpackage with zero circular imports.
  - **Consolidated AI Core**: Unified repositories into `repository.go` and use cases into `usecase.go`, reducing root `domain/ai` file count from ~15 files down to 7 clean, focused files.
- `[Golang & Architecture Overhaul]` Full Clean Architecture & Readability Refactoring:
  - **DTO Standardization**: Created dedicated `dto.go` in all domain modules (`auth`, `product`, `order`, `customer`, `dashboard`, `ai`), strictly separating database entities from presentation DTO schemas.
  - **Domain Errors & Auto HTTP Mapping**: Created `internal/domain/errors.go` and `internal/response/response.go` with automatic error-to-status code translation (404, 401, 403, 409, 400).
  - **Removed Utils Anti-Pattern**: Replaced `internal/utils` with dedicated `internal/security` (`jwt.go`, `hash.go`, `slug.go`) and `internal/response`.
  - **Modular Route Registration**: Implemented `RegisterRoutes(rg *gin.RouterGroup)` on all domain handlers; simplified `internal/router/router.go` to ~50 lines.
- `[Golang & Code Architecture]` Removed `ai_client.go` & Consolidated Recommendations into `product.Repository`:
  - Removed redundant `ai_client.go` file and interface entirely.
  - Implemented `GetRecommendations(ctx, productID, limit)` natively in `product.Repository` using PostgreSQL pgvector cosine similarity (`<=>`) and category soft-boost.
  - Cleaned `product.UseCase` constructor to standard `NewUseCase(repo, rdb)` adhering to Clean Architecture principles.
- `[Golang & Code Architecture]` Refactored AI Domain Structs into Dedicated Module (`models.go`):
  - Extracted all inline function-scoped structs into package-level DTOs: `CatalogProductRow`, `CatalogProductDetail`, `CartProductInfo`, `AnalyticsTopProduct`, `AnalyticsOrderRow`, `InventoryLowStockProduct`, `InventoryProductRecord`, `InventoryStockAdjustmentLog`.
  - Zero inline struct definitions remaining in function bodies across all tool implementations.
- `[Frontend & Fullstack Integration]` Unified 100% of Web API Requests to Golang Backend (`:8080`):
  - Configured `tirenn-frontend` to route all AI endpoints (`/chat/shopper`, `/chat/admin`, `/knowledge/*`, `/catalog/search`) directly through the compiled Golang backend on port `:8080`.
  - Updated `src/services/api.ts`, `AdminAIChatDrawer.tsx`, and `KnowledgeManagement.tsx` to default all AI API requests to `getApiBaseUrl()`.
  - Set `VITE_AI_SERVICE_URL=http://localhost:8080/api/v1` in `tirenn-frontend/.env`.
  - Verified Vite production build (`npm run build`) succeeded with 0 errors in 1.36s.
- `[Golang & AI Engine]` Migrated AI Service Engine to Native Golang (`tirenn-backend/internal/domain/ai`):
  - **Embedding Consistency**: Standardized on `sentence-transformers/paraphrase-multilingual-MiniLM-L12-v2` across both Golang and Python to guarantee 100% vector space consistency for 560 catalog products.
  - **ReAct Agent Harness**: Autonomous multi-turn tool-calling loop (`AgentHarness`) with Ollama `/api/chat` integration and 0.2 temperature.
  - **Ollama Client**: Native Go HTTP client for Ollama LLM chat and 384-dimensional text embeddings (`/api/embeddings`).
  - **Customer Tools Layer**: `SearchProductsTool` (pgvector `<=>` cosine similarity + category filter), `GetProductDetailTool`, `CheckProductStockTool`, `AddToCartTool`, `ViewCartTool`, `SearchStorePoliciesAndSOPTool` (`doc_type = 'SOP_CUSTOMER'`).
  - **Admin Tools Layer**: `GetExecutiveDashboardMetricsTool`, `GetRecentOrdersOverviewTool`, `GetLowStockProductsTool`, `AdjustProductStockTool` (2-step confirmation guardrail with atomic GORM SQL transaction + `stock_adjustment_logs`), `SearchAdminInternalSOPTool` (`doc_type = 'SOP_ADMIN'`).
  - **Redis Sessions & 2-Tier RAG Cache**: Sliding window buffer (`chat:session:{session_id}`) with 24h TTL and exact normalized SHA-256 hash cache (`rag:exact:{doc_type}:{hash}`) with 2h TTL.
  - **Endpoints Registered**: `POST /api/v1/chat/shopper`, `POST /api/v1/chat/admin`, `DELETE /api/v1/chat/session/:id`, `GET /api/v1/catalog/search`, `POST /api/v1/knowledge/upload`, `POST /api/v1/knowledge/ask`, `GET /api/v1/knowledge/documents`, `DELETE /api/v1/knowledge/documents/:id`.
  - **Preserved Existing Codebase**: Original Python `tirenn-ai-service/` preserved 100% intact.
  - **Test Suite**: All unit tests in `internal/domain/ai/ai_test.go` and full Go test suite PASSED.
- `[AI Service]` Implemented Canonical 4-Layer Single Agent Harness Architecture:
  - **Layer 1 (Chat UI)**: Pure frontend rendering connected strictly to Agent API (`POST /api/v1/chat/shopper`).
  - **Layer 2 (Agent Orchestrator)**: Domain-agnostic ReAct tool-calling loop with configurable max iteration cap (`max_iterations = 5`) and stateless per-session processing.
  - **Layer 3 (Tool Layer)**: Fixed, deterministic tools (`search_products`, `explain_product_details`, `check_product_stock`, `add_to_cart`, `view_cart`, `search_store_policies_and_sop`) returning **100% structured factual JSON facts and zero subjective opinions**. The LLM forms all conversational explanations.
  - **Layer 4 (Commerce Backend)**: Go Backend & PostgreSQL/pgvector acting as immutable Source of Truth with server-side stock re-validation at cart mutation time.
- `[Teamwork Multi-Agent]` Successfully Built & Verified End-to-End Real-Time AI Product Recommendation & Similar Items System:
  - **`[AI Service]`**: Implemented high-performance pgvector cosine distance search (`<=>`) on 384-dimensional embeddings with category/subcategory affinity soft-boost (+0.15 subcategory, +0.08 category), dynamic price corridor ($0.4x - 2.5x$), and `order_items` co-occurrence aggregation.
  - **`[Golang]`**: Implemented `GET /api/v1/products/:id/recommendations` with Redis Cache-Aside layer (key `recommendations:product:{id}`, 1-hour TTL `3600s`) and deterministic fallback to category top-sellers. 14/14 Go unit/integration tests passed.
  - **`[Frontend]`**: Integrated horizontal recommendation carousel in Product Detail Modal (`ProductDetailModal.tsx`) and contextual add-ons in Cart Drawer (`CartDrawer.tsx`) with 1-click Add to Cart, currency formatting (`IDR`/`USD`), and real-time badge updates.
  - **`[QA & E2E]`**: Added E2E Test #9 in `tirenn-infra/qa/e2e/storefront.spec.ts`. All 18 Playwright E2E tests passing.
  - **`[Victory Audit]`**: Verified live response <100ms when cached, zero container errors, and 100% acceptance criteria fulfillment.
- `[AI Service]` Cleaned and pruned dead code across `tirenn-ai-service`:
  - Deleted legacy in-memory session folder `app/harness/memory/` (superseded by Redis `SessionRepository`).
  - Deleted ad-hoc test file `app/harness/test_relevance_guardrail.py`.
  - Deleted unused domain model `app/domain/category.py` and duplicate schemas `app/schemas/search.py`.
  - Removed unused `get_product_by_sku_or_name` from `ProductRepository`.
  - Removed backward-compatibility aliases and unused regex modules from tools.
- `[AI Service]` Integrated Redis Session Management (`SessionRepository`):
  - Session chat history stored under `chat:session:{session_id}` with auto-expiring 24h TTL.
  - Added `DELETE /api/v1/chat/session/{session_id}` endpoint to purge sessions on demand.
- `[Frontend]` Integrated Redis Chat Session Lifecycle into `AIChatModal.tsx`:
  - Maintained unique `sessionId` in `localStorage` and dispatched `session_id` payload on `/chat/shopper`.
  - Added asynchronous `DELETE /api/v1/chat/session/{sessionId}` call when the user clicks the "Reset Chat" button, instantly purging conversation memory from Redis and regenerating a clean session token.
- `[Frontend]` Enhanced `AIChatModal.tsx` `cart_action` handler to reliably dispatch items into `CartContext` supporting both direct top-level payload attributes and nested `cartAction.product` objects, ensuring items added via AI chat immediately appear in Cart Drawer and update the Cart badge.
- `[Golang]` Fixed Admin Product Management Catalog Retrieval:
  - Registered `admin.GET("/products", handlers.Product.AdminListProducts)` in `internal/router/router.go`.
  - Updated `tirenn-backend/.env` with container hostnames (`DB_HOST=postgres`, `REDIS_HOST=redis`) for Docker network connectivity.
  - Successfully validated retrieval of all 560 catalog products in Admin Product Management.
- `[AI Service]` Enforced Hard Data Isolation for Customer SOP (`doc_type='SOP_CUSTOMER'`):
  - Locked `SearchStorePoliciesAndSOPTool` to strictly query customer-facing knowledge documents (`doc_type="SOP_CUSTOMER"`), physically barring the public AI Shopper from retrieving internal merchant or administrative SOPs from PostgreSQL.
  - Enhanced `SYSTEM_PROMPT` with strict customer-only data scope directives.
- `[AI Service]` Pure LLM Native Reasoning for 2-Step Confirmation (Zero Hardcoded Dictionaries):
  - Removed all hardcoded affirmative/cancel keyword sets and interceptors.
  - The model natively inspects conversational history, determines whether user agreed/confirmed (in any language or phrasing), and autonomously executes `adjust_product_stock` with `confirmed=true` or acknowledges cancellation.
  - `ProductRepository.adjust_stock` commits atomic SQL transactions directly in PostgreSQL with audit log recording in `stock_adjustment_logs`.
  - **`AnalyticsRepository` (`app/repositories/analytics_repository.py`)**: Direct PostgreSQL queries for real-time dashboard KPIs, financial totals, order items aggregation, and order histories (`get_dashboard_summary`, `get_top_selling_products`, `get_recent_orders`).
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
- `[Frontend]` Implemented Stock Adjustment Confirmation Guardrail (`StockAdjustmentModal.tsx`):
  - Added 2-step verification guardrail displaying projected warehouse stock changes (Current Stock $\rightarrow$ Projected New Stock), operation type, and audit reasons with warning banners before submitting mutations.
  - Fully localized in Indonesian & English via `react-i18next` under `"admin"`.
- `[Frontend]` Implemented Admin AI Copilot UI & Rich Markdown Formatting (`AdminAIChatDrawer.tsx`):
  - Added `AdminFormattedMessage` component supporting `**bold text**`, `*italic*`, `` `inline code / SKU` ``, bullet lists (`- `, `* `), numbered lists (`1. `), and header formatting (`### `), fixing unparsed markdown asterisks.
  - Created interactive sliding drawer for Admin Control Panel (`/admin`) with one-click quick action chips (Revenue KPIs, Low Stock Warnings, Warehouse Picking SOP, Recent Orders).
  - Added "⚡ Admin AI Copilot" button in Admin Navigation Tabs and floating action button on bottom-right of Admin Dashboard.
  - Fully localized in Indonesian & English via `react-i18next` under `"admin_copilot"`.
- `[AI Service]` Implemented 2-Tier Redis Semantic RAG Cache in `KnowledgeUseCase`:
  - **Tier 1 (Exact Hash Match)**: Stores hash keys (`rag:exact:{scope}:{hash}`) in Redis with 24h TTL for $< 1\text{ms}$ instantaneous responses.
  - **Tier 2 (Semantic Vector Similarity Match)**: Stores query embeddings in Redis List (`rag:semantic:{scope}`) and evaluates cosine similarity against active clusters in in-memory RAM. When $\ge 92\%$ similarity is reached, returns cached excerpts in $\approx 5\text{ms}$ without invoking PostgreSQL.
  - **Auto-Invalidation**: Flushes all `rag:exact:*` and `rag:semantic:*` keys automatically whenever an Admin uploads or deletes SOP PDF documents in `KnowledgeManagement`.
- `[Frontend]` Full i18n Localization Refactor (`KnowledgeManagement.tsx` & `AIChatModal.tsx`):
  - Extracted all remaining hardcoded strings, toasts, badges, and playground UI text into `src/locales/en.json` and `src/locales/id.json` under `knowledge` and `ai_chat` keys.
  - Replaced inline `isEn ? ... : ...` ternary logic with declarative `t('...')` translation calls with dynamic interpolation (`{{title}}`, `{{chunks}}`, `{{name}}`, `{{qty}}`).
- `[Frontend]` Enhanced `KnowledgeManagement.tsx` with Clear Access Scope Badges (`🛍️ Customer Guide` vs `🔒 Admin Internal`) and Added Scope Filter to the Semantic Vector RAG Playground.
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
  - Added adaptive category relaxation fallback in `SearchProductsTool` to prevent zero results if the LLM provides an erroneous category ID.
- `[AI Service]` Enforced minimum query length requirement (>= 3 characters) across semantic search (`SearchUseCase.execute`, `SearchProductsTool`, `AddToCartTool`, `_resolve_target_product`).
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
- `[Frontend]` Added dynamic language synchronization effect for the initial welcome message in `AIChatModal.tsx` when switching between ID and EN.
- `[AI Service]` Fixed product resolution hierarchy in `AddToCartTool` so explicit/partial product names (`masukan keranjang [nama produk]`) always take precedence over ordinal heuristics.
- `[AI Service]` Restricted AI Chat product recommendation output to maximum **6 items** (`CHAT_SEARCH_LIMIT=6`) across `config.py`, `agent.py`, `catalog_tools.py`, and `chat_handler.py`.
- `[AI Service]` Enhanced `AddToCartTool` with conversational history context and ordinal/numeric index resolution (e.g. automatically resolving `"masukkan kekeranjang 1"`, `"nomor 1"`, `"pertama"` to the target item from previous recommendations).
- `[AI Service]` Updated `AgentHarness` with cart intent fallback to guarantee `add_to_cart` execution and prevent accidental invocation of `search_products` on cart commands.
- `[Frontend]` Sliced `suggestedProducts` to maximum 6 items in `AIChatModal.tsx`.
- `[Infra]` Removed all hardcoded environment values across `docker-compose.yml` files (`tirenn-infra`, `tirenn-backend`, `tirenn-ai-service`, `tirenn-frontend`), switching entirely to `env_file: .env`.
- `[Infra]` Cleaned all `.env.example` templates to ensure actual environment values are stored exclusively in `.env` files.
- `[AI Service]` Configured `INTERNAL_API_KEY` to secure internal machine-to-machine catalog synchronization and vector re-indexing endpoints (`/index-products`, `/sync-from-backend`).
- `[Golang]` Configured `INTERNAL_API_KEY` in `internal/config/config.go` and injected `X-API-Key` headers in `internal/domain/product/ai_client.go`.
- `[Golang]` Created Goose SQL migration `migrations/20260828000002_create_knowledge_tables.sql` for `knowledge_documents` and `knowledge_chunks` with HNSW vector cosine indexing.
- `[Golang]` Created `KnowledgeDocument` and `KnowledgeChunk` GORM domain entities in `internal/domain/knowledge/entity.go` and integrated them into `AutoMigrate`, removing all ad-hoc raw `db.Exec` DDL statements.
- `[AI Service]` Removed DDL schema creation statements from `tirenn-ai-service/app/repositories/knowledge_repository.py`, establishing Golang as the single source of truth for database migrations.
- `[PM]` Designed and structured Hybrid Agentic RAG architecture combining structured SQL tools with vector knowledge base for store SOPs and policy management.
- `[PM]` Created official bilingual Standard Operating Procedures:
  - `docs/sop/SOP_Customer_Guide.pdf` & `SOP_Customer_Guide.md` (Customer shopping guide, payment methods, warranty, and 7-day return policy).
  - `docs/sop/SOP_Admin_Operations.pdf` & `SOP_Admin_Operations.md` (Merchant operations, order fulfillment SLA, picking/packing, and stock adjustments).
- `[PM]` Configured multi-tier GitHub `CODEOWNERS` routing for root, frontend, backend, ai-service, and infra under `@tirenn`.
- `[AI Service]` Implemented `AgentHarness` with `RelevanceGuardrail` to filter contradictory/irrelevant product search returns (e.g. removing shorts when searching long pants).
- `[AI Service]` Built in-memory PDF extraction with `pypdf.PdfReader(io.BytesIO)` — zero disk writes.
- `[AI Service]` Implemented direct JWT validation (`verify_admin_jwt`) using `PyJWT` with `HS256` matching Golang backend's `JWT_SECRET`.
- `[AI Service]` Created PostgreSQL `pgvector` knowledge repository (`knowledge_documents` and `knowledge_chunks`) with 384-dimensional dense vectors and HNSW cosine distance indexing.
- `[AI Service]` Added `SearchStorePoliciesAndSOPTool` into the Shopper Agent Harness to answer store policy, SLA, and return queries directly from vector chunks.
- `[AI Service]` Pre-indexed 23 vector chunks from Customer SOP (13 chunks) and Admin SOP (10 chunks).
- `[Frontend]` Built Admin Knowledge Management page (`src/components/admin/KnowledgeManagement.tsx`) featuring drag-and-drop PDF upload, indexed document catalog, and real-time Semantic Vector RAG playground.
- `[Frontend]` Added `"📚 SOP & AI Knowledge"` navigation tab in Admin Navbar and linked quick action in Executive Dashboard.
- `[QA]` Added Admin Knowledge Management E2E test to Playwright suite (`tirenn-infra/qa/e2e/admin.spec.ts`).
- `[QA]` Configured Playwright with 2 workers for local CPU LLM stability; verified **12 of 12 E2E tests passing 100%**.
