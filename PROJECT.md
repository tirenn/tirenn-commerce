# Project: Tirenn Commerce — Real-Time AI Product Recommendation & Similar Items Discovery

## Architecture
Tirenn Commerce Real-Time AI Product Recommendation is composed of four clean layers:
1. **Python AI Engine (`tirenn-ai-service`)**:
   - Computes high-performance vector-similarity and cross-category recommendations.
   - Leverages `vector(384)` embeddings in PostgreSQL with HNSW cosine distance indexing (`products_embedding_hnsw_idx`).
   - Implements Similar Items (cosine similarity + category affinity + price corridor) and Frequently Bought Together / Cross-Category co-occurrence with fallback.
2. **Go Backend Core (`tirenn-backend`)**:
   - Exposes dedicated REST endpoint `GET /api/v1/products/:id/recommendations`.
   - Redis cache-aside caching under key `recommendations:product:{id}` with 1-hour TTL (`3600s`).
   - Resilient deterministic fallback to category top-sellers if AI service is unavailable or times out.
3. **React Storefront (`tirenn-storefront`)**:
   - Product Detail Modal (PDP) horizontal recommendation carousel ("Produk Serupa / You May Also Like").
   - Cart Drawer contextual add-ons for active cart items.
   - Dynamic multi-currency formatting (`IDR`/`USD`) and 1-click Add to Cart with live badge updates and toast notifications.
4. **QA & E2E Verification (`tirenn-infra/qa`)**:
   - Playwright test runner executing 14 end-to-end test scenarios (13 regression baseline + 1 recommendation suite).

```
Storefront (React 19)
   │
   ├─► GET /api/v1/products/:id/recommendations ──► Go Backend (:8080)
   │                                                     │
   │                                                     ├─► Redis Cache (recommendations:product:{id}, 1h TTL)
   │                                                     │
   │                                                     ├─► (Miss) ──► Python AI Service (:8000)
   │                                                     │                   │
   │                                                     │                   └─► PostgreSQL (pgvector HNSW cosine search)
   │                                                     │
   │                                                     └─► (Fallback) ─► PostgreSQL (Category Top-Sellers)
```

## Feature Inventory
| # | Feature | Description | Milestone | Source |
|---|---------|-------------|-----------|--------|
| 1 | AI Vector Recommendation Engine | Compute cosine similarity with category boost and price corridor in Python AI service | M1 | ORIGINAL_REQUEST §R1 |
| 2 | Frequently Bought Together / Cross-Category | Compute order co-occurrence with cross-category fallback | M1 | ORIGINAL_REQUEST §R1 |
| 3 | AI Service Recommendations API | Expose `GET /api/v1/products/{id}/recommendations` in FastAPI | M1 | ORIGINAL_REQUEST §R1 |
| 4 | Go AI Client Extension | Add recommendation query method with timeout and API key headers in Go backend | M2 | ORIGINAL_REQUEST §R2 |
| 5 | Deterministic Fallback Logic | Query category top-sellers (badges & rating) when AI service is unavailable | M2 | ORIGINAL_REQUEST §R2 |
| 6 | Redis Recommendation Cache | Cache recommendation responses under `recommendations:product:{id}` for 1 hour | M2 | ORIGINAL_REQUEST §R2 |
| 7 | Go REST Endpoint | Expose `GET /api/v1/products/:id/recommendations` with 4-8 limit validation | M2 | ORIGINAL_REQUEST §R2 |
| 8 | PDP Recommendation Carousel | Render horizontal carousel in Product Detail Modal with quick-add & prices | M3 | ORIGINAL_REQUEST §R3 |
| 9 | Cart Drawer Contextual Add-ons | Render contextual recommendations when items exist in cart | M3 | ORIGINAL_REQUEST §R3 |
| 10 | 1-Click Add & Currency Parity | Support dynamic IDR/USD formatting and instant cart badge updates | M3 | ORIGINAL_REQUEST §R3 |
| 11 | Playwright Recommendation E2E Test | Add Test #9 to `storefront.spec.ts` verifying PDP and Cart Drawer quick add | M4 | ORIGINAL_REQUEST §Acceptance |
| 12 | Regression & Container Verification | Verify 13 baseline E2E tests pass, zero build errors across all services | M4 | ORIGINAL_REQUEST §Acceptance |

