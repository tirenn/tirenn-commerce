# 🧪 Tirenn Commerce - Quality Assurance (QA) Suite

This directory contains the automated testing suites for **Tirenn Commerce**:

1. **🎭 Playwright Browser E2E Tests** (`qa/e2e/`):
   - Full user-facing journey testing in Chromium.
   - Catalog browsing, category filtering, and real-time search.
   - Product Detail Modal (PDP) and quantity adjustments.
   - Cart drawer operations (add to cart, badge counter, quantity modification, remove).
   - Checkout flow and order placement.
   - Customer Order History tracking.
   - Admin Back-Office (Executive KPIs, Products & Stock Controller, Orders fulfillment, Customer CRM).

2. **⚡ Go Concurrency & API Integration Tests** (`qa/e2e_api_test.go` & `qa/main.go`):
   - Atomic database locking (`SELECT FOR UPDATE`) preventing inventory overselling under high concurrency.
   - RBAC security checks (preventing unauthorized access to `/api/v1/admin/*`).
   - Order lifecycle state transitions and automatic stock deduction / restock on cancellation.

---

## 🚀 How to Run the QA Tests

### 1. Run Playwright Browser Tests:
```bash
# From project root:
make test-e2e

# Or inside qa directory:
cd qa
npm run test:e2e
```

### 2. Run Interactive API QA Runner:
```bash
# From project root:
make qa-run

# Or inside qa directory:
cd qa
go run main.go
```

### 3. Run Go Test Suite:
```bash
make qa-test
```
