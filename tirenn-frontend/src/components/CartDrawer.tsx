import React, { useState, useEffect } from 'react';
import { useTranslation } from 'react-i18next';
import { useCart } from '../context/CartContext';
import { useCurrency } from '../context/CurrencyContext';
import { getRecommendations } from '../services/api';
import type { Product } from '../types';

interface CartDrawerProps {
  isOpen: boolean;
  onClose: () => void;
  onCheckout: () => void;
}

export const CartDrawer: React.FC<CartDrawerProps> = ({ isOpen, onClose, onCheckout }) => {
  const { t, i18n } = useTranslation();
  const { items, cartTotal, addToCart, updateQuantity, removeFromCart, clearCart } = useCart();
  const { formatPrice } = useCurrency();

  const [recommendations, setRecommendations] = useState<Product[]>([]);
  const [loadingRecs, setLoadingRecs] = useState(false);

  // Target product is the last added item in the cart
  const lastItem = items.length > 0 ? items[items.length - 1] : null;
  const lastProductId = lastItem?.product?.id;

  useEffect(() => {
    if (!isOpen || !lastProductId) {
      setRecommendations([]);
      return;
    }

    let isMounted = true;
    setLoadingRecs(true);
    getRecommendations(lastProductId, 4)
      .then((data) => {
        if (isMounted) {
          setRecommendations(data || []);
        }
      })
      .catch((err) => {
        console.error('Failed to load cart recommendations', err);
        if (isMounted) setRecommendations([]);
      })
      .finally(() => {
        if (isMounted) setLoadingRecs(false);
      });

    return () => {
      isMounted = false;
    };
  }, [isOpen, lastProductId]);

  if (!isOpen) return null;

  // Filter out products already present in the cart
  const cartProductIds = new Set(items.map((item) => item.product?.id));
  const filteredRecommendations = recommendations.filter(
    (rec) => rec && !cartProductIds.has(rec.id)
  );

  return (
    <div className="fixed inset-0 z-50 overflow-hidden">
      {/* Backdrop */}
      <div
        className="absolute inset-0 bg-slate-900/50 backdrop-blur-xs transition-opacity"
        onClick={onClose}
      />

      <div className="fixed inset-y-0 right-0 max-w-full flex pl-10">
        <div
          data-testid="cart-drawer"
          className="w-screen max-w-md bg-white border-l border-slate-200 flex flex-col justify-between p-6 shadow-2xl relative animate-modal"
        >
          {/* Header */}
          <div className="pb-4 border-b border-slate-100 flex items-center justify-between">
            <h2 className="font-bold text-lg text-slate-900">
              {t('cart.title')} ({items.length})
            </h2>
            <button
              data-testid="cart-drawer-close"
              onClick={onClose}
              className="text-slate-400 hover:text-slate-700 w-8 h-8 rounded-full bg-slate-100 flex items-center justify-center cursor-pointer transition-colors"
            >
              ✕
            </button>
          </div>

          {/* Items & Contextual Recommendations */}
          <div className="flex-1 overflow-y-auto py-4 space-y-3">
            {items.length === 0 ? (
              <div className="h-full flex flex-col items-center justify-center text-center p-6 space-y-2">
                <span className="text-3xl text-slate-300">🛒</span>
                <h3 className="font-semibold text-sm text-slate-800">{t('cart.empty_title')}</h3>
                <p className="text-xs text-slate-500">{t('cart.empty_desc')}</p>
              </div>
            ) : (
              <>
                {items.map((item) => (
                  <div
                    key={item.product.id}
                    data-testid={`cart-item-${item.product.id}`}
                    className="bg-slate-50 border border-slate-200/80 rounded-xl p-3 flex gap-3 items-center"
                  >
                    <img
                      src={
                        item.product.image_url ||
                        'https://images.unsplash.com/photo-1505740420928-5e560c06d30e?w=400&q=80'
                      }
                      alt={item.product.name}
                      className="w-14 h-14 object-contain bg-white border border-slate-200 rounded-lg p-1"
                    />

                    <div className="flex-1 min-w-0">
                      <h4 className="font-semibold text-xs text-slate-900 truncate">
                        {item.product.name}
                      </h4>
                      <span className="text-xs text-slate-500 font-medium">
                        {formatPrice(item.product.price, item.product.currency)}
                      </span>

                      <div className="flex items-center gap-2 mt-1.5">
                        <button
                          data-testid={`cart-decrement-${item.product.id}`}
                          onClick={() => updateQuantity(item.product.id, item.quantity - 1)}
                          className="w-5 h-5 bg-white hover:bg-slate-200 border border-slate-200 rounded text-xs font-bold flex items-center justify-center cursor-pointer"
                        >
                          -
                        </button>
                        <span
                          data-testid={`cart-quantity-${item.product.id}`}
                          className="text-xs font-semibold px-1 text-slate-900"
                        >
                          {item.quantity}
                        </span>
                        <button
                          data-testid={`cart-increment-${item.product.id}`}
                          onClick={() => updateQuantity(item.product.id, item.quantity + 1)}
                          className="w-5 h-5 bg-white hover:bg-slate-200 border border-slate-200 rounded text-xs font-bold flex items-center justify-center cursor-pointer"
                        >
                          +
                        </button>
                      </div>
                    </div>

                    <div className="text-right flex flex-col justify-between items-end self-stretch">
                      <span className="font-bold text-xs text-slate-900">
                        {formatPrice(item.product.price * item.quantity, item.product.currency)}
                      </span>
                      <button
                        data-testid={`cart-remove-${item.product.id}`}
                        onClick={() => removeFromCart(item.product.id)}
                        className="text-[11px] text-slate-400 hover:text-rose-600 cursor-pointer"
                      >
                        ✕
                      </button>
                    </div>
                  </div>
                ))}

                {/* Contextual Recommendations Section */}
                {(loadingRecs || filteredRecommendations.length > 0) && (
                  <div
                    data-testid="cart-recommendations-section"
                    className="mt-4 pt-4 border-t border-slate-200 space-y-2.5"
                  >
                    <div className="flex items-center justify-between">
                      <h3 className="text-xs font-bold text-slate-900 flex items-center gap-1.5">
                        <span className="text-amber-500">💡</span>
                        <span>
                          {t(
                            'cart.recommendations_title',
                            i18n.language === 'en' ? 'Frequently Bought Together' : 'Pelengkap Belanja'
                          )}
                        </span>
                      </h3>
                      <span className="text-[10px] text-slate-400 font-medium">
                        {t(
                          'cart.recommendations_subtitle',
                          i18n.language === 'en' ? 'Contextual Add-ons' : 'Rekomendasi Tambahan'
                        )}
                      </span>
                    </div>

                    {loadingRecs ? (
                      <div className="space-y-2">
                        <div className="h-14 bg-slate-100 rounded-xl animate-pulse" />
                        <div className="h-14 bg-slate-100 rounded-xl animate-pulse" />
                      </div>
                    ) : (
                      filteredRecommendations.map((rec) => (
                        <div
                          key={rec.id}
                          data-testid={`cart-recommendation-card-${rec.id}`}
                          className="bg-slate-50 hover:bg-slate-100/90 border border-slate-200/80 rounded-xl p-2.5 flex items-center justify-between gap-3 transition-colors"
                        >
                          <img
                            src={
                              rec.image_url ||
                              'https://images.unsplash.com/photo-1505740420928-5e560c06d30e?w=400&q=80'
                            }
                            alt={rec.name}
                            className="w-12 h-12 object-contain bg-white border border-slate-200 rounded-lg p-1 shrink-0"
                          />
                          <div className="flex-1 min-w-0">
                            <h4 className="font-semibold text-xs text-slate-900 truncate" title={rec.name}>
                              {rec.name}
                            </h4>
                            <span className="text-xs font-bold text-slate-700 block">
                              {formatPrice(rec.price, rec.currency)}
                            </span>
                          </div>
                          <button
                            data-testid={`cart-recommendation-add-${rec.id}`}
                            disabled={(rec.stock_quantity ?? 0) <= 0}
                            onClick={() => {
                              if ((rec.stock_quantity ?? 0) > 0) {
                                addToCart(rec, 1);
                              }
                            }}
                            className={`text-xs font-semibold py-1.5 px-2.5 rounded-lg transition-colors shrink-0 cursor-pointer flex items-center gap-1 ${
                              (rec.stock_quantity ?? 0) <= 0
                                ? 'bg-slate-100 text-slate-400 cursor-not-allowed'
                                : 'bg-blue-600 hover:bg-blue-700 text-white shadow-2xs'
                            }`}
                          >
                            <span>+</span>
                            <span>{t('product.add_to_cart', 'Tambah')}</span>
                          </button>
                        </div>
                      ))
                    )}
                  </div>
                )}
              </>
            )}
          </div>

          {/* Footer Breakdown */}
          {items.length > 0 && (
            <div className="pt-4 border-t border-slate-100 space-y-3">
              <div className="flex justify-between text-sm font-bold text-slate-900">
                <span>{t('cart.total')}:</span>
                <span data-testid="cart-total-price" className="text-base text-blue-600 font-extrabold">
                  {formatPrice(cartTotal)}
                </span>
              </div>

              <button
                data-testid="cart-checkout-button"
                onClick={() => {
                  onClose();
                  onCheckout();
                }}
                className="w-full bg-blue-600 hover:bg-blue-700 text-white font-semibold text-sm py-2.5 rounded-lg shadow-sm cursor-pointer transition-colors"
              >
                {t('cart.checkout')}
              </button>

              <button
                onClick={clearCart}
                className="w-full text-center text-xs text-slate-400 hover:text-slate-600 cursor-pointer pt-1"
              >
                {t('cart.clear')}
              </button>
            </div>
          )}
        </div>
      </div>
    </div>
  );
};

