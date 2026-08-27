# 📋 Project Context: Tirenn Backend

This document describes the package organization, domain models, and API interfaces of `tirenn-backend`.

---

## 🏛️ Project Structure & Clean Domain Packages

The Go backend follows Clean Domain Architecture:

```
internal/
├── database/            # PostgreSQL connection, pgvector extension, connection pooling
├── logger/              # Structured JSON logging compatible with Grafana Loki & Promtail
├── middleware/          # JWT auth, CORS, Redis sliding-window rate limiting, request tracing
├── router/              # Gin HTTP router mounting domain endpoints
├── utils/               # Standardized JSON response envelopes, pagination, error formatters
└── domain/
    ├── admin/           # Executive KPIs, analytics aggregation, dashboard overview
    ├── auth/            # JWT authentication, login, registration, /auth/me profile
    ├── customer/        # Customer directory, user management
    ├── order/           # Order processing, transactional checkout, status workflows
    └── product/         # Product catalog, categories, stock adjustment audit logs
```

---

## 🗄️ PostgreSQL Database Entities

- **`users`**: User identity, hashed passwords, roles (`ADMIN` / `CUSTOMER`).
- **`categories`**: Product taxonomy (e.g. *Elektronik & Gadget*, *Fashion Pria*, *Fashion Wanita*).
- **`products`**: Product inventory, SKU, pricing, stock levels, `embedding vector(384)`, full-text search fields.
- **`stock_adjustment_logs`**: Complete audit trail of stock adjustments (`ADD`, `SUBTRACT`, `SET`, admin ID, reason).
- **`orders` & `order_items`**: Order transactions, total amount, shipping addresses, items snapshot.

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
