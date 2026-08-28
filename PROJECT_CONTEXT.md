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

### 📅 2026-08-28

- `[AI Service]` Implemented Canonical 4-Layer Single Agent Harness Architecture:
  - **Layer 1 (Chat UI)**: Pure frontend rendering connected strictly to Agent API (`POST /api/v1/chat/shopper`).
  - **Layer 2 (Agent Orchestrator)**: Domain-agnostic ReAct tool-calling loop with configurable max iteration cap (`max_iterations = 5`) and stateless per-session processing.
  - **Layer 3 (Tool Layer)**: Fixed, deterministic tools (`search_products`, `explain_product_details`, `check_product_stock`, `add_to_cart`, `view_cart`, `search_store_policies_and_sop`) returning **100% structured factual JSON facts and zero subjective opinions**. The LLM forms all conversational explanations.
  - **Layer 4 (Commerce Backend)**: Go Backend & PostgreSQL/pgvector acting as immutable Source of Truth with server-side stock re-validation at cart mutation time.
- `[AI Service]` Cleaned and pruned dead code across `tirenn-ai-service`:
  - Deleted legacy in-memory session folder `app/harness/memory/` (superseded by Redis `SessionRepository`).
  - Deleted ad-hoc test file `app/harness/test_relevance_guardrail.py`.
  - Deleted unused domain model `app/domain/category.py` and duplicate schemas `app/schemas/search.py`.
  - Removed unused `get_product_by_sku_or_name` from `ProductRepository`.
  - Removed backward-compatibility aliases and unused regex modules from tools.
- `[AI Service]` Integrated Redis Session Management (`SessionRepository`):
  - Session chat history stored under `chat:session:{session_id}` with auto-expiring 24h TTL.
  - Added `DELETE /api/v1/chat/session/{session_id}` endpoint to purge sessions on demand.
- `[Frontend]` Enhanced `AIChatModal.tsx` `cart_action` handler to reliably dispatch items into `CartContext` supporting both direct top-level payload attributes and nested `cartAction.product` objects, ensuring items added via AI chat immediately appear in Cart Drawer and update the Cart badge.
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
