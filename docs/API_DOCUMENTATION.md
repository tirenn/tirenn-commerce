# 📚 Tirenn Commerce - REST API Specification & Integration Guide

> **Target Audience**: Frontend Engineers, Mobile Developers, and QA Automation Engineers.  
> **API Version**: `v1`  
> **Base URL**: `http://localhost:8080/api/v1` (Production / Staging / Local)  
> **Protocol**: HTTP/1.1 or HTTP/2, RESTful JSON over TCP  

---

## 📑 Table of Contents

1. [General Information & Base URLs](#1-general-information--base-urls)
2. [Authentication & Authorization (JWT & RBAC)](#2-authentication--authorization-jwt--rbac)
3. [Standard Response Envelope](#3-standard-response-envelope)
4. [Pre-Seeded Demo Accounts](#4-pre-seeded-demo-accounts)
5. [Endpoints Specification](#5-endpoints-specification)
   - [A. System & Healthcheck](#a-system--healthcheck)
   - [B. Authentication Domain](#b-authentication-domain)
   - [C. Products & Storefront Catalog Domain](#c-products--storefront-catalog-domain)
   - [D. Orders & Checkout Domain (Customer)](#d-orders--checkout-domain-customer)
   - [E. Merchant Admin Domain (Requires ADMIN Role)](#e-merchant-admin-domain-requires-admin-role)
6. [TypeScript Interfaces for Frontend](#6-typescript-interfaces-for-frontend)
7. [Frontend Integration Examples (Fetch API Client)](#7-frontend-integration-examples-fetch-api-client)

---

## 1. General Information & Base URLs

| Environment | Base URL | Health Check |
| :--- | :--- | :--- |
| **Local Development** | `http://localhost:8080/api/v1` | `http://localhost:8080/healthz` |
| **Docker Network** | `http://backend:8080/api/v1` | `http://backend:8080/healthz` |

### Required Headers
- `Content-Type: application/json`
- `Accept: application/json`
- `Authorization: Bearer <JWT_TOKEN>` *(for authenticated endpoints)*

---

## 2. Authentication & Authorization (JWT & RBAC)

The API uses **JSON Web Tokens (JWT)** for stateless session management:
- When a user logs in via `POST /api/v1/auth/login` or registers via `POST /api/v1/auth/register`, the backend returns a signed JWT token with a 24-hour expiration.
- Store the token in `localStorage` (`tirenn_token`) or secure cookies.
- Send the token in the HTTP `Authorization` header:
  ```http
  Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
  ```

### Role-Based Access Control (RBAC):
- **`CUSTOMER`**: Can browse public catalog, add to cart, perform atomic checkout, and view their own order history (`/orders/*`).
- **`ADMIN`**: Has exclusive access to the `/admin/*` routes (Analytics Dashboard, Product CRUD, Stock Adjuster, Order Fulfillment status transitions, Customer CRM).
- Non-admin attempts to access `/admin/*` will receive `403 Forbidden`.

---

## 3. Standard Response Envelope

All API responses conform to a predictable envelope structure:

### Successful Response (Single Entity or List)
```json
{
  "success": true,
  "message": "Operation completed successfully",
  "data": { ... }
}
```

### Successful Response with Pagination Metadata
```json
{
  "success": true,
  "message": "Products retrieved successfully",
  "data": [ ... ],
  "meta": {
    "page": 1,
    "limit": 12,
    "total_rows": 9,
    "total_pages": 1
  }
}
```

### Error Response
```json
{
  "success": false,
  "message": "Human-readable error summary",
  "error": "Detailed validation or system error reason"
}
```

### Common HTTP Status Codes
| Code | Meaning | When it occurs |
| :--- | :--- | :--- |
| **`200 OK`** | Success | GET, PUT, PATCH, DELETE operations succeed |
| **`201 Created`** | Created | POST operations (registration, product creation, checkout) |
| **`400 Bad Request`** | Validation Error | Missing required fields, invalid types, negative stock |
| **`401 Unauthorized`** | Auth Missing / Expired | Token missing, invalid signature, or expired |
| **`403 Forbidden`** | Insufficient Role | Customer attempting to call Admin endpoint |
| **`404 Not Found`** | Resource Missing | Product, Order, or Category not found |
| **`409 Conflict`** | Out of Stock | Concurrency lock detected insufficient inventory during checkout |
| **`500 Internal Error`** | Server Error | Database failure or unhandled panic |

---

## 4. Pre-Seeded Demo Accounts

| Role | Email | Password | Access Rights |
| :--- | :--- | :--- | :--- |
| **👑 Merchant Admin** | `admin@gocommerce.com` | `Admin@123` | Full Back-Office, Stock Adjustments, Order Fulfillment, CRM, Analytics |
| **🛍️ Shopper (Customer)** | `shopper@gocommerce.com` | `Shopper@123` | Storefront Catalog, Cart, Checkout, Order History |
| **⭐ Customer 2** | `sarah.jenkins@example.com` | `Sarah@123` | Storefront Catalog, Cart, Checkout, Order History |

---

## 5. Endpoints Specification

---

### A. System & Healthcheck

#### `GET /healthz`
Public system health check and database connectivity verification.

- **Auth**: None
- **Response `200 OK`**:
```json
{
  "success": true,
  "message": "GoCommerce API is healthy 🚀",
  "data": {
    "database": "mysql",
    "environment": "development",
    "status": "online",
    "timestamp": "2026-08-27T10:02:45.317Z"
  }
}
```

---

### B. Authentication Domain

#### 1. `POST /api/v1/auth/register`
Register a new customer account.

- **Auth**: None
- **Request Body**:
```json
{
  "name": "Alex Rivera",
  "email": "alex.rivera@example.com",
  "password": "SecurePassword123!",
  "phone": "+1-555-019-2834",
  "address": "742 Evergreen Terrace, Springfield, OR"
}
```
- **Response `201 Created`**:
```json
{
  "success": true,
  "message": "Registration successful",
  "data": {
    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "user": {
      "id": 4,
      "name": "Alex Rivera",
      "email": "alex.rivera@example.com",
      "role": "CUSTOMER",
      "phone": "+1-555-019-2834",
      "address": "742 Evergreen Terrace, Springfield, OR",
      "status": "ACTIVE",
      "created_at": "2026-08-27T10:15:00Z"
    }
  }
}
```

---

#### 2. `POST /api/v1/auth/login`
Authenticate an existing user (Admin or Customer).

- **Auth**: None
- **Request Body**:
```json
{
  "email": "shopper@gocommerce.com",
  "password": "Shopper@123"
}
```
- **Response `200 OK`**:
```json
{
  "success": true,
  "message": "Login successful",
  "data": {
    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "user": {
      "id": 2,
      "name": "Alex Mercer",
      "email": "shopper@gocommerce.com",
      "role": "CUSTOMER",
      "status": "ACTIVE",
      "created_at": "2026-08-27T09:43:35Z"
    }
  }
}
```

---

#### 3. `GET /api/v1/auth/me`
Retrieve authenticated user profile and permissions.

- **Auth**: `Bearer <JWT_TOKEN>`
- **Response `200 OK`**:
```json
{
  "success": true,
  "message": "User profile retrieved",
  "data": {
    "id": 2,
    "name": "Alex Mercer",
    "email": "shopper@gocommerce.com",
    "role": "CUSTOMER",
    "phone": "+1-555-019-2834",
    "address": "742 Evergreen Terrace, Springfield, OR",
    "status": "ACTIVE",
    "created_at": "2026-08-27T09:43:35Z"
  }
}
```

---

### C. Products & Storefront Catalog Domain

#### 1. `GET /api/v1/products`
List, search, filter, and paginate storefront products.

- **Auth**: None
- **Query Parameters**:
  - `search` *(string, optional)*: Match product title or SKU (e.g. `?search=Headphones`)
  - `category_id` *(int, optional)*: Filter by department ID (e.g. `?category_id=1`)
  - `sort` *(string, optional)*:
    - `newest` *(default)*
    - `price_asc` (Price: Low to High)
    - `price_desc` (Price: High to Low)
    - `name_asc` (Name: A to Z)
  - `in_stock` *(boolean, optional)*: `true` to return only products with `stock_quantity > 0`
  - `page` *(int, optional, default: 1)*: Page number
  - `limit` *(int, optional, default: 12)*: Number of items per page (e.g. `?limit=50`)
- **Response `200 OK`**:
```json
{
  "success": true,
  "message": "Products retrieved successfully",
  "data": [
    {
      "id": 1,
      "category_id": 1,
      "category": {
        "id": 1,
        "name": "Electronics & Tech",
        "slug": "electronics-tech",
        "description": "High-performance headphones, smart wearables, and computer peripherals.",
        "icon": "⚡",
        "created_at": "2026-08-27T09:43:35Z"
      },
      "name": "AuraPro Active Noise-Cancelling Wireless Headphones",
      "slug": "aurapro-anc-headphones",
      "sku": "TECH-AP-001",
      "description": "Premium over-ear wireless headphones with hybrid active noise cancellation.",
      "price": 149.99,
      "stock_quantity": 35,
      "low_stock_threshold": 8,
      "image_url": "https://images.unsplash.com/photo-1505740420928-5e560c06d30e?w=600",
      "is_active": true,
      "badge": "BESTSELLER",
      "rating": 4.9,
      "created_at": "2026-08-27T09:43:35Z",
      "updated_at": "2026-08-27T09:43:35Z"
    }
  ],
  "meta": {
    "page": 1,
    "limit": 50,
    "total_rows": 9,
    "total_pages": 1
  }
}
```

---

#### 2. `GET /api/v1/products/:id`
Get single product details by Numeric ID or URL Slug.

- **Auth**: None
- **URL Parameters**:
  - `:id` -> Either `1` (numeric ID) or `aurapro-anc-headphones` (slug)
- **Response `200 OK`**:
```json
{
  "success": true,
  "message": "Product retrieved",
  "data": {
    "id": 1,
    "category_id": 1,
    "name": "AuraPro Active Noise-Cancelling Wireless Headphones",
    "slug": "aurapro-anc-headphones",
    "sku": "TECH-AP-001",
    "description": "Premium over-ear wireless headphones with hybrid active noise cancellation.",
    "price": 149.99,
    "stock_quantity": 35,
    "low_stock_threshold": 8,
    "image_url": "https://images.unsplash.com/photo-1505740420928-5e560c06d30e?w=600",
    "is_active": true,
    "rating": 4.9,
    "created_at": "2026-08-27T09:43:35Z"
  }
}
```

---

#### 3. `GET /api/v1/categories`
Get list of all retail department categories.

- **Auth**: None
- **Response `200 OK`**:
```json
{
  "success": true,
  "message": "Categories retrieved successfully",
  "data": [
    {
      "id": 1,
      "name": "Electronics & Tech",
      "slug": "electronics-tech",
      "description": "High-performance tech and accessories",
      "icon": "⚡",
      "created_at": "2026-08-27T09:43:35Z"
    },
    {
      "id": 2,
      "name": "Fashion & Apparel",
      "slug": "fashion-apparel",
      "description": "Everyday apparel and outerwear",
      "icon": "👕",
      "created_at": "2026-08-27T09:43:35Z"
    }
  ]
}
```

---

### D. Orders & Checkout Domain (Customer)

#### 1. `POST /api/v1/orders/checkout`
Perform atomic transactional checkout. Automatically acquires row-level locks (`SELECT FOR UPDATE`), verifies available inventory, decrements product stock quantities, creates order items, and records order history.

- **Auth**: `Bearer <JWT_TOKEN>`
- **Request Body**:
```json
{
  "items": [
    {
      "product_id": 1,
      "quantity": 2
    },
    {
      "product_id": 4,
      "quantity": 1
    }
  ],
  "shipping_name": "Alex Mercer",
  "shipping_phone": "+1-555-019-2834",
  "shipping_address": "742 Evergreen Terrace, Springfield, OR",
  "payment_method": "CARD",
  "notes": "Please leave package at front door"
}
```
- **Response `201 Created`**:
```json
{
  "success": true,
  "message": "Order created successfully",
  "data": {
    "id": 5,
    "order_number": "ORD-20260827-9481",
    "user_id": 2,
    "total_amount": 353.98,
    "status": "PAID",
    "shipping_name": "Alex Mercer",
    "shipping_phone": "+1-555-019-2834",
    "shipping_address": "742 Evergreen Terrace, Springfield, OR",
    "payment_method": "CARD",
    "payment_status": "COMPLETED",
    "notes": "Please leave package at front door",
    "items": [
      {
        "id": 8,
        "order_id": 5,
        "product_id": 1,
        "product_name": "AuraPro Active Noise-Cancelling Wireless Headphones",
        "product_sku": "TECH-AP-001",
        "product_image": "https://images.unsplash.com/photo-1505740420928-5e560c06d30e?w=600",
        "quantity": 2,
        "unit_price": 149.99,
        "subtotal": 299.98
      },
      {
        "id": 9,
        "order_id": 5,
        "product_id": 4,
        "product_name": "UrbanCraft Heavyweight French Terry Hoodie",
        "product_sku": "FASH-HD-101",
        "product_image": "https://images.unsplash.com/photo-1556905055-8f358a7a47b2?w=600",
        "quantity": 1,
        "unit_price": 54.00,
        "subtotal": 54.00
      }
    ],
    "created_at": "2026-08-27T10:20:00Z",
    "updated_at": "2026-08-27T10:20:00Z"
  }
}
```
- **Error Response `400 Bad Request` (Insufficient Stock)**:
```json
{
  "success": false,
  "message": "Product 'AuraPro Active Noise-Cancelling Wireless Headphones' is out of stock (Requested: 50, Available: 35)",
  "error": "Insufficient stock"
}
```

---

#### 2. `GET /api/v1/orders/my-orders`
List order invoices belonging to the currently authenticated customer.

- **Auth**: `Bearer <JWT_TOKEN>`
- **Response `200 OK`**:
```json
{
  "success": true,
  "message": "Orders retrieved",
  "data": [
    {
      "id": 5,
      "order_number": "ORD-20260827-9481",
      "total_amount": 353.98,
      "status": "PAID",
      "shipping_name": "Alex Mercer",
      "shipping_address": "742 Evergreen Terrace, Springfield, OR",
      "payment_method": "CARD",
      "payment_status": "COMPLETED",
      "items": [ ... ],
      "created_at": "2026-08-27T10:20:00Z"
    }
  ]
}
```

---

### E. Merchant Admin Domain (Requires ADMIN Role)

---

#### 1. `GET /api/v1/admin/dashboard`
Get executive revenue KPIs, 7-day velocity chart data, low-stock radar, and recent orders.

- **Auth**: `Bearer <ADMIN_JWT_TOKEN>`
- **Query Parameters**:
  - `days` *(int, optional, default: 7)*: Analysis window
- **Response `200 OK`**:
```json
{
  "success": true,
  "message": "Dashboard data retrieved",
  "data": {
    "summary": {
      "total_revenue": 1428.50,
      "total_orders": 8,
      "total_customers": 4,
      "low_stock_count": 2,
      "pending_orders_count": 1
    },
    "revenue_trends": [
      { "date": "2026-08-21", "revenue": 120.00, "order_count": 1 },
      { "date": "2026-08-27", "revenue": 353.98, "order_count": 2 }
    ],
    "top_selling_products": [
      {
        "product_id": 1,
        "product_name": "AuraPro ANC Headphones",
        "product_sku": "TECH-AP-001",
        "product_image": "https://images.unsplash.com/...",
        "total_sold": 6,
        "total_revenue": 899.94
      }
    ],
    "recent_orders": [
      {
        "id": 5,
        "order_number": "ORD-20260827-9481",
        "customer_name": "Alex Mercer",
        "total_amount": 353.98,
        "status": "PAID",
        "created_at": "2026-08-27T10:20:00Z"
      }
    ],
    "low_stock_alerts": [
      {
        "id": 8,
        "product_name": "Nomad 35L Waterproof Travel Backpack",
        "product_sku": "SPRT-BP-301",
        "stock_quantity": 4,
        "low_stock_threshold": 5
      }
    ]
  }
}
```

---

#### 2. `GET /api/v1/admin/products`
Admin catalog listing including inactive/hidden items.

- **Auth**: `Bearer <ADMIN_JWT_TOKEN>`
- **Query Parameters**: `search`, `category_id`, `page`, `limit`
- **Response `200 OK`**: Same structure as storefront product list.

---

#### 3. `POST /api/v1/admin/products`
Create a new product SKU in the catalog.

- **Auth**: `Bearer <ADMIN_JWT_TOKEN>`
- **Request Body**:
```json
{
  "name": "Smart Ambient Glow Lamp",
  "sku": "SKU-9402",
  "category_id": 3,
  "description": "Dimmable wireless bedside ambient lamp.",
  "price": 39.99,
  "stock_quantity": 30,
  "low_stock_threshold": 5,
  "image_url": "https://images.unsplash.com/photo-1507473885765-e6ed057f782c",
  "is_active": true
}
```
- **Response `201 Created`**: Returns created product entity.

---

#### 4. `PUT /api/v1/admin/products/:id`
Update an existing product's title, price, description, threshold, or active visibility.

- **Auth**: `Bearer <ADMIN_JWT_TOKEN>`
- **Response `200 OK`**: Returns updated product entity.

---

#### 5. `DELETE /api/v1/admin/products/:id`
Soft-delete / remove a product from the catalog.

- **Auth**: `Bearer <ADMIN_JWT_TOKEN>`
- **Response `200 OK`**: `{"success": true, "message": "Product deleted successfully"}`

---

#### 6. `POST /api/v1/admin/products/:id/adjust-stock`
Adjust inventory counts with immutable audit tracking.

- **Auth**: `Bearer <ADMIN_JWT_TOKEN>`
- **Request Body**:
```json
{
  "type": "ADD",
  "amount": 25,
  "reason": "Restock Shipment from Supplier"
}
```
*Valid `type` values*:
- `"ADD"`: Increment current stock by `amount`
- `"SUBTRACT"`: Deduct `amount` from current stock
- `"SET"`: Set stock to exact `amount`

- **Response `200 OK`**: Returns updated product entity with new `stock_quantity`.

---

#### 7. `GET /api/v1/admin/products/:id/stock-logs`
Get historical stock audit logs for a product.

- **Auth**: `Bearer <ADMIN_JWT_TOKEN>`
- **Response `200 OK`**:
```json
{
  "success": true,
  "message": "Stock adjustment logs retrieved",
  "data": [
    {
      "id": 12,
      "product_id": 1,
      "change_amount": 25,
      "previous_stock": 10,
      "current_stock": 35,
      "reason": "Restock Shipment from Supplier",
      "created_at": "2026-08-27T10:25:00Z"
    }
  ]
}
```

---

#### 8. `GET /api/v1/admin/orders`
List customer orders across the platform with filtering.

- **Auth**: `Bearer <ADMIN_JWT_TOKEN>`
- **Query Parameters**:
  - `search` *(string, optional)*: Search order number, customer name, phone
  - `status` *(string, optional)*: `PAID`, `PROCESSING`, `SHIPPED`, `COMPLETED`, `CANCELLED`
  - `page`, `limit`
- **Response `200 OK`**: Returns array of orders with customer shipping details.

---

#### 9. `PATCH /api/v1/admin/orders/:id/status`
Update order lifecycle fulfillment status. If set to `CANCELLED`, previously deducted product stocks are **automatically restored**.

- **Auth**: `Bearer <ADMIN_JWT_TOKEN>`
- **Request Body**:
```json
{
  "status": "SHIPPED"
}
```
*Valid `status` values*: `"PAID"`, `"PROCESSING"`, `"SHIPPED"`, `"COMPLETED"`, `"CANCELLED"`.

- **Response `200 OK`**: Returns updated order entity.

---

#### 10. `GET /api/v1/admin/customers`
Retrieve Customer CRM directory with lifetime spend metrics.

- **Auth**: `Bearer <ADMIN_JWT_TOKEN>`
- **Query Parameters**: `search`, `status` (`ACTIVE` / `SUSPENDED`), `limit`
- **Response `200 OK`**:
```json
{
  "success": true,
  "message": "Customers retrieved",
  "data": [
    {
      "id": 2,
      "name": "Alex Mercer",
      "email": "shopper@gocommerce.com",
      "phone": "+1-555-019-2834",
      "address": "742 Evergreen Terrace, Springfield, OR",
      "status": "ACTIVE",
      "total_orders": 3,
      "total_spent": 524.97,
      "created_at": "2026-08-27T09:43:35Z"
    }
  ]
}
```

---

#### 11. `PATCH /api/v1/admin/customers/:id/status`
Suspend or Reactivate a customer account.

- **Auth**: `Bearer <ADMIN_JWT_TOKEN>`
- **Request Body**:
```json
{
  "status": "SUSPENDED"
}
```
- **Response `200 OK`**: `{"success": true, "message": "Customer status updated"}`

---

## 6. TypeScript Interfaces for Frontend

Copy and paste these interfaces into your frontend codebase (`src/types/index.ts`):

```typescript
export type UserRole = 'ADMIN' | 'CUSTOMER';
export type AccountStatus = 'ACTIVE' | 'SUSPENDED';
export type OrderStatus = 'PENDING' | 'PAID' | 'PROCESSING' | 'SHIPPED' | 'COMPLETED' | 'CANCELLED';

export interface User {
  id: number;
  name: string;
  email: string;
  role: UserRole;
  phone?: string;
  address?: string;
  status: AccountStatus;
  created_at: string;
}

export interface Category {
  id: number;
  name: string;
  slug: string;
  description: string;
  icon?: string;
  created_at: string;
}

export interface Product {
  id: number;
  category_id: number;
  category?: Category;
  name: string;
  slug: string;
  sku: string;
  description: string;
  price: number;
  stock_quantity: number;
  low_stock_threshold: number;
  image_url: string;
  is_active: boolean;
  badge?: string;
  rating: number;
  created_at: string;
  updated_at: string;
}

export interface OrderItem {
  id: number;
  order_id: number;
  product_id: number;
  product_name: string;
  product_sku: string;
  product_image?: string;
  quantity: number;
  unit_price: number;
  subtotal: number;
}

export interface Order {
  id: number;
  order_number: string;
  user_id: number;
  user?: User;
  total_amount: number;
  status: OrderStatus;
  shipping_name: string;
  shipping_phone: string;
  shipping_address: string;
  payment_method: string;
  payment_status: string;
  notes?: string;
  items: OrderItem[];
  created_at: string;
  updated_at: string;
}

export interface PaginationMeta {
  page: number;
  limit: number;
  total_rows: number;
  total_pages: number;
}

export interface ApiResponse<T = any> {
  success: boolean;
  message?: string;
  data?: T;
  error?: string;
  meta?: PaginationMeta;
}
```

---

## 7. Frontend Integration Examples (Fetch API Client)

```typescript
// src/services/api.ts
const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080/api/v1';

export function getAuthToken(): string | null {
  return localStorage.getItem('tirenn_token');
}

export async function apiRequest<T>(
  endpoint: string,
  options: RequestInit = {}
): Promise<ApiResponse<T>> {
  const token = getAuthToken();
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
    ...(options.headers as Record<string, string>),
  };

  if (token) {
    headers['Authorization'] = `Bearer ${token}`;
  }

  const url = endpoint.startsWith('http') ? endpoint : `${API_BASE_URL}${endpoint}`;

  try {
    const res = await fetch(url, { ...options, headers });
    const data: ApiResponse<T> = await res.json();
    if (!res.ok && !data.error) {
      data.error = data.message || `HTTP error ${res.status}`;
    }
    return data;
  } catch (err: any) {
    return {
      success: false,
      error: err.message || 'Network error connecting to backend API',
    };
  }
}
```

---

### Sample Call: Fetching Catalog
```typescript
const res = await apiRequest<Product[]>('/products?category_id=1&sort=price_asc');
if (res.success && res.data) {
  console.log('Products:', res.data);
  console.log('Total Count:', res.meta?.total_rows);
}
```

### Sample Call: Checking Out
```typescript
const checkoutRes = await apiRequest<Order>('/orders/checkout', {
  method: 'POST',
  body: JSON.stringify({
    items: [{ product_id: 1, quantity: 2 }],
    shipping_name: 'Jane Doe',
    shipping_phone: '+1-555-010-9988',
    shipping_address: '123 Main St, New York, NY',
    payment_method: 'CARD',
  }),
});
```
