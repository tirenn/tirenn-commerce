import React from 'react';
import { useTranslation } from 'react-i18next';
import type { Category } from '../types';

interface FilterBarProps {
  categories: Category[];
  selectedCategoryId: number;
  selectedSubCategoryId: number;
  selectedSort: string;
  onlyInStock: boolean;
  isSemanticSearch: boolean;
  totalProductsCount: number;
  onSelectCategory: (id: number) => void;
  onSelectSubCategory: (id: number) => void;
  onSelectSort: (sort: string) => void;
  onToggleInStock: (inStock: boolean) => void;
  onToggleSemanticSearch: (isSemantic: boolean) => void;
  onResetFilters: () => void;
}

export const FilterBar: React.FC<FilterBarProps> = ({
  categories,
  selectedCategoryId,
  selectedSubCategoryId,
  selectedSort,
  onlyInStock,
  isSemanticSearch,
  totalProductsCount,
  onSelectCategory,
  onSelectSubCategory,
  onSelectSort,
  onToggleInStock,
  onToggleSemanticSearch,
  onResetFilters,
}) => {
  const { t } = useTranslation();
  const isFiltered = selectedCategoryId > 0 || selectedSubCategoryId > 0 || onlyInStock || selectedSort !== 'newest';

  // Find active category's subcategories
  const activeCategory = categories.find((c) => c.id === selectedCategoryId);
  const subCategories = activeCategory?.sub_categories || [];

  return (
    <div className="bg-white border border-slate-200 rounded-xl p-3 sm:p-4 mb-5 sm:mb-6 space-y-2.5 sm:space-y-3 shadow-xs">
      
      {/* 1. Main Category Tabs - Horizontal Scroll on Mobile, Wrap on Desktop */}
      <div className="flex items-center gap-1.5 sm:gap-2 overflow-x-auto no-scrollbar pb-1 sm:pb-0 flex-nowrap sm:flex-wrap">
        <button
          data-testid="category-tab-all"
          onClick={() => {
            onSelectCategory(0);
            onSelectSubCategory(0);
          }}
          className={`px-3 py-1.5 rounded-lg text-xs font-semibold transition-colors cursor-pointer shrink-0 ${
            selectedCategoryId === 0
              ? 'bg-slate-900 text-white'
              : 'bg-slate-100 text-slate-600 hover:bg-slate-200'
          }`}
        >
          {t('filter.all_categories')}
        </button>

        {categories.map((cat) => (
          <button
            key={cat.id}
            data-testid={`category-tab-${cat.id}`}
            onClick={() => {
              if (cat.id === selectedCategoryId) {
                onSelectCategory(0);
                onSelectSubCategory(0);
              } else {
                onSelectCategory(cat.id);
                onSelectSubCategory(0);
              }
            }}
            className={`px-3 py-1.5 rounded-lg text-xs font-semibold transition-colors cursor-pointer flex items-center gap-1 shrink-0 ${
              selectedCategoryId === cat.id
                ? 'bg-blue-600 text-white shadow-xs'
                : 'bg-slate-100 text-slate-600 hover:bg-slate-200'
            }`}
          >
            {cat.icon && <span>{cat.icon}</span>}
            <span>{t(`categories.${cat.slug}`, cat.name)}</span>
          </button>
        ))}
      </div>

      {/* 2. Sub-Category Pills - Horizontal Scroll on Mobile */}
      {subCategories.length > 0 && (
        <div className="flex items-center gap-1.5 pt-2 pb-1 border-t border-dashed border-slate-200 overflow-x-auto no-scrollbar flex-nowrap sm:flex-wrap">
          <span className="text-[10px] sm:text-[11px] font-bold text-slate-400 uppercase tracking-wider mr-1 shrink-0">
            {t('product.subcategory')}:
          </span>
          <button
            data-testid="subcategory-tab-all"
            onClick={() => onSelectSubCategory(0)}
            className={`px-2.5 py-1 rounded-md text-[11px] font-medium transition-colors cursor-pointer shrink-0 ${
              selectedSubCategoryId === 0
                ? 'bg-blue-100 text-blue-800 font-bold border border-blue-200'
                : 'bg-slate-50 text-slate-600 hover:bg-slate-100 border border-slate-200'
            }`}
          >
            {t('filter.all_subcategories')}
          </button>

          {subCategories.map((sub) => (
            <button
              key={sub.id}
              data-testid={`subcategory-tab-${sub.id}`}
              onClick={() => onSelectSubCategory(sub.id === selectedSubCategoryId ? 0 : sub.id)}
              className={`px-2.5 py-1 rounded-md text-[11px] font-medium transition-colors cursor-pointer flex items-center gap-1 shrink-0 ${
                selectedSubCategoryId === sub.id
                  ? 'bg-blue-600 text-white font-semibold'
                  : 'bg-slate-50 text-slate-600 hover:bg-slate-100 border border-slate-200'
              }`}
            >
              {sub.icon && <span>{sub.icon}</span>}
              <span>{t(`subcategories.${sub.slug}`, sub.name)}</span>
            </button>
          ))}
        </div>
      )}

      {/* 3. Bottom bar: Status, Count & Sort */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-2.5 pt-2 border-t border-slate-100 text-xs">
        <div className="flex items-center justify-between sm:justify-start gap-2 text-slate-600">
          <span data-testid="products-count">
            <b>{totalProductsCount}</b> {t('filter.results_count', { count: totalProductsCount }).replace(/Menampilkan |Showing /, '')}
          </span>
          {isFiltered && (
            <button
              onClick={onResetFilters}
              className="text-blue-600 hover:underline cursor-pointer font-medium ml-1"
            >
              {t('filter.reset')}
            </button>
          )}
        </div>

        <div className="flex flex-wrap items-center justify-between sm:justify-end gap-2 sm:gap-3">
          {/* AI Semantic Search Toggle */}
          <button
            type="button"
            data-testid="semantic-search-toggle"
            onClick={() => onToggleSemanticSearch(!isSemanticSearch)}
            className={`flex items-center gap-1.5 px-2.5 py-1 rounded-lg text-xs font-semibold transition-all cursor-pointer border ${
              isSemanticSearch
                ? 'bg-purple-600 text-white border-purple-600 shadow-xs ring-2 ring-purple-200'
                : 'bg-slate-50 text-slate-600 hover:bg-slate-100 border-slate-200'
            }`}
            title={isSemanticSearch ? t('filter.semantic_search_on_hint') : t('filter.semantic_search_off_hint')}
          >
            <span>🧠</span>
            <span className="hidden xs:inline">{t('filter.semantic_search')}</span>
            <span
              className={`text-[10px] px-1.5 py-0.5 rounded font-bold uppercase ${
                isSemanticSearch ? 'bg-purple-800 text-purple-100' : 'bg-slate-200 text-slate-600'
              }`}
            >
              {isSemanticSearch ? 'AI Service' : 'Go Backend'}
            </span>
          </button>

          <label className="flex items-center gap-1.5 cursor-pointer text-slate-700 select-none">
            <input
              type="checkbox"
              data-testid="in-stock-checkbox"
              className="w-3.5 h-3.5 accent-blue-600 rounded cursor-pointer"
              checked={onlyInStock}
              onChange={(e) => onToggleInStock(e.target.checked)}
            />
            <span className="text-[11px] sm:text-xs">{t('filter.in_stock_only')}</span>
          </label>

          <div className="flex items-center gap-1">
            <select
              data-testid="sort-select"
              className="bg-slate-50 border border-slate-200 rounded-lg px-2 py-1 text-slate-800 outline-none cursor-pointer text-xs"
              value={selectedSort}
              onChange={(e) => onSelectSort(e.target.value)}
            >
              <option value="newest">{t('filter.sort_newest')}</option>
              <option value="price_asc">{t('filter.sort_price_asc')}</option>
              <option value="price_desc">{t('filter.sort_price_desc')}</option>
              <option value="name_asc">{t('filter.sort_name_asc')}</option>
            </select>
          </div>
        </div>
      </div>
    </div>
  );
};
