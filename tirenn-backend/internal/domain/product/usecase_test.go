package product

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"gocommerce-backend/internal/utils"
)

type mockProductRepo struct {
	products      map[uint]*Product
	categories    map[uint]*Category
	subCategories map[uint]*SubCategory
	stockLogs     []StockAdjustmentLog
}

func newMockProductRepo() *mockProductRepo {
	return &mockProductRepo{
		products:      make(map[uint]*Product),
		categories:    make(map[uint]*Category),
		subCategories: make(map[uint]*SubCategory),
	}
}

func (m *mockProductRepo) Create(ctx context.Context, p *Product) error {
	if p.ID == 0 {
		p.ID = uint(len(m.products) + 1)
	}
	m.products[p.ID] = p
	return nil
}

func (m *mockProductRepo) FindByID(ctx context.Context, id uint) (*Product, error) {
	if p, ok := m.products[id]; ok {
		return p, nil
	}
	return nil, errors.New("record not found")
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
	return nil, errors.New("record not found")
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
	return nil, errors.New("record not found")
}

func (m *mockProductRepo) FindCategoryBySlug(ctx context.Context, slug string) (*Category, error) {
	for _, c := range m.categories {
		if c.Slug == slug {
			return c, nil
		}
	}
	return nil, errors.New("record not found")
}

func (m *mockProductRepo) CreateSubCategory(ctx context.Context, sc *SubCategory) error {
	sc.ID = uint(len(m.subCategories) + 1)
	m.subCategories[sc.ID] = sc
	return nil
}

func (m *mockProductRepo) ListSubCategories(ctx context.Context, categoryID uint) ([]SubCategory, error) {
	var list []SubCategory
	for _, sc := range m.subCategories {
		if categoryID == 0 || sc.CategoryID == categoryID {
			list = append(list, *sc)
		}
	}
	return list, nil
}

func (m *mockProductRepo) FindSubCategoryByID(ctx context.Context, id uint) (*SubCategory, error) {
	if sc, ok := m.subCategories[id]; ok {
		return sc, nil
	}
	return nil, errors.New("record not found")
}

func (m *mockProductRepo) FindSubCategoryBySlug(ctx context.Context, slug string) (*SubCategory, error) {
	for _, sc := range m.subCategories {
		if sc.Slug == slug {
			return sc, nil
		}
	}
	return nil, errors.New("record not found")
}

func (m *mockProductRepo) GetCategoryTopSellers(ctx context.Context, categoryID uint, excludeID uint, limit int) ([]Product, error) {
	var list []Product
	for _, p := range m.products {
		if p.CategoryID == categoryID && p.ID != excludeID && p.IsActive {
			list = append(list, *p)
			if len(list) >= limit {
				break
			}
		}
	}
	return list, nil
}

func (m *mockProductRepo) GetOverallTopSellers(ctx context.Context, excludeID uint, limit int) ([]Product, error) {
	var list []Product
	for _, p := range m.products {
		if p.ID != excludeID && p.IsActive {
			list = append(list, *p)
			if len(list) >= limit {
				break
			}
		}
	}
	return list, nil
}

type mockAIClient struct {
	getRecommendationsFn func(ctx context.Context, productID uint, limit int) ([]uint, error)
	searchSemanticFn     func(ctx context.Context, query string, categoryID uint, limit int) ([]uint, error)
	syncProductsFn       func(ctx context.Context, products []Product) error
}

func (m *mockAIClient) GetRecommendations(ctx context.Context, productID uint, limit int) ([]uint, error) {
	if m.getRecommendationsFn != nil {
		return m.getRecommendationsFn(ctx, productID, limit)
	}
	return nil, nil
}

func (m *mockAIClient) SearchSemantic(ctx context.Context, query string, categoryID uint, limit int) ([]uint, error) {
	if m.searchSemanticFn != nil {
		return m.searchSemanticFn(ctx, query, categoryID, limit)
	}
	return nil, nil
}

func (m *mockAIClient) SyncProducts(ctx context.Context, products []Product) error {
	if m.syncProductsFn != nil {
		return m.syncProductsFn(ctx, products)
	}
	return nil
}

func TestProductUseCase_AdjustStock(t *testing.T) {
	repo := newMockProductRepo()
	uc := NewUseCase(repo, nil, nil)

	p := &Product{
		ID:            1,
		Name:          "Headphone Nirkabel AuraPro",
		Price:         1499000,
		StockQuantity: 10,
		IsActive:      true,
	}
	repo.products[1] = p

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

	_, err = uc.AdjustStock(ctx, 1, &StockAdjustRequest{
		Type:   "SUBTRACT",
		Amount: 100,
		Reason: "Invalid subtraction",
	}, 1)
	if err == nil {
		t.Error("Expected error for subtracting more than available stock, got nil")
	}
}

