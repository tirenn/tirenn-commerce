import React, { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { useCart } from '../context/CartContext';
import { useAuth } from '../context/AuthContext';
import { useCurrency } from '../context/CurrencyContext';
import { useToast } from '../context/ToastContext';
import { apiRequest } from '../services/api';
import type { Order, AppView } from '../types';

interface CheckoutModalProps {
  isOpen: boolean;
  onClose: () => void;
  onAuthRequired: () => void;
  onSelectView: (view: AppView) => void;
}

export const CheckoutModal: React.FC<CheckoutModalProps> = ({
  isOpen,
  onClose,
  onAuthRequired,
  onSelectView,
}) => {
  const { t, i18n } = useTranslation();
  const { items, cartTotal, clearCart } = useCart();
  const { currentUser } = useAuth();
  const { formatPrice, currency } = useCurrency();
  const { showToast } = useToast();

  const [shippingName, setShippingName] = useState(currentUser?.name || 'Budi Santoso');
  const [shippingPhone, setShippingPhone] = useState(currentUser?.phone || '+62 812-3456-7890');
  const [shippingAddress, setShippingAddress] = useState(
    currentUser?.address || 'Jl. Sudirman No. 45, Jakarta Pusat'
  );
  const [paymentMethod, setPaymentMethod] = useState('CARD');
  const [notes, setNotes] = useState('');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');

  if (!isOpen) return null;

  const handlePlaceOrder = async (e: React.FormEvent) => {
    e.preventDefault();

    if (!currentUser) {
      onClose();
      onAuthRequired();
      showToast(i18n.language === 'en' ? 'Please sign in to place your order' : 'Silakan masuk untuk menyelesaikan pesanan', 'warning');
      return;
    }

    if (items.length === 0) {
      setError(t('cart.empty_title'));
      return;
    }

    setError('');
    setLoading(true);

    const payload = {
      items: items.map((i) => ({
        product_id: i.product.id,
        quantity: i.quantity,
      })),
      shipping_name: shippingName,
      shipping_phone: shippingPhone,
      shipping_address: shippingAddress,
      payment_method: paymentMethod,
      currency: currency,
      notes,
    };

    const res = await apiRequest<Order>('/orders/checkout', {
      method: 'POST',
      body: JSON.stringify(payload),
    });

    setLoading(false);

    if (res.success && res.data) {
      clearCart();
      showToast(t('checkout.success_title'), 'success');
      onClose();
      onSelectView('my-orders');
    } else {
      setError(res.error || 'Failed to place order. Please check stock availability.');
    }
  };

  return (
    <div className="fixed inset-0 bg-slate-900/50 backdrop-blur-xs z-50 flex items-center justify-center p-4">
      <div data-testid="checkout-modal" className="bg-white rounded-2xl w-full max-w-xl p-6 sm:p-8 relative max-h-[90vh] overflow-y-auto shadow-xl border border-slate-200 animate-modal">
        {/* Close Button */}
        <button
          data-testid="checkout-close"
          onClick={onClose}
          className="absolute top-4 right-4 text-slate-400 hover:text-slate-700 w-8 h-8 rounded-full bg-slate-100 flex items-center justify-center cursor-pointer transition-colors"
        >
          ✕
        </button>

        <h2 className="text-xl font-bold text-slate-900 mb-1">{t('checkout.title')}</h2>
        <p className="text-xs text-slate-500 mb-4 pb-3 border-b border-slate-100">
          {t('checkout.shipping_info')}
        </p>

        {error && (
          <div className="bg-rose-50 text-rose-700 p-3 rounded-lg border border-rose-200 text-xs font-medium mb-4">
            ⚠️ {error}
          </div>
        )}

        <form onSubmit={handlePlaceOrder} className="space-y-3.5 text-xs">
          <div>
            <label className="font-medium text-slate-700 block mb-1">{t('checkout.full_name')}</label>
            <input
              type="text"
              data-testid="checkout-name"
              className="w-full bg-slate-50 border border-slate-200 rounded-lg px-3 py-2 text-slate-900 outline-none focus:border-blue-600 focus:bg-white"
              value={shippingName}
              onChange={(e) => setShippingName(e.target.value)}
              required
            />
          </div>

          <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
            <div>
              <label className="font-medium text-slate-700 block mb-1">{t('checkout.phone')}</label>
              <input
                type="tel"
                data-testid="checkout-phone"
                className="w-full bg-slate-50 border border-slate-200 rounded-lg px-3 py-2 text-slate-900 outline-none focus:border-blue-600 focus:bg-white"
                value={shippingPhone}
                onChange={(e) => setShippingPhone(e.target.value)}
                required
              />
            </div>
            <div>
              <label className="font-medium text-slate-700 block mb-1">{t('checkout.payment_method')}</label>
              <select
                data-testid="checkout-payment-method"
                className="w-full bg-slate-50 border border-slate-200 rounded-lg px-2.5 py-2 text-slate-800 outline-none focus:border-blue-600 cursor-pointer"
                value={paymentMethod}
                onChange={(e) => setPaymentMethod(e.target.value)}
              >
                <option value="CARD">{t('checkout.simulated_card')}</option>
                <option value="QRIS">{t('checkout.qris')}</option>
                <option value="BANK_TRANSFER">{t('checkout.bank_transfer')}</option>
              </select>
            </div>
          </div>

          <div>
            <label className="font-medium text-slate-700 block mb-1">{t('checkout.address')}</label>
            <textarea
              data-testid="checkout-address"
              className="w-full h-16 resize-none bg-slate-50 border border-slate-200 rounded-lg px-3 py-2 text-slate-900 outline-none focus:border-blue-600 focus:bg-white"
              value={shippingAddress}
              onChange={(e) => setShippingAddress(e.target.value)}
              required
            />
          </div>

          <div>
            <label className="font-medium text-slate-700 block mb-1">{t('checkout.notes')}</label>
            <input
              type="text"
              data-testid="checkout-notes"
              className="w-full bg-slate-50 border border-slate-200 rounded-lg px-3 py-2 text-slate-900 outline-none focus:border-blue-600 focus:bg-white"
              value={notes}
              onChange={(e) => setNotes(e.target.value)}
              placeholder="e.g. Tolong titipkan paket ke satpam"
            />
          </div>

          <div className="pt-3 border-t border-slate-100 flex items-center justify-between">
            <span className="text-sm font-bold text-slate-900">Total: {formatPrice(cartTotal)}</span>
            <button
              type="submit"
              data-testid="checkout-submit"
              disabled={loading}
              className="bg-blue-600 hover:bg-blue-700 text-white font-semibold text-xs py-2.5 px-5 rounded-lg shadow-sm cursor-pointer transition-colors"
            >
              {loading ? t('checkout.processing') : `${t('checkout.pay_now')} (${formatPrice(cartTotal)})`}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
};
