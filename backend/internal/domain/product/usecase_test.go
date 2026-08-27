package product

import (
	"context"
	"testing"
)

type mockProductRepo struct {
	products   map[uint]*Product
	categories map[uint]*Category
	stockLogs  []StockAdjustmentLog
}

func newMockProductRepo() *mockProductRepo {
	return &mockProductRepo{
		products:   make(map[uint]*Product),
		categories: make(map[uint]*Category),
	}
}

func (m *mockProductRepo) Create(ctx context.Context, p *Product) error {
	p.ID = uint(len(m.products) + 1)
	m.products[p.ID] = p
	return nil
}
func (m *mockProductRepo) FindByID(ctx context.Context, id uint) (*Product, error) {
	if p, ok := m.products[id]; ok {
		return p, nil
	}
	return nil, nil
}
func (m *mockProductRepo) FindByIDs(ctx context.Context, ids []uint) ([]Product, error) {
	var list []Product
	for _, id := range ids {
		if p, ok := m.products[id]; ok {
			list = append(list, *p)
		}
	}
	return list, nil
}
func (m *mockProductRepo) FindBySlug(ctx context.Context, slug string) (*Product, error) {
	for _, p := range m.products {
		if p.Slug == slug {
			return p, nil
		}
	}
	return nil, nil
}
func (m *mockProductRepo) Update(ctx context.Context, p *Product) error {
	m.products[p.ID] = p
	return nil
}
func (m *mockProductRepo) Delete(ctx context.Context, id uint) error {
	delete(m.products, id)
	return nil
}
func (m *mockProductRepo) ListAll(ctx context.Context) ([]Product, error) {
	var list []Product
	for _, p := range m.products {
		list = append(list, *p)
	}
	return list, nil
}
func (m *mockProductRepo) List(ctx context.Context, filter ProductFilterQuery) ([]Product, int64, error) {
	var list []Product
	for _, p := range m.products {
		list = append(list, *p)
	}
	return list, int64(len(list)), nil
}
func (m *mockProductRepo) AdjustStock(ctx context.Context, log *StockAdjustmentLog) error {
	if p, ok := m.products[log.ProductID]; ok {
		p.StockQuantity = log.NewStock
		m.stockLogs = append(m.stockLogs, *log)
	}
	return nil
}
func (m *mockProductRepo) GetStockLogs(ctx context.Context, productID uint) ([]StockAdjustmentLog, error) {
	return m.stockLogs, nil
}
func (m *mockProductRepo) CreateCategory(ctx context.Context, c *Category) error {
	c.ID = uint(len(m.categories) + 1)
	m.categories[c.ID] = c
	return nil
}
func (m *mockProductRepo) ListCategories(ctx context.Context) ([]Category, error) {
	var list []Category
	for _, c := range m.categories {
		list = append(list, *c)
	}
	return list, nil
}
func (m *mockProductRepo) FindCategoryByID(ctx context.Context, id uint) (*Category, error) {
	if c, ok := m.categories[id]; ok {
		return c, nil
	}
	return nil, nil
}
func (m *mockProductRepo) FindCategoryBySlug(ctx context.Context, slug string) (*Category, error) {
	for _, c := range m.categories {
		if c.Slug == slug {
			return c, nil
		}
	}
	return nil, nil
}

func TestProductUseCase_AdjustStock(t *testing.T) {
	repo := newMockProductRepo()
	uc := NewUseCase(repo, nil)

	// Setup initial product
	p := &Product{
		ID:            1,
		Name:          "Headphone Nirkabel AuraPro",
		Price:         1499000,
		StockQuantity: 10,
	}
	repo.products[1] = p

	// Test ADD
	ctx := context.Background()
	updated, err := uc.AdjustStock(ctx, 1, &StockAdjustRequest{
		Type:   "ADD",
		Amount: 5,
		Reason: "Restock Shipment",
	}, 1)

	if err != nil {
		t.Fatalf("AdjustStock ADD failed: %v", err)
	}
	if updated.StockQuantity != 15 {
		t.Errorf("Expected stock to be 15, got %d", updated.StockQuantity)
	}

	// Test SUBTRACT
	updated, err = uc.AdjustStock(ctx, 1, &StockAdjustRequest{
		Type:   "SUBTRACT",
		Amount: 3,
		Reason: "Damaged box",
	}, 1)
	if err != nil {
		t.Fatalf("AdjustStock SUBTRACT failed: %v", err)
	}
	if updated.StockQuantity != 12 {
		t.Errorf("Expected stock to be 12, got %d", updated.StockQuantity)
	}

	// Test Over-SUBTRACT (should error)
	_, err = uc.AdjustStock(ctx, 1, &StockAdjustRequest{
		Type:   "SUBTRACT",
		Amount: 100,
		Reason: "Invalid subtraction",
	}, 1)
	if err == nil {
		t.Error("Expected error for subtracting more than available stock, got nil")
	}
}
