import React from 'react';
import { useTranslation } from 'react-i18next';
import type { Product } from '../types';
import { useCart } from '../context/CartContext';
import { useCurrency } from '../context/CurrencyContext';

interface ProductCardProps {
  product: Product;
  onSelect: (product: Product) => void;
}

export const ProductCard: React.FC<ProductCardProps> = ({ product, onSelect }) => {
  const { t } = useTranslation();
  const { addToCart } = useCart();
  const { formatPrice } = useCurrency();

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
      className="bg-white border border-slate-200 rounded-xl overflow-hidden flex flex-col justify-between cursor-pointer transition-all hover:border-slate-300 hover:shadow-xs group"
    >
      {/* Product Image Box */}
      <div className="relative bg-slate-50 border-b border-slate-100 h-48 p-4 flex items-center justify-center overflow-hidden">
        {isOutOfStock ? (
          <span className="absolute top-2.5 left-2.5 bg-rose-100 text-rose-800 text-[10px] font-semibold px-2 py-0.5 rounded-md border border-rose-200 z-10">
            {t('product.out_of_stock')}
          </span>
        ) : product.badge ? (
          <span className="absolute top-2.5 left-2.5 bg-blue-100 text-blue-800 text-[10px] font-semibold px-2 py-0.5 rounded-md border border-blue-200 z-10">
            {product.badge}
          </span>
        ) : null}

        <img
          src={product.image_url || 'https://images.unsplash.com/photo-1505740420928-5e560c06d30e?w=400&q=80'}
          alt={product.name || 'Product Image'}
          className="max-h-full max-w-full object-contain group-hover:scale-105 transition-transform duration-200"
          loading="lazy"
        />
      </div>

      {/* Product Info */}
      <div className="p-4 flex-1 flex flex-col justify-between">
        <div>
          <div className="flex items-center gap-1 text-[11px] text-slate-500 font-medium uppercase tracking-wider mb-1">
            <span>{product.category ? t(`categories.${product.category.slug}`, product.category.name) : 'General'}</span>
            {product.sub_category && (
              <>
                <span>•</span>
                <span className="text-blue-600 font-semibold">{t(`subcategories.${product.sub_category.slug}`, product.sub_category.name)}</span>
              </>
            )}
          </div>

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
              {formatPrice(product.price, product.currency)}
            </span>
            <span className="text-[10px] text-slate-500 block">
              {isOutOfStock ? t('product.out_of_stock') : `Stock: ${product.stock_quantity ?? 0}`}
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
            {isOutOfStock ? t('product.out_of_stock') : `+ ${t('product.add_to_cart')}`}
          </button>
        </div>
      </div>
    </div>
  );
};
