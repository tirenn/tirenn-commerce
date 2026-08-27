import React, { createContext, useContext, useState, useEffect } from 'react';
import type { Product } from '../types';
import { useToast } from './ToastContext';

export interface CartItem {
  product: Product;
  quantity: number;
}

interface CartContextType {
  items: CartItem[];
  cartCount: number;
  cartTotal: number;
  addToCart: (product: Product, quantity?: number) => void;
  removeFromCart: (productId: number) => void;
  updateQuantity: (productId: number, quantity: number) => void;
  clearCart: () => void;
}

const CartContext = createContext<CartContextType | undefined>(undefined);

export const CartProvider: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const { showToast } = useToast();
  const [items, setItems] = useState<CartItem[]>(() => {
    try {
      const saved = localStorage.getItem('tirenn_cart');
      const parsed = saved ? JSON.parse(saved) : [];
      return Array.isArray(parsed) ? parsed.filter((i) => i && i.product && typeof i.product.price === 'number') : [];
    } catch {
      return [];
    }
  });

  useEffect(() => {
    try {
      localStorage.setItem('tirenn_cart', JSON.stringify(items));
    } catch (err) {
      console.error('Failed to persist cart', err);
    }
  }, [items]);

  const cartCount = Array.isArray(items)
    ? items.reduce((sum, item) => sum + (Number(item?.quantity) || 0), 0)
    : 0;

  const cartTotal = Array.isArray(items)
    ? items.reduce(
        (sum, item) => sum + (Number(item?.product?.price) || 0) * (Number(item?.quantity) || 0),
        0
      )
    : 0;

  const addToCart = (product: Product, quantity: number = 1) => {
    if (!product || typeof product.id === 'undefined') return;
    setItems((prev) => {
      const safePrev = Array.isArray(prev) ? prev : [];
      const existing = safePrev.find((item) => item.product?.id === product.id);
      if (existing) {
        const newQty = Math.min((existing.quantity || 0) + quantity, product.stock_quantity || 99);
        return safePrev.map((item) =>
          item.product?.id === product.id ? { ...item, quantity: newQty } : item
        );
      }
      return [...safePrev, { product, quantity: Math.min(quantity, product.stock_quantity || 99) }];
    });
    showToast(`Added ${quantity}x "${product.name}" to your cart`, 'success');
  };

  const removeFromCart = (productId: number) => {
    setItems((prev) => (Array.isArray(prev) ? prev.filter((item) => item.product?.id !== productId) : []));
    showToast('Item removed from cart', 'info');
  };

  const updateQuantity = (productId: number, quantity: number) => {
    if (quantity <= 0) {
      removeFromCart(productId);
      return;
    }
    setItems((prev) =>
      (Array.isArray(prev) ? prev : []).map((item) => {
        if (item.product?.id === productId) {
          const validQty = Math.min(quantity, item.product?.stock_quantity || 99);
          return { ...item, quantity: validQty };
        }
        return item;
      })
    );
  };

  const clearCart = () => {
    setItems([]);
  };

  return (
    <CartContext.Provider
      value={{ items: Array.isArray(items) ? items : [], cartCount, cartTotal, addToCart, removeFromCart, updateQuantity, clearCart }}
    >
      {children}
    </CartContext.Provider>
  );
};

export const useCart = () => {
  const context = useContext(CartContext);
  if (!context) {
    throw new Error('useCart must be used within a CartProvider');
  }
  return context;
};
