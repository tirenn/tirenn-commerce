# 📋 Project Context: Tirenn Infrastructure & QA

This document outlines the topology, observability pipeline, and QA test specifications of `tirenn-infra`.

---

## 🏛️ Infrastructure Structure

```
tirenn-infra/
├── docker-compose.yml       # Core infrastructure services definition
├── Makefile                 # Automation targets for infra & QA
├── logging/
│   ├── grafana-datasources.yml # Auto-provisioned Loki data source for Grafana
│   ├── loki-config.yml      # Loki retention, storage, and server configuration
│   └── promtail-config.yml  # Promtail Docker socket log scraper rules
└── qa/
    ├── e2e/
    │   ├── admin.spec.ts    # Admin Control Panel E2E test suite
    │   └── storefront.spec.ts # Storefront catalog, search, cart & checkout test suite
    ├── scripts/
    │   └── clean-reports.js # Test report sanitizer
    ├── main.go              # Go interactive API verification runner
    ├── package.json         # QA dependencies (Playwright, TypeScript)
    └── playwright.config.ts # Playwright multi-worker configuration
```

---

## 🧪 QA Test Specifications (100% Coverage Target)

1. **Storefront Specs (`qa/e2e/storefront.spec.ts`)**:
   - `1. Homepage Render`: Verifies product grid, badges, prices in IDR.
   - `2. Category Filter`: Tests department pill filtering (e.g. *Elektronik & Gadget*, *Fashion Pria*).
   - `3. Real-Time Search`: Tests instant search input with live catalog filtering.
   - `4. Product Detail Modal (PDP)`: Tests modal launch, stock check, quantity adjusters.
   - `5. Cart Drawer`: Tests adding to cart, opening drawer, modifying item quantity.
   - `6. 1-Click Shopper Auth & Checkout`: Tests 1-click shopper authentication, order submission, and invoice verification.
2. **Admin Specs (`qa/e2e/admin.spec.ts`)**:
   - `1. Executive Dashboard`: Tests KPI cards, gross revenue, inventory value.
   - `2. Product & Stock Management`: Tests catalog table, stock adjustment modal, audit reason logging.
   - `3. Customer & Orders CRM`: Tests customer orders and customer directory metrics.
