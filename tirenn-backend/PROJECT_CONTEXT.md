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

## 📜 Service Changelog

- `[Golang]` Implemented Product Recommendation Endpoint & Redis Caching (`GET /api/v1/products/:id/recommendations`):
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
