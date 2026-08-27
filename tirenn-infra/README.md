# 🏗️ Tirenn Commerce - Infrastructure & QA Orchestrator

Core Infrastructure Stack, Observability Suite, and Automated End-to-End Quality Assurance (QA) Engine for Tirenn Commerce.

---

## 🌟 Contained Services & Stacks

1. **🐘 PostgreSQL 16 (`pgvector`)**:
   - Primary relational datastore with `pgvector` and `pg_trgm` extensions enabled.
   - Port `5432:5432`.
2. **⚡ Redis 7 In-Memory Store**:
   - Distributed rate limiter state and caching layer.
   - Port `6379:6379`.
3. **🦙 Ollama Local LLM Runner**:
   - Host for `qwen2.5:3b` model used by Tirenn AI Shopper.
   - Port `11434:11434`.
4. **📊 Observability Stack (Loki, Promtail, Grafana)**:
   - **Grafana Loki (`:3100`)**: Centralized log aggregation engine.
   - **Promtail**: Docker socket container log shipper.
   - **Grafana (`:3001`)**: Log exploration dashboard (`admin/admin`).
5. **🧪 QA Testing Engine (`qa/`)**:
   - **Playwright E2E**: Multi-browser end-to-end storefront and admin tests.
   - **Go Integration Tests**: Interactive API test runner.

---

## 🚀 Getting Started

### 1. Start Infrastructure
```bash
# Start all infrastructure containers
make up

# Check container status
make ps

# Stop infrastructure
make down
```

### 2. Run Automated QA Tests
```bash
# Run Playwright E2E test suite (Chromium, Storefront + Admin)
make test-e2e

# Run Go API integration test runner
make qa-run

# Run Go integration tests
make qa-test
```
