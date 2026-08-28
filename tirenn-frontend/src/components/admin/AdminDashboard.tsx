import React, { useState, useEffect } from 'react';
import { apiRequest } from '../../services/api';
import { useCurrency } from '../../context/CurrencyContext';
import type { DashboardData, AppView } from '../../types';

interface AdminDashboardProps {
  onSelectView: (view: AppView) => void;
}

export const AdminDashboard: React.FC<AdminDashboardProps> = ({ onSelectView }) => {
  const { formatPrice } = useCurrency();
  const [data, setData] = useState<DashboardData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  const loadDashboard = async () => {
    setLoading(true);
    setError('');

    const res = await apiRequest<DashboardData>('/admin/dashboard?days=7');
    setLoading(false);

    if (res.success && res.data) {
      setData(res.data);
    } else {
      setError(res.error || 'Failed to load executive dashboard data');
    }
  };

  useEffect(() => {
    loadDashboard();
  }, []);

  if (loading) {
    return (
      <div className="bg-white border border-slate-200 rounded-2xl p-16 text-center shadow-xs">
        <span className="text-3xl animate-spin block mb-3">⚡</span>
        <span className="font-medium text-sm text-slate-600">Calculating business analytics...</span>
      </div>
    );
  }

  if (error || !data) {
    return (
      <div className="bg-rose-50 border border-rose-200 text-rose-700 p-6 rounded-2xl text-sm font-medium">
        ⚠️ {error || 'Error loading dashboard'}
      </div>
    );
  }

  const maxDailyRevenue = Math.max(...(data.revenue_trends?.map((t) => t.revenue) || [1]), 1);

  return (
    <div className="space-y-6">
      {/* Top Banner */}
      <div className="flex flex-wrap items-center justify-between gap-4">
        <div>
          <h2 className="text-xl sm:text-2xl font-black text-slate-900 tracking-tight">
            Merchant Executive Dashboard
          </h2>
          <p className="text-xs text-slate-500">Live operational metrics and store performance</p>
        </div>
        <div className="flex items-center gap-2">
          <button
            onClick={() => onSelectView('admin-products')}
            className="bg-blue-600 hover:bg-blue-700 text-white font-semibold text-xs py-2 px-3.5 rounded-lg shadow-xs transition-colors cursor-pointer"
          >
            Manage Products
          </button>
          <button
            onClick={() => onSelectView('admin-orders')}
            className="bg-slate-900 hover:bg-black text-white font-semibold text-xs py-2 px-3.5 rounded-lg shadow-xs transition-colors cursor-pointer"
          >
            Fulfillment Queue
          </button>
        </div>
      </div>

      {/* KPI Cards */}
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
        {/* Total Revenue */}
        <div className="bg-white border border-slate-200 rounded-xl p-5 shadow-xs">
          <span className="text-xs font-bold text-slate-400 uppercase tracking-wider block mb-1">
            TOTAL REVENUE
          </span>
          <div className="text-2xl font-black text-slate-900">
            {formatPrice(data.summary?.total_revenue || 0)}
          </div>
          <span className="text-[11px] text-emerald-600 font-semibold mt-1 block">
            ↑ 14.2% vs previous period
          </span>
        </div>

        {/* Total Orders */}
        <div className="bg-white border border-slate-200 rounded-xl p-5 shadow-xs">
          <span className="text-xs font-bold text-slate-400 uppercase tracking-wider block mb-1">
            TOTAL ORDERS
          </span>
          <div className="text-2xl font-black text-slate-900">
            {data.summary?.total_orders || 0}
          </div>
          <span className="text-[11px] text-blue-600 font-semibold mt-1 block">
            {data.summary?.pending_orders_count || 0} pending fulfillment
          </span>
        </div>

        {/* Registered Customers */}
        <div className="bg-white border border-slate-200 rounded-xl p-5 shadow-xs">
          <span className="text-xs font-bold text-slate-400 uppercase tracking-wider block mb-1">
            ACTIVE CUSTOMERS
          </span>
          <div className="text-2xl font-black text-slate-900">
            {data.summary?.total_customers || 0}
          </div>
          <span className="text-[11px] text-purple-600 font-semibold mt-1 block">
            Verified accounts
          </span>
        </div>

        {/* Low Stock Alerts */}
        <div className="bg-white border border-slate-200 rounded-xl p-5 shadow-xs">
          <span className="text-xs font-bold text-slate-400 uppercase tracking-wider block mb-1">
            LOW STOCK RADAR
          </span>
          <div className="text-2xl font-black text-slate-900">
            {data.summary?.low_stock_count || 0}
          </div>
          <span className={`text-[11px] font-semibold mt-1 block ${
            (data.summary?.low_stock_count || 0) > 0 ? 'text-amber-600' : 'text-emerald-600'
          }`}>
            {(data.summary?.low_stock_count || 0) > 0 ? 'Requires restock shipment' : 'All SKUs healthy'}
          </span>
        </div>
      </div>

      {/* Grid: 7-Day Velocity & Top Products */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* 7-Day Revenue Velocity Chart */}
        <div className="lg:col-span-2 bg-white border border-slate-200 rounded-xl p-5 sm:p-6 shadow-xs">
          <div className="flex items-center justify-between mb-6">
            <div>
              <h3 className="font-bold text-sm text-slate-900">7-Day Sales Velocity</h3>
              <p className="text-xs text-slate-500">Daily gross turnover and completed orders</p>
            </div>
            <span className="text-xs font-bold text-slate-400 bg-slate-100 px-2.5 py-1 rounded-md">
              Past 7 Days
            </span>
          </div>

          <div className="h-48 flex items-end gap-2 sm:gap-4 pt-4 border-b border-slate-100">
            {data.revenue_trends?.map((item, idx) => {
              const heightPercent = Math.max(Math.round((item.revenue / maxDailyRevenue) * 100), 10);
              return (
                <div key={idx} className="flex-1 flex flex-col items-center gap-1.5 h-full justify-end group">
                  <div className="text-[10px] font-bold text-slate-600 opacity-0 group-hover:opacity-100 transition-opacity">
                    {formatPrice(item.revenue)}
                  </div>
                  <div
                    style={{ height: `${heightPercent}%` }}
                    className="w-full bg-blue-600 hover:bg-blue-700 rounded-t-md transition-all duration-300 relative cursor-pointer"
                  />
                  <span className="text-[10px] text-slate-400 truncate max-w-full">
                    {item.date?.slice(5) || `D${idx + 1}`}
                  </span>
                </div>
              );
            })}
          </div>
        </div>

        {/* Top Selling Products */}
        <div className="bg-white border border-slate-200 rounded-xl p-5 sm:p-6 shadow-xs flex flex-col justify-between">
          <div>
            <h3 className="font-bold text-sm text-slate-900 mb-1">Top Selling SKUs</h3>
            <p className="text-xs text-slate-500 mb-4">Highest grossing catalog items</p>

            <div className="space-y-3">
              {data.top_selling_products?.length === 0 ? (
                <p className="text-xs text-slate-400 py-6 text-center">No sales recorded yet</p>
              ) : (
                data.top_selling_products?.slice(0, 4).map((p: any, idx) => (
                  <div key={idx} className="flex items-center justify-between text-xs pb-2 border-b border-slate-50">
                    <div className="flex items-center gap-2.5 min-w-0">
                      <span className="w-5 h-5 rounded-full bg-slate-100 text-slate-600 font-bold flex items-center justify-center text-[10px]">
                        {idx + 1}
                      </span>
                      <div className="truncate">
                        <h4 className="font-semibold text-slate-900 truncate">{p.product_name}</h4>
                        <span className="text-[10px] text-slate-400">{p.units_sold || p.total_sold} units sold</span>
                      </div>
                    </div>
                    <span className="font-bold text-slate-900 whitespace-nowrap">
                      {formatPrice(p.revenue || p.total_revenue)}
                    </span>
                  </div>
                ))
              )}
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};
