# 🐘 Tirenn Commerce - Backend Microservice

Enterprise-grade, high-performance E-Commerce Core REST API written in **Golang (Gin)**, **GORM**, and **PostgreSQL (pgvector)**.

---

## 🌟 Key Features

1. **🛍️ Product Catalog & Stock Management**:
   - High-throughput catalog queries with full-text search, pagination, and department category filtering.
   - Atomic stock adjustments (`ADD`, `SUBTRACT`, `SET`) with audit logging.
2. **🛒 Order Management & Atomic Checkout**:
   - Multi-item checkout processing within strict PostgreSQL database transactions (`BEGIN ... COMMIT`).
   - Dynamic order status tracking (`PENDING`, `PAID`, `SHIPPED`, `DELIVERED`, `CANCELLED`).
3. **📊 Executive Analytics & Admin APIs**:
   - Executive Dashboard metrics: Total gross revenue, order volume, low stock alerts, CRM directory.
4. **🔐 Authentication & Security**:
   - JWT authentication (`RS256` / `HS256`) with role-based access control (`ADMIN` vs `USER`).
   - Redis Sliding Window Rate Limiter with standard `X-RateLimit-*` headers.
   - Strict CORS middleware.
5. **🗄️ Database Migrations (Goose)**:
   - Version-controlled SQL schema migrations and seed data.

---

## 🛠️ Tech Stack

- **Language**: Go 1.22+
- **HTTP Web Framework**: Gin (`github.com/gin-gonic/gin`)
- **ORM**: GORM (`gorm.io/gorm` with `gorm.io/driver/postgres`)
- **Database**: PostgreSQL 16 with `pgvector`
- **Cache & Rate Limiting**: Redis 7 Alpine
- **Database Migrations**: Goose (`github.com/pressly/goose/v3`)

---

## 🚀 Getting Started

### 1. Environment Configuration
Copy `.env.example` to `.env`:
```bash
SERVER_PORT=8080
ENV=development

DB_HOST=localhost
DB_PORT=5432
DB_USER=gouser
DB_PASSWORD=gopassword
DB_NAME=gocommerce_db
DB_SSLMODE=disable

REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=

JWT_SECRET=supersecretjwtkey_for_dev_change_in_prod
AI_SERVICE_URL=http://localhost:8000
```

### 2. Database Migrations
```bash
# Run all pending migrations
make migrate-up

# Check migration status
make migrate-status
```

### 3. Local Bare-Metal Execution
```bash
# Start backend server
make run
# -> Server running on port 8080
```

### 4. Docker Containerization
```bash
# Build and run standalone Go container
make docker-up

# Stop container
make docker-down
```
