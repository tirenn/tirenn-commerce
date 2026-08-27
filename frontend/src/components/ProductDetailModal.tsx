import React, { useState } from 'react';
import type { Product } from '../types';
import { useCart } from '../context/CartContext';
import { formatRupiah } from '../utils/format';

interface ProductDetailModalProps {
  product: Product | null;
  onClose: () => void;
  onDirectCheckout: () => void;
}

export const ProductDetailModal: React.FC<ProductDetailModalProps> = ({
  product,
  onClose,
  onDirectCheckout,
}) => {
  const { addToCart } = useCart();
  const [quantity, setQuantity] = useState(1);

  if (!product) return null;

  const increment = () => {
    if (quantity < (product.stock_quantity ?? 0)) {
      setQuantity((q) => q + 1);
    }
  };

  const decrement = () => {
    if (quantity > 1) {
      setQuantity((q) => q - 1);
    }
  };

  const handleAdd = () => {
    if ((product.stock_quantity ?? 0) > 0) {
      addToCart(product, quantity);
      onClose();
    }
  };

  const handleBuyNow = () => {
    if ((product.stock_quantity ?? 0) > 0) {
      addToCart(product, quantity);
      onClose();
      onDirectCheckout();
    }
  };

  return (
    <div className="fixed inset-0 bg-slate-900/50 backdrop-blur-xs z-50 flex items-center justify-center p-4">
      <div data-testid="pdp-modal" className="bg-white rounded-2xl w-full max-w-2xl p-6 sm:p-8 relative max-h-[90vh] overflow-y-auto shadow-xl border border-slate-200 animate-modal">
        {/* Close Button */}
        <button
          data-testid="pdp-close"
          onClick={onClose}
          className="absolute top-4 right-4 text-slate-400 hover:text-slate-700 w-8 h-8 rounded-full bg-slate-100 flex items-center justify-center cursor-pointer transition-colors"
        >
          ✕
        </button>

        <div className="grid grid-cols-1 sm:grid-cols-2 gap-6 items-start">
          {/* Product Image */}
          <div className="bg-slate-50 border border-slate-100 rounded-xl h-64 flex items-center justify-center p-4 overflow-hidden">
            <img
              src={product.image_url}
              alt={product.name}
              className="max-h-full max-w-full object-contain"
            />
          </div>

          {/* Details */}
          <div className="space-y-4">
            <div>
              <span className="text-xs font-semibold text-slate-500 uppercase tracking-wider block mb-1">
                {product.category?.name || 'Department'} • SKU: {product.sku}
              </span>
              <h2 className="text-xl font-bold text-slate-900 leading-snug">
                {product.name}
              </h2>
            </div>

            <div className="font-extrabold text-2xl text-slate-900">
              {formatRupiah(product.price)}
            </div>

            <p className="text-xs text-slate-600 leading-relaxed">
              {product.description}
            </p>

            <div className="text-xs text-slate-500">
              <b>Stock:</b> {product.stock_quantity ?? 0} units available
            </div>

            {/* Quantity Selector & Actions */}
            {(product.stock_quantity ?? 0) > 0 ? (
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
                    Add to Cart
                  </button>
                  <button
                    data-testid="pdp-buy-now"
                    onClick={handleBuyNow}
                    className="bg-blue-600 hover:bg-blue-700 text-white font-semibold py-2 px-3 rounded-lg text-xs shadow-xs transition-colors cursor-pointer"
                  >
                    Buy Now
                  </button>
                </div>
              </div>
            ) : (
              <div className="bg-rose-50 border border-rose-200 p-3 rounded-lg text-center text-xs text-rose-700 font-medium">
                Out of Stock
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  );
};