func TestProductUseCase_GetRecommendations_AISuccess(t *testing.T) {
	repo := newMockProductRepo()
	repo.products[1] = &Product{ID: 1, CategoryID: 1, Name: "Base Product", IsActive: true}
	repo.products[2] = &Product{ID: 2, CategoryID: 1, Name: "Rec 1", IsActive: true}
	repo.products[3] = &Product{ID: 3, CategoryID: 1, Name: "Rec 2", IsActive: true}
	repo.products[4] = &Product{ID: 4, CategoryID: 2, Name: "Rec 3", IsActive: true}
	repo.products[5] = &Product{ID: 5, CategoryID: 2, Name: "Rec 4", IsActive: true}

	aiMock := &mockAIClient{
		getRecommendationsFn: func(ctx context.Context, productID uint, limit int) ([]uint, error) {
			return []uint{2, 3, 4, 5}, nil
		},
	}

	uc := NewUseCase(repo, aiMock, nil)
	recs, err := uc.GetRecommendations(context.Background(), 1, 4)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if len(recs) != 4 {
		t.Fatalf("Expected 4 recommendations, got %d", len(recs))
	}
	if recs[0].ID != 2 || recs[1].ID != 3 || recs[2].ID != 4 || recs[3].ID != 5 {
		t.Errorf("Unexpected recommendation order: %+v", recs)
	}
}

func TestProductUseCase_GetRecommendations_AIFallbackToCategory(t *testing.T) {
	repo := newMockProductRepo()
	repo.products[1] = &Product{ID: 1, CategoryID: 1, Name: "Base Product", IsActive: true}
	repo.products[2] = &Product{ID: 2, CategoryID: 1, Name: "Category Item 1", IsActive: true, Badge: "Terlaris", Rating: 4.9}
	repo.products[3] = &Product{ID: 3, CategoryID: 1, Name: "Category Item 2", IsActive: true, Badge: "Best Seller", Rating: 4.8}
	repo.products[4] = &Product{ID: 4, CategoryID: 1, Name: "Category Item 3", IsActive: true, Rating: 4.7}
	repo.products[5] = &Product{ID: 5, CategoryID: 1, Name: "Category Item 4", IsActive: true, Rating: 4.6}

	aiMock := &mockAIClient{
		getRecommendationsFn: func(ctx context.Context, productID uint, limit int) ([]uint, error) {
			return nil, errors.New("ai service unavailable")
		},
	}

	uc := NewUseCase(repo, aiMock, nil)
	recs, err := uc.GetRecommendations(context.Background(), 1, 4)
	if err != nil {
		t.Fatalf("Expected no error on fallback, got %v", err)
	}
	if len(recs) != 4 {
		t.Fatalf("Expected 4 recommendations from fallback, got %d", len(recs))
	}
	for _, r := range recs {
		if r.ID == 1 {
			t.Errorf("Fallback recommendations should exclude base product ID 1")
		}
		if r.CategoryID != 1 {
			t.Errorf("Fallback recommendations should match category 1, got category %d", r.CategoryID)
		}
	}
}

func TestProductUseCase_GetRecommendations_AIFallbackSupplementOverall(t *testing.T) {
	repo := newMockProductRepo()
	repo.products[1] = &Product{ID: 1, CategoryID: 1, Name: "Base Product", IsActive: true}
	// Only 1 item in same category
	repo.products[2] = &Product{ID: 2, CategoryID: 1, Name: "Category Item 1", IsActive: true}
	// Other categories
	repo.products[10] = &Product{ID: 10, CategoryID: 2, Name: "Overall Item 1", IsActive: true}
	repo.products[11] = &Product{ID: 11, CategoryID: 2, Name: "Overall Item 2", IsActive: true}
	repo.products[12] = &Product{ID: 12, CategoryID: 3, Name: "Overall Item 3", IsActive: true}

	aiMock := &mockAIClient{
		getRecommendationsFn: func(ctx context.Context, productID uint, limit int) ([]uint, error) {
			return nil, errors.New("ai service timeout")
		},
	}

	uc := NewUseCase(repo, aiMock, nil)
	recs, err := uc.GetRecommendations(context.Background(), 1, 4)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if len(recs) != 4 {
		t.Fatalf("Expected 4 supplemented recommendations, got %d", len(recs))
	}

	// Verify no duplicates
	seen := make(map[uint]bool)
	for _, r := range recs {
		if seen[r.ID] {
			t.Errorf("Duplicate recommendation ID found: %d", r.ID)
		}
		seen[r.ID] = true
		if r.ID == 1 {
			t.Errorf("Recommendations should not contain target product ID 1")
		}
	}
}

func TestProductUseCase_GetRecommendations_ProductNotFound(t *testing.T) {
	repo := newMockProductRepo()
	uc := NewUseCase(repo, nil, nil)

	_, err := uc.GetRecommendations(context.Background(), 9999, 6)
	if err == nil {
		t.Fatalf("Expected error for non-existent product, got nil")
	}
	if err.Error() != "product not found" {
		t.Errorf("Expected 'product not found' error, got %v", err)
	}
}

