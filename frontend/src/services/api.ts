import type { ApiResponse, Product } from '../types';

function getApiBaseUrl(): string {
  if (typeof window !== 'undefined' && window.location.hostname) {
    const host = window.location.hostname;
    if (import.meta.env.VITE_API_BASE_URL) {
      return import.meta.env.VITE_API_BASE_URL.replace(/localhost|127\.0\.0\.1/, host).replace(/\/+$/, '');
    }
    return `http://${host}:8080/api/v1`;
  }
  return 'http://localhost:8080/api/v1';
}

function getAIServiceUrl(): string {
  if (typeof window !== 'undefined' && window.location.hostname) {
    const host = window.location.hostname;
    if (import.meta.env.VITE_AI_SERVICE_URL) {
      return import.meta.env.VITE_AI_SERVICE_URL.replace(/localhost|127\.0\.0\.1/, host).replace(/\/+$/, '');
    }
    return `http://${host}:8000/api/v1`;
  }
  return 'http://localhost:8000/api/v1';
}

export const API_BASE_URL = getApiBaseUrl();
export const AI_API_BASE_URL = getAIServiceUrl();

export function getAuthToken(): string | null {
  return localStorage.getItem('tirenn_token') || localStorage.getItem('gocommerce_token');
}

export function setAuthToken(token: string | null) {
  if (token) {
    localStorage.setItem('tirenn_token', token);
  } else {
    localStorage.removeItem('tirenn_token');
    localStorage.removeItem('gocommerce_token');
  }
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
    const res = await fetch(url, {
      ...options,
      headers,
    });

    const data: ApiResponse<T> = await res.json();
    if (!res.ok && !data.error) {
      data.error = data.message || `Request failed with status ${res.status}`;
    }
    return data;
  } catch (err: any) {
    return {
      success: false,
      error: err.message || 'Network error occurred. Please ensure backend is running.',
    };
  }
}

export async function getRecommendations(
  productId: number,
  limit: number = 6
): Promise<Product[]> {
  if (!productId) return [];
  try {
    const res = await apiRequest<Product[]>(`/products/${productId}/recommendations?limit=${limit}`);
    if (res.success && Array.isArray(res.data)) {
      return res.data;
    }
    return [];
  } catch (err) {
    console.error(`Failed to fetch recommendations for product ${productId}:`, err);
    return [];
  }
}

export async function searchSemanticAI(
  query: string,
  options: {
    limit?: number;
    categoryId?: number;
    scoreThreshold?: number;
    minPrice?: number;
    maxPrice?: number;
    inStock?: boolean;
  } = {}
): Promise<ApiResponse<Product[]>> {
  const url = `${AI_API_BASE_URL}/search/semantic`;
  try {
    const res = await fetch(url, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
        query,
        limit: options.limit ?? 12,
        category_id: options.categoryId ?? 0,
        score_threshold: options.scoreThreshold,
        min_price: options.minPrice,
        max_price: options.maxPrice,
        in_stock: options.inStock,
      }),
    });

    const data = await res.json();
    if (!res.ok) {
      return {
        success: false,
        error: data.detail || `AI Search failed with status ${res.status}`,
      };
    }
    return {
      success: true,
      data: data.data || [],
      meta: {
        total_rows: data.total_results || (data.data ? data.data.length : 0),
        total_page: 1,
        page: 1,
        limit: options.limit ?? 12,
      },
    };
  } catch (err: any) {
    return {
      success: false,
      error: err.message || 'Failed to connect to AI Service',
    };
  }
}

