# 🛍️ Tirenn Commerce - Master Project Context & Development Log

> **Note for AI Assistant & Developers**: This document contains the full architecture, decisions, file maps, migrations, testing matrices, and operational context for **Tirenn Commerce**. Keep this file updated as new features are added.

---

## 📌 1. Project Overview & Architecture

**Tirenn Commerce** is a full-stack, production-grade **modern e-commerce marketplace and department store platform** designed with an ultra-simple, minimal, and high-converting UI:

- **Branding**: **Tirenn Commerce** (`tirenn commerce`) - The Modern Online Marketplace.
- **Auto-Pagination (Infinite Scrolling)**: Native `IntersectionObserver`-based auto pagination on the Storefront. As the shopper scrolls down, subsequent pages (`12 products/batch`) are fetched and appended seamlessly with zero lag, showing animated loading spinners and completion indicators (`✓ Semua 100 produk telah ditampilkan`).
- **Catalog & Localization**: **100 Products across 8 Categories in Bahasa Indonesia** with authentic Indonesian names, product descriptions, localized user profiles, and Indonesian Rupiah (`Rp` / `IDR`) pricing.
- **Search Engine**: **Pure MySQL Full-Text Search (FTS)** with zero `LIKE` queries. Indexes configured across:
  - **Products**: `idx_products_fulltext (name, description, sku)`
  - **Categories**: `idx_categories_fulltext (name, description)`
  - Executing: `MATCH(products.name, products.description, products.sku) AGAINST (? IN BOOLEAN MODE) OR MATCH(categories.name, categories.description) AGAINST (? IN BOOLEAN MODE)`.
- **Security & Git Hygiene**: Zero hardcoded credentials in Makefiles. `.gitignore` configured to ignore `.env`, binaries, test reports, and AI assistant artifacts (`.agents/`, `.gemini/`, `.claude/`, `.cursor/`).
- **Currency**: **Indonesian Rupiah (IDR / Rp)** formatted across all storefront and admin operations (e.g. `Rp 1.499.000`).
- **Role Isolation**:
  - **👑 Admin**: When logging in as `ADMIN`, the user is strictly and exclusively routed to the Merchant Console (`admin-dashboard`, `admin-products`, `admin-orders`, `admin-customers`). All non-admin storefront elements (cart, catalog filters, banners) are hidden from Admin.
  - **🛍️ Customer / Public**: Full access to clean storefront, real-time search, cart drawer, checkout, and order history.
- **Frontend Stack**: **React 19 + TypeScript + Vite + Tailwind CSS 4** (clean component architecture in `frontend/src/`).
- **Backend Stack**: **Golang (Go 1.24+)**, Gin framework, GORM ORM, strict MySQL database, Viper configuration loader, JWT RBAC, and Goose database migrations.
- **API Documentation**:
  - **Comprehensive Markdown Reference**: [`docs/API_DOCUMENTATION.md`](docs/API_DOCUMENTATION.md) (All endpoints, query params, DTOs, response schemas, and TypeScript interfaces).
  - **OpenAPI 3.0.3 Spec**: [`docs/openapi.yaml`](docs/openapi.yaml) (Ready to import into Postman, Swagger UI, Insomnia).
- **QA Automation Workspace (`qa/`)**:
  - **Browser E2E Testing**: **Playwright** located in `qa/` (`qa/playwright.config.ts`, `qa/e2e/`) testing all user-facing storefront and admin journeys in Chromium.
  - **Automatic Post-Test Cleanup**: Configured `posttest:e2e` hook (`qa/scripts/clean-reports.js`) that automatically purges all `test-results/` and `playwright-report/` directories after every test run.
  - **API & Concurrency Testing**: Dedicated Go testing suite in `qa/` (`qa/e2e_api_test.go` and `qa/main.go`) verifying atomic `SELECT FOR UPDATE` overselling prevention.
- **Infrastructure & CI/CD**: Multi-stage Dockerfiles, Docker Compose (`mysql` + `backend` + `frontend`), and GitHub Actions CI/CD pipeline (`.github/workflows/ci-cd.yml`).

---

## 🎨 2. 100 Seeded Indonesian Products Across 8 Categories

