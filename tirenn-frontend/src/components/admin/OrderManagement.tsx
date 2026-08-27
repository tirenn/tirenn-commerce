import React, { useState, useEffect } from 'react';
import { apiRequest } from '../../services/api';
import { useToast } from '../../context/ToastContext';
import { formatRupiah } from '../../utils/format';
import type { Order } from '../../types';

export const OrderManagement: React.FC = () => {
  const { showToast } = useToast();
  const [orders, setOrders] = useState<Order[]>([]);
  const [loading, setLoading] = useState(true);
  const [statusFilter, setStatusFilter] = useState('');
  const [search, setSearch] = useState('');

  const loadOrders = async () => {
    setLoading(true);
    const params = new URLSearchParams();
    if (statusFilter) params.append('status', statusFilter);
    if (search) params.append('search', search);

    const res = await apiRequest<Order[]>(`/admin/orders?${params.toString()}`);
    setLoading(false);

    if (res.success && Array.isArray(res.data)) {
      setOrders(res.data);
    }
  };

  useEffect(() => {
    loadOrders();
  }, [statusFilter, search]);

  const handleUpdateStatus = async (orderId: number, newStatus: string) => {
    const res = await apiRequest(`/admin/orders/${orderId}/status`, {
      method: 'PATCH',
      body: JSON.stringify({ status: newStatus }),
    });

    if (res.success) {
      showToast(`Order status updated to ${newStatus}`, 'success');
      loadOrders();
    } else {
      showToast(res.error || 'Failed to update order status', 'error');
    }
  };

  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-xl font-bold text-slate-900">Order Fulfillment & Tracking</h2>
        <p className="text-xs text-slate-500">Manage customer order workflows and fulfillment states</p>
      </div>

      {/* Filter Bar */}
      <div className="flex flex-wrap items-center gap-3">
        <input
          type="text"
          placeholder="Search by order # or customer..."
          className="bg-white border border-slate-200 rounded-lg px-3 py-1.5 text-xs text-slate-900 outline-none w-64"
          value={search}
          onChange={(e) => setSearch(e.target.value)}
        />

        <div className="flex items-center gap-1.5">
          {['', 'PAID', 'PROCESSING', 'SHIPPED', 'COMPLETED', 'CANCELLED'].map((st) => (
            <button
              key={st}
              onClick={() => setStatusFilter(st)}
              className={`px-3 py-1 rounded-lg text-xs font-semibold cursor-pointer transition-colors ${
                statusFilter === st
                  ? 'bg-slate-900 text-white'
                  : 'bg-white border border-slate-200 text-slate-600 hover:bg-slate-50'
              }`}
            >
              {st || 'All Orders'}
            </button>
          ))}
        </div>
      </div>

      {/* Orders Table */}
      <div className="bg-white border border-slate-200 rounded-xl overflow-hidden shadow-xs">
        <div className="overflow-x-auto">
          <table className="w-full text-left text-xs text-slate-600">
            <thead className="bg-slate-50 border-b border-slate-200 text-[11px] uppercase font-bold text-slate-400">
              <tr>
                <th className="p-3.5">Order #</th>
                <th className="p-3.5">Customer & Shipping</th>
                <th className="p-3.5">Total Amount</th>
                <th className="p-3.5">Payment</th>
                <th className="p-3.5">Fulfillment Status</th>
                <th className="p-3.5 text-right">Update Status</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-100">
              {loading ? (
                <tr>
                  <td colSpan={6} className="text-center p-8">
                    Loading orders queue...
                  </td>
                </tr>
              ) : orders.length === 0 ? (
                <tr>
                  <td colSpan={6} className="text-center p-8 text-slate-400">
                    No matching orders
                  </td>
                </tr>
              ) : (
                orders.map((o) => (
                  <tr key={o.id} className="hover:bg-slate-50/80 transition-colors">
                    <td className="p-3.5 font-bold font-mono text-slate-900">
                      #{o.order_number}
                    </td>
                    <td className="p-3.5">
                      <div className="font-semibold text-slate-900">{o.shipping_name}</div>
                      <div className="text-[10px] text-slate-400 truncate max-w-xs">{o.shipping_address}</div>
                    </td>
                    <td className="p-3.5 font-bold text-slate-900">
                      {formatRupiah(o.total_amount)}
                    </td>
                    <td className="p-3.5">
                      <span className="font-medium text-slate-700">{o.payment_method}</span>
                      <span className="block text-[10px] text-emerald-600 font-bold">{o.payment_status}</span>
                    </td>
                    <td className="p-3.5">
                      <span className="bg-slate-100 text-slate-800 font-bold px-2 py-0.5 rounded text-[10px] border border-slate-200">
                        {o.status}
                      </span>
                    </td>
                    <td className="p-3.5 text-right">
                      <select
                        className="bg-slate-50 border border-slate-200 rounded px-2 py-1 text-[11px] font-semibold text-slate-800 outline-none cursor-pointer"
                        value={o.status}
                        onChange={(e) => handleUpdateStatus(o.id, e.target.value)}
                      >
                        <option value="PAID">PAID</option>
                        <option value="PROCESSING">PROCESSING</option>
                        <option value="SHIPPED">SHIPPED</option>
                        <option value="COMPLETED">COMPLETED</option>
                        <option value="CANCELLED">CANCELLED (Restock)</option>
                      </select>
                    </td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
};
