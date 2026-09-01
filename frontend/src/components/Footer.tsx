import React from 'react';
import type { AppView } from '../types';

interface FooterProps {
  onSelectCategory: (id: number) => void;
  onSelectView: (view: AppView) => void;
}

export const Footer: React.FC<FooterProps> = ({ onSelectCategory, onSelectView }) => {
  return (
    <footer className="bg-white border-t border-slate-200 mt-16 py-8 text-slate-500 text-xs">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 flex flex-col sm:flex-row items-center justify-between gap-4">
        <div className="flex items-center gap-2">
          <div className="w-6 h-6 bg-blue-600 text-white rounded-md flex items-center justify-center font-bold text-xs">
            T
          </div>
          <span className="font-bold text-slate-800">Tirenn Commerce</span>
          <span className="text-slate-400">© 2026</span>
        </div>

        <div className="flex items-center gap-4 text-xs font-medium">
          <button
            onClick={() => onSelectView('storefront')}
            className="hover:text-slate-900 cursor-pointer"
          >
            Catalog
          </button>
          <button
            onClick={() => onSelectView('my-orders')}
            className="hover:text-slate-900 cursor-pointer"
          >
            Orders
          </button>
        </div>
      </div>
    </footer>
  );
};
