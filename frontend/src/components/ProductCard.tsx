import React from 'react';
import type { Product } from '../types';
import { useCart } from '../context/CartContext';
import { formatRupiah } from '../utils/format';

interface ProductCardProps {
  product: Product;
  onSelect: (product: Product) => void;
}

export const ProductCard: React.FC<ProductCardProps> = ({ product, onSelect }) => {
  const { addToCart } = useCart();

  if (!product) return null;

  const handleAddToCart = (e: React.MouseEvent) => {
    e.stopPropagation();
    if ((product.stock_quantity ?? 0) > 0) {
      addToCart(product, 1);
    }
  };

  const isOutOfStock = (product.stock_quantity ?? 0) <= 0;

  return (
    <div
      data-testid={`product-card-${product.id}`}
      onClick={() => onSelect(product)}
      className="bg-white border border-slate-200 rounded-xl overflow-hidden flex flex-col justify-between cursor-pointer transition-all hover:border-slate-300 hover:shadow-xs"
    >
      {/* Product Image Box */}
      <div className="relative bg-slate-50 border-b border-slate-100 h-48 p-4 flex items-center justify-center overflow-hidden">
        {isOutOfStock && (
          <span className="absolute top-2.5 left-2.5 bg-rose-100 text-rose-800 text-[10px] font-semibold px-2 py-0.5 rounded-md border border-rose-200">
            Out of Stock
          </span>
        )}

        <img
          src={product.image_url || 'https://images.unsplash.com/photo-1505740420928-5e560c06d30e?w=400&q=80'}
          alt={product.name || 'Product Image'}
          className="max-h-full max-w-full object-contain"
          loading="lazy"
        />
      </div>

      {/* Product Info */}
      <div className="p-4 flex-1 flex flex-col justify-between">
        <div>
          <span className="text-[11px] text-slate-500 font-medium uppercase tracking-wider block mb-1">
            {product.category?.name || 'General'}
          </span>

          <h3 className="font-semibold text-sm text-slate-900 leading-snug mb-1.5 line-clamp-2">
            {product.name}
          </h3>

          <p className="text-xs text-slate-500 line-clamp-2 mb-3">
            {product.description}
          </p>
        </div>

        {/* Pricing & Add to Cart */}
        <div className="pt-3 border-t border-slate-100 flex items-center justify-between gap-2">
          <div>
            <span className="font-bold text-sm sm:text-base text-slate-900 block">
              {formatRupiah(product.price)}
            </span>
            <span className="text-[10px] text-slate-500 block">
              {isOutOfStock ? 'Sold out' : `Stock: ${product.stock_quantity ?? 0}`}
            </span>
          </div>

          <button
            data-testid={`add-to-cart-${product.id}`}
            disabled={isOutOfStock}
            onClick={handleAddToCart}
            className={`text-xs font-semibold py-1.5 px-3 rounded-lg transition-colors cursor-pointer ${
              isOutOfStock
                ? 'bg-slate-100 text-slate-400 cursor-not-allowed'
                : 'bg-blue-600 hover:bg-blue-700 text-white'
            }`}
          >
            {isOutOfStock ? 'Unavailable' : '+ Add'}
          </button>
        </div>
      </div>
    </div>
  );
};
