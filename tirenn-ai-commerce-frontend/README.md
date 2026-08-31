# 🎨 Tirenn Commerce - Frontend Microservice

Modern, ultra-fast E-Commerce Storefront & Executive Admin Panel built with **React 19**, **Vite 6**, **TypeScript**, and **Tailwind CSS**.

---

## 🌟 Key Features

1. **🛍️ Public Storefront**:
   - Dynamic product catalog with infinite scroll and department category filters.
   - Real-time instant search input.
   - Interactive Product Detail Modal (PDP) with real-time stock indicator and IDR currency formatting.
   - Interactive Cart Drawer with quantity adjusters and instant subtotal calculation.
   - Checkout Modal with instant order validation.
2. **🤖 Embedded AI Shopper Modal**:
   - Direct connection to **Tirenn AI Service** (`http://localhost:8000/api/v1/chat/shopper`).
   - Guest-accessible consultation and product discovery.
   - Authentication-gated `add_to_cart` tool integration with automatic CartContext sync.
   - Visual tool call badges and product recommendation carousels.
3. **📊 Executive Admin Control Panel**:
   - Executive Dashboard with KPI cards, revenue metrics in Rupiah, and sales analytics.
   - Inventory Management: Stock adjustment modal (ADD, SUBTRACT, SET) with audit logs.
   - Order Management & CRM directory.
4. **🔐 Authentication & Security**:
   - 1-Click Shopper & Admin Demo logins.
   - Client-side JWT persistence and automatic profile refresh.

---

## 🛠️ Tech Stack

- **Framework**: React 19 + TypeScript
- **Bundler**: Vite 6 (ESBuild)
- **Styling**: Tailwind CSS 4
- **State Management**: React Context API (`CartContext`, `AuthContext`, `ToastContext`)
- **Web Server / Production Container**: Nginx Alpine

---

## 🚀 Getting Started

### 1. Environment Configuration
Copy `.env.example` to `.env`:
```bash
VITE_API_BASE_URL=http://localhost:8080/api/v1
VITE_AI_API_BASE_URL=http://localhost:8000/api/v1
```

### 2. Local Development
```bash
# Install dependencies
npm install

# Run development server with HMR
make dev
# -> Accessible at http://localhost:3000
```

### 3. Build & Preview
```bash
# Compile and build production bundle
make build

# Preview build locally
make preview
```

### 4. Docker Container
```bash
# Build and run standalone Nginx container
make docker-up

# Stop container
make docker-down
```