func TestProductUseCase_GetRecommendations_LimitClamping(t *testing.T) {
	repo := newMockProductRepo()
	repo.products[1] = &Product{ID: 1, CategoryID: 1, Name: "Base Product", IsActive: true}
	for i := uint(2); i <= 15; i++ {
		repo.products[i] = &Product{ID: i, CategoryID: 1, Name: "Product", IsActive: true}
	}

	uc := NewUseCase(repo, nil, nil)

	// Under-min limit (2 -> clamped to 4)
	recs1, err := uc.GetRecommendations(context.Background(), 1, 2)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(recs1) != 4 {
		t.Errorf("Expected limit clamped to 4, got %d items", len(recs1))
	}

	// Over-max limit (12 -> clamped to 8)
	recs2, err := uc.GetRecommendations(context.Background(), 1, 12)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(recs2) != 8 {
		t.Errorf("Expected limit clamped to 8, got %d items", len(recs2))
	}
}

func TestAIClient_GetRecommendations(t *testing.T) {
	// 1. Success case with API key validation
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/products/42/recommendations" {
			t.Errorf("Unexpected path: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if r.URL.Query().Get("limit") != "6" {
			t.Errorf("Unexpected limit query: %s", r.URL.Query().Get("limit"))
		}
		if r.Header.Get("X-API-Key") != "test-api-key" {
			t.Errorf("Missing or invalid X-API-Key header: %s", r.Header.Get("X-API-Key"))
		}

		resp := recommendationResp{
			Success:   true,
			ProductID: 42,
			Recommendations: []recommendationItem{
				{ID: 101, Score: 0.95, Reason: "similar_embedding"},
				{ID: 102, Score: 0.89, Reason: "category_match"},
			},
			Total: 2,
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewAIClient(server.URL, "test-api-key")
	ids, err := client.GetRecommendations(context.Background(), 42, 6)
	if err != nil {
		t.Fatalf("Expected no error from AIClient, got %v", err)
	}
	if len(ids) != 2 || ids[0] != 101 || ids[1] != 102 {
		t.Errorf("Unexpected IDs returned: %v", ids)
	}

	// 2. Unconfigured URL error
	emptyClient := NewAIClient("", "")
	_, err = emptyClient.GetRecommendations(context.Background(), 42, 6)
	if err == nil {
		t.Errorf("Expected error for empty AI service URL, got nil")
	}

	// 3. HTTP 500 error from server
	errServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer errServer.Close()

	errClient := NewAIClient(errServer.URL, "")
	_, err = errClient.GetRecommendations(context.Background(), 42, 6)
	if err == nil {
		t.Errorf("Expected error for HTTP 500 from AI service, got nil")
	}
}

func TestHandler_GetRecommendations(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := newMockProductRepo()
	repo.products[1] = &Product{ID: 1, CategoryID: 1, Name: "Test Phone", Price: 5000000, Currency: "IDR", IsActive: true}
	repo.products[2] = &Product{ID: 2, CategoryID: 1, Name: "Phone Case", Price: 150000, Currency: "IDR", IsActive: true}
	repo.products[3] = &Product{ID: 3, CategoryID: 1, Name: "Screen Protector", Price: 50000, Currency: "IDR", IsActive: true}
	repo.products[4] = &Product{ID: 4, CategoryID: 1, Name: "Charger", Price: 250000, Currency: "IDR", IsActive: true}
	repo.products[5] = &Product{ID: 5, CategoryID: 1, Name: "Wireless Earbuds", Price: 750000, Currency: "IDR", IsActive: true}

	uc := NewUseCase(repo, nil, nil)
	h := NewHandler(uc)

	r := gin.New()
	r.GET("/api/v1/products/:id/recommendations", h.GetRecommendations)

	// 1. Success request
	req := httptest.NewRequest(http.MethodGet, "/api/v1/products/1/recommendations?limit=4", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d. Body: %s", w.Code, w.Body.String())
	}

	var resp utils.APIResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to parse JSON response: %v", err)
	}
	if !resp.Success {
		t.Errorf("Expected success=true, got %v", resp.Success)
	}
	if resp.Message != "Recommendations retrieved successfully" {
		t.Errorf("Unexpected message: %s", resp.Message)
	}

	// 2. Invalid ID request
	reqBad := httptest.NewRequest(http.MethodGet, "/api/v1/products/invalid-id/recommendations", nil)
	wBad := httptest.NewRecorder()
	r.ServeHTTP(wBad, reqBad)
	if wBad.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for invalid ID, got %d", wBad.Code)
	}

	// 3. Not Found request
	req404 := httptest.NewRequest(http.MethodGet, "/api/v1/products/9999/recommendations", nil)
	w404 := httptest.NewRecorder()
	r.ServeHTTP(w404, req404)
	if w404.Code != http.StatusNotFound {
		t.Errorf("Expected status 404 for not found product, got %d", w404.Code)
	}
}
