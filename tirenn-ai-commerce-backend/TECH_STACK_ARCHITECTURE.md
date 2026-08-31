# 🏗️ Tech Stack & Architecture: Tirenn Backend

This document outlines the technical architecture, data storage, rate limiting, and containerization design of `tirenn-backend`.

---

## 💻 Technical Stack

| Layer | Technology | Details |
| :--- | :--- | :--- |
| **Language** | Go 1.22 | High-concurrency compiled backend |
| **HTTP Framework** | Gin Web Framework | High-performance routing & middleware |
| **ORM / Data Access** | GORM | PostgreSQL dialect, connection pooling |
| **Database** | PostgreSQL 16 + pgvector | ACID relational database with vector embeddings |
| **Cache & Limiter** | Redis 7 Alpine | Distributed Sliding Window Rate Limiting |
| **Migrations** | Goose v3 | Declarative SQL migrations & seeds |

---

## 🐳 Container Architecture (`docker-compose.yml`)

```yaml
version: '3.8'

services:
  backend:
    build:
      context: .
      dockerfile: Dockerfile
    container_name: tirenn-backend
    restart: always
    ports:
      - "8080:8080"
    environment:
      - SERVER_PORT=8080
      - ENV=production
      - DB_HOST=postgres
      - DB_PORT=5432
      - DB_USER=gouser
      - DB_PASSWORD=gopassword
      - DB_NAME=gocommerce_db
      - REDIS_HOST=redis
      - REDIS_PORT=6379
      - JWT_SECRET=supersecretjwtkey_for_dev_change_in_prod
      - AI_SERVICE_URL=http://ai-service:8000
    networks:
      - tirenn-net

networks:
  tirenn-net:
    external: true
```

### Multi-Stage Dockerfile
1. **Builder (`golang:alpine`)**:
   - `CGO_ENABLED=1 GOOS=linux go build -ldflags="-s -w" -o /app/server ./cmd/server`
2. **Runtime (`alpine:3.19`)**:
   - Minimal attack surface (~25MB binary container).
   - Installs `ca-certificates` and `tzdata`.
