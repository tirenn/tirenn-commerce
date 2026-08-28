# 📋 Project Context: Tirenn Infrastructure & QA

This document outlines the topology, observability pipeline, and QA test specifications of `tirenn-infra`.

---

## 🏛️ Infrastructure Structure

```
tirenn-infra/
├── docker-compose.yml       # Core infrastructure services definition (PostgreSQL pgvector, Redis, Ollama, Grafana, Loki, Promtail)
├── Makefile                 # Automation targets for infra & QA
├── logging/
│   ├── grafana-datasources.yml # Auto-provisioned Loki data source for Grafana
│   ├── loki-config.yml      # Loki retention, storage, and server configuration
│   └── promtail-config.yml  # Promtail Docker socket log scraper rules
└── qa/
    ├── e2e/
    │   ├── admin.spec.ts    # Admin Control Panel & Knowledge Management E2E test suite
    │   └── storefront.spec.ts # Storefront catalog, i18n, AI chat, cart & checkout test suite
    ├── scripts/
    │   └── clean-reports.js # Test report sanitizer
    ├── main.go              # Go interactive API verification runner
    ├── package.json         # QA dependencies (Playwright, TypeScript)
    └── playwright.config.ts # Playwright configuration (workers: 2 for CPU stability)
```

---

## 🧪 QA Test Specifications (12 E2E Scenarios)

1. **Storefront Specs (`qa/e2e/storefront.spec.ts`)**:
   - `1. Homepage Render`: Verifies product grid, badges, prices in active currency.
   - `2. Category Filter`: Tests department pill filtering and subcategory selection.
   - `3. i18n & Dynamic Currency`: Tests real-time language and currency toggling (IDR / USD).
   - `4. Real-Time Search`: Tests instant search input with live catalog filtering.
   - `5. Product Detail Modal (PDP)`: Tests modal launch, stock check, quantity adjusters.
   - `6. Cart Drawer`: Tests adding to cart, opening drawer, modifying item quantity.
   - `7. Guest AI Shopper Chat`: Tests AI recommendation, relevance guardrails, and adding product to cart from chat.
   - `8. 1-Click Shopper Auth & Checkout`: Tests 1-click shopper authentication, order submission, and order history verification.
2. **Admin Specs (`qa/e2e/admin.spec.ts`)**:
   - `1. Executive Dashboard`: Tests KPI cards, gross revenue, inventory value in IDR.
   - `2. Product & Stock Management`: Tests catalog table, stock adjustment modal, audit reason logging.
   - `3. Customer & Orders CRM`: Tests customer orders and customer directory metrics.
   - `4. SOP & AI Knowledge Base`: Tests navigation to Knowledge Management, upload view, and semantic playground.

---

## 📜 Service Changelog

### 📅 2026-08-28
- `[QA]` Added Admin Knowledge Management E2E test scenario in `e2e/admin.spec.ts`.
- `[QA]` Configured Playwright with `workers: 2` in `playwright.config.ts` for consistent local CPU model execution.
- `[QA]` Verified 12 of 12 end-to-end tests passing at 100% success rate.
