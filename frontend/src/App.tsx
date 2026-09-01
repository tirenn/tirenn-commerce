import React, { useState, useEffect, useRef, useCallback } from 'react';
import { useTranslation } from 'react-i18next';
import { apiRequest } from './services/api';
import { useAuth } from './context/AuthContext';
import type { Product, Category, AppView } from './types';

// Storefront Components
import { Navbar } from './components/Navbar';
import { HeroBanner } from './components/HeroBanner';
import { FilterBar } from './components/FilterBar';
import { ProductCard } from './components/ProductCard';
import { ProductDetailModal } from './components/ProductDetailModal';
import { CartDrawer } from './components/CartDrawer';
import { CheckoutModal } from './components/CheckoutModal';
import { AuthModal } from './components/AuthModal';
import { OrderHistory } from './components/OrderHistory';
import { Footer } from './components/Footer';
import { AIChatModal } from './components/AIChatModal';

// Admin Components
import { AdminDashboard } from './components/admin/AdminDashboard';
import { ProductManagement } from './components/admin/ProductManagement';
import { OrderManagement } from './components/admin/OrderManagement';
import { CustomerManagement } from './components/admin/CustomerManagement';
import { KnowledgeManagement } from './components/admin/KnowledgeManagement';
import { AdminAIChatDrawer } from './components/admin/AdminAIChatDrawer';

const PRODUCTS_PER_PAGE = 12;

