import React, { useState, useEffect } from 'react';
import { useTranslation } from 'react-i18next';
import type { Product } from '../types';
import { useCart } from '../context/CartContext';
import { useCurrency } from '../context/CurrencyContext';
import { getRecommendations } from '../services/api';

interface ProductDetailModalProps {
  product: Product | null;
  onClose: () => void;
  onDirectCheckout: () => void;
  onSelectProduct?: (product: Product) => void;
}

export const ProductDetailModal: React.FC<ProductDetailModalProps> = ({
  product,
  onClose,
  onDirectCheckout,
  onSelectProduct,
}) => {
  const { t, i18n } = useTranslation();
  const { addToCart } = useCart();
  const { formatPrice } = useCurrency();

  const [activeProduct, setActiveProduct] = useState<Product | null>(product);
  const [quantity, setQuantity] = useState(1);
  const [recommendations, setRecommendations] = useState<Product[]>([]);
  const [loadingRecs, setLoadingRecs] = useState(false);

  useEffect(() => {
    setActiveProduct(product);
    setQuantity(1);
  }, [product]);

  const currentProduct = activeProduct || product;

  useEffect(() => {
    if (!currentProduct?.id) {
      setRecommendations([]);
      return;
    }

    let isMounted = true;
    setLoadingRecs(true);
    getRecommendations(currentProduct.id, 6)
      .then((data) => {
        if (isMounted) {
          setRecommendations(data || []);
        }
      })
      .catch((err) => {
        console.error('Failed to load PDP recommendations', err);
        if (isMounted) setRecommendations([]);
      })
      .finally(() => {
        if (isMounted) setLoadingRecs(false);
      });

    return () => {
      isMounted = false;
    };
  }, [currentProduct?.id]);

  if (!currentProduct) return null;

  const increment = () => {
    if (quantity < (currentProduct.stock_quantity ?? 0)) {
      setQuantity((q) => q + 1);
    }
  };

  const decrement = () => {
    if (quantity > 1) {
      setQuantity((q) => q - 1);
    }
  };

  const handleAdd = () => {
    if ((currentProduct.stock_quantity ?? 0) > 0) {
      addToCart(currentProduct, quantity);
      onClose();
    }
  };

  const handleBuyNow = () => {
    if ((currentProduct.stock_quantity ?? 0) > 0) {
      addToCart(currentProduct, quantity);
      onClose();
      onDirectCheckout();
    }
  };

  const handleSelectRecommendation = (rec: Product) => {
    setActiveProduct(rec);
    setQuantity(1);
    if (onSelectProduct) {
      onSelectProduct(rec);
    }
  };

  const handleQuickAdd = (rec: Product, e: React.MouseEvent) => {
    e.stopPropagation();
    if ((rec.stock_quantity ?? 0) > 0) {
      addToCart(rec, 1);
    }
  };

  return (
    <div className="fixed inset-0 bg-slate-900/60 backdrop-blur-xs z-50 flex items-end sm:items-center justify-center p-0 sm:p-4">
      <div
        data-testid="pdp-modal"
        className="bg-white rounded-t-2xl sm:rounded-2xl w-full max-w-3xl p-4 sm:p-8 relative max-h-[92dvh] sm:max-h-[90vh] overflow-y-auto shadow-2xl border border-slate-200 animate-modal"
      >
        {/* Close Button */}
        <button
          data-testid="pdp-close"
          onClick={onClose}
          className="absolute top-3 sm:top-4 right-3 sm:right-4 text-slate-400 hover:text-slate-700 w-8 h-8 rounded-full bg-slate-100 flex items-center justify-center cursor-pointer transition-colors z-10"
        >
          ✕
        </button>

        <div className="grid grid-cols-1 sm:grid-cols-2 gap-4 sm:gap-6 items-start pt-2 sm:pt-0">
          {/* Product Image */}
          <div className="bg-slate-50 border border-slate-100 rounded-xl h-52 sm:h-64 flex items-center justify-center p-3 sm:p-4 overflow-hidden">
            <img
              src={currentProduct.image_url}
              alt={currentProduct.name}
              className="max-h-full max-w-full object-contain"
            />
          </div>

          {/* Details */}
          <div className="space-y-4">
            <div>
              <div className="flex items-center gap-1.5 text-xs font-semibold text-slate-500 uppercase tracking-wider mb-1">
                <span>
                  {currentProduct.category
                    ? t(`categories.${currentProduct.category.slug}`, currentProduct.category.name)
                    : 'Department'}
                </span>
                {currentProduct.sub_category && (
                  <>
                    <span>•</span>
                    <span className="text-blue-600 font-bold">
                      {t(
                        `subcategories.${currentProduct.sub_category.slug}`,
                        currentProduct.sub_category.name
                      )}
                    </span>
                  </>
                )}
                <span>•</span>
                <span>SKU: {currentProduct.sku}</span>
              </div>
              <h2 className="text-xl font-bold text-slate-900 leading-snug">
                {currentProduct.name}
              </h2>
            </div>

            <div className="font-extrabold text-2xl text-slate-900">
              {formatPrice(currentProduct.price, currentProduct.currency)}
            </div>

            <p className="text-xs text-slate-600 leading-relaxed">
              {currentProduct.description}
            </p>

            <div className="text-xs text-slate-500">
              <b>{t('product.in_stock')}:</b> {currentProduct.stock_quantity ?? 0}{' '}
              {t('product.remaining_stock', { count: currentProduct.stock_quantity ?? 0 }).replace(
                /[0-9]+ /,
                ''
              )}
            </div>

            {/* Quantity Selector & Actions */}
            {(currentProduct.stock_quantity ?? 0) > 0 ? (
              <div className="space-y-3 pt-2">
                <div className="flex items-center gap-3">
                  <span className="text-xs font-medium text-slate-700">Quantity:</span>
                  <div className="flex items-center border border-slate-200 rounded-lg overflow-hidden bg-white">
                    <button
                      className="px-3 py-1 text-sm font-bold text-slate-600 hover:bg-slate-100 transition-colors cursor-pointer"
                      onClick={decrement}
                    >
                      -
                    </button>
                    <span className="px-3 py-1 text-xs font-semibold text-slate-900 select-none">
                      {quantity}
                    </span>
                    <button
                      className="px-3 py-1 text-sm font-bold text-slate-600 hover:bg-slate-100 transition-colors cursor-pointer"
                      onClick={increment}
                    >
                      +
                    </button>
                  </div>
                </div>

                <div className="grid grid-cols-2 gap-2 pt-1">
                  <button
                    data-testid="pdp-add-to-cart"
                    onClick={handleAdd}
                    className="border border-slate-200 hover:bg-slate-50 text-slate-800 font-semibold py-2 px-3 rounded-lg text-xs transition-colors cursor-pointer"
                  >
                    + {t('product.add_to_cart')}
                  </button>
                  <button
                    data-testid="pdp-buy-now"
                    onClick={handleBuyNow}
                    className="bg-blue-600 hover:bg-blue-700 text-white font-semibold py-2 px-3 rounded-lg text-xs shadow-xs transition-colors cursor-pointer"
                  >
                    {t('checkout.pay_now')}
                  </button>
                </div>
              </div>
            ) : (
              <div className="bg-rose-50 border border-rose-200 p-3 rounded-lg text-center text-xs text-rose-700 font-medium">
                {t('product.out_of_stock')}
              </div>
            )}
          </div>
        </div>

        {/* Recommendations Carousel Section */}
        {(loadingRecs || recommendations.length > 0) && (
          <div
            data-testid="pdp-recommendations-section"
            className="mt-6 pt-5 border-t border-slate-200"
          >
            <div className="flex items-center justify-between mb-3">
              <h3 className="text-sm font-bold text-slate-900 flex items-center gap-1.5">
                <span className="text-amber-500">✨</span>
                <span>
                  {t(
                    'product.similar_products',
                    i18n.language === 'en' ? 'You May Also Like' : 'Produk Serupa'
                  )}
                </span>
              </h3>
              <span className="text-[11px] text-slate-400 font-medium">
                {i18n.language === 'en' ? 'Similar Items Discovery' : 'Penemuan Produk Serupa'}
              </span>
            </div>

            {loadingRecs ? (
              <div className="flex gap-3 overflow-x-auto pb-2">
                {[1, 2, 3, 4].map((i) => (
                  <div
                    key={i}
                    className="w-40 sm:w-44 shrink-0 bg-slate-50 border border-slate-100 rounded-xl p-3 h-48 animate-pulse flex flex-col justify-between"
                  >
                    <div className="bg-slate-200 rounded-lg h-24 w-full" />
                    <div className="space-y-1.5 pt-2">
                      <div className="bg-slate-200 rounded h-3 w-3/4" />
                      <div className="bg-slate-200 rounded h-3 w-1/2" />
                    </div>
                  </div>
                ))}
              </div>
            ) : (
              <div className="flex gap-3 overflow-x-auto pb-2 pt-1 scrollbar-thin">
                {recommendations.map((rec) => (
                  <div
                    key={rec.id}
                    data-testid={`pdp-recommendation-card-${rec.id}`}
                    onClick={() => handleSelectRecommendation(rec)}
                    className="bg-slate-50 hover:bg-slate-100/90 border border-slate-200/80 rounded-xl p-3 flex flex-col justify-between cursor-pointer transition-all duration-200 group shrink-0 w-40 sm:w-44 hover:shadow-xs hover:border-slate-300"
                  >
                    {/* Product Image & Badge */}
                    <div className="relative bg-white border border-slate-100 rounded-lg h-28 flex items-center justify-center p-2 mb-2 overflow-hidden">
                      {(rec.stock_quantity ?? 0) <= 0 ? (
                        <span className="absolute top-1.5 left-1.5 bg-rose-100 text-rose-800 text-[9px] font-bold px-1.5 py-0.5 rounded border border-rose-200 z-10">
                          {t('product.out_of_stock')}
                        </span>
                      ) : rec.badge ? (
                        <span className="absolute top-1.5 left-1.5 bg-blue-100 text-blue-800 text-[9px] font-bold px-1.5 py-0.5 rounded border border-blue-200 z-10">
                          {rec.badge}
                        </span>
                      ) : null}
                      <img
                        src={
                          rec.image_url ||
                          'https://images.unsplash.com/photo-1505740420928-5e560c06d30e?w=400&q=80'
                        }
                        alt={rec.name}
                        className="max-h-full max-w-full object-contain group-hover:scale-105 transition-transform duration-200"
                        loading="lazy"
                      />
                    </div>

                    {/* Info & Quick Add */}
                    <div className="flex-1 flex flex-col justify-between">
                      <div>
                        <h4
                          className="font-semibold text-xs text-slate-900 line-clamp-1 mb-1 group-hover:text-blue-600 transition-colors"
                          title={rec.name}
                        >
                          {rec.name}
                        </h4>
                        <div className="font-bold text-xs text-slate-900 mb-2">
                          {formatPrice(rec.price, rec.currency)}
                        </div>
                      </div>

                      <button
                        data-testid={`pdp-recommendation-add-${rec.id}`}
                        disabled={(rec.stock_quantity ?? 0) <= 0}
                        onClick={(e) => handleQuickAdd(rec, e)}
                        className={`w-full py-1.5 px-2 rounded-lg text-xs font-semibold transition-colors flex items-center justify-center gap-1 cursor-pointer ${
                          (rec.stock_quantity ?? 0) <= 0
                            ? 'bg-slate-100 text-slate-400 cursor-not-allowed'
                            : 'bg-blue-600 hover:bg-blue-700 text-white shadow-2xs'
                        }`}
                      >
                        <span>+</span>
                        <span>{t('product.add_to_cart', 'Tambah')}</span>
                      </button>
                    </div>
                  </div>
                ))}
              </div>
            )}
          </div>
        )}
      </div>
    </div>
  );
};

