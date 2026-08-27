import React, { useState, useEffect } from 'react';
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

// Admin Components
import { AdminDashboard } from './components/admin/AdminDashboard';
import { ProductManagement } from './components/admin/ProductManagement';
import { OrderManagement } from './components/admin/OrderManagement';
import { CustomerManagement } from './components/admin/CustomerManagement';

export const App: React.FC = () => {
  const { currentUser } = useAuth();
  const isAdmin = currentUser?.role === 'ADMIN';

  const [currentView, setCurrentView] = useState<AppView>(() => {
    return isAdmin ? 'admin-dashboard' : 'storefront';
  });

  // When user role changes (e.g. login as admin or logout), update view
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

  // Products & Categories
  const [products, setProducts] = useState<Product[]>([]);
  const [categories, setCategories] = useState<Category[]>([]);
  const [totalProducts, setTotalProducts] = useState(0);
  const [loadingProducts, setLoadingProducts] = useState(false);
  const [error, setError] = useState('');

  // Filters & Search
  const [searchTerm, setSearchTerm] = useState('');
  const [selectedCategoryId, setSelectedCategoryId] = useState(0);
  const [selectedSort, setSelectedSort] = useState('newest');
  const [onlyInStock, setOnlyInStock] = useState(false);

  // Modals
  const [isCartOpen, setIsCartOpen] = useState(false);
  const [isAuthOpen, setIsAuthOpen] = useState(false);
  const [isCheckoutOpen, setIsCheckoutOpen] = useState(false);
  const [selectedProduct, setSelectedProduct] = useState<Product | null>(null);

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

  const loadProducts = async () => {
    setLoadingProducts(true);
    setError('');

    try {
      const params = new URLSearchParams();
      if (searchTerm) params.append('search', searchTerm);
      if (selectedCategoryId > 0) params.append('category_id', selectedCategoryId.toString());
      if (selectedSort) params.append('sort', selectedSort);
      if (onlyInStock) params.append('in_stock', 'true');
      params.append('limit', '50');

      const res = await apiRequest<Product[]>(`/products?${params.toString()}`);
      setLoadingProducts(false);

      if (res.success && Array.isArray(res.data)) {
        setProducts(res.data);
        setTotalProducts(res.meta?.total_rows || res.data.length);
      } else {
        setError(res.error || 'Failed to load products');
      }
    } catch (err: any) {
      setLoadingProducts(false);
      setError(err?.message || 'Network error occurred while fetching products');
    }
  };

  useEffect(() => {
    if (!isAdmin) {
      loadCategories();
    }
  }, [isAdmin]);

  useEffect(() => {
    if (!isAdmin) {
      loadProducts();
    }
  }, [searchTerm, selectedCategoryId, selectedSort, onlyInStock, isAdmin]);

  const handleResetFilters = () => {
    setSearchTerm('');
    setSelectedCategoryId(0);
    setSelectedSort('newest');
    setOnlyInStock(false);
  };

  return (
    <div className="min-h-screen flex flex-col justify-between bg-slate-50 text-slate-900">
      {/* Navbar */}
      <Navbar
        currentView={currentView}
        onSelectView={setCurrentView}
        onOpenCart={() => setIsCartOpen(true)}
        onOpenAuth={() => setIsAuthOpen(true)}
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
          ) : (
            <AdminDashboard onSelectView={setCurrentView} />
          )
        ) : (
          /* Customer / Public Storefront Views */
          currentView === 'storefront' ? (
            <>
              {!searchTerm && selectedCategoryId === 0 && <HeroBanner />}

              <FilterBar
                categories={Array.isArray(categories) ? categories : []}
                selectedCategoryId={selectedCategoryId}
                selectedSort={selectedSort}
                onlyInStock={onlyInStock}
                totalProductsCount={totalProducts}
                onSelectCategory={setSelectedCategoryId}
                onSelectSort={setSelectedSort}
                onToggleInStock={setOnlyInStock}
                onResetFilters={handleResetFilters}
              />

              {/* Product Grid */}
              {loadingProducts ? (
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
                    Try adjusting your search terms or filters.
                  </p>
                  <button
                    onClick={handleResetFilters}
                    className="bg-blue-600 hover:bg-blue-700 text-white font-semibold text-xs py-1.5 px-3 rounded-lg cursor-pointer mt-2"
                  >
                    Clear Filters
                  </button>
                </div>
              ) : (
                <div data-testid="products-grid" className="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4 gap-5">
                  {products.map((product) => (
                    <ProductCard
                      key={product.id}
                      product={product}
                      onSelect={(p) => setSelectedProduct(p)}
                    />
                  ))}
                </div>
              )}
            </>
          ) : currentView === 'my-orders' ? (
            <OrderHistory onSelectView={setCurrentView} />
          ) : null
        )}
      </main>

      {/* Footer */}
      {!isAdmin && <Footer onSelectCategory={setSelectedCategoryId} onSelectView={setCurrentView} />}

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
          />
        </>
      )}

      {/* Auth Modal */}
      <AuthModal isOpen={isAuthOpen} onClose={() => setIsAuthOpen(false)} />
    </div>
  );
};
