# 📋 Project Context: Tirenn Frontend

This document outlines the state, structure, and operational context of the `tirenn-frontend` service.

---

## 🏛️ Application Architecture & Views

The application renders two primary view suites depending on user role and navigation:

### 1. Storefront Suite (`AppView: 'storefront' | 'my-orders'`)
- **Navbar (`src/components/Navbar.tsx`)**: Global brand navigation, bilingual toggle (IDR/USD, ID/EN), search bar, category shortcuts, Cart counter badge, AI Shopper trigger button, and User Auth state.
- **HeroBanner (`src/components/HeroBanner.tsx`)**: High-impact promotional hero with quick action triggers.
- **FilterBar (`src/components/FilterBar.tsx`)**: Category pills, sort selectors (newest, price, rating), in-stock toggles, AI semantic toggle.
- **ProductCard (`src/components/ProductCard.tsx`)**: Clean eCommerce product tile with badge, price in active currency, stock status, and PDP trigger.
- **ProductDetailModal (`src/components/ProductDetailModal.tsx`)**: Modal view for product specs, SKU, stock indicator, quantity adjuster, and direct checkout.
- **CartDrawer (`src/components/CartDrawer.tsx`)**: Slide-over drawer with item listing, quantity modifier, subtotal calculation, and checkout trigger.
- **CheckoutModal (`src/components/CheckoutModal.tsx`)**: Shipping address form, order summary, auth-gating check, and atomic order placement.
- **AIChatModal (`src/components/AIChatModal.tsx`)**: Autonomous AI Shopper interface connecting directly to Python AI service (`:8000`), supporting tool calling feedback, guest consultation, and login-gated cart addition.
- **OrderHistory (`src/components/OrderHistory.tsx`)**: User purchase history with order status badges and item summaries.

### 2. Admin Control Suite (`AppView: 'admin-dashboard' | 'admin-products' | 'admin-orders' | 'admin-customers' | 'admin-knowledge'`)
- **AdminDashboard (`src/components/admin/AdminDashboard.tsx`)**: Executive KPIs, gross revenue, order volume, inventory valuation, and quick actions.
- **ProductManagement (`src/components/admin/ProductManagement.tsx`)**: Product listing, CRUD modal, stock adjustment triggers with reason audit logging.
- **OrderManagement (`src/components/admin/OrderManagement.tsx`)**: Order status management (PENDING, PAID, SHIPPED, DELIVERED, CANCELLED).
- **CustomerManagement (`src/components/admin/CustomerManagement.tsx`)**: CRM user directory with spending metrics.
- **KnowledgeManagement (`src/components/admin/KnowledgeManagement.tsx`)**: In-memory PDF upload zone, document catalog, and real-time Semantic Vector RAG playground.

---

## 🔄 Global State & Context Providers

- **`AuthContext` (`src/context/AuthContext.tsx`)**: Manages `currentUser`, `token`, `login()`, `logout()`, and `refreshProfile()`. Persists credentials in `localStorage` (`tirenn_user`, `tirenn_token`).
- **`CartContext` (`src/context/CartContext.tsx`)**: Client-side shopping cart state (`items`, `cartCount`, `cartTotal`, `addToCart`, `removeFromCart`, `updateQuantity`, `clearCart`). Persisted in `localStorage` (`tirenn_cart`).
- **`CurrencyContext` (`src/context/CurrencyContext.tsx`)**: Currency conversion state (IDR <-> USD) and price formatting helper (`formatPrice`).
- **`ToastContext` (`src/context/ToastContext.tsx`)**: Global unobtrusive notification banners for errors, info, and success actions.

---

## 🌐 API Integrations

- **Golang Backend (`http://localhost:8080/api/v1`)**:
  - `/products`, `/categories`, `/orders/checkout`, `/orders/my`, `/auth/login`, `/auth/register`, `/auth/me`, `/admin/*`.
- **Python AI Service (`http://localhost:8000/api/v1`)**:
  - `/chat/shopper`: Conversational agent with tool calling.
  - `/knowledge/upload-pdf`: In-memory PDF vectorization (JWT protected).
  - `/knowledge/documents`: Document list & deletion (JWT protected).
  - `/knowledge/query`: Vector RAG search.
  - `/search/semantic`: Vector & hybrid semantic search.

---

## 📜 Service Changelog

### 📅 2026-08-28
- `[Frontend]` Updated `AIChatModal.tsx` to render the initial welcome message dynamically through `getWelcomeMessage(i18n.language)` directly in the JSX tree, guaranteeing instant re-render upon language toggle (ID $\leftrightarrow$ EN) in both directions.
- `[Frontend]` Rebuilt production Docker container `tirenn-frontend`.
- `[Frontend]` Enhanced `AIChatModal.tsx` `cart_action` handler to reliably dispatch items into `CartContext` supporting both direct top-level payload attributes and nested `cartAction.product` objects, ensuring items added via AI chat immediately appear in Cart Drawer and update the Cart badge.
- `[Frontend]` Rebuilt production bundle and updated container `tirenn-frontend`.
