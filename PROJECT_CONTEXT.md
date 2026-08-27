# 🛍️ Tirenn Commerce - Master Project Context & Development Log

> **Note for AI Assistant & Developers**: This document contains the full architecture, decisions, file maps, migrations, testing matrices, and operational context for **Tirenn Commerce**. Keep this file updated as new features are added.

---

## 📌 1. Project Overview & Architecture

**Tirenn Commerce** is a full-stack, production-grade **modern e-commerce marketplace and department store platform** designed with an ultra-simple, minimal, and high-converting UI:

- **Branding**: **Tirenn Commerce** (`tirenn commerce`) - The Modern Online Marketplace.
- **Security & Git Hygiene**: Zero hardcoded credentials in Makefiles. `.gitignore` configured to ignore `.env`, binaries, test reports, and AI assistant artifacts (`.agents/`, `.gemini/`, `.claude/`, `.cursor/`).
- **Currency**: **Indonesian Rupiah (IDR / Rp)** formatted across all storefront and admin operations (e.g. `Rp 2.249.850`).
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

## 🎨 2. Simplified Minimalist UI Design

All marketing clutter, discount strike-throughs, coupon popups, and fake review badges have been removed for an ultra-clean user experience:

- **Header**: Logo, clean search bar, Store tab, Orders / Admin tab, Cart with badge counter, and Sign In / Logout.
- **Storefront**: Clean Category tabs, In-Stock filter, Price sort dropdown, and 4-Column responsive product grid.
- **Product Card**: High-res image, title, category, clean price (`Rp XX.XXX`), stock status, and `+ Add` button.
- **Product Detail Modal (PDP)**: Product photo, title, SKU, price (`Rp`), description, stock count, quantity counter, `Add to Cart`, and `Buy Now`.
- **Slide-over Cart Drawer**: Clean item list with quantity increment/decrement, remove, total amount (`Rp`), and `Checkout` button.
- **Checkout Modal**: Recipient Name, Phone, Address, Payment Method, and `Confirm Order` button.
- **Admin Control Panel**: Real-time sales analytics, product inventory & stock adjustment modal, order fulfillment status updater, and customer CRM.

---

## 🧪 3. QA Automation Matrix (`qa/`)

Run tests from project root:
- **Playwright Browser Tests (with automatic report purge)**: `make test-e2e` (or `cd qa && npm run test:e2e`)
- **API & Concurrency Tests**: `make qa-test` (or `make qa-run`)

| Suite | Runner | Verification Scope | Status |
| :--- | :--- | :--- | :--- |
| **Storefront Browsing** | Playwright (Chromium) | Homepage title, category filters, and real-time search | ✅ PASS |
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

## 🗂️ 5. Repository Structure

