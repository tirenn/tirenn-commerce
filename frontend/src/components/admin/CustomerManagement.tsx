import React, { useState, useEffect } from 'react';
import { apiRequest } from '../../services/api';
import { useToast } from '../../context/ToastContext';
import { formatRupiah } from '../../utils/format';
import type { CustomerWithStats } from '../../types';

export const CustomerManagement: React.FC = () => {
  const { showToast } = useToast();
  const [customers, setCustomers] = useState<CustomerWithStats[]>([]);
  const [loading, setLoading] = useState(true);
  const [search, setSearch] = useState('');

  const loadCustomers = async () => {
    setLoading(true);
    const res = await apiRequest<CustomerWithStats[]>(`/admin/customers?search=${search}`);
    setLoading(false);

    if (res.success && Array.isArray(res.data)) {
      setCustomers(res.data);
    }
  };

  useEffect(() => {
    loadCustomers();
  }, [search]);

  const handleToggleStatus = async (customerId: number, currentStatus: string) => {
    const newStatus = currentStatus === 'ACTIVE' ? 'SUSPENDED' : 'ACTIVE';
    const res = await apiRequest(`/admin/customers/${customerId}/status`, {
      method: 'PATCH',
      body: JSON.stringify({ status: newStatus }),
    });

    if (res.success) {
      showToast(`Customer account ${newStatus.toLowerCase()}`, 'info');
      loadCustomers();
    } else {
      showToast(res.error || 'Failed to update customer status', 'error');
    }
  };

  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-xl font-bold text-slate-900">Customer CRM Directory</h2>
        <p className="text-xs text-slate-500">Manage registered shopper accounts and lifetime metrics</p>
      </div>

      <div className="max-w-sm">
        <input
          type="text"
          placeholder="Search by customer name or email..."
          className="w-full bg-white border border-slate-200 rounded-lg px-3 py-2 text-xs text-slate-900 outline-none"
          value={search}
          onChange={(e) => setSearch(e.target.value)}
        />
      </div>

      <div className="bg-white border border-slate-200 rounded-xl overflow-hidden shadow-xs">
        <div className="overflow-x-auto">
          <table className="w-full text-left text-xs text-slate-600">
            <thead className="bg-slate-50 border-b border-slate-200 text-[11px] uppercase font-bold text-slate-400">
              <tr>
                <th className="p-3.5">Customer Profile</th>
                <th className="p-3.5">Contact Info</th>
                <th className="p-3.5">Total Orders</th>
                <th className="p-3.5">Lifetime Spend</th>
                <th className="p-3.5">Account Status</th>
                <th className="p-3.5 text-right">Actions</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-100">
              {loading ? (
                <tr>
                  <td colSpan={6} className="text-center p-8">
                    Loading customer accounts...
                  </td>
                </tr>
              ) : customers.length === 0 ? (
                <tr>
                  <td colSpan={6} className="text-center p-8 text-slate-400">
                    No customers found
                  </td>
                </tr>
              ) : (
                customers.map((c) => (
                  <tr key={c.id} className="hover:bg-slate-50/80 transition-colors">
                    <td className="p-3.5">
                      <div className="font-semibold text-slate-900">{c.name}</div>
                      <div className="text-[10px] text-slate-400">Joined {new Date(c.created_at).toLocaleDateString()}</div>
                    </td>
                    <td className="p-3.5">
                      <div>{c.email}</div>
                      <div className="text-[10px] text-slate-400">{c.phone || 'No phone'}</div>
                    </td>
                    <td className="p-3.5 font-bold text-slate-800">{c.total_orders} orders</td>
                    <td className="p-3.5 font-extrabold text-slate-900">
                      {formatRupiah(c.total_spent)}
                    </td>
                    <td className="p-3.5">
                      <span
                        className={`px-2 py-0.5 rounded-full text-[10px] font-bold ${
                          c.status === 'ACTIVE'
                            ? 'bg-emerald-50 text-emerald-700 border border-emerald-200'
                            : 'bg-rose-50 text-rose-700 border border-rose-200'
                        }`}
                      >
                        {c.status}
                      </span>
                    </td>
                    <td className="p-3.5 text-right">
                      <button
                        onClick={() => handleToggleStatus(c.id, c.status)}
                        className={`px-2.5 py-1 rounded text-[11px] font-semibold border cursor-pointer ${
                          c.status === 'ACTIVE'
                            ? 'bg-rose-50 text-rose-700 border-rose-200 hover:bg-rose-100'
                            : 'bg-emerald-50 text-emerald-700 border-emerald-200 hover:bg-emerald-100'
                        }`}
                      >
                        {c.status === 'ACTIVE' ? 'Suspend Account' : 'Reactivate'}
                      </button>
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
