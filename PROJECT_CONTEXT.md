# 🛍️ Tirenn Commerce - Master Project Context & Development Log

> **Note for AI Assistant & Developers**: This document contains the full architecture, decisions, file maps, migrations, testing matrices, and operational context for **Tirenn Commerce**. Keep this file updated as new features are added.

---

## 📌 1. Project Overview & Architecture

**Tirenn Commerce** is a full-stack, production-grade **modern e-commerce marketplace and department store platform** designed with an ultra-simple, minimal, and high-converting UI:

- **Branding**: **Tirenn Commerce** (`tirenn commerce`) - The Modern Online Marketplace.
- **🐘 Unified Database & Vector Engine (PostgreSQL 16 + pgvector)**:
  - Container `tirenn-postgres` running `pgvector/pgvector:pg16` on Port `5432`.
  - Database name: `gocommerce_db`, user: `gouser`.
  - Storing both relational business data (Users, Categories, Products, Orders, Order Items, Stock Logs) and 384-dimensional dense vectors (`products.embedding`) with **`HNSW`** index.
  - Replaces external vector DBs (e.g. Qdrant) with unified SQL Cosine distance (`<=>`) queries, eliminating dual-write sync issues and stale search data.
- **🤖 Conversational AI Shopper (Agentic Tool Calling + RAG)**:
  - **Local LLM Engine**: **Ollama** running `qwen2.5:3b` in Docker (`tirenn-ollama` on Port `11434`).
  - **Tool Calling Architecture**: The LLM calls 3 autonomous tools:
    1. `search_products`: Native `pgvector` Cosine distance similarity with `min_price`, `max_price`, `in_stock`, and `category_id` filters.
    2. `check_product_stock`: Real-time inventory and pricing lookup via Go backend API.
    3. `add_to_cart`: Autonomous item addition to customer cart.
  - **Conversational Modal UI**: Interactive chat with native Markdown support (`**bold**`, lists, bullets), product card previews with thumbnails, SKU badges, dynamic tool execution indicators, and **Reset Chat** functionality.
- **🧠 Python AI Semantic Search Microservice (`ai-service/`)**:
  - **Framework**: **FastAPI** + **Uvicorn** running on Port 8000.
  - **Local Embedding Engine**: **SentenceTransformers** (`sentence-transformers/paraphrase-multilingual-MiniLM-L12-v2`) running locally with native **Bahasa Indonesia** understanding (~220MB model size, 384 dimensions, string-literal pgvector serialization).
  - **PostgreSQL Vector Adapter**: Direct connection via `psycopg2` and `pgvector` Python client.
  - **Endpoints**: `POST /api/v1/search/semantic`, `POST /api/v1/chat/shopper`, `POST /api/v1/index-products`, `POST /api/v1/sync-from-backend`, `GET /healthz`.
- **📊 Centralized Observability & Logging Stack (`logging/`)**:
  - **Grafana Loki 3.0** (Port `3100`): Centralized log aggregation engine.
  - **Grafana Promtail 3.0** (Port `9080`): Automatic Docker container log scraper via `/var/run/docker.sock`.
  - **Grafana Dashboard** (Port `3001`): Web UI for LogQL search, live streaming, error filtering, and trace correlation (`admin`/`admin`).
- **🆔 End-to-End Tracing & Structured Error Logging**:
  - **RequestID Middleware**: Generates/preserves `X-Request-ID`, injects it into Go `context.Context`, and returns it in headers and JSON responses.
  - **Layer-by-Layer Error Logging**: `logger.Error(ctx, layer, msg, err)` records structured `APP_LOG` entries across Handlers, UseCases, and Repositories with caller file/line numbers.
  - **AI Response Audit Trail**: `AI_RESPONSE_LOG` records prompt, tool parameters, execution output, latency, and full AI synthesized answers.
- **Auto-Pagination (Infinite Scrolling)**: Native `IntersectionObserver`-based auto pagination on the Storefront (`12 products/batch`) with animated loading spinners and completion indicators (`✓ Semua 100 produk telah ditampilkan`).
- **Catalog & Localization**: **100 Products across 8 Categories in Bahasa Indonesia** with authentic Indonesian names, product descriptions, localized user profiles, and Indonesian Rupiah (`Rp` / `IDR`) pricing.
- **Frontend Stack**: **React 19 + TypeScript + Vite + Tailwind CSS 4** (clean component architecture in `frontend/src/`).
- **Backend Stack**: **Golang (Go 1.24+)**, Gin framework, GORM (PostgreSQL Driver), Viper configuration loader, and JWT RBAC.
- **QA Automation Workspace (`qa/`)**:
  - **Browser E2E Testing**: **Playwright** located in `qa/` (`qa/playwright.config.ts`, `qa/e2e/`) testing all user-facing storefront and admin journeys in Chromium.
  - **Automatic Post-Test Cleanup**: Configured `posttest:e2e` hook (`qa/scripts/clean-reports.js`) that automatically purges all `test-results/` and `playwright-report/` directories after every test run.