| No | Kategori (Category) | Jumlah Produk | Contoh Produk (Sample Product) |
| :--- | :--- | :--- | :--- |
| 1 | **⚡ Elektronik & Gadget** | 15 Produk | Headphone Nirkabel AuraPro ANC, Smartwatch TitanFit, Keyboard ApexCraft RGB 75% |
| 2 | **👔 Fashion Pria** | 13 Produk | Hoodie Heavyweight UrbanCraft, Jaket Windbreaker AeroFlex, Kaos Oversized 24s |
| 3 | **👗 Fashion Wanita** | 12 Produk | Blouse Katun Linen, Celana Kulot High Waist, Cardigan Rajut Korean Style |
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
- **API & Concurrency Tests**: `make qa-test` (or `make qa-run`)

| Suite | Runner | Verification Scope | Status |
| :--- | :--- | :--- | :--- |
| **Storefront Browsing & Infinite Scroll** | Playwright (Chromium) | Homepage title, infinite scrolling, 8 category filters, and pure FTS search across 100 products | ✅ PASS |
| **PDP Modal** | Playwright (Chromium) | Product detail modal opening, quantity counter adjustments | ✅ PASS |
| **Cart Drawer** | Playwright (Chromium) | Item addition, badge count, drawer open, quantity +/- controls | ✅ PASS |
| **Shopper Checkout** | Playwright (Chromium) | 1-Click Shopper Demo auth, checkout form submission, order history redirect | ✅ PASS |
| **Admin Control & IDR**| Playwright (Chromium) | 1-Click Admin Demo login, direct admin view locking, Rupiah KPIs, product CRUD, orders, and CRM | ✅ PASS |
| **API & Concurrency** | Go test (`qa/`) | 10 concurrent requests on 1 unit inventory, verifying zero overselling with atomic locks | ✅ PASS |

**Result**: All Playwright browser tests and Go integration suites pass with 100% success rate, followed by automatic cleanup of all report files.

---

## ⚙️ 4. Environment Configurations

### Backend (`backend/.env`):
- `PORT=8080`
- `ENVIRONMENT=development`
- `DB_HOST=127.0.0.1` (or `mysql` in Docker)
- `DB_PORT=3306`
- `DB_USER=root`
- `DB_PASSWORD=rootpassword`
- `DB_NAME=gocommerce_db`
- `JWT_SECRET=super-secret-tirenn-jwt-key-2026`
- `JWT_EXPIRE_HOURS=24`

### Frontend (`frontend/.env`):
- `VITE_API_BASE_URL=http://localhost:8080/api/v1`
- `VITE_BACKEND_URL=http://localhost:8080`

---

## 🔑 5. Default Seeded Credentials

| Role | Name | Email | Password | Access Level |
| :--- | :--- | :--- | :--- | :--- |
| **👑 Admin (Merchant)** | `Tirenn Merchant Admin` | `admin@gocommerce.com` | `Admin@123` | Exclusive Back-Office Access (Dashboard, Products, Orders, CRM) |
| **🛍️ Shopper (Customer)** | `Budi Santoso` | `shopper@gocommerce.com` | `Shopper@123` | Storefront, Cart, Checkout, Order History |
| **⭐ Customer 2** | `Siti Rahmawati` | `sarah.jenkins@example.com` | `Sarah@123` | Storefront, Cart, Checkout, Order History |

---

## 💻 6. Makefile Command Cheat Sheet

```bash
# Automated Browser Testing (Playwright from qa/ with auto report purge)
make test-e2e                          # Run all Playwright E2E browser tests

# Database Migrations (Goose)
make migrate-create name=add_feature   # Create new migration file
make migrate-up                        # Run all pending migrations
make migrate-down                      # Roll back the single latest migration
make migrate-status                    # Check migration version and status
make migrate-reset                     # Roll back all migrations

# Automated API QA Testing
make qa-run                            # Run interactive QA automation runner
make qa-test                           # Run Go testing suite in qa/

# Docker Compose Services
make docker-up                         # Launch MySQL, Backend, and Frontend
make docker-down                       # Stop all services
docker compose down -v --rmi all       # Stop and delete containers, images, and volumes

# Local Dev
make backend-run                       # Run Go backend on :8080
make frontend-run                      # Run React dev server on :3000
```
