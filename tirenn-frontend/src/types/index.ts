export type UserRole = 'ADMIN' | 'CUSTOMER';

export interface User {
  id: number;
  name: string;
  email: string;
  role: UserRole;
  phone?: string;
  address?: string;
  status: 'ACTIVE' | 'SUSPENDED';
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
  status: 'PENDING' | 'PAID' | 'PROCESSING' | 'SHIPPED' | 'COMPLETED' | 'CANCELLED';
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

export interface StockAdjustmentLog {
  id: number;
  product_id: number;
  user_id: number;
  user?: User;
  change_amount: number;
  previous_stock: number;
  current_stock: number;
  reason: string;
  created_at: string;
}

export interface DashboardData {
  summary: {
    total_revenue: number;
    total_orders: number;
    total_customers: number;
    low_stock_count: number;
    pending_orders_count: number;
  };
  revenue_trends: Array<{
    date: string;
    revenue: number;
    order_count: number;
  }>;
  top_selling_products: Array<{
    product_id: number;
    product_name: string;
    product_sku: string;
    product_image: string;
    total_sold: number;
    total_revenue: number;
  }>;
  recent_orders: Array<{
    id: number;
    order_number: string;
    customer_name: string;
    total_amount: number;
    status: string;
    created_at: string;
  }>;
  low_stock_alerts: Array<{
    id: number;
    product_name: string;
    product_sku: string;
    stock_quantity: number;
    low_stock_threshold: number;
  }>;
}

export interface CustomerWithStats {
  id: number;
  name: string;
  email: string;
  phone?: string;
  address?: string;
  status: 'ACTIVE' | 'SUSPENDED';
  total_orders: number;
  total_spent: number;
  last_order_at?: string;
  created_at: string;
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

export interface AuthResponse {
  token: string;
  user: User;
}

export type AppView = 
  | 'storefront' 
  | 'my-orders' 
  | 'admin-dashboard' 
  | 'admin-products' 
  | 'admin-orders' 
  | 'admin-customers';
