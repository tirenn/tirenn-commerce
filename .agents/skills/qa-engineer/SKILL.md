---
name: qa-engineer
description: >-
  Provides end-to-end expertise in Quality Assurance, automated test strategies, unit/integration/E2E testing (Playwright, Vitest, Go test),
  API testing, performance/load testing (k6), security audits (OWASP), and CI/CD quality gates.
---

# Senior QA Automation Engineer Skill & Agent Guide

This skill equips the agent with senior-level Quality Assurance (QA) and test automation expertise covering test strategy design, unit and integration testing, API contract testing, End-to-End (E2E) automation, performance load testing, security audits, and continuous testing quality gates.

---

## 1. Core Testing Strategy & Pyramid Framework

```text
               /\
              /  \     E2E / UI Tests (Playwright, Cypress)
             /----\    - Critical user journeys (Checkout, Auth, Admin workflows)
            /      \   
           /--------\  Integration & API Tests (httptest, Testcontainers, Postman)
          /          \ - Database transactions, RBAC, HTTP handlers, third-party integrations
         /------------\
        /              \ Unit Tests (Go test, Vitest, Jest)
       /----------------\ - Business logic, domain usecases, validation, calculation algorithms
```

### Test Level Responsibility Matrix
| Level | Scope | Tools | Execution Target | Gate Requirement |
| :--- | :--- | :--- | :--- | :--- |
| **Unit** | Individual functions, domain usecases, utils | Go `testing`, `testify`, `Vitest` | Local & Pre-commit | > 80% coverage |
| **API / Integration** | Handlers, DB queries, Middleware, Transactions | `httptest`, `testcontainers-go`, `k6` | CI Pipeline | Zero race conditions (`-race`) |
| **E2E / UI** | End-to-end user browser interactions | `Playwright`, `Cypress` | Staging / Nightly CI | 100% critical path pass |
| **Performance** | Latency, throughput, concurrency limits | `k6`, `Locust`, `vegeta` | Pre-Release Audit | P95 < 200ms at target RPS |

---

## 2. Go Backend Testing Standards & Patterns

### A. HTTP Handler Testing (`net/http/httptest` + Gin)
```go
func TestProductHandler_GetProduct(t *testing.T) {
    gin.SetMode(gin.TestMode)
    mockUseCase := new(MockProductUseCase)
    handler := NewHandler(mockUseCase)

    router := gin.New()
    router.GET("/api/v1/products/:id", handler.GetProduct)

    expectedProduct := &Product{ID: 1, Name: "Comic Issue #1", Price: 14.99}
    mockUseCase.On("GetProductByID", mock.Anything, uint(1)).Return(expectedProduct, nil)

    req, _ := http.NewRequest(http.MethodGet, "/api/v1/products/1", nil)
    w := httptest.NewRecorder()
    router.ServeHTTP(w, req)

    assert.Equal(t, http.StatusOK, w.Code)
    var resp APIResponse
    err := json.Unmarshal(w.Body.Bytes(), &resp)
    assert.NoError(t, err)
    assert.True(t, resp.Success)
}
```

### B. Concurrency & Race Condition Verification
Always execute tests with the `-race` detector enabled:
```bash
go test -v -race -count=1 ./...
```

### C. Database Transaction Isolation in Tests
Wrap integration tests in isolated database transactions and roll them back automatically at test teardown:
```go
func RunInTestTransaction(db *gorm.DB, testFunc func(tx *gorm.DB)) {
    tx := db.Begin()
    defer tx.Rollback()
    testFunc(tx)
}
```

---

## 3. Frontend & E2E Test Automation (Playwright)

### Critical Path Test Template (Storefront Checkout Journey)
```typescript
import { test, expect } from '@playwright/test';

test.describe('E-Commerce Storefront Critical Journey', () => {
  test('User can browse catalog, add item to cart, and checkout', async ({ page }) => {
    // 1. Visit homepage
    await page.goto('http://localhost:3000');
    await expect(page.locator('h1')).toContainText('GOCOMMERCE');

    // 2. Search for a product
    const searchInput = page.locator('input[placeholder*="Search"]');
    await searchInput.fill('Cyber Sentinel');
    await expect(page.locator('.comic-box')).toHaveCount(1);

    // 3. Add item to cart
    await page.locator('button:has-text("🛒 + CART")').first().click();
    await expect(page.locator('.comic-badge:has-text("Added")')).toBeVisible();

    // 4. Open Cart Drawer
    await page.locator('button:has-text("CART")').click();
    await expect(page.locator('h2')).toContainText('YOUR HERO CART');

    // 5. Trigger Checkout
    await page.locator('button:has-text("PROCEED TO CHECKOUT")').click();
    await expect(page.locator('h2')).toContainText('CONFIRM YOUR HERO ORDER');

    // 6. Submit Checkout Form
    await page.locator('input[placeholder="Full Name"]').fill('Test Superhero');
    await page.locator('button:has-text("AUTHORIZE PAYMENT")').click();

    // 7. Verification: Redirect to Order History
    await expect(page.locator('h2')).toContainText('YOUR HERO ORDER HISTORY');
  });
});
```

---

## 4. API Load & Performance Testing (k6 Blueprint)

```javascript
import http from 'k6/http';
import { check, sleep } from 'k6';

export const options = {
  stages: [
    { duration: '30s', target: 20 },  // Ramp up to 20 virtual users
    { duration: '1m', target: 50 },   // Stress test at 50 virtual users
    { duration: '20s', target: 0 },   // Ramp down
  ],
  thresholds: {
    http_req_duration: ['p(95)<250'], // 95% of requests must complete below 250ms
    http_req_failed: ['rate<0.01'],   // Error rate below 1%
  },
};

export default function () {
  const res = http.get('http://localhost:8080/api/v1/products');
  check(res, {
    'status is 200': (r) => r.status === 200,
    'response has data': (r) => JSON.parse(r.body).data.length > 0,
  });
  sleep(1);
}
```

---

## 5. Security & Vulnerability Checklist (OWASP Top 10)

- **SQL Injection**: Validate parameterized queries across all database interactions (GORM / sqlx).
- **Authentication & RBAC**: Test token expiry, token tampering, and privilege escalation (e.g. customer attempting to call `/api/v1/admin/*`).
- **Input Validation & Sanitization**: Ensure negative stock, invalid numbers, and script injection strings (`<script>alert(1)</script>`) are rejected with `400 Bad Request`.
- **Atomic Concurrency (Overselling)**: Test 10 simultaneous checkout requests against a product with stock quantity = 1; exactly 1 must succeed and 9 must fail.

---

## 6. Bug Report & Defect Submission Template

```markdown
### 🐞 Bug: [Concise summary of issue]

**Severity**: Critical / High / Medium / Low  
**Environment**: Staging (v1.2.0) | Browser: Chrome 124 | OS: Windows 11  

#### 📋 Steps to Reproduce
1. Log in as `shopper@gocommerce.com`.
2. Add product with 1 unit remaining in stock to the cart.
3. Open two tabs and initiate checkout simultaneously.

#### ❌ Expected vs Actual Behavior
- **Expected**: One order completes; the second tab displays an out-of-stock warning.
- **Actual**: Both orders succeed, resulting in negative stock (-1).

#### 📸 Evidence / Logs
- Attached backend error logs / network trace.
```
