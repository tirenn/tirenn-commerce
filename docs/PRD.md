# Product Requirement Document (PRD)
## Project: GoCommerce - Lightweight E-Commerce Platform

---

## 1. Executive Summary & Vision

### 1.1 Overview
**GoCommerce** is an end-to-end, high-performance, single-binary (or lightweight modular) e-commerce web application built entirely in **Golang**. It provides both a Customer-Facing Storefront and a dedicated Merchant Admin Back-Office.

### 1.2 Core Objectives
- **Simplicity & Performance**: Sub-50ms response times powered by native Go HTTP handlers, server-side rendering (SSR with `html/template` or `templ` + `htmx`), and minimal asset overhead.
- **Merchant Empowerment**: Full self-service back-office for inventory control, product catalog management, order fulfillment, customer insights, and real-time sales dashboards.
- **Seamless Shopping Experience**: Instant product search, multi-attribute filtering, cart management, streamlined checkout, and transparent order tracking for shoppers.

---

## 2. User Personas & Roles

| Persona | Role | Primary Goals | Key Pain Points |
| :--- | :--- | :--- | :--- |
| **Store Admin / Merchant** | `ADMIN` | Manage catalog, adjust inventory levels, fulfill customer orders, monitor revenue & stock health. | Clunky interfaces, slow order processing, lack of stock alerts. |
| **Shopper / Customer** | `CUSTOMER` | Discover products easily, filter by category/price, place orders securely, view past purchase history. | Complex checkout, slow search, opaque order statuses. |
| **Guest Shopper** | `GUEST` | Browse catalog, search items, add to cart prior to login/registration. | Forced registration before browsing. |

---

## 3. System Architecture & Tech Stack

```text
                           ┌──────────────────────────────────────────────┐
                           │              GoCommerce System               │
                           │                                              │
  ┌──────────────────┐     │  ┌─────────────────┐    ┌─────────────────┐  │     ┌──────────────────┐
  │   Shopper Web    │────▶│  │ Storefront HTTP │    │ Admin Back-     │  │────▶│ PostgreSQL /     │
  │ (HTML+HTMX+CSS)  │     │  │ Handlers (SSR)  │    │ office Handlers │  │     │ SQLite Database  │
  └──────────────────┘     │  └────────┬────────┘    └────────┬────────┘  │     └──────────────────┘
                           │           │                      │           │
  ┌──────────────────┐     │           ▼                      ▼           │
  │   Admin Web      │────▶│  ┌────────────────────────────────────────┐  │
  │ (Admin Dashboard)│     │  │       Domain Services / Logic Layer     │  │
  └──────────────────┘     │  │ (Catalog, Orders, Inventory, Users)    │  │
                           │  └────────────────────────────────────────┘  │
                           └──────────────────────────────────────────────┘
```

- **Backend Runtime**: Golang 1.22+ (`net/http` / `chi` / `gin`)
- **Frontend Layer**: Golang Server-Side Rendering (SSR) with `html/template` or `templ`, enhanced by **HTMX** (for reactive SPA-like interactions without heavy JS frameworks) and **Tailwind CSS**.
- **Database**: PostgreSQL (Production) / SQLite (Embedded / Local Dev) with migrations (`golang-migrate` or `goose`).
- **Session / Auth**: HTTP-only secure Cookie sessions + JWT/HMAC tokens, bcrypt password hashing.

---

## 4. Functional Requirements (FRD)

### Epic 1: Authentication & Role-Based Access Control (RBAC)
- **FR-1.1**: User registration with email, password, full name, and phone number.
- **FR-1.2**: Secure login with password hashing (`golang.org/x/crypto/bcrypt`) and session persistence.
- **FR-1.3**: Distinct middleware protections: `/admin/*` requires `Role == "ADMIN"`, `/account/*` requires `Role == "CUSTOMER"` or `"ADMIN"`.
- **FR-1.4**: Guest session capability for cart retention prior to authentication.

---

### Epic 2: Admin Product & Catalog Management
- **FR-2.1 - Product CRUD**: Admin can create, view, update, and soft-delete products (Name, SKU, Description, Price, Image URL, Category, Status).
- **FR-2.2 - Category Management**: Create, edit, and organize hierarchical product categories.
- **FR-2.3 - Stock & Inventory Control**: 
  - Direct stock adjustments with reason logging (Restock, Damage, Manual Correction).
  - Low-stock threshold configuration per item.
  - Automatic inventory decrement on completed checkout and increment on cancellation.

---

### Epic 3: Storefront & Product Discovery
- **FR-3.1 - Catalog Browsing**: Responsive product grid showing thumbnail, title, price, category badge, and stock availability.
- **FR-3.2 - Live Search**: Keyword-based full-text search across product title, SKU, and description.
- **FR-3.3 - Dynamic Filtering & Sorting**:
  - Filter by Category, Price Range (`min_price`, `max_price`), and In-Stock availability.
  - Sort by Newest, Price (Low to High / High to Low), and Best Sellers.
- **FR-3.4 - Product Detail Page (PDP)**: Rich view with image gallery, detailed description, specs, real-time stock status, and "Add to Cart" CTA.

---

### Epic 4: Shopping Cart & Checkout Flow
- **FR-4.1 - Cart Management**: Add item, update quantities, remove item, and display subtotal/tax/grand total.
- **FR-4.2 - Inventory Locking / Verification**: Validate real-time stock availability before initiating checkout.
- **FR-4.3 - Checkout Experience**:
  - Shipping address collection.
  - Payment method selection (Simulated / Gateway integration ready: Credit Card, Bank Transfer, Cash on Delivery).
