import React from 'react';
import { useTranslation } from 'react-i18next';
import { useAuth } from '../context/AuthContext';
import { useCart } from '../context/CartContext';
import { useCurrency } from '../context/CurrencyContext';
import { useToast } from '../context/ToastContext';
import type { AppView } from '../types';

interface NavbarProps {
  currentView: AppView;
  onSelectView: (view: AppView) => void;
  onOpenCart: () => void;
  onOpenAuth: () => void;
  onOpenAdminAI?: () => void;
  searchTerm: string;
  onSearchChange: (term: string) => void;
}

export const Navbar: React.FC<NavbarProps> = ({
  currentView,
  onSelectView,
  onOpenCart,
  onOpenAuth,
  onOpenAdminAI,
  searchTerm,
  onSearchChange,
}) => {
  const { t, i18n } = useTranslation();
  const { currentUser, logout } = useAuth();
  const { cartCount } = useCart();
  const { currency } = useCurrency();
  const { showToast } = useToast();

  const isAdmin = currentUser?.role === 'ADMIN';

  const handleLogout = () => {
    logout();
    onSelectView('storefront');
    showToast(i18n.language === 'en' ? 'Signed out successfully' : 'Berhasil keluar', 'info');
  };

  const toggleLanguage = () => {
    const nextLang = i18n.language === 'id' ? 'en' : 'id';
    i18n.changeLanguage(nextLang);
    localStorage.setItem('tirenn_lang', nextLang);
  };

  return (
    <header className="sticky top-0 z-40 bg-white border-b border-slate-200 shadow-xs">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <div className="flex items-center justify-between h-16 gap-4">
          
          {/* Logo */}
          <div
            onClick={() => onSelectView(isAdmin ? 'admin-dashboard' : 'storefront')}
            className="flex items-center gap-2 cursor-pointer select-none"
          >
            <div className={`w-8 h-8 rounded-lg flex items-center justify-center font-bold text-base text-white ${
              isAdmin ? 'bg-purple-700' : 'bg-blue-600'
            }`}>
              T
            </div>
            <div>
              <span className="font-bold text-lg text-slate-900 tracking-tight block leading-none">
                Tirenn {isAdmin ? 'Admin' : 'Commerce'}
              </span>
              {isAdmin && (
                <span className="text-[10px] font-semibold text-purple-700 uppercase tracking-wider">
                  {t('nav.role_admin')}
                </span>
              )}
            </div>
          </div>

          {/* Search Bar - Only for Storefront */}
          {!isAdmin ? (
            <div className="flex-1 max-w-md hidden sm:block">
              <div className="relative">
                <input
                  type="text"
                  data-testid="search-input"
                  className="w-full text-sm bg-slate-50 border border-slate-200 rounded-lg pl-9 pr-8 py-2 text-slate-900 outline-none focus:bg-white focus:border-blue-600 transition-all"
                  placeholder={t('nav.search_placeholder')}
                  value={searchTerm}
                  onChange={(e) => onSearchChange(e.target.value)}
                />
                <svg className="w-4 h-4 text-slate-400 absolute left-3 top-1/2 -translate-y-1/2" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
                </svg>
                {searchTerm && (
                  <button
                    onClick={() => onSearchChange('')}
                    className="absolute right-2.5 top-1/2 -translate-y-1/2 text-xs font-bold text-slate-400 hover:text-slate-600"
                  >
                    ✕
                  </button>
                )}
              </div>
            </div>
          ) : (
            /* Admin Navigation Tabs in Header */
            <div className="flex-1 flex items-center gap-1.5 ml-4">
              <button
                data-testid="admin-tab-dashboard"
                onClick={() => onSelectView('admin-dashboard')}
                className={`px-3 py-1.5 rounded-lg text-xs font-semibold transition-colors cursor-pointer ${
                  currentView === 'admin-dashboard' ? 'bg-slate-900 text-white' : 'text-slate-600 hover:bg-slate-100'
                }`}
              >
                Dashboard
              </button>
              <button
                data-testid="admin-tab-products"
                onClick={() => onSelectView('admin-products')}
                className={`px-3 py-1.5 rounded-lg text-xs font-semibold transition-colors cursor-pointer ${
                  currentView === 'admin-products' ? 'bg-slate-900 text-white' : 'text-slate-600 hover:bg-slate-100'
                }`}
              >
                {t('admin.manage_products')}
              </button>
              <button
                data-testid="admin-tab-orders"
                onClick={() => onSelectView('admin-orders')}
                className={`px-3 py-1.5 rounded-lg text-xs font-semibold transition-colors cursor-pointer ${
                  currentView === 'admin-orders' ? 'bg-slate-900 text-white' : 'text-slate-600 hover:bg-slate-100'
                }`}
              >
                {t('orders.title')}
              </button>
              <button
                data-testid="admin-tab-customers"
                onClick={() => onSelectView('admin-customers')}
                className={`px-3 py-1.5 rounded-lg text-xs font-semibold transition-colors cursor-pointer ${
                  currentView === 'admin-customers' ? 'bg-slate-900 text-white' : 'text-slate-600 hover:bg-slate-100'
                }`}
              >
                Customers
              </button>
              <button
                data-testid="admin-tab-knowledge"
                onClick={() => onSelectView('admin-knowledge')}
                className={`px-3 py-1.5 rounded-lg text-xs font-semibold transition-colors cursor-pointer flex items-center gap-1 ${
                  currentView === 'admin-knowledge' ? 'bg-purple-700 text-white' : 'text-slate-600 hover:bg-purple-50 hover:text-purple-700'
                }`}
              >
                <span>📚</span>
                <span>{i18n.language === 'en' ? 'SOP & AI Knowledge' : 'SOP & Pengetahuan'}</span>
              </button>
              <button
                data-testid="admin-tab-ai-copilot"
                onClick={onOpenAdminAI}
                className="px-3 py-1.5 rounded-lg text-xs font-semibold transition-colors cursor-pointer flex items-center gap-1 bg-gradient-to-r from-purple-700 to-indigo-700 hover:from-purple-800 hover:to-indigo-800 text-white shadow-xs"
              >
                <span>⚡</span>
                <span>{t('admin_copilot.btn_open')}</span>
              </button>
            </div>
          )}

          {/* Right Action Items */}
          <div className="flex items-center gap-2 sm:gap-3">
            
            {/* Language & Currency Switcher Toggle */}
            <button
              data-testid="lang-currency-toggle"
              onClick={toggleLanguage}
              title="Switch Language & Currency (IDR / USD)"
              className="flex items-center gap-1.5 px-2.5 py-1 text-xs font-semibold bg-slate-100 hover:bg-slate-200 text-slate-700 rounded-lg transition-colors cursor-pointer border border-slate-200"
            >
              <span className="text-sm">{i18n.language === 'en' ? '🇺🇸' : '🇮🇩'}</span>
              <span className="uppercase">{i18n.language}</span>
              <span className="text-slate-400">|</span>
              <span className="text-blue-600 font-bold">{currency === 'USD' ? '$' : 'Rp'}</span>
            </button>

            {!isAdmin ? (
              <>
                <button
                  data-testid="nav-store"
                  onClick={() => onSelectView('storefront')}
                  className={`px-3 py-1.5 text-sm font-medium rounded-lg transition-colors cursor-pointer ${
                    currentView === 'storefront' ? 'text-blue-600 bg-blue-50' : 'text-slate-700 hover:bg-slate-100'
                  }`}
                >
                  Store
                </button>

                {currentUser && (
                  <button
                    data-testid="nav-orders"
                    onClick={() => onSelectView('my-orders')}
                    className={`px-3 py-1.5 text-sm font-medium rounded-lg transition-colors cursor-pointer ${
                      currentView === 'my-orders' ? 'text-blue-600 bg-blue-50' : 'text-slate-700 hover:bg-slate-100'
                    }`}
                  >
                    {t('nav.my_orders')}
                  </button>
                )}

                {/* Cart Button */}
                <button
                  data-testid="cart-button"
                  onClick={onOpenCart}
                  className="bg-blue-600 hover:bg-blue-700 text-white font-medium text-sm py-1.5 px-3 rounded-lg flex items-center gap-1.5 cursor-pointer shadow-xs transition-colors"
                >
                  <span>{t('nav.cart')}</span>
                  {cartCount > 0 && (
                    <span data-testid="cart-badge" className="bg-white text-blue-600 text-xs px-1.5 py-0.2 rounded-full font-bold">
                      {cartCount}
                    </span>
                  )}
                </button>
              </>
            ) : null}

            {/* Auth Menu */}
            {currentUser ? (
              <div className="flex items-center gap-2 pl-2 border-l border-slate-200">
                <span className={`text-xs font-semibold px-2 py-0.5 rounded-md ${
                  isAdmin ? 'bg-purple-100 text-purple-800' : 'text-slate-700'
                }`}>
                  {currentUser.name} {isAdmin && '👑'}
                </span>
                <button
                  data-testid="logout-button"
                  onClick={handleLogout}
                  className="bg-slate-100 hover:bg-slate-200 text-slate-700 text-xs font-medium py-1.5 px-2.5 rounded-lg transition-colors cursor-pointer"
                >
                  {t('nav.logout')}
                </button>
              </div>
            ) : (
              <button
                data-testid="login-button"
                onClick={onOpenAuth}
                className="border border-slate-200 hover:bg-slate-50 text-slate-700 text-sm font-medium py-1.5 px-3 rounded-lg transition-colors cursor-pointer"
              >
                {t('nav.login')}
              </button>
            )}
          </div>
        </div>

        {/* Mobile Search */}
        {!isAdmin && (
          <div className="pb-3 sm:hidden">
            <input
              type="text"
              className="w-full text-xs bg-slate-50 border border-slate-200 rounded-lg px-3 py-1.5 text-slate-900 outline-none"
              placeholder={t('nav.search_placeholder')}
              value={searchTerm}
              onChange={(e) => onSearchChange(e.target.value)}
            />
          </div>
        )}
      </div>
    </header>
  );
};
