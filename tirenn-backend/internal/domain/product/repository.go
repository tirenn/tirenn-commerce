package product

import (
	"context"
	"strings"

	"gorm.io/gorm"
)

type Repository interface {
	// Product operations
	Create(ctx context.Context, p *Product) error
	FindByID(ctx context.Context, id uint) (*Product, error)
	FindByIDs(ctx context.Context, ids []uint) ([]Product, error)
	FindBySlug(ctx context.Context, slug string) (*Product, error)
	Update(ctx context.Context, p *Product) error
	Delete(ctx context.Context, id uint) error
	List(ctx context.Context, filter ProductFilterQuery) ([]Product, int64, error)
	ListAll(ctx context.Context) ([]Product, error)

	// Stock adjustment
	AdjustStock(ctx context.Context, log *StockAdjustmentLog) error
	GetStockLogs(ctx context.Context, productID uint) ([]StockAdjustmentLog, error)

	// Category operations
	CreateCategory(ctx context.Context, c *Category) error
	ListCategories(ctx context.Context) ([]Category, error)
	FindCategoryByID(ctx context.Context, id uint) (*Category, error)
	FindCategoryBySlug(ctx context.Context, slug string) (*Category, error)

	// SubCategory operations
	CreateSubCategory(ctx context.Context, sc *SubCategory) error
	ListSubCategories(ctx context.Context, categoryID uint) ([]SubCategory, error)
	FindSubCategoryByID(ctx context.Context, id uint) (*SubCategory, error)
	FindSubCategoryBySlug(ctx context.Context, slug string) (*SubCategory, error)

	// Top sellers / recommendations fallback
	GetCategoryTopSellers(ctx context.Context, categoryID uint, excludeID uint, limit int) ([]Product, error)
	GetOverallTopSellers(ctx context.Context, excludeID uint, limit int) ([]Product, error)
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

func (r *repository) Create(ctx context.Context, p *Product) error {
	return r.db.WithContext(ctx).Create(p).Error
}

func (r *repository) FindByID(ctx context.Context, id uint) (*Product, error) {
	var p Product
	err := r.db.WithContext(ctx).Preload("Category").Preload("SubCategory").First(&p, id).Error
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *repository) FindByIDs(ctx context.Context, ids []uint) ([]Product, error) {
	if len(ids) == 0 {
		return []Product{}, nil
	}
	var products []Product
	err := r.db.WithContext(ctx).Preload("Category").Preload("SubCategory").Where("id IN ?", ids).Find(&products).Error
	if err != nil {
		return nil, err
	}

	// Preserve ordering from input IDs
	productMap := make(map[uint]Product)
	for _, p := range products {
		productMap[p.ID] = p
	}

	var ordered []Product
	for _, id := range ids {
		if p, ok := productMap[id]; ok {
			ordered = append(ordered, p)
		}
	}
	return ordered, nil
}

func (r *repository) FindBySlug(ctx context.Context, slug string) (*Product, error) {
	var p Product
	err := r.db.WithContext(ctx).Preload("Category").Preload("SubCategory").Where("slug = ?", slug).First(&p).Error
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *repository) Update(ctx context.Context, p *Product) error {
	return r.db.WithContext(ctx).Save(p).Error
}

func (r *repository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&Product{}, id).Error
}

func (r *repository) ListAll(ctx context.Context) ([]Product, error) {
	var products []Product
	err := r.db.WithContext(ctx).Preload("Category").Preload("SubCategory").Where("is_active = ?", true).Find(&products).Error
	return products, err
}

func (r *repository) List(ctx context.Context, filter ProductFilterQuery) ([]Product, int64, error) {
	var products []Product
	var total int64

	query := r.db.WithContext(ctx).Model(&Product{}).
		Joins("LEFT JOIN categories ON categories.id = products.category_id").
		Joins("LEFT JOIN sub_categories ON sub_categories.id = products.sub_category_id").
		Preload("Category").
		Preload("SubCategory")

	if !filter.IsAdmin {
		query = query.Where("products.is_active = ?", true)
	}

	// PostgreSQL Case-Insensitive ILIKE Substring Search
	if filter.Search != "" {
		cleanSearch := strings.TrimSpace(filter.Search)
		tokens := strings.Fields(cleanSearch)
		for _, token := range tokens {
			pattern := "%" + token + "%"
			query = query.Where(
				"(products.name ILIKE ? OR products.description ILIKE ? OR products.sku ILIKE ? OR categories.name ILIKE ? OR sub_categories.name ILIKE ?)",
				pattern, pattern, pattern, pattern, pattern,
			)
		}
	}

	if filter.CategoryID > 0 {
		query = query.Where("products.category_id = ?", filter.CategoryID)
	}

	if filter.SubCategoryID > 0 {
		query = query.Where("products.sub_category_id = ?", filter.SubCategoryID)
	}

	if filter.MinPrice > 0 {
		query = query.Where("products.price >= ?", filter.MinPrice)
	}

	if filter.MaxPrice > 0 {
		query = query.Where("products.price <= ?", filter.MaxPrice)
	}

	if filter.InStock != nil {
		if *filter.InStock {
			query = query.Where("products.stock_quantity > 0")
		} else {
			query = query.Where("products.stock_quantity = 0")
		}
	}

	// Count total records matching criteria
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Apply Sorting
	switch filter.Sort {
	case "price_asc":
		query = query.Order("products.price ASC")
	case "price_desc":
		query = query.Order("products.price DESC")
	case "name_asc":
		query = query.Order("products.name ASC")
	case "newest":
		query = query.Order("products.id ASC")
	default:
		query = query.Order("products.id ASC")
	}

	// Pagination
	page := filter.Page
	if page < 1 {
		page = 1
	}
	limit := filter.Limit
	if limit < 1 {
		limit = 12
	}
	offset := (page - 1) * limit

	err := query.Offset(offset).Limit(limit).Find(&products).Error
	if err != nil {
		return nil, 0, err
	}

	return products, total, nil
}

