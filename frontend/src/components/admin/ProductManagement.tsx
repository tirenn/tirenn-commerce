import React, { useState, useEffect } from 'react';
import { apiRequest } from '../../services/api';
import { useToast } from '../../context/ToastContext';
import { StockAdjustmentModal } from './StockAdjustmentModal';
import { formatRupiah } from '../../utils/format';
import type { Product, Category } from '../../types';

export const ProductManagement: React.FC = () => {
  const { showToast } = useToast();

  const [products, setProducts] = useState<Product[]>([]);
  const [categories, setCategories] = useState<Category[]>([]);
  const [loading, setLoading] = useState(true);
  const [search, setSearch] = useState('');
  const [selectedProductForStock, setSelectedProductForStock] = useState<Product | null>(null);

  // Create / Edit modal state
  const [isEditModalOpen, setIsEditModalOpen] = useState(false);
  const [editingProduct, setEditingProduct] = useState<Product | null>(null);

  // Form Fields
  const [name, setName] = useState('');
  const [sku, setSku] = useState('');
  const [categoryId, setCategoryId] = useState<number>(1);
  const [description, setDescription] = useState('');
  const [price, setPrice] = useState('');
  const [stockQuantity, setStockQuantity] = useState('');
  const [lowStockThreshold, setLowStockThreshold] = useState('5');
  const [imageUrl, setImageUrl] = useState('');
  const [isActive, setIsActive] = useState(true);

  const loadData = async () => {
    setLoading(true);
    const [prodRes, catRes] = await Promise.all([
      apiRequest<Product[]>(`/admin/products?limit=50&search=${search}`),
      apiRequest<Category[]>('/categories'),
    ]);
    setLoading(false);

    if (prodRes.success && Array.isArray(prodRes.data)) {
      setProducts(prodRes.data);
    }
    if (catRes.success && Array.isArray(catRes.data)) {
      setCategories(catRes.data);
    }
  };

  useEffect(() => {
    loadData();
  }, [search]);

  const openCreateModal = () => {
    setEditingProduct(null);
    setName('');
    setSku(`SKU-${Math.floor(1000 + Math.random() * 9000)}`);
    setCategoryId(categories[0]?.id || 1);
    setDescription('');
    setPrice('150000');
    setStockQuantity('20');
    setLowStockThreshold('5');
    setImageUrl('https://images.unsplash.com/photo-1505740420928-5e560c06d30e?w=600');
    setIsActive(true);
    setIsEditModalOpen(true);
  };

  const openEditModal = (p: Product) => {
    setEditingProduct(p);
    setName(p.name);
    setSku(p.sku);
    setCategoryId(p.category_id);
    setDescription(p.description);
    setPrice(p.price.toString());
    setStockQuantity(p.stock_quantity.toString());
    setLowStockThreshold(p.low_stock_threshold.toString());
    setImageUrl(p.image_url);
    setIsActive(p.is_active);
    setIsEditModalOpen(true);
  };

  const handleSaveProduct = async (e: React.FormEvent) => {
    e.preventDefault();

    const payload = {
      name,
      sku,
      category_id: Number(categoryId),
      description,
      price: parseFloat(price) || 0,
      stock_quantity: parseInt(stockQuantity, 10) || 0,
      low_stock_threshold: parseInt(lowStockThreshold, 10) || 5,
      image_url: imageUrl,
      is_active: isActive,
    };

    let res;
    if (editingProduct) {
      res = await apiRequest<Product>(`/admin/products/${editingProduct.id}`, {
        method: 'PUT',
        body: JSON.stringify(payload),
      });
    } else {
      res = await apiRequest<Product>('/admin/products', {
        method: 'POST',
        body: JSON.stringify(payload),
      });
    }

    if (res.success) {
      showToast(editingProduct ? 'Product updated' : 'Product created', 'success');
      setIsEditModalOpen(false);
      loadData();
    } else {
      showToast(res.error || 'Failed to save product', 'error');
    }
  };

  const handleDelete = async (id: number) => {
    if (!window.confirm('Are you sure you want to delete this product SKU?')) return;

    const res = await apiRequest(`/admin/products/${id}`, { method: 'DELETE' });
    if (res.success) {
      showToast('Product deleted', 'info');
      loadData();
    } else {
      showToast(res.error || 'Failed to delete product', 'error');
    }
  };

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex flex-wrap items-center justify-between gap-4">
        <div>
          <h2 className="text-xl font-bold text-slate-900">Product & Stock Management</h2>
          <p className="text-xs text-slate-500">Create, edit catalog items and adjust warehouse inventory</p>
        </div>
        <button
          onClick={openCreateModal}
          className="bg-blue-600 hover:bg-blue-700 text-white font-semibold text-xs py-2 px-4 rounded-lg shadow-xs transition-colors cursor-pointer"
        >
          + Add New Product
        </button>
      </div>

      {/* Search Input */}
      <div className="max-w-sm">
        <input
          type="text"
          placeholder="Filter by title or SKU..."
          className="w-full bg-white border border-slate-200 rounded-lg px-3 py-2 text-xs text-slate-900 outline-none focus:border-blue-600"
          value={search}
          onChange={(e) => setSearch(e.target.value)}
        />
      </div>

      {/* Products Table */}
      <div className="bg-white border border-slate-200 rounded-xl overflow-hidden shadow-xs">
        <div className="overflow-x-auto">
          <table className="w-full text-left text-xs text-slate-600">
            <thead className="bg-slate-50 border-b border-slate-200 text-[11px] uppercase font-bold text-slate-400">
              <tr>
                <th className="p-3.5">Product</th>
                <th className="p-3.5">Category</th>
                <th className="p-3.5">Price</th>
                <th className="p-3.5">Stock</th>
                <th className="p-3.5">Status</th>
                <th className="p-3.5 text-right">Actions</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-100">
              {loading ? (
                <tr>
                  <td colSpan={6} className="text-center p-8">
                    Loading inventory catalog...
                  </td>
                </tr>
              ) : products.length === 0 ? (
                <tr>
                  <td colSpan={6} className="text-center p-8 text-slate-400">
                    No products found
                  </td>
                </tr>
              ) : (
                products.map((p) => {
                  const isLow = p.stock_quantity <= p.low_stock_threshold;
                  return (
                    <tr key={p.id} className="hover:bg-slate-50/80 transition-colors">
                      <td className="p-3.5">
                        <div className="flex items-center gap-3">
                          <img
                            src={p.image_url}
                            alt={p.name}
                            className="w-10 h-10 object-contain rounded-lg bg-slate-50 border border-slate-200 p-0.5"
                          />
                          <div>
                            <div className="font-semibold text-slate-900 line-clamp-1">{p.name}</div>
                            <div className="text-[10px] text-slate-400 font-mono">SKU: {p.sku}</div>
                          </div>
                        </div>
                      </td>
                      <td className="p-3.5">{p.category?.name || 'General'}</td>
                      <td className="p-3.5 font-bold text-slate-900">{formatRupiah(p.price)}</td>
                      <td className="p-3.5">
                        <div className="flex items-center gap-1.5">
                          <span
                            className={`font-extrabold ${
                              isLow ? 'text-amber-600' : 'text-slate-900'
                            }`}
                          >
                            {p.stock_quantity}
                          </span>
                          {isLow && (
                            <span className="text-[10px] bg-amber-50 text-amber-700 px-1.5 py-0.5 rounded border border-amber-200">
                              Low
                            </span>
                          )}
                        </div>
                      </td>
                      <td className="p-3.5">
                        <span
                          className={`px-2 py-0.5 rounded-full text-[10px] font-bold ${
                            p.is_active
                              ? 'bg-emerald-50 text-emerald-700 border border-emerald-200'
                              : 'bg-slate-100 text-slate-500 border border-slate-200'
                          }`}
                        >
                          {p.is_active ? 'Active' : 'Inactive'}
                        </span>
                      </td>
                      <td className="p-3.5 text-right space-x-2">
                        <button
                          onClick={() => setSelectedProductForStock(p)}
                          className="bg-purple-50 text-purple-700 hover:bg-purple-100 px-2.5 py-1 rounded text-[11px] font-semibold border border-purple-200 cursor-pointer"
                        >
                          Adjust Stock
                        </button>
                        <button
                          onClick={() => openEditModal(p)}
                          className="text-slate-600 hover:text-slate-900 font-semibold text-[11px] cursor-pointer"
                        >
                          Edit
                        </button>
                        <button
                          onClick={() => handleDelete(p.id)}
                          className="text-rose-600 hover:text-rose-800 font-semibold text-[11px] cursor-pointer"
                        >
                          Delete
                        </button>
                      </td>
                    </tr>
                  );
                })
              )}
            </tbody>
          </table>
        </div>
      </div>

      {/* Stock Adjustment Modal */}
      {selectedProductForStock && (
        <StockAdjustmentModal
          product={selectedProductForStock}
          onClose={() => setSelectedProductForStock(null)}
          onSuccess={() => {
            setSelectedProductForStock(null);
            loadData();
          }}
        />
      )}

      {/* Create / Edit Modal */}
      {isEditModalOpen && (
        <div className="fixed inset-0 bg-slate-900/50 backdrop-blur-xs z-50 flex items-center justify-center p-4">
          <div className="bg-white rounded-2xl w-full max-w-lg p-6 relative max-h-[90vh] overflow-y-auto shadow-xl border border-slate-200 animate-modal">
            <h3 className="font-bold text-base text-slate-900 mb-4">
              {editingProduct ? 'Edit Product SKU' : 'Create New Product'}
            </h3>

            <form onSubmit={handleSaveProduct} className="space-y-3 text-xs">
              <div>
                <label className="font-medium text-slate-700 block mb-1">Product Title</label>
                <input
                  type="text"
                  className="w-full bg-slate-50 border border-slate-200 rounded-lg px-3 py-2 text-slate-900 outline-none focus:border-blue-600 focus:bg-white"
                  value={name}
                  onChange={(e) => setName(e.target.value)}
                  required
                />
              </div>

              <div className="grid grid-cols-2 gap-3">
                <div>
                  <label className="font-medium text-slate-700 block mb-1">SKU</label>
                  <input
                    type="text"
                    className="w-full bg-slate-50 border border-slate-200 rounded-lg px-3 py-2 text-slate-900 outline-none font-mono"
                    value={sku}
                    onChange={(e) => setSku(e.target.value)}
                    required
                  />
                </div>
                <div>
                  <label className="font-medium text-slate-700 block mb-1">Category</label>
                  <select
                    className="w-full bg-slate-50 border border-slate-200 rounded-lg px-2.5 py-2 text-slate-800 outline-none"
                    value={categoryId}
                    onChange={(e) => setCategoryId(Number(e.target.value))}
                  >
                    {categories.map((c) => (
                      <option key={c.id} value={c.id}>
                        {c.name}
                      </option>
                    ))}
                  </select>
                </div>
              </div>

              <div className="grid grid-cols-3 gap-3">
                <div>
                  <label className="font-medium text-slate-700 block mb-1">Price (Rp)</label>
                  <input
                    type="number"
                    step="1000"
                    className="w-full bg-slate-50 border border-slate-200 rounded-lg px-3 py-2 text-slate-900 outline-none"
                    value={price}
                    onChange={(e) => setPrice(e.target.value)}
                    required
                  />
                </div>
                <div>
                  <label className="font-medium text-slate-700 block mb-1">Stock</label>
                  <input
                    type="number"
                    className="w-full bg-slate-50 border border-slate-200 rounded-lg px-3 py-2 text-slate-900 outline-none"
                    value={stockQuantity}
                    onChange={(e) => setStockQuantity(e.target.value)}
                    required
                  />
                </div>
                <div>
                  <label className="font-medium text-slate-700 block mb-1">Low Alert</label>
                  <input
                    type="number"
                    className="w-full bg-slate-50 border border-slate-200 rounded-lg px-3 py-2 text-slate-900 outline-none"
                    value={lowStockThreshold}
                    onChange={(e) => setLowStockThreshold(e.target.value)}
                    required
                  />
                </div>
              </div>

              <div>
                <label className="font-medium text-slate-700 block mb-1">Image URL</label>
                <input
                  type="url"
                  className="w-full bg-slate-50 border border-slate-200 rounded-lg px-3 py-2 text-slate-900 outline-none"
                  value={imageUrl}
                  onChange={(e) => setImageUrl(e.target.value)}
                  required
                />
              </div>

              <div>
                <label className="font-medium text-slate-700 block mb-1">Description</label>
                <textarea
                  className="w-full h-16 bg-slate-50 border border-slate-200 rounded-lg px-3 py-2 text-slate-900 outline-none resize-none"
                  value={description}
                  onChange={(e) => setDescription(e.target.value)}
                  required
                />
              </div>

              <div className="flex items-center gap-2 pt-1">
                <input
                  type="checkbox"
                  id="is_active"
                  className="w-4 h-4 accent-blue-600 rounded cursor-pointer"
                  checked={isActive}
                  onChange={(e) => setIsActive(e.target.checked)}
                />
                <label htmlFor="is_active" className="text-slate-700 select-none cursor-pointer">
                  Item is active and visible in storefront catalog
                </label>
              </div>

              <div className="flex justify-end gap-2 pt-4 border-t border-slate-100">
                <button
                  type="button"
                  onClick={() => setIsEditModalOpen(false)}
                  className="bg-slate-100 hover:bg-slate-200 text-slate-700 font-semibold px-4 py-2 rounded-lg cursor-pointer"
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  className="bg-blue-600 hover:bg-blue-700 text-white font-semibold px-4 py-2 rounded-lg cursor-pointer"
                >
                  Save Product
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
};