- **FR-4.4 - Order Creation**: Atomic database transaction creating Order + OrderItems and updating inventory counts.

---

### Epic 5: Order Lifecycle & Management
- **FR-5.1 - Order State Machine**:
  `PENDING` ➔ `PAID` ➔ `PROCESSING` ➔ `SHIPPED` ➔ `DELIVERED` | `CANCELLED`
- **FR-5.2 - Admin Order Back-Office**:
  - Filter orders by status, date range, customer, and order ID.
  - Update fulfillment status, input tracking numbers, and process cancellations/refunds.
- **FR-5.3 - Customer Order History**:
  - Customer can list past orders with status badges and invoice totals.
  - Detailed order receipt page with itemized breakdown and shipping address.

---

### Epic 6: Customer Relationship Management (CRM)
- **FR-6.1 - Customer Directory**: Admin can search and inspect registered customers (Account creation date, total spend, total orders).
- **FR-6.2 - Account Management**: Admin can activate, suspend, or reset customer account statuses.
- **FR-6.3 - Customer Profile**: Customers can update contact details and saved delivery addresses.

---

### Epic 7: Admin Analytics & Business Dashboard
- **FR-7.1 - Key Performance Indicators (KPIs)**:
  - Total Revenue (Day / Month / Lifetime).
  - Total Orders & Conversion metrics.
  - Total Registered Customers.
  - Low Stock Item Count.
- **FR-7.2 - Visual Reports & Trends**:
  - Sales revenue chart over time (Last 7 days / 30 days).
  - Top 5 Best-Selling Products by volume and revenue.
  - Recent Order Feed with quick-action status updates.

---

## 5. Domain Data Model (Entity Relationship)

```text
  ┌──────────────────┐          ┌──────────────────┐
  │      Users       │1        *│      Orders      │
  ├──────────────────┤──────────├──────────────────┤
  │ id (UUID/INT)    │          │ id (UUID/INT)    │
  │ name             │          │ user_id          │
  │ email            │          │ total_amount     │
  │ password_hash    │          │ status (ENUM)    │
  │ role (ENUM)      │          │ shipping_address │
  │ status (ACTIVE..)│          │ payment_method   │
  │ created_at       │          │ created_at       │
  └──────────────────┘          └────────┬─────────┘
                                         │ 1
                                         │ *
  ┌──────────────────┐          ┌────────▼─────────┐
  │    Categories    │1        *│   Order_Items    │
  ├──────────────────┤──────────├──────────────────┤
  │ id               │          │ id               │
  │ name             │          │ order_id         │
  │ slug             │          │ product_id       │
  │ description      │          │ quantity         │
  └────────┬─────────┘          │ unit_price       │
           │ 1                  │ subtotal         │
           │ *                  └──────────────────┘
  ┌────────▼─────────┐                   │ *
  │     Products     │                   │ 1
  ├──────────────────┤───────────────────┘
  │ id (UUID/INT)    │
  │ category_id      │
  │ name             │
  │ sku              │
  │ description      │
  │ price            │
  │ stock_quantity   │
  │ image_url        │
  │ is_active        │
  │ created_at       │
  └──────────────────┘
```

---

## 6. User Stories & Acceptance Criteria (Gherkin Sample)

### Scenario 1: Customer Places an Order with Stock Validation
```gherkin
Feature: Checkout & Stock Deduction
  As a Customer
  I want to checkout my cart items
  So that I can purchase the products

  Scenario: Successful order placement
    Given I have 2 units of "Wireless Mouse" in my cart with available stock of 5
    When I submit the checkout form with valid shipping details
    Then an Order should be created with status "PENDING"
    And the available stock of "Wireless Mouse" should become 3
    And my shopping cart should be emptied
    And I should be redirected to the order confirmation page
```

### Scenario 2: Admin Adjusts Stock and Low Stock Trigger
```gherkin
Feature: Inventory Adjustment
  As an Admin
  I want to adjust product stock levels
  So that inventory counts match physical warehouse audits

  Scenario: Stock drops below threshold
    Given Product "Mechanical Keyboard" has 5 units in stock with threshold 3
    When Admin updates stock to 2 units
    Then the product inventory is set to 2
    And the Dashboard displays a "Low Stock Warning" for "Mechanical Keyboard"
```

---

## 7. Non-Functional Requirements (NFR)

- **Performance**: P95 Server Response Time < 60ms under standard loads.
- **Concurrency & Atomicity**: Database transactions with row-level locks (`SELECT FOR UPDATE`) during checkout to prevent overselling race conditions.
- **Security**:
  - CSRF protection across all state-mutating forms.
  - SQL Injection prevention via parameterized queries.
  - XSS prevention through context-aware Go HTML escaping.
- **Reliability & Observability**: Structured JSON logging (`log/slog`), healthcheck endpoint (`/healthz`), and graceful shutdown.

---

## 8. Phased Roadmap & Milestones

| Phase | Milestone | Deliverables |
| :--- | :--- | :--- |
| **Phase 1** | **Foundations & Domain** | Project layout, DB migrations, User models, Auth & RBAC middleware. |
| **Phase 2** | **Admin Back-Office** | Product/Category CRUD, Inventory adjustments, Customer management UI. |
| **Phase 3** | **Storefront & Catalog** | Product grid, Category filtering, Live search, PDP. |
| **Phase 4** | **Cart & Checkout** | Cart session engine, Checkout flow, Stock atomic deduction, Order creation. |
| **Phase 5** | **Order Lifecycle & Dashboard** | Admin order management, Customer order history, Dashboard KPI charts. |
| **Phase 6** | **Polish & Verification** | E2E test suite, responsive UI audit, performance tuning. |
