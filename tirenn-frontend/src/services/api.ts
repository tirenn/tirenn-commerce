import type { ApiResponse } from '../types';

function getApiBaseUrl(): string {
  if (import.meta.env.VITE_API_BASE_URL) {
    return import.meta.env.VITE_API_BASE_URL.replace(/\/+$/, '');
  }
  if (import.meta.env.VITE_BACKEND_URL) {
    const root = import.meta.env.VITE_BACKEND_URL.replace(/\/+$/, '');
    return `${root}/api/v1`;
  }
  return 'http://localhost:8080/api/v1';
}

function getAIServiceUrl(): string {
  if (import.meta.env.VITE_AI_SERVICE_URL) {
    return import.meta.env.VITE_AI_SERVICE_URL.replace(/\/+$/, '');
  }
  if (import.meta.env.VITE_AI_API_BASE_URL) {
    return import.meta.env.VITE_AI_API_BASE_URL.replace(/\/+$/, '');
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
