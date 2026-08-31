# 📋 Project Context: Tirenn Backend

This document describes the package organization, domain models, and API interfaces of `tirenn-backend`.

---

## 🏛️ Project Structure & Clean Domain Packages

The Go backend follows Clean Domain Architecture:

```
internal/
├── database/            # PostgreSQL connection, pgvector extension, AutoMigrate & seeders
├── logger/              # Structured JSON logging compatible with Grafana Loki & Promtail
├── middleware/          # JWT auth, CORS, Redis sliding-window rate limiting, request tracing
├── router/              # Gin HTTP router mounting domain endpoints
├── utils/               # Standardized JSON response envelopes, pagination, error formatters
└── domain/
    ├── admin/           # Executive KPIs, analytics aggregation, dashboard overview
    ├── auth/            # JWT authentication, login, registration, /auth/me profile
    ├── customer/        # Customer directory, user management
    ├── knowledge/       # KnowledgeDocument & KnowledgeChunk GORM models & migrations
    ├── order/           # Order processing, transactional checkout, status workflows
    └── product/         # Product catalog, categories, stock adjustment audit logs
```

---

## 🗄️ PostgreSQL Database Entities & Migrations

> **Rule**: All database changes **MUST** be created using Goose SQL migrations via `make migrate-create name=<migration_name>` and placed in `migrations/`, alongside corresponding GORM domain models.

- **`users`**: User identity, hashed passwords, roles (`ADMIN` / `CUSTOMER`).
- **`categories`** & **`sub_categories`**: Product taxonomy (e.g. *Elektronik & Gadget*, *Fashion Pria*, *Fashion Wanita*).
- **`products`**: Product inventory, SKU, pricing, stock levels, `embedding vector(384)`, full-text search fields.
- **`stock_adjustment_logs`**: Complete audit trail of stock adjustments (`ADD`, `SUBTRACT`, `SET`, admin ID, reason).
- **`orders` & `order_items`**: Order transactions, total amount, shipping addresses, items snapshot.
- **`knowledge_documents`**: RAG Knowledge Base documents (title, doc_type, filename, total_pages, total_chunks, created_at).
- **`knowledge_chunks`**: RAG Knowledge Chunks (document_id FK, chunk_index, content, page_number, `embedding vector(384)`, HNSW vector cosine index `idx_knowledge_chunks_embedding_hnsw`).

---

## 🌐 API Endpoints

### Public Catalog
- `GET /api/v1/products`: List products with pagination, search, category filter, in-stock filter.
- `GET /api/v1/products/:id`: Get product details.
- `GET /api/v1/categories`: List all categories.

### Authentication
- `POST /api/v1/auth/login`: Authenticate and receive JWT token.
- `POST /api/v1/auth/register`: Create customer account.
- `GET /api/v1/auth/me`: Get current authenticated user profile.

### Orders
- `POST /api/v1/orders/checkout`: Place an atomic order with stock deduction.
- `GET /api/v1/orders/my`: Get current user order history.

### Admin
- `GET /api/v1/admin/analytics`: Executive KPI summaries and revenue metrics.
- `POST /api/v1/admin/products`: Create new product.
- `PUT /api/v1/admin/products/:id`: Update product.
- `DELETE /api/v1/admin/products/:id`: Soft-delete product.
- `POST /api/v1/admin/products/:id/stock`: Adjust stock with reason.
- `GET /api/v1/admin/orders`: List customer orders.
- `PATCH /api/v1/admin/orders/:id/status`: Update order status.
- `GET /api/v1/admin/customers`: List customer directory with metrics.

---

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
  - **Seeder CLI Entrypoint (`cmd/seed/main.go`)**: Created dedicated seeder tool supporting `-force` reset.
  - **Host Agnostic Connection (`database.go`)**: Implemented automatic `127.0.0.1` fallback when executed outside Docker network.
  - **Make Targets (`Makefile`)**: Added `make seed`, `make db-seed`, and `make db-reset`.
