# E2E Test Infra: Tirenn Commerce

## Test Philosophy
- Opaque-box, requirement-driven end-to-end verification.
- Validates user workflows, catalog browsing, cart operations, currency switching, AI copilot, recommendations, admin management, and data consistency across all microservices.

## Feature Inventory & Test Mapping
| # | Feature | Source | Test File | Test Case Name / Number |
|---|---------|--------|-----------|-------------------------|
| 1 | Catalog & Product Browsing | ORIGINAL_REQUEST | `storefront.spec.ts` | 1. Should load storefront and render product catalog |
| 2 | Category Navigation | ORIGINAL_REQUEST | `storefront.spec.ts` | 2. Should filter products by category |
| 3 | Multi-Currency & i18n | ORIGINAL_REQUEST | `storefront.spec.ts` | 3. Should switch language and currency (IDR / USD) |
| 4 | Semantic Search | ORIGINAL_REQUEST | `storefront.spec.ts` | 4. Should search products via semantic and keyword search |
| 5 | PDP Modal & Quantity | ORIGINAL_REQUEST | `storefront.spec.ts` | 5. Should open product detail modal, inspect details, and adjust quantity |
| 6 | Cart Drawer & Quantity Management | ORIGINAL_REQUEST | `storefront.spec.ts` | 6. Should add product to cart and adjust quantity in cart drawer |
| 7 | AI Shopper Copilot | ORIGINAL_REQUEST | `storefront.spec.ts` | 7. Should interact with AI Shopper Assistant |
| 8 | Shopper Demo Checkout | ORIGINAL_REQUEST | `storefront.spec.ts` | 8. Should complete full shopper demo checkout flow |
| 9 | AI Recommendation & Quick Add | ORIGINAL_REQUEST §Acceptance | `storefront.spec.ts` | 9. Should display AI product recommendations in PDP and Cart Drawer with 1-click quick add |
| 10 | Admin Dashboard KPIs | ORIGINAL_REQUEST | `admin.spec.ts` | 1. Should load admin dashboard and display financial KPIs |
| 11 | Admin Inventory & Low Stock | ORIGINAL_REQUEST | `admin.spec.ts` | 2. Should navigate to products inventory and filter low stock |
| 12 | Admin Customer CRM | ORIGINAL_REQUEST | `admin.spec.ts` | 3. Should view customer CRM and order history |
| 13 | Admin Knowledge Base RAG | ORIGINAL_REQUEST | `admin.spec.ts` | 4. Should view AI knowledge base documents |
| 14 | Admin AI Copilot | ORIGINAL_REQUEST | `admin.spec.ts` | 5. Should interact with Admin AI Copilot |

## Test Architecture
- **Framework**: Playwright `@playwright/test`
- **Execution Command**: `npm run test:e2e` (in `tirenn-infra/qa`)
- **Total Tests**: 14 (13 baseline + 1 new recommendation scenario)
- **Pass Semantics**: All 14 tests pass with exit code 0.