func (r *repository) AdjustStock(ctx context.Context, log *StockAdjustmentLog) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Update product stock quantity
		if err := tx.Model(&Product{}).Where("id = ?", log.ProductID).
			Update("stock_quantity", log.NewStock).Error; err != nil {
			return err
		}

		// Insert audit adjustment log
		if err := tx.Create(log).Error; err != nil {
			return err
		}

		return nil
	})
}

func (r *repository) GetStockLogs(ctx context.Context, productID uint) ([]StockAdjustmentLog, error) {
	var logs []StockAdjustmentLog
	err := r.db.WithContext(ctx).Where("product_id = ?", productID).
		Order("created_at DESC").Find(&logs).Error
	return logs, err
}

func (r *repository) CreateCategory(ctx context.Context, c *Category) error {
	return r.db.WithContext(ctx).Create(c).Error
}

func (r *repository) ListCategories(ctx context.Context) ([]Category, error) {
	var categories []Category
	err := r.db.WithContext(ctx).Preload("SubCategories").Order("id ASC").Find(&categories).Error
	return categories, err
}

func (r *repository) FindCategoryByID(ctx context.Context, id uint) (*Category, error) {
	var c Category
	err := r.db.WithContext(ctx).Preload("SubCategories").First(&c, id).Error
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *repository) FindCategoryBySlug(ctx context.Context, slug string) (*Category, error) {
	var c Category
	err := r.db.WithContext(ctx).Preload("SubCategories").Where("slug = ?", slug).First(&c).Error
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *repository) CreateSubCategory(ctx context.Context, sc *SubCategory) error {
	return r.db.WithContext(ctx).Create(sc).Error
}

func (r *repository) ListSubCategories(ctx context.Context, categoryID uint) ([]SubCategory, error) {
	var subCategories []SubCategory
	query := r.db.WithContext(ctx).Order("id ASC")
	if categoryID > 0 {
		query = query.Where("category_id = ?", categoryID)
	}
	err := query.Find(&subCategories).Error
	return subCategories, err
}

func (r *repository) FindSubCategoryByID(ctx context.Context, id uint) (*SubCategory, error) {
	var sc SubCategory
	err := r.db.WithContext(ctx).Preload("Category").First(&sc, id).Error
	if err != nil {
		return nil, err
	}
	return &sc, nil
}

func (r *repository) FindSubCategoryBySlug(ctx context.Context, slug string) (*SubCategory, error) {
	var sc SubCategory
	err := r.db.WithContext(ctx).Preload("Category").Where("slug = ?", slug).First(&sc).Error
	if err != nil {
		return nil, err
	}
	return &sc, nil
}

func (r *repository) GetCategoryTopSellers(ctx context.Context, categoryID uint, excludeID uint, limit int) ([]Product, error) {
	var products []Product
	err := r.db.WithContext(ctx).
		Preload("Category").
		Preload("SubCategory").
		Where("category_id = ? AND id != ? AND is_active = ?", categoryID, excludeID, true).
		Order("CASE WHEN badge ILIKE '%Terlaris%' OR badge ILIKE '%Best Seller%' THEN 0 ELSE 1 END ASC, rating DESC, id ASC").
		Limit(limit).
		Find(&products).Error
	return products, err
}

func (r *repository) GetOverallTopSellers(ctx context.Context, excludeID uint, limit int) ([]Product, error) {
	var products []Product
	err := r.db.WithContext(ctx).
		Preload("Category").
		Preload("SubCategory").
		Where("id != ? AND is_active = ?", excludeID, true).
		Order("CASE WHEN badge ILIKE '%Terlaris%' OR badge ILIKE '%Best Seller%' THEN 0 ELSE 1 END ASC, rating DESC, id ASC").
		Limit(limit).
		Find(&products).Error
	return products, err
}