## Milestones
| # | Name | Scope | Dependencies | Status |
|---|------|-------|-------------|--------|
| M1 | AI Recommendation Engine | `tirenn-ai-service`: Vector similarity, category boost, price corridor, co-occurrence, FastAPI handler | none | DONE |
| M2 | Backend API & Caching | `tirenn-backend`: Redis caching (1h TTL), AI client, top-sellers fallback, REST endpoint | M1 (Interface agreed) | DONE |
| M3 | Storefront UI Integration | `tirenn-frontend`: PDP carousel, Cart Drawer add-ons, 1-click add, currency formatting | M2 (Interface agreed) | DONE |
| M4 | E2E Testing & Verification | `tirenn-infra/qa`: Add recommendation E2E test, run 14 Playwright tests, audit & container builds | M1, M2, M3 | DONE |

## Interface Contracts

### AI Service ↔ Go Backend
- **Endpoint**: `GET /api/v1/products/{id}/recommendations?limit={limit}&type={similar|cross_category}`
- **Headers**: `X-API-Key: <INTERNAL_API_KEY>`
- **Response Format**:
```json
{
  "success": true,
  "product_id": 1,
  "recommendations": [
    {
      "id": 2,
      "score": 0.892,
      "reason": "similar_category_price"
    }
  ],
  "total": 6
}
```

### Go Backend ↔ React Storefront
- **Endpoint**: `GET /api/v1/products/:id/recommendations?limit=6`
- **Response Format**:
```json
{
  "success": true,
  "message": "Recommendations retrieved successfully",
  "data": [
    {
      "id": 2,
      "category_id": 1,
      "sub_category_id": 2,
      "name": "Wireless Noise-Canceling Headphones",
      "slug": "wireless-noise-canceling-headphones",
      "sku": "AUD-WNC-002",
      "description": "...",
      "price": 1499000,
      "currency": "IDR",
      "stock_quantity": 25,
      "image_url": "https://...",
      "is_active": true,
      "badge": "Terlaris",
      "rating": 4.8
    }
  ]
}
```
- **Redis Cache Key**: `recommendations:product:{id}` (TTL: 3600 seconds)

### Storefront UI Test Selectors Contract
- PDP Recommendations Container: `[data-testid="pdp-recommendations-section"]`
- PDP Recommendation Item Card: `[data-testid="pdp-recommendation-card-${product.id}"]`
- PDP Quick Add Button: `[data-testid="pdp-recommendation-add-${product.id}"]`
- Cart Drawer Recommendations Container: `[data-testid="cart-recommendations-section"]`
- Cart Recommendation Item Card: `[data-testid="cart-recommendation-card-${product.id}"]`
- Cart Quick Add Button: `[data-testid="cart-recommendation-add-${product.id}"]`
- Cart Counter Badge: `[data-testid="cart-badge"]`

## Code Layout
- `tirenn-ai-service/`:
  - `app/repositories/product_repository.py`: Vector search and co-occurrence SQL queries
  - `app/usecases/recommendation_usecase.py`: Recommendation scoring and ranking logic
  - `app/handlers/recommendation_handler.py`: FastAPI routes
  - `app/main.py`: Dependency injection wiring
  - `tests/test_recommendations.py`: Pytest suite
- `tirenn-backend/`:
  - `internal/domain/product/ai_client.go`: AI client recommendation method
  - `internal/domain/product/repository.go`: Top-sellers and fallback queries
  - `internal/domain/product/usecase.go`: Recommendation usecase with Redis caching and fallback
  - `internal/domain/product/handler.go`: HTTP handler `GetRecommendations`
  - `internal/router/router.go`: Route registration
  - `internal/domain/product/usecase_test.go`: Unit tests and mock fixes
- `tirenn-frontend/`:
  - `src/services/api.ts`: Recommendation API service call
  - `src/components/ProductDetailModal.tsx`: PDP carousel integration
  - `src/components/CartDrawer.tsx`: Contextual cart add-ons integration
- `tirenn-infra/qa/`:
  - `e2e/storefront.spec.ts`: Test #9 for recommendation display and quick add