- **Infrastructure & Containerization**: 8 containerized Docker services (`tirenn-postgres`, `tirenn-ollama`, `tirenn-ai-service`, `tirenn-backend`, `tirenn-frontend`, `tirenn-loki`, `tirenn-promtail`, `tirenn-grafana`).

---

## 🎨 2. 100 Seeded Indonesian Products Across 8 Categories

| No | Kategori (Category) | Jumlah Produk | Contoh Produk (Sample Product) |
| :--- | :--- | :--- | :--- |
| 1 | **⚡ Elektronik & Gadget** | 15 Produk | Headphone Nirkabel AuraPro ANC, Smartwatch TitanFit, Keyboard ApexCraft RGB 75% |
| 2 | **👔 Fashion Pria** | 13 Produk | Celana Jeans Pria Slim Fit Washed, Hoodie Heavyweight, Jaket Windbreaker |
| 3 | **👗 Fashion Wanita** | 12 Produk | Celana Jeans Wanita High Waist, Blouse Katun Linen, Piyama Satin Silk |
| 4 | **🏡 Peralatan Rumah Tangga** | 13 Produk | Penggiling Kopi BaristaCraft, Lampu Nordic Glow, Air Fryer Digital 4.5L |
| 5 | **🎒 Olahraga & Outdoor** | 12 Produk | Tas Ransel Nomad 35L Rolltop, Tumbler Termos 1000ml, Matras Yoga TPE |
| 6 | **✨ Kecantikan & Perawatan** | 12 Produk | Serum Niacinamide 10%, Sunscreen Gel SPF 50+, Gentle Cleanser Ceramide |
| 7 | **☕ Makanan & Minuman Sehat**| 12 Produk | Kopi Arabika Gayo Specialty, Madu Hutan Sumbawa, Granola Coklat Mete |
| 8 | **📚 Buku & Alat Tulis** | 11 Produk | Jurnal Kulit Dotted A5, Pulpen Gel 0.5mm, Daily Planner Undated |
| **Total**| **8 Kategori** | **100 Produk** | **100% Bahasa Indonesia & Real IDR Pricing** |

---

## 🧪 3. QA Automation Matrix (`qa/`)

Run tests from project root:
- **Playwright Browser Tests (with automatic report purge)**: `make test-e2e` (or `cd qa && npm run test:e2e`)

| Suite | Runner | Verification Scope | Status |
| :--- | :--- | :--- | :--- |
| **Storefront Browsing & Semantic Toggle** | Playwright (Chromium) | Homepage title, infinite scrolling, AI semantic toggle, 8 category filters, and search | ✅ PASS |
| **PDP Modal** | Playwright (Chromium) | Product detail modal opening, quantity counter adjustments | ✅ PASS |
| **Cart Drawer** | Playwright (Chromium) | Item addition, badge count, drawer open, quantity +/- controls | ✅ PASS |
| **Shopper Checkout** | Playwright (Chromium) | 1-Click Shopper Demo auth, checkout form submission, order history redirect | ✅ PASS |
| **Admin Control & IDR**| Playwright (Chromium) | 1-Click Admin Demo login, direct admin view locking, Rupiah KPIs, product CRUD, orders, and CRM | ✅ PASS |

---

## 💻 4. Makefile & Docker Command Cheat Sheet

```bash
# Automated Browser Testing (Playwright from qa/ with auto report purge)
make test-e2e                          # Run all Playwright E2E browser tests

# Docker Compose Full Stack (All 8 Containers)
make docker-up                         # Launch Postgres, Ollama, AI Service, Backend, Frontend, Loki, Promtail, Grafana
make docker-down                       # Stop all services
docker compose up -d --build           # Rebuild and start updated containers

# Observability (Grafana Loki)
# Open http://localhost:3001 in browser (admin/admin) to search and stream logs with LogQL

# Python AI Semantic Search & Shopper
make ai-run                            # Start FastAPI AI service on :8000

# Local Dev
make backend-run                       # Run Go backend on :8080
make frontend-run                      # Run React dev server on :3000
```
