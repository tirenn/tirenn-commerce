# 🛍️ Tirenn Commerce - Modern Pop E-Commerce & Merchant Back-Office

A high-performance, full-stack e-commerce web platform engineered with a **Golang Backend** (Clean Architecture, Gin, GORM, MySQL, Goose Migrations, Viper Config, JWT RBAC) and a **Svelte Frontend** featuring a polished, high-converting **E-Commerce Layout** with Comic / Pop-Art / Neo-Brutalist design tokens.

---

## 🏛️ Architecture Highlights

```text
ai-commerce/
├── .github/
│   └── workflows/
│       └── ci-cd.yml                # Automated GitHub Actions CI/CD Pipeline
├── backend/
│   ├── cmd/
│   │   ├── server/main.go           # Server bootstrap & graceful shutdown
│   │   └── migrate/main.go          # Dedicated Goose migration CLI runner for MySQL
│   ├── migrations/                  # Goose SQL migrations (Up & Down)
│   ├── internal/
│   │   ├── router/router.go         # Dedicated separate routing layer
│   │   ├── config/config.go         # Viper environment loader with strongly-typed structs
│   │   ├── database/                # MySQL connection pool and demo seeder
│   │   ├── middleware/              # JWT auth, RBAC (ADMIN/CUSTOMER), CORS, Logger
│   │   └── domain/                  # Clean Architecture per domain (Auth, Product, Order, Customer, Dashboard)
│   ├── Makefile                     # Makefile for migrations and server development
│   └── Dockerfile
├── frontend/
│   ├── src/
│   │   ├── lib/
│   │   │   ├── api.ts               # API client with token interceptor
│   │   │   ├── stores.ts            # Svelte stores (Auth, Cart, Toasts, Views)
│   │   │   └── components/
│   │   │       ├── comic/           # Navbar (Announcement + Search), ToastContainer, AuthModal
│   │   │       ├── storefront/      # HeroBanner, FilterBar, ProductCard, CartDrawer, CheckoutModal, OrderHistory
│   │   │       └── admin/           # AdminDashboard, ProductManagement, StockAdjustmentModal, OrderManagement, CustomerManagement
│   │   ├── App.svelte               # Root e-commerce view router & layout
│   │   └── app.css                  # Modern E-Commerce Neo-Brutalist CSS design system
│   ├── Dockerfile
│   └── vite.config.ts
├── qa/                              # Automated QA End-to-End & Concurrency test suite
├── Makefile                         # Root Makefile forwarding commands
├── PROJECT_CONTEXT.md               # Master context memory document
└── docker-compose.yml               # Multi-container orchestration (MySQL + Backend + Frontend)
```

---

## 🚀 Quick Start with Docker

```bash
# Launch entire stack
make docker-up

# Stop all services and clean up
docker compose down -v --rmi all
```

- **Frontend Storefront**: `http://localhost:3000`
- **Backend API**: `http://localhost:8080`
- **MySQL Database**: `localhost:3306`

---

## 🗄️ Database Migrations with Goose & Makefile

```bash
# 1. Create a New Migration
make migrate-create name=add_discount_coupons

# 2. Run All Pending Up Migrations
make migrate-up

# 3. Roll Back the Latest Migration (Down)
make migrate-down

# 4. Check Migration Status & Version
make migrate-status

# 5. Reset All Migrations
make migrate-reset
```

---

## 🧪 Automated QA Testing

```bash
# Interactive QA Runner
make qa-run

# Go Test Suite
make qa-test
```

---

## 🔑 Pre-Seeded Demo Accounts

| Role | Email | Password | Access Level |
| :--- | :--- | :--- | :--- |
| **👑 Admin (Merchant)** | `admin@gocommerce.com` | `Admin@123` | Full Back-Office, Stock Controller, Order Fulfillment, CRM |
| **🛍️ Shopper (Customer)** | `shopper@gocommerce.com` | `Shopper@123` | Storefront, Shopping Cart, Checkout, Order History |
| **🦸 Customer 2** | `peter.parker@dailybugle.com` | `Spidey@123` | Storefront, Shopping Cart, Checkout, Order History |
