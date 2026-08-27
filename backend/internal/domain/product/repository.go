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
	FindBySlug(ctx context.Context, slug string) (*Product, error)
	Update(ctx context.Context, p *Product) error
	Delete(ctx context.Context, id uint) error
	List(ctx context.Context, filter ProductFilterQuery) ([]Product, int64, error)

	// Stock adjustment
	AdjustStock(ctx context.Context, log *StockAdjustmentLog) error
	GetStockLogs(ctx context.Context, productID uint) ([]StockAdjustmentLog, error)

	// Category operations
	CreateCategory(ctx context.Context, c *Category) error
	ListCategories(ctx context.Context) ([]Category, error)
	FindCategoryByID(ctx context.Context, id uint) (*Category, error)
	FindCategoryBySlug(ctx context.Context, slug string) (*Category, error)
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
	err := r.db.WithContext(ctx).Preload("Category").First(&p, id).Error
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *repository) FindBySlug(ctx context.Context, slug string) (*Product, error) {
	var p Product
	err := r.db.WithContext(ctx).Preload("Category").Where("slug = ?", slug).First(&p).Error
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

// formatBooleanFullText parses user search terms into MySQL Full-Text boolean search syntax
// e.g. "wireless headphones" -> "+wireless* +headphones*", "TECH-AP-001" -> "+TECH* +AP* +001*"
func formatBooleanFullText(input string) string {
	replacer := strings.NewReplacer("-", " ", "_", " ", "/", " ", ".", " ")
	cleanedInput := replacer.Replace(input)
	words := strings.Fields(cleanedInput)
	if len(words) == 0 {
		return ""
	}
	var formatted []string
	for _, w := range words {
		clean := strings.Map(func(r rune) rune {
			if strings.ContainsRune("+-~*<>\"()@", r) {
				return -1
			}
			return r
		}, w)
		if len(clean) > 0 {
			formatted = append(formatted, "+"+clean+"*")
		}
	}
	return strings.Join(formatted, " ")
}

func (r *repository) List(ctx context.Context, filter ProductFilterQuery) ([]Product, int64, error) {
	var products []Product
	var total int64

	query := r.db.WithContext(ctx).Model(&Product{}).
		Joins("LEFT JOIN categories ON categories.id = products.category_id").
		Preload("Category")

	if !filter.IsAdmin {
		query = query.Where("products.is_active = ?", true)
	}

	// Pure Full-Text Search across Products (name, description, sku) and Categories (name, description)
	if filter.Search != "" {
		cleanSearch := strings.TrimSpace(filter.Search)
		ftsQuery := formatBooleanFullText(cleanSearch)
		if ftsQuery != "" {
			query = query.Where(
				"MATCH(products.name, products.description, products.sku) AGAINST (? IN BOOLEAN MODE) OR MATCH(categories.name, categories.description) AGAINST (? IN BOOLEAN MODE)",
				ftsQuery, ftsQuery,
			)
		}
	}

	if filter.CategoryID > 0 {
		query = query.Where("products.category_id = ?", filter.CategoryID)
	}

	if filter.MinPrice > 0 {
		query = query.Where("products.price >= ?", filter.MinPrice)
	}

	if filter.MaxPrice > 0 {
		query = query.Where("products.price <= ?", filter.MaxPrice)
	}

	if filter.InStock != nil && *filter.InStock {
		query = query.Where("products.stock_quantity > 0")
	}

	// Count total matching
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Sorting
	switch filter.Sort {
	case "price_asc":
		query = query.Order("products.price ASC")
	case "price_desc":
		query = query.Order("products.price DESC")
	case "name_asc":
		query = query.Order("products.name ASC")
	case "rating_desc":
		query = query.Order("products.rating DESC")
	default:
		query = query.Order("products.created_at DESC")
	}

	// Pagination
	page := filter.Page
	if page < 1 {
		page = 1
	}
	limit := filter.Limit
	if limit < 1 || limit > 100 {
		limit = 12
	}
	offset := (page - 1) * limit

	err := query.Offset(offset).Limit(limit).Find(&products).Error
	return products, total, err
}

func (r *repository) AdjustStock(ctx context.Context, log *StockAdjustmentLog) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Update product stock
		if err := tx.Model(&Product{}).Where("id = ?", log.ProductID).Update("stock_quantity", log.NewStock).Error; err != nil {
			return err
		}
		// Save log
		return tx.Create(log).Error
	})
}

func (r *repository) GetStockLogs(ctx context.Context, productID uint) ([]StockAdjustmentLog, error) {
	var logs []StockAdjustmentLog
	err := r.db.WithContext(ctx).Where("product_id = ?", productID).Order("created_at DESC").Find(&logs).Error
	return logs, err
}

func (r *repository) CreateCategory(ctx context.Context, c *Category) error {
	return r.db.WithContext(ctx).Create(c).Error
}

func (r *repository) ListCategories(ctx context.Context) ([]Category, error) {
	var categories []Category
	err := r.db.WithContext(ctx).Order("name ASC").Find(&categories).Error
	return categories, err
}

func (r *repository) FindCategoryByID(ctx context.Context, id uint) (*Category, error) {
	var c Category
	err := r.db.WithContext(ctx).First(&c, id).Error
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *repository) FindCategoryBySlug(ctx context.Context, slug string) (*Category, error) {
	var c Category
	err := r.db.WithContext(ctx).Where("slug = ?", slug).First(&c).Error
	if err != nil {
		return nil, err
	}
	return &c, nil
}