- `[Golang & AI Config]` LLM Temperature, Similarity Thresholds & Hybrid Search Parameters:
  - **Dynamic Environment Binding (`.env` & `internal/config`)**: Mapped `LLM_TOOL_TEMPERATURE`, `LLM_CHAT_TEMPERATURE`, `DEFAULT_SEARCH_SCORE_THRESHOLD`, `CHAT_SEARCH_SCORE_THRESHOLD`, `CHAT_SEARCH_FALLBACK_THRESHOLD`, `ENABLE_HYBRID_SEARCH`, `HYBRID_VECTOR_WEIGHT`, `HYBRID_TEXT_WEIGHT`, `SEARCH_LIMIT`, `CHAT_SEARCH_LIMIT`.
  - **Hybrid Dense-Vector + Keyword SQL Ranking**: Implemented weighted combination score formula in `SearchProductsTool` with automatic fallback.
  - **Dynamic ReAct Temperatures**: Configured `toolTemperature` (0.0) for deterministic tool parameter invocation and `chatTemperature` (0.3) for natural conversations in `AgentHarness`.
- `[Golang & Observability]` Distributed Tracing, AI & RAG Telemetry, and Grafana Loki Integration:
  - **High-Precision Structured Logger (`internal/logger`)**: Standardized telemetry payload with `func`, `duration_ms`, `event`, `is_slow`, `metadata`. Added `Track(ctx, layer, funcName)` for zero-overhead function timing and bottleneck identification ($> 200\text{ms}$).
  - **AI & LLM Model Telemetry**: Integrated prompt tracking, token/message count, model inference latency in ms (`LLM_RESPONSE`), and ReAct tool execution time (`TOOL_EXECUTION`).
  - **RAG Stage Breakdown**: Created `logger.NewRAGTracker` capturing exact latency across Stage 1 (Vector Retrieval), Stage 2 (Context Augmentation), and Stage 3 (LLM Generation).
  - **Grafana Observability Dashboard**: Pre-provisioned Grafana dashboard (`tirenn-observability`) with P95 latency charts, slow operation alerts, and live log stream.
- `[Golang]` AI Domain Reorganization & External Client Extraction:
  - **External Adapter Extraction (`internal/client/ollama`)**: Relocated Ollama HTTP client logic out of `domain/ai` into dedicated infrastructure client package with 100% self-contained request/response schemas.
  - **Modular ReAct Tools Subpackage (`internal/domain/ai/tools`)**: Moved all ReAct tool implementations (`catalog.go`, `cart.go`, `analytics.go`, `inventory.go`, `knowledge.go`, `models.go`) into dedicated subpackage.
  - **Consolidated AI Core**: Unified repositories into `repository.go` and use cases into `usecase.go`, reducing root `domain/ai` file count from ~15 files down to 7 clean, focused files.
- `[Golang]` Full Clean Architecture & Readability Refactoring:
  - **DTO Standardization**: Created dedicated `dto.go` in all domain modules (`auth`, `product`, `order`, `customer`, `dashboard`, `ai`), separating domain models from presentation schemas.
  - **Domain Errors & Auto HTTP Mapping**: Created `internal/domain/errors.go` and `internal/response/response.go` with automatic error-to-status code translation.
  - **Removed Utils Anti-Pattern**: Replaced `internal/utils` with dedicated `internal/security` (`jwt.go`, `hash.go`, `slug.go`) and `internal/response`.
  - **Modular Route Registration**: Implemented `RegisterRoutes(rg *gin.RouterGroup)` on all domain handlers; simplified `internal/router/router.go` to ~50 lines.
- `[Golang]` Removed `ai_client.go` & Consolidated Recommendations into `product.Repository`:
  - Removed redundant `ai_client.go` file and interface entirely.
  - Implemented `GetRecommendations(ctx, productID, limit)` natively in `product.Repository` using PostgreSQL pgvector cosine similarity (`<=>`) and category soft-boost.
  - Cleaned `product.UseCase` constructor to standard `NewUseCase(repo, rdb)` adhering to Clean Architecture principles.
- `[Golang]` Refactored AI Domain Structs into Dedicated Module (`models.go`):
  - Extracted all inline function-scoped structs into package-level DTOs: `CatalogProductRow`, `CatalogProductDetail`, `CartProductInfo`, `AnalyticsTopProduct`, `AnalyticsOrderRow`, `InventoryLowStockProduct`, `InventoryProductRecord`, `InventoryStockAdjustmentLog`.
  - Zero inline struct definitions remaining in function bodies across all tool implementations.