```text
ai-commerce/
├── .github/
│   └── workflows/
│       └── ci-cd.yml                    # Automated GitHub Actions CI/CD Pipeline
├── backend/
│   ├── cmd/
│   │   ├── server/main.go               # Server lifecycle bootstrap & graceful shutdown
│   │   └── migrate/main.go              # Dedicated Goose migration CLI runner (Viper-powered with retry loop)
│   ├── migrations/                      # Goose SQL migration files (Up & Down)
│   ├── internal/
│   │   ├── config/                      # Viper environment & config loader with strongly-typed structs
│   │   ├── database/                    # MySQL connection pooling & general retail seeder
│   │   ├── middleware/                  # JWT auth, RBAC guards, CORS, Logger
│   │   ├── router/                      # Dedicated separate routing tree registering all domain endpoints
│   │   ├── utils/                       # JWT generation/validation, bcrypt hashing, JSON response helpers
│   │   └── domain/                      # Clean Architecture per domain (entity, repo, usecase, handler)
│   ├── Dockerfile                       # Multi-stage Docker build (golang:alpine -> alpine:3.19)
│   ├── Makefile                         # Backend Makefile (migrations, build, dev, test)
│   ├── go.mod                           # Go module definition
│   └── .env.example                     # Environment variables template
├── frontend/
│   ├── src/
│   │   ├── components/                  # Clean modular React components
│   │   │   ├── admin/                   # AdminDashboard, ProductManagement, StockAdjustmentModal, OrderManagement, CustomerManagement
│   │   │   ├── Navbar.tsx               # Top header
│   │   │   ├── HeroBanner.tsx           # Simple banner
│   │   │   ├── FilterBar.tsx            # Clean Category pills & sort
│   │   │   ├── ProductCard.tsx          # Clean Product Card
│   │   │   ├── ProductDetailModal.tsx   # Clean PDP Modal
│   │   │   ├── CartDrawer.tsx           # Clean Cart Drawer
│   │   │   ├── CheckoutModal.tsx        # Clean Checkout Modal
│   │   │   ├── AuthModal.tsx            # Clean Sign In / 1-Click Demo Login
│   │   │   ├── OrderHistory.tsx         # Customer Order History
│   │   │   ├── Footer.tsx               # Minimal Footer
│   │   │   └── ErrorBoundary.tsx        # React ErrorBoundary
│   │   ├── context/                     # AuthContext, CartContext, ToastContext
│   │   ├── services/                    # api.ts fetch client
│   │   ├── types/                       # TypeScript interfaces
│   │   ├── utils/                       # format.ts (Indonesian Rupiah / formatRupiah)
│   │   ├── App.tsx                      # Main app router with role isolation
│   │   ├── main.tsx                     # React root
│   │   └── index.css                    # Tailwind CSS
│   ├── .env                             # Frontend environment variables
│   ├── .env.example                     # Frontend environment template
│   ├── Dockerfile                       # Multi-stage Docker build (Node.js build -> Nginx Alpine)
│   ├── nginx.conf                       # Nginx SPA router
│   ├── package.json                     # React + Tailwind
│   ├── tsconfig.json
│   ├── tsconfig.app.json
│   ├── tsconfig.node.json
│   └── vite.config.ts                   # Vite configuration
├── qa/
│   ├── e2e/                             # Playwright Browser Automated E2E Tests
│   │   ├── storefront.spec.ts           # Storefront user journey (Catalog, Search, Cart, Checkout)
│   │   └── admin.spec.ts                # Admin control panel tests (Dashboard, Products, Orders, CRM)
│   ├── scripts/
│   │   └── clean-reports.js             # Automated post-test report cleaner
│   ├── e2e_api_test.go                  # End-to-end integration and concurrency testing suite
│   ├── main.go                          # Standalone interactive QA test runner
│   ├── package.json                     # Playwright & TypeScript dependencies for QA with posttest hook
│   ├── playwright.config.ts             # Playwright QA configuration
│   ├── README.md                        # QA test matrix and instructions
│   └── go.mod
├── docs/
│   ├── API_DOCUMENTATION.md             # Complete REST API specification & integration guide
│   ├── openapi.yaml                     # Standard OpenAPI 3.0.3 specification
│   └── PRD.md                           # Formal Product Requirement Document
├── .gitignore                           # Excludes node_modules, binaries, .env, test-results, .agents/
├── Makefile                             # Root Makefile with test-e2e, docker, migrate commands
├── docker-compose.yml                   # Docker Compose orchestrating MySQL 8.0, Backend, Frontend
├── PROJECT_CONTEXT.md                   # THIS MASTER CONTEXT FILE
└── README.md                            # Public repository README
```

---

## 🔑 6. Default Seeded Credentials

| Role | Email | Password | Access Level |
| :--- | :--- | :--- | :--- |
| **👑 Admin (Merchant)** | `admin@gocommerce.com` | `Admin@123` | Exclusive Back-Office Access (Dashboard, Products, Orders, CRM) |
| **🛍️ Shopper (Customer)** | `shopper@gocommerce.com` | `Shopper@123` | Storefront, Cart, Checkout, Order History |
| **⭐ Customer 2** | `sarah.jenkins@example.com` | `Sarah@123` | Storefront, Cart, Checkout, Order History |

---

## 💻 7. Makefile Command Cheat Sheet

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