export const App: React.FC = () => {
  const { t, i18n } = useTranslation();
  const { currentUser } = useAuth();
  const isAdmin = currentUser?.role === 'ADMIN';

  const [currentView, setCurrentView] = useState<AppView>(() => {
    return isAdmin ? 'admin-dashboard' : 'storefront';
  });

  // When user role changes, update view
  useEffect(() => {
    if (isAdmin) {
      if (!currentView.startsWith('admin')) {
        setCurrentView('admin-dashboard');
      }
    } else {
      if (currentView.startsWith('admin')) {
        setCurrentView('storefront');
      }
    }
  }, [currentUser]);

  // Products, Infinite Scrolling & Categories
  const [products, setProducts] = useState<Product[]>([]);
  const [categories, setCategories] = useState<Category[]>([]);
  const [currentPage, setCurrentPage] = useState(1);
  const [totalPages, setTotalPages] = useState(1);
  const [totalProducts, setTotalProducts] = useState(0);
  const [loadingInitial, setLoadingInitial] = useState(false);
  const [loadingMore, setLoadingMore] = useState(false);
  const [error, setError] = useState('');

  // Filters & Search
  const [searchTerm, setSearchTerm] = useState('');
  const [selectedCategoryId, setSelectedCategoryId] = useState(0);
  const [selectedSubCategoryId, setSelectedSubCategoryId] = useState(0);
  const [selectedSort, setSelectedSort] = useState('newest');
  const [onlyInStock, setOnlyInStock] = useState(false);

  // Modals
  const [isCartOpen, setIsCartOpen] = useState(false);
  const [isAuthOpen, setIsAuthOpen] = useState(false);
  const [isCheckoutOpen, setIsCheckoutOpen] = useState(false);
  const [isAIChatOpen, setIsAIChatOpen] = useState(false);
  const [isAdminChatOpen, setIsAdminChatOpen] = useState(false);
  const [selectedProduct, setSelectedProduct] = useState<Product | null>(null);

  // Sentinel Ref for IntersectionObserver
  const sentinelRef = useRef<HTMLDivElement | null>(null);

  const loadCategories = async () => {
    try {
      const res = await apiRequest<Category[]>('/categories');
      if (res.success && Array.isArray(res.data)) {
        setCategories(res.data);
      }
    } catch (err) {
      console.error('Failed to load categories', err);
    }
  };

  const isFetchingRef = useRef(false);

  // Fetch products function (supports initial replace or append for infinite scrolling)
  const fetchProducts = useCallback(
    async (pageToFetch: number, isInitial: boolean = false) => {
      if (isFetchingRef.current) return;
      isFetchingRef.current = true;

      if (isInitial) {
        setLoadingInitial(true);
      } else {
        setLoadingMore(true);
      }
      setError('');

      try {
        const params = new URLSearchParams();
        const cleanSearch = searchTerm.trim();
        if (cleanSearch.length >= 3) {
          params.append('search', cleanSearch);
        }
        if (selectedCategoryId > 0) params.append('category_id', selectedCategoryId.toString());
        if (selectedSubCategoryId > 0) params.append('sub_category_id', selectedSubCategoryId.toString());
        if (selectedSort) params.append('sort', selectedSort);
        if (onlyInStock) params.append('in_stock', 'true');
        params.append('page', pageToFetch.toString());
        params.append('limit', PRODUCTS_PER_PAGE.toString());

        const res = await apiRequest<Product[]>(`/products?${params.toString()}`);

        if (res.success && Array.isArray(res.data)) {
          const incoming = res.data;
          if (isInitial || pageToFetch === 1) {
            setProducts(incoming);
          } else {
            // Append new products, preventing any duplicate IDs
            setProducts((prev) => {
              const existingIds = new Set(prev.map((p) => p.id));
              const fresh = incoming.filter((p) => !existingIds.has(p.id));
              return [...prev, ...fresh];
            });
          }
          const totalRows = res.meta?.total_rows ?? incoming.length;
          const pages = res.meta?.total_page ?? res.meta?.total_pages ?? Math.ceil(totalRows / PRODUCTS_PER_PAGE) ?? 1;
          setTotalProducts(totalRows);
          setTotalPages(pages);
        } else {
          setError(res.error || 'Failed to load products');
        }
      } catch (err: any) {
        setError(err?.message || 'Network error occurred while fetching products');
      } finally {
        setLoadingInitial(false);
        setLoadingMore(false);
        isFetchingRef.current = false;
      }
    },
    [searchTerm, selectedCategoryId, selectedSubCategoryId, selectedSort, onlyInStock]
  );

  const loadNextPage = useCallback(() => {
    if (isFetchingRef.current || loadingInitial || loadingMore) return;
    if (currentPage >= totalPages) return;
    if (products.length >= totalProducts && totalProducts > 0) return;

    const nextPage = currentPage + 1;
    setCurrentPage(nextPage);
    fetchProducts(nextPage, false);
  }, [currentPage, totalPages, totalProducts, products.length, loadingInitial, loadingMore, fetchProducts]);

  useEffect(() => {
    if (!isAdmin) {
      loadCategories();
    }
  }, [isAdmin]);

  // When filters or search change, reset page to 1 and reload products
  useEffect(() => {
    if (!isAdmin) {
      setCurrentPage(1);
      fetchProducts(1, true);
    }
  }, [fetchProducts, isAdmin]);

  // Dual Trigger 1: Window Scroll Event for reliable infinite scroll
  useEffect(() => {
    if (isAdmin) return;

    const handleScroll = () => {
      const scrollPosition = window.innerHeight + window.scrollY;
      const threshold = document.documentElement.scrollHeight - 600;
      if (scrollPosition >= threshold) {
        loadNextPage();
      }
    };

    window.addEventListener('scroll', handleScroll, { passive: true });
    return () => window.removeEventListener('scroll', handleScroll);
  }, [isAdmin, loadNextPage]);

  // Dual Trigger 2: IntersectionObserver for sentinel element
  useEffect(() => {
    if (isAdmin) return;
    const currentSentinel = sentinelRef.current;
    if (!currentSentinel) return;

    const observer = new IntersectionObserver(
      (entries) => {
        if (entries[0].isIntersecting) {
          loadNextPage();
        }
      },
      { rootMargin: '400px' }
    );

    observer.observe(currentSentinel);
    return () => observer.unobserve(currentSentinel);
  }, [isAdmin, loadNextPage]);

  const handleResetFilters = () => {
    setSearchTerm('');
    setSelectedCategoryId(0);
    setSelectedSubCategoryId(0);
    setSelectedSort('newest');
    setOnlyInStock(false);
    setCurrentPage(1);
  };

  return (
    <div className="min-h-screen flex flex-col justify-between bg-slate-50 text-slate-900 relative">
      {/* Navbar */}
      <Navbar
        currentView={currentView}
        onSelectView={setCurrentView}
        onOpenCart={() => setIsCartOpen(true)}
        onOpenAuth={() => setIsAuthOpen(true)}
        onOpenAdminAI={() => setIsAdminChatOpen(true)}
        searchTerm={searchTerm}
        onSearchChange={setSearchTerm}
      />

      {/* Main Content */}
      <main className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-6 flex-1 w-full">
        {isAdmin ? (
          /* When logged in as Admin, strictly show admin views */
          currentView === 'admin-products' ? (
            <ProductManagement />
          ) : currentView === 'admin-orders' ? (
            <OrderManagement />
          ) : currentView === 'admin-customers' ? (
            <CustomerManagement />
          ) : currentView === 'admin-knowledge' ? (
            <KnowledgeManagement />
          ) : (
            <AdminDashboard onSelectView={setCurrentView} />
          )
        ) : (
          /* Customer / Public Storefront Views */
          currentView === 'storefront' ? (
            <>
              {!searchTerm && selectedCategoryId === 0 && <HeroBanner onExplore={() => {
                const el = document.getElementById('catalog-anchor');
                el?.scrollIntoView({ behavior: 'smooth' });
              }} onOpenAI={() => setIsAIChatOpen(true)} />}

              <div id="catalog-anchor"></div>

              <FilterBar
                categories={Array.isArray(categories) ? categories : []}
                selectedCategoryId={selectedCategoryId}
                selectedSubCategoryId={selectedSubCategoryId}
                selectedSort={selectedSort}
                onlyInStock={onlyInStock}
                totalProductsCount={totalProducts}
                onSelectCategory={setSelectedCategoryId}
                onSelectSubCategory={setSelectedSubCategoryId}
                onSelectSort={setSelectedSort}
                onToggleInStock={setOnlyInStock}
                onResetFilters={handleResetFilters}
              />

              {/* Initial Loading State */}
              {loadingInitial ? (
                <div className="bg-white border border-slate-200 rounded-xl p-12 text-center">
                  <span className="text-2xl animate-spin block mb-2">⚡</span>
                  <span className="text-xs text-slate-600">Loading catalog...</span>
                </div>
              ) : error ? (
                <div className="bg-rose-50 border border-rose-200 text-rose-700 p-4 rounded-xl text-xs font-medium">
                  ⚠️ {error}
                </div>
              ) : !Array.isArray(products) || products.length === 0 ? (
                <div className="bg-white border border-slate-200 rounded-xl p-12 text-center space-y-2">
                  <h3 className="font-semibold text-sm text-slate-900">No matching products found</h3>
                  <p className="text-xs text-slate-500">
                    Try adjusting your search keywords or category filters.
                  </p>
                  <button
                    onClick={handleResetFilters}
                    className="bg-blue-600 hover:bg-blue-700 text-white font-semibold text-xs py-1.5 px-3 rounded-lg cursor-pointer mt-2"
                  >
                    {t('filter.reset')}
                  </button>
                </div>
              ) : (
                <>
                  {/* Products Grid */}
                  <div
                    data-testid="products-grid"
                    className="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4 gap-5"
                  >
                    {products.map((product) => (
                      <ProductCard
                        key={product.id}
                        product={product}
                        onSelect={(p) => setSelectedProduct(p)}
                      />
                    ))}
                  </div>

                  {/* Auto-Pagination Infinite Scroll Sentinel & Loading Indicator */}
                  <div ref={sentinelRef} className="py-10 text-center min-h-[90px] flex flex-col items-center justify-center">
                    {loadingMore ? (
                      <div className="inline-flex items-center gap-2 bg-white border border-slate-200 px-5 py-2.5 rounded-full shadow-xs text-xs font-semibold text-slate-600 animate-pulse">
                        <span className="w-4 h-4 border-2 border-blue-600 border-t-transparent rounded-full animate-spin"></span>
                        {i18n.language === 'en' ? 'Loading more products...' : 'Memuat lebih banyak produk...'}
                      </div>
                    ) : products.length < totalProducts && totalProducts > 0 ? (
                      <button
                        onClick={loadNextPage}
                        data-testid="load-more-products-btn"
                        className="bg-white hover:bg-slate-100 border border-slate-200 text-slate-700 text-xs font-semibold py-2 px-5 rounded-full shadow-xs transition-colors cursor-pointer flex items-center gap-1.5"
                      >
                        <span>⬇️</span>
                        <span>{i18n.language === 'en' ? `Load More Products (${products.length}/${totalProducts})` : `Muat Lebih Banyak (${products.length}/${totalProducts})`}</span>
                      </button>
                    ) : products.length >= totalProducts && totalProducts > 0 ? (
                      <div className="text-xs text-slate-400 font-medium">
                        ✓ {i18n.language === 'en' ? `All ${totalProducts} products loaded` : `Semua ${totalProducts} produk telah dimuat`}
                      </div>
                    ) : null}
                  </div>
                </>
              )}
            </>
          ) : currentView === 'my-orders' ? (
            <OrderHistory onSelectView={setCurrentView} />
          ) : null
        )}
      </main>

      {/* Footer */}
      {!isAdmin && <Footer onSelectCategory={setSelectedCategoryId} onSelectView={setCurrentView} />}

      {/* Floating AI Shopper Button (Storefront Only) */}
      {!isAdmin && (
        <button
          data-testid="ai-shopper-floating-btn"
          onClick={() => setIsAIChatOpen(true)}
          className="fixed bottom-6 right-6 z-40 bg-purple-600 hover:bg-purple-700 text-white font-bold text-xs py-3 px-4 rounded-full shadow-lg flex items-center gap-2 transition-transform hover:scale-105 cursor-pointer border-2 border-white ring-4 ring-purple-300/40"
        >
          <span className="text-base animate-bounce">🤖</span>
          <span>{t('hero.cta_ai')}</span>
        </button>
      )}

      {/* Floating Admin AI Copilot Button (Admin Only) */}
      {isAdmin && (
        <button
          data-testid="admin-ai-floating-btn"
          onClick={() => setIsAdminChatOpen(true)}
          className="fixed bottom-6 right-6 z-40 bg-gradient-to-r from-purple-700 via-indigo-700 to-slate-900 hover:from-purple-800 hover:to-slate-950 text-white font-bold text-xs py-3 px-4 rounded-full shadow-xl flex items-center gap-2 transition-transform hover:scale-105 cursor-pointer border-2 border-white/80 ring-4 ring-purple-400/40"
        >
          <span className="text-base animate-bounce">⚡</span>
          <span>{t('admin_copilot.btn_open')}</span>
        </button>
      )}

      {/* Admin AI Copilot Drawer */}
      {isAdmin && (
        <AdminAIChatDrawer
          isOpen={isAdminChatOpen}
          onClose={() => setIsAdminChatOpen(false)}
        />
      )}

      {/* Drawers & Modals (Storefront Only) */}
      {!isAdmin && (
        <>
          <CartDrawer
            isOpen={isCartOpen}
            onClose={() => setIsCartOpen(false)}
            onCheckout={() => setIsCheckoutOpen(true)}
          />

          <CheckoutModal
            isOpen={isCheckoutOpen}
            onClose={() => setIsCheckoutOpen(false)}
            onAuthRequired={() => setIsAuthOpen(true)}
            onSelectView={setCurrentView}
          />

          <ProductDetailModal
            product={selectedProduct}
            onClose={() => setSelectedProduct(null)}
            onDirectCheckout={() => setIsCheckoutOpen(true)}
            onSelectProduct={(p) => setSelectedProduct(p)}
          />

          <AIChatModal
            isOpen={isAIChatOpen}
            onClose={() => setIsAIChatOpen(false)}
            onAuthRequired={() => setIsAuthOpen(true)}
          />
        </>
      )}

      {/* Auth Modal */}
      <AuthModal isOpen={isAuthOpen} onClose={() => setIsAuthOpen(false)} />
    </div>
  );
};