- `[Golang]` Migrated and Integrated Native AI Engine (`internal/domain/ai`):
  - **Embedding Consistency**: Configured `sentence-transformers/paraphrase-multilingual-MiniLM-L12-v2` as the unified 384-dimensional multilingual embedding model matching existing catalog embeddings.
  - **ReAct Agent Harness**: Autonomous multi-turn tool-calling loop (`AgentHarness`) with Ollama `/api/chat` integration and 0.2 temperature.
  - **Ollama Client**: Native Go HTTP client for Ollama LLM chat and 384-dimensional text embeddings (`/api/embeddings`).
  - **Customer Tools Layer**: `SearchProductsTool` (pgvector `<=>` cosine similarity + category filter), `GetProductDetailTool`, `CheckProductStockTool`, `AddToCartTool`, `ViewCartTool`, `SearchStorePoliciesAndSOPTool` (`doc_type = 'SOP_CUSTOMER'`).
  - **Admin Tools Layer**: `GetExecutiveDashboardMetricsTool`, `GetRecentOrdersOverviewTool`, `GetLowStockProductsTool`, `AdjustProductStockTool` (2-step confirmation guardrail with atomic GORM SQL transaction + `stock_adjustment_logs`), `SearchAdminInternalSOPTool` (`doc_type = 'SOP_ADMIN'`).
  - **Redis Sessions & 2-Tier RAG Cache**: Sliding window buffer (`chat:session:{session_id}`) with 24h TTL and exact normalized SHA-256 hash cache (`rag:exact:{doc_type}:{hash}`) with 2h TTL.
  - **Endpoints Registered**: `POST /api/v1/chat/shopper`, `POST /api/v1/chat/admin`, `DELETE /api/v1/chat/session/:id`, `GET /api/v1/catalog/search`, `POST /api/v1/knowledge/upload`, `POST /api/v1/knowledge/ask`, `GET /api/v1/knowledge/documents`, `DELETE /api/v1/knowledge/documents/:id`.
  - **Preserved Existing Codebase**: Original Python `tirenn-ai-service/` preserved 100% intact.
  - **Test Suite**: All unit tests in `internal/domain/ai/ai_test.go` and full Go test suite PASSED.
  - Added `AIClient.GetRecommendations` calling `tirenn-ai-service` with `X-API-Key` authentication and timeout safeguards.
  - Implemented Cache-Aside pattern in Redis under `recommendations:product:{id}` with 1-hour TTL (`3600s`) and mutation invalidation.
  - Implemented deterministic fallback querying category/store top-sellers if AI service is temporarily unreachable.
  - 14/14 Go unit/integration tests passing (including 1,000 concurrent request load test).
- `[Golang]` Configured `INTERNAL_API_KEY` in `internal/config/config.go` and wired `X-API-Key` headers in `internal/domain/product/ai_client.go`.
- `[Golang]` Fixed Admin Product Management Catalog Retrieval:
  - Registered `admin.GET("/products", handlers.Product.AdminListProducts)` in `internal/router/router.go`.
  - Updated `tirenn-backend/.env` with container hostnames (`DB_HOST=postgres`, `REDIS_HOST=redis`) for Docker network connectivity.
  - Successfully validated retrieval of all 560 catalog products in Admin Product Management.
- `[Golang]` Enforced Clean Architecture & Migration Rules:`migrations/20260828000002_create_knowledge_tables.sql` for `knowledge_documents` and `knowledge_chunks` with HNSW vector cosine indexing.
- `[Golang]` Added `KnowledgeDocument` and `KnowledgeChunk` GORM models under `internal/domain/knowledge/entity.go`.
- `[Golang]` Integrated `&knowledge.KnowledgeDocument{}` and `&knowledge.KnowledgeChunk{}` into `db.AutoMigrate(...)` in `internal/database/seeder.go`, removing all ad-hoc raw DDL executions.
- `[Golang]` Rebuilt and restarted backend Docker container.
