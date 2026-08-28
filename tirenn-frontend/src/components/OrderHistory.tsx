import React, { useState, useEffect } from 'react';
import { useTranslation } from 'react-i18next';
import { apiRequest } from '../services/api';
import { useCurrency } from '../context/CurrencyContext';
import type { Order, AppView } from '../types';

interface OrderHistoryProps {
  onSelectView: (view: AppView) => void;
}

export const OrderHistory: React.FC<OrderHistoryProps> = ({ onSelectView }) => {
  const { t } = useTranslation();
  const { formatPrice } = useCurrency();
  const [orders, setOrders] = useState<Order[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  const loadOrders = async () => {
    setLoading(true);
    setError('');

    const res = await apiRequest<Order[]>('/orders/my-orders');
    setLoading(false);

    if (res.success && Array.isArray(res.data)) {
      setOrders(res.data);
    } else {
      setError(res.error || 'Failed to load order history');
    }
  };

  useEffect(() => {
    loadOrders();
  }, []);

  const getStatusBadge = (status: string) => {
    switch (status) {
      case 'PAID':
      case 'COMPLETED':
        return 'bg-emerald-50 text-emerald-700 border-emerald-200';
      case 'SHIPPED':
      case 'PROCESSING':
        return 'bg-blue-50 text-blue-700 border-blue-200';
      case 'CANCELLED':
        return 'bg-rose-50 text-rose-700 border-rose-200';
      default:
        return 'bg-amber-50 text-amber-700 border-amber-200';
    }
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-xl font-bold text-slate-900">{t('orders.title')}</h2>
          <p className="text-xs text-slate-500">Track and review your previous purchases</p>
        </div>
        <button
          onClick={() => onSelectView('storefront')}
          className="text-xs font-semibold text-blue-600 hover:text-blue-700 bg-blue-50 px-3 py-1.5 rounded-lg border border-blue-100 transition-colors cursor-pointer"
        >
          ← {t('checkout.continue_shopping')}
        </button>
      </div>

      {loading ? (
        <div className="bg-white border border-slate-200 rounded-2xl p-16 text-center shadow-xs">
          <span className="text-3xl animate-spin block mb-3">⚡</span>
          <span className="font-medium text-sm text-slate-600">Retrieving order invoices...</span>
        </div>
      ) : error ? (
        <div className="bg-rose-50 border border-rose-200 text-rose-700 p-4 rounded-xl text-sm font-medium">
          ⚠️ {error}
        </div>
      ) : orders.length === 0 ? (
        <div className="bg-white border border-slate-200 rounded-2xl p-16 text-center space-y-3 shadow-xs">
          <span className="text-4xl block">📦</span>
          <h3 className="font-bold text-lg text-slate-900">{t('orders.empty')}</h3>
          <button
            onClick={() => onSelectView('storefront')}
            className="bg-blue-600 hover:bg-blue-700 text-white font-semibold text-xs py-2 px-4 rounded-lg shadow-xs cursor-pointer mt-2"
          >
            {t('checkout.continue_shopping')}
          </button>
        </div>
      ) : (
        <div className="space-y-4">
          {orders.map((order) => (
            <div
              key={order.id}
              className="bg-white border border-slate-200 rounded-xl p-5 shadow-xs transition-shadow hover:shadow-md"
            >
              {/* Order Header */}
              <div className="flex flex-wrap items-center justify-between gap-3 pb-3 border-b border-slate-100 text-xs">
                <div className="flex items-center gap-3">
                  <span className="font-bold text-slate-900 text-sm">
                    {t('orders.order_number')} #{order.order_number}
                  </span>
                  <span className={`px-2.5 py-0.5 rounded-full text-[10px] font-bold border ${getStatusBadge(order.status)}`}>
                    {order.status}
                  </span>
                </div>
                <span className="text-slate-400">
                  {new Date(order.created_at).toLocaleDateString(undefined, {
                    year: 'numeric',
                    month: 'short',
                    day: 'numeric',
                    hour: '2-digit',
                    minute: '2-digit',
                  })}
                </span>
              </div>

              {/* Order Items */}
              <div className="py-3 space-y-2">
                {order.items?.map((item) => (
                  <div key={item.id} className="flex items-center justify-between text-xs">
                    <div className="flex items-center gap-3">
                      {item.product_image && (
                        <img
                          src={item.product_image}
                          alt={item.product_name}
                          className="w-10 h-10 object-contain bg-slate-50 border border-slate-100 rounded-lg p-0.5"
                        />
                      )}
                      <div>
                        <span className="font-medium text-slate-800">{item.product_name}</span>
                        <span className="text-slate-400 block text-[11px]">
                          Qty: {item.quantity} × {formatPrice(item.unit_price)}
                        </span>
                      </div>
                    </div>
                    <span className="font-semibold text-slate-900">
                      {formatPrice(item.subtotal)}
                    </span>
                  </div>
                ))}
              </div>

              {/* Order Footer */}
              <div className="pt-3 border-t border-slate-100 flex flex-wrap items-center justify-between gap-2 text-xs">
                <div className="text-slate-500">
                  <span>Ship to: </span>
                  <b className="text-slate-700">{order.shipping_name}</b> ({order.shipping_address})
                </div>
                <div className="font-extrabold text-sm text-slate-900">
                  Total: {formatPrice(order.total_amount)}
                </div>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
};
