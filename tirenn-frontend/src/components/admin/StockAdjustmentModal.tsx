import React, { useEffect, useState } from 'react';
import { apiRequest } from '../../services/api';
import { useToast } from '../../context/ToastContext';
import type { Product, StockAdjustmentLog } from '../../types';

interface StockAdjustmentModalProps {
  product: Product;
  onClose: () => void;
  onSuccess: () => void;
}

export const StockAdjustmentModal: React.FC<StockAdjustmentModalProps> = ({
  product,
  onClose,
  onSuccess,
}) => {
  const { showToast } = useToast();

  const [adjustmentType, setAdjustmentType] = useState<'ADD' | 'SUBTRACT' | 'SET'>('ADD');
  const [amount, setAmount] = useState(10);
  const [reason, setReason] = useState('Restock Shipment');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');

  const [logs, setLogs] = useState<StockAdjustmentLog[]>([]);
  const [loadingLogs, setLoadingLogs] = useState(true);

  const reasons = [
    'Restock Shipment',
    'Warehouse Inventory Count Correction',
    'Damaged Goods Write-off',
    'Returned Order Inventory Restock',
    'Promotion Allocation',
    'Other / Custom Audit',
  ];

  const loadLogs = async () => {
    setLoadingLogs(true);
    const res = await apiRequest<StockAdjustmentLog[]>(`/admin/products/${product.id}/stock-logs`);
    setLoadingLogs(false);
    if (res.success && res.data) {
      setLogs(res.data);
    }
  };

  useEffect(() => {
    loadLogs();
  }, [product.id]);

  const handleAdjustStock = async (e: React.FormEvent) => {
    e.preventDefault();
    if (amount <= 0 && adjustmentType !== 'SET') {
      setError('Amount must be greater than 0');
      return;
    }

    setError('');
    setLoading(true);

    const res = await apiRequest(`/admin/products/${product.id}/adjust-stock`, {
      method: 'POST',
      body: JSON.stringify({
        type: adjustmentType,
        amount: Number(amount),
        reason,
      }),
    });

    setLoading(false);

    if (res.success) {
      showToast('Inventory stock updated successfully!', 'success');
      onSuccess();
    } else {
      setError(res.error || 'Failed to adjust stock');
    }
  };

  return (
    <div className="fixed inset-0 bg-slate-900/50 backdrop-blur-xs z-50 flex items-center justify-center p-4">
      <div className="bg-white rounded-2xl w-full max-w-2xl p-6 sm:p-8 relative max-h-[90vh] overflow-y-auto shadow-2xl border border-slate-200 animate-modal">
        <button
          onClick={onClose}
          className="absolute top-4 right-4 text-slate-400 hover:text-slate-700 w-8 h-8 rounded-full bg-slate-100 flex items-center justify-center cursor-pointer transition-colors"
        >
          ✕
        </button>

        <div className="mb-5 pb-3 border-b border-slate-100">
          <h3 className="text-xl font-bold text-slate-900">Inventory Stock Controller</h3>
          <p className="text-xs text-slate-500 mt-0.5">Adjust warehouse counts with immutable audit logs</p>
        </div>

        {/* Product Summary */}
        <div className="bg-slate-50 border border-slate-200 p-3 rounded-xl flex items-center justify-between mb-5">
          <div className="flex items-center gap-3">
            <img
              src={product.image_url}
              alt={product.name}
              className="w-12 h-12 object-contain border border-slate-200 rounded-md bg-white p-1"
            />
            <div>
              <h4 className="font-bold text-xs text-slate-900">{product.name}</h4>
              <span className="text-[10px] text-slate-400 font-mono">SKU: {product.sku}</span>
            </div>
          </div>
          <div className="text-right">
            <span className="text-[10px] font-bold text-slate-400 uppercase block">CURRENT STOCK</span>
            <span className="font-extrabold text-lg text-slate-900">{product.stock_quantity} units</span>
          </div>
        </div>

        {error && (
          <div className="bg-rose-50 border border-rose-200 text-rose-700 p-3 rounded-lg text-xs font-medium mb-4">
            ⚠️ {error}
          </div>
        )}

        {/* Form */}
        <form onSubmit={handleAdjustStock} className="space-y-4 text-xs mb-6 pb-6 border-b border-slate-200">
          <div>
            <label className="font-semibold text-slate-700 block mb-1.5">Operation Type</label>
            <div className="grid grid-cols-3 gap-2">
              <button
                type="button"
                className={`py-2 font-semibold rounded-lg border transition-all cursor-pointer ${
                  adjustmentType === 'ADD'
                    ? 'bg-emerald-600 text-white border-emerald-600 shadow-xs'
                    : 'bg-slate-50 text-slate-700 border-slate-200 hover:bg-slate-100'
                }`}
                onClick={() => setAdjustmentType('ADD')}
              >
                + ADD UNITS
              </button>
              <button
                type="button"
                className={`py-2 font-semibold rounded-lg border transition-all cursor-pointer ${
                  adjustmentType === 'SUBTRACT'
                    ? 'bg-rose-600 text-white border-rose-600 shadow-xs'
                    : 'bg-slate-50 text-slate-700 border-slate-200 hover:bg-slate-100'
                }`}
                onClick={() => setAdjustmentType('SUBTRACT')}
              >
                - DEDUCT UNITS
              </button>
              <button
                type="button"
                className={`py-2 font-semibold rounded-lg border transition-all cursor-pointer ${
                  adjustmentType === 'SET'
                    ? 'bg-blue-600 text-white border-blue-600 shadow-xs'
                    : 'bg-slate-50 text-slate-700 border-slate-200 hover:bg-slate-100'
                }`}
                onClick={() => setAdjustmentType('SET')}
              >
                = SET EXACT
              </button>
            </div>
          </div>

          <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
            <div>
              <label className="font-semibold text-slate-700 block mb-1">Quantity</label>
              <input
                type="number"
                min="0"
                className="w-full text-xs font-sans bg-slate-50 border border-slate-200 rounded-lg px-3 py-2 text-slate-900 outline-none focus:border-blue-600 focus:bg-white"
                value={amount}
                onChange={(e) => setAmount(Number(e.target.value))}
                required
              />
            </div>
            <div>
              <label className="font-semibold text-slate-700 block mb-1">Audit Reason</label>
              <select
                className="w-full text-xs font-medium bg-slate-50 border border-slate-200 rounded-lg px-2.5 py-2 text-slate-800 outline-none focus:border-blue-600 cursor-pointer"
                value={reason}
                onChange={(e) => setReason(e.target.value)}
              >
                {reasons.map((r) => (
                  <option key={r} value={r}>
                    {r}
                  </option>
                ))}
              </select>
            </div>
          </div>

          <button
            type="submit"
            disabled={loading}
            className="w-full bg-blue-600 hover:bg-blue-700 text-white font-semibold text-xs py-2.5 rounded-lg shadow-sm cursor-pointer transition-colors"
          >
            {loading ? 'Submitting...' : 'Apply Stock Adjustment'}
          </button>
        </form>

        {/* Historical Logs */}
        <div>
          <h4 className="font-bold text-xs text-slate-900 mb-2 uppercase tracking-wider">Audit Log History</h4>
          {loadingLogs ? (
            <span className="text-xs text-slate-400">Loading audit trail...</span>
          ) : logs.length === 0 ? (
            <span className="text-xs text-slate-400">No stock adjustment logs recorded for this item yet.</span>
          ) : (
            <div className="space-y-1.5 max-h-36 overflow-y-auto pr-1">
              {logs.map((log) => (
                <div
                  key={log.id}
                  className="bg-slate-50 p-2.5 rounded-lg border border-slate-200/80 flex items-center justify-between text-[11px]"
                >
                  <div>
                    <span className="font-semibold text-slate-800">{log.reason}</span>
                    <span className="text-slate-400 block text-[10px]">
                      {new Date(log.created_at).toLocaleString()}
                    </span>
                  </div>
                  <div className="text-right">
                    <span className={`font-bold ${log.change_amount >= 0 ? 'text-emerald-600' : 'text-rose-600'}`}>
                      {log.change_amount >= 0 ? `+${log.change_amount}` : log.change_amount} units
                    </span>
                    <span className="text-[10px] text-slate-400 block font-mono">
                      {log.previous_stock} → {log.current_stock}
                    </span>
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
      </div>
    </div>
  );
};
