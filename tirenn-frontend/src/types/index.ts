export type UserRole = 'ADMIN' | 'CUSTOMER';

export type AppView =
  | 'storefront'
  | 'my-orders'
  | 'admin-dashboard'
  | 'admin-products'
  | 'admin-orders'
  | 'admin-customers'
  | 'admin-knowledge';

export interface KnowledgeDocument {
  id: number;
  title: string;
  doc_type: 'SOP_CUSTOMER' | 'SOP_ADMIN' | 'POLICY' | 'GENERAL' | string;
  filename: string;
  total_pages: number;
  total_chunks: number;
  created_at: string;
}

export interface KnowledgeChunkResult {
  chunk_id: number;
  document_id: number;
  document_title: string;
  doc_type: string;
  filename: string;
  chunk_index: number;
  page_number: number;
  content: string;
  score: number;
}

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

export interface AuthResponse {
  user: User;
  token: string;
}

export interface CustomerWithStats extends User {
  total_orders: number;
  total_spent: number;
  last_order_at?: string;
}

export interface SubCategory {
  id: number;
  category_id: number;
  name: string;
  slug: string;
  description?: string;
  icon?: string;
  created_at?: string;
}

export interface Category {
  id: number;
  name: string;
  slug: string;
  description: string;
  icon?: string;
  sub_categories?: SubCategory[];
  created_at: string;
}

export interface Product {
  id: number;
  category_id: number;
  category?: Category;
  sub_category_id?: number;
  sub_category?: SubCategory;
  name: string;
  slug: string;
  sku: string;
  description: string;
  price: number;
  currency?: string;
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
  currency?: string;
}

export interface Order {
  id: number;
  order_number: string;
  user_id: number;
  user?: User;
  total_amount: number;
  currency?: string;
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
    units_sold?: number;
    total_sold?: number;
    revenue?: number;
    total_revenue?: number;
  }>;
}

export interface ApiResponse<T> {
  success: boolean;
  message?: string;
  data?: T;
  meta?: {
    page: number;
    limit: number;
    total_rows: number;
    total_page?: number;
    total_pages?: number;
  };
  error?: string;
}

export interface ProductFilters {
  category_id?: number;
  sub_category_id?: number;
  search?: string;
  sort?: string;
  min_price?: number;
  max_price?: number;
  in_stock?: boolean;
  page?: number;
  limit?: number;
}
