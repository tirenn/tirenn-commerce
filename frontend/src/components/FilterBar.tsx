import React from 'react';
import type { Category } from '../types';

interface FilterBarProps {
  categories: Category[];
  selectedCategoryId: number;
  selectedSort: string;
  onlyInStock: boolean;
  isSemantic: boolean;
  totalProductsCount: number;
  onSelectCategory: (id: number) => void;
  onSelectSort: (sort: string) => void;
  onToggleInStock: (inStock: boolean) => void;
  onToggleSemantic: (semantic: boolean) => void;
  onResetFilters: () => void;
}

export const FilterBar: React.FC<FilterBarProps> = ({
  categories,
  selectedCategoryId,
  selectedSort,
  onlyInStock,
  isSemantic,
  totalProductsCount,
  onSelectCategory,
  onSelectSort,
  onToggleInStock,
  onToggleSemantic,
  onResetFilters,
}) => {
  const isFiltered = selectedCategoryId > 0 || onlyInStock || isSemantic || selectedSort !== 'newest';

  return (
    <div className="bg-white border border-slate-200 rounded-xl p-4 mb-6 space-y-3">
      {/* Top bar: Category tabs */}
      <div className="flex flex-wrap items-center gap-2">
        <button
          data-testid="category-tab-all"
          onClick={() => onSelectCategory(0)}
          className={`px-3 py-1.5 rounded-lg text-xs font-semibold transition-colors cursor-pointer ${
            selectedCategoryId === 0
              ? 'bg-slate-900 text-white'
              : 'bg-slate-100 text-slate-600 hover:bg-slate-200'
          }`}
        >
          All
        </button>

        {categories.map((cat) => (
          <button
            key={cat.id}
            data-testid={`category-tab-${cat.id}`}
            onClick={() => onSelectCategory(cat.id === selectedCategoryId ? 0 : cat.id)}
            className={`px-3 py-1.5 rounded-lg text-xs font-semibold transition-colors cursor-pointer ${
              selectedCategoryId === cat.id
                ? 'bg-blue-600 text-white'
                : 'bg-slate-100 text-slate-600 hover:bg-slate-200'
            }`}
          >
            {cat.name}
          </button>
        ))}
      </div>

      {/* Bottom bar: Status, Count & Sort */}
      <div className="flex flex-wrap items-center justify-between gap-3 pt-2 border-t border-slate-100 text-xs">
        <div className="flex items-center gap-2 text-slate-600">
          <span data-testid="products-count"><b>{totalProductsCount}</b> products</span>
          {isFiltered && (
            <button
              onClick={onResetFilters}
              className="text-blue-600 hover:underline cursor-pointer ml-2"
            >
              Reset
            </button>
          )}
        </div>

        <div className="flex items-center gap-4">
          {/* AI Semantic Search Toggle */}
          <button
            type="button"
            data-testid="semantic-search-toggle"
            onClick={() => onToggleSemantic(!isSemantic)}
            className={`flex items-center gap-1.5 px-2.5 py-1 rounded-lg font-semibold text-xs transition-all cursor-pointer border ${
              isSemantic
                ? 'bg-purple-600 text-white border-purple-700 shadow-xs ring-2 ring-purple-300'
                : 'bg-purple-50 text-purple-700 border-purple-200 hover:bg-purple-100'
            }`}
          >
            <span>🧠</span>
            <span>AI Semantic {isSemantic ? 'ON' : 'OFF'}</span>
          </button>

          <label className="flex items-center gap-1.5 cursor-pointer text-slate-700 select-none">
            <input
              type="checkbox"
              data-testid="in-stock-checkbox"
              className="w-3.5 h-3.5 accent-blue-600 rounded cursor-pointer"
              checked={onlyInStock}
              onChange={(e) => onToggleInStock(e.target.checked)}
            />
            <span>In-Stock Only</span>
          </label>

          <div className="flex items-center gap-1.5">
            <span className="text-slate-500 hidden sm:inline">Sort:</span>
            <select
              data-testid="sort-select"
              className="bg-slate-50 border border-slate-200 rounded-lg px-2 py-1 text-slate-800 outline-none cursor-pointer"
              value={selectedSort}
              onChange={(e) => onSelectSort(e.target.value)}
            >
              <option value="newest">Newest</option>
              <option value="price_asc">Price: Low to High</option>
              <option value="price_desc">Price: High to Low</option>
              <option value="name_asc">Name: A to Z</option>
            </select>
          </div>
        </div>
      </div>
    </div>
  );
};
