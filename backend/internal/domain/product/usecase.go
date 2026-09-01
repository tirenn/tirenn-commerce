package product

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/pgvector/pgvector-go"
	"github.com/redis/go-redis/v9"
	"tirenn-ai-commerce/internal/client/ollama"
	"tirenn-ai-commerce/internal/domain"
	"tirenn-ai-commerce/internal/logger"
	"tirenn-ai-commerce/internal/security"
)

type UseCase interface {
	CreateProduct(ctx context.Context, req *CreateProductRequest) (*Product, error)
	GetProductByID(ctx context.Context, id uint) (*Product, error)
	GetProductBySlug(ctx context.Context, slug string) (*Product, error)
	UpdateProduct(ctx context.Context, id uint, req *UpdateProductRequest) (*Product, error)
	DeleteProduct(ctx context.Context, id uint) error
	ListProducts(ctx context.Context, filter ProductFilterQuery) ([]Product, int64, error)
	GetRecommendations(ctx context.Context, productID uint, limit int) ([]Product, error)

	AdjustStock(ctx context.Context, productID uint, req *StockAdjustRequest, adminID uint) (*Product, error)
	GetStockLogs(ctx context.Context, productID uint) ([]StockAdjustmentLog, error)

	CreateCategory(ctx context.Context, req *CreateCategoryRequest) (*Category, error)
	ListCategories(ctx context.Context) ([]Category, error)

	CreateSubCategory(ctx context.Context, req *CreateSubCategoryRequest) (*SubCategory, error)
	ListSubCategories(ctx context.Context, categoryID uint) ([]SubCategory, error)
}

type useCase struct {
	repo         Repository
	rdb          *redis.Client
	ollamaClient *ollama.Client
}

func NewUseCase(repo Repository, rdb *redis.Client, ollamaClients ...*ollama.Client) UseCase {
	var ollamaClient *ollama.Client
	if len(ollamaClients) > 0 {
		ollamaClient = ollamaClients[0]
	}

	return &useCase{
		repo:         repo,
		rdb:          rdb,
		ollamaClient: ollamaClient,
	}
}

func (u *useCase) CreateProduct(ctx context.Context, req *CreateProductRequest) (*Product, error) {
	// Verify category exists
	cat, err := u.repo.FindCategoryByID(ctx, req.CategoryID)
	if err != nil {
		logger.Error(ctx, "usecase", "selected category does not exist", err)
		return nil, domain.ErrNotFound
	}

	slug := security.Slugify(req.Name)
	existing, _ := u.repo.FindBySlug(ctx, slug)
	if existing != nil {
		slug = fmt.Sprintf("%s-%d", slug, time.Now().Unix())
	}

	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	currency := "IDR"
	if req.Currency != "" {
		currency = req.Currency
	}

	lowStock := 5
	if req.LowStockThreshold > 0 {
		lowStock = req.LowStockThreshold
	}

	product := &Product{
		CategoryID:        cat.ID,
		SubCategoryID:     req.SubCategoryID,
		Name:              req.Name,
		Slug:              slug,
		SKU:               req.SKU,
		Description:       req.Description,
		Price:             req.Price,
		Currency:          currency,
		StockQuantity:     req.StockQuantity,
		LowStockThreshold: lowStock,
		ImageURL:          req.ImageURL,
		IsActive:          isActive,
		Badge:             req.Badge,
		Rating:            5.0,
	}

	// Automatically calculate pgvector 384-dimensional dense embedding
	if u.ollamaClient != nil {
		textToEmbed := fmt.Sprintf("%s. %s", product.Name, product.Description)
		if vec, err := u.ollamaClient.GenerateEmbedding(ctx, textToEmbed); err == nil && len(vec) > 0 {
			product.Embedding = pgvector.NewVector(vec)
		} else if err != nil {
			logger.Warn(ctx, "usecase.product", "failed to generate product embedding", err)
		}
	}

	if err := u.repo.Create(ctx, product); err != nil {
		logger.Error(ctx, "usecase", "failed to create product in repository", err)
		return nil, err
	}

	return u.repo.FindByID(ctx, product.ID)
}

func (u *useCase) GetProductByID(ctx context.Context, id uint) (*Product, error) {
	p, err := u.repo.FindByID(ctx, id)
	if err != nil {
		logger.Warn(ctx, "usecase", fmt.Sprintf("product not found with id %d", id), domain.ErrNotFound)
		return nil, domain.ErrNotFound
	}
	return p, nil
}

func (u *useCase) GetProductBySlug(ctx context.Context, slug string) (*Product, error) {
	p, err := u.repo.FindBySlug(ctx, slug)
	if err != nil {
		logger.Warn(ctx, "usecase", fmt.Sprintf("product not found with slug %s", slug), domain.ErrNotFound)
		return nil, domain.ErrNotFound
	}
	return p, nil
}

func (u *useCase) UpdateProduct(ctx context.Context, id uint, req *UpdateProductRequest) (*Product, error) {
	p, err := u.repo.FindByID(ctx, id)
	if err != nil {
		logger.Warn(ctx, "usecase", fmt.Sprintf("product not found for update with id %d", id), domain.ErrNotFound)
		return nil, domain.ErrNotFound
	}

	if req.CategoryID != nil {
		_, err := u.repo.FindCategoryByID(ctx, *req.CategoryID)
		if err != nil {
			logger.Error(ctx, "usecase", "selected category does not exist", err)
			return nil, domain.ErrNotFound
		}
		p.CategoryID = *req.CategoryID
	}

	if req.SubCategoryID != nil {
		p.SubCategoryID = req.SubCategoryID
	}

	nameOrDescChanged := false
	if req.Name != nil {
		p.Name = *req.Name
		p.Slug = security.Slugify(*req.Name)
		nameOrDescChanged = true
	}

	if req.SKU != nil {
		p.SKU = *req.SKU
	}

	if req.Description != nil {
		p.Description = *req.Description
		nameOrDescChanged = true
	}

	if req.Price != nil {
		p.Price = *req.Price
	}

	if req.Currency != nil {
		p.Currency = *req.Currency
	}

	if req.StockQuantity != nil {
		p.StockQuantity = *req.StockQuantity
	}

	if req.LowStockThreshold != nil {
		p.LowStockThreshold = *req.LowStockThreshold
	}

	if req.ImageURL != nil {
		p.ImageURL = *req.ImageURL
	}

	if req.IsActive != nil {
		p.IsActive = *req.IsActive
	}

	if req.Badge != nil {
		p.Badge = *req.Badge
	}

	// Re-calculate pgvector embedding if Name or Description was updated
	if nameOrDescChanged && u.ollamaClient != nil {
		textToEmbed := fmt.Sprintf("%s. %s", p.Name, p.Description)
		if vec, err := u.ollamaClient.GenerateEmbedding(ctx, textToEmbed); err == nil && len(vec) > 0 {
			p.Embedding = pgvector.NewVector(vec)
		}
	}

	if err := u.repo.Update(ctx, p); err != nil {
		logger.Error(ctx, "usecase", fmt.Sprintf("failed to update product %d in repository", id), err)
		return nil, err
	}

	updated, err := u.repo.FindByID(ctx, p.ID)
	if err == nil && u.rdb != nil {
		_ = u.rdb.Del(ctx, fmt.Sprintf("recommendations:product:%d", id)).Err()
	}

	return updated, err
}

func (u *useCase) DeleteProduct(ctx context.Context, id uint) error {
	_, err := u.repo.FindByID(ctx, id)
	if err != nil {
		logger.Warn(ctx, "usecase", fmt.Sprintf("product not found for deletion with id %d", id), domain.ErrNotFound)
		return domain.ErrNotFound
	}

	if err := u.repo.Delete(ctx, id); err != nil {
		logger.Error(ctx, "usecase", fmt.Sprintf("failed to delete product %d in repository", id), err)
		return err
	}

	if u.rdb != nil {
		_ = u.rdb.Del(ctx, fmt.Sprintf("recommendations:product:%d", id)).Err()
	}

	return nil
}

func (u *useCase) ListProducts(ctx context.Context, filter ProductFilterQuery) (products []Product, total int64, err error) {
	defer logger.Track(ctx, "usecase.product", "ListProducts")(&err, map[string]interface{}{"search": filter.Search, "category_id": filter.CategoryID})

	cleanSearch := strings.TrimSpace(filter.Search)
	// Enforce minimum 3 characters for product search and calculate Hybrid embedding vector
	if len(cleanSearch) >= 3 && u.ollamaClient != nil {
		if vec, embErr := u.ollamaClient.GenerateEmbedding(ctx, cleanSearch); embErr == nil && len(vec) > 0 {
			filter.Embedding = vec
		} else if embErr != nil {
			logger.Warn(ctx, "usecase.product", "failed generating search query embedding, falling back to trigram/keyword", embErr)
		}
	} else if len(cleanSearch) < 3 {
		filter.Search = "" // Ignore searches under 3 characters to protect DB load
	}

	return u.repo.List(ctx, filter)
}

func (u *useCase) AdjustStock(ctx context.Context, productID uint, req *StockAdjustRequest, adminID uint) (*Product, error) {
	p, err := u.repo.FindByID(ctx, productID)
	if err != nil {
		logger.Warn(ctx, "usecase", fmt.Sprintf("product not found for stock adjustment with id %d", productID), domain.ErrNotFound)
		return nil, domain.ErrNotFound
	}

	prevStock := p.StockQuantity
	var newStock int

	switch req.Type {
	case "ADD":
		newStock = prevStock + req.Amount
	case "SUBTRACT":
		if prevStock < req.Amount {
			logger.Warn(ctx, "usecase", fmt.Sprintf("insufficient stock for product %d", productID), domain.ErrInsufficientStock)
			return nil, domain.ErrInsufficientStock
		}
		newStock = prevStock - req.Amount
	case "SET":
		newStock = req.Amount
	default:
		logger.Warn(ctx, "usecase", "invalid adjustment type provided", domain.ErrInvalidAdjustment)
		return nil, domain.ErrInvalidAdjustment
	}

	logEntry := &StockAdjustmentLog{
		ProductID:      productID,
		AdjustmentType: req.Type,
		Quantity:       req.Amount,
		PreviousStock:  prevStock,
		NewStock:       newStock,
		Reason:         req.Reason,
		AdjustedBy:     adminID,
	}

	if err := u.repo.AdjustStock(ctx, logEntry); err != nil {
		logger.Error(ctx, "usecase", fmt.Sprintf("failed to adjust stock for product %d in repository", productID), err)
		return nil, err
	}

	return u.repo.FindByID(ctx, productID)
}

func (u *useCase) GetStockLogs(ctx context.Context, productID uint) ([]StockAdjustmentLog, error) {
	return u.repo.GetStockLogs(ctx, productID)
}

func (u *useCase) CreateCategory(ctx context.Context, req *CreateCategoryRequest) (*Category, error) {
	slug := security.Slugify(req.Name)

	category := &Category{
		Name:        req.Name,
		Slug:        slug,
		Description: req.Description,
		Icon:        req.Icon,
	}

	if err := u.repo.CreateCategory(ctx, category); err != nil {
		logger.Error(ctx, "usecase", "failed to create category in repository", err)
		return nil, err
	}

	return category, nil
}

func (u *useCase) ListCategories(ctx context.Context) ([]Category, error) {
	return u.repo.ListCategories(ctx)
}

func (u *useCase) CreateSubCategory(ctx context.Context, req *CreateSubCategoryRequest) (*SubCategory, error) {
	slug := security.Slugify(req.Name)

	subCategory := &SubCategory{
		CategoryID:  req.CategoryID,
		Name:        req.Name,
		Slug:        slug,
		Description: req.Description,
		Icon:        req.Icon,
	}

	if err := u.repo.CreateSubCategory(ctx, subCategory); err != nil {
		logger.Error(ctx, "usecase", "failed to create sub category in repository", err)
		return nil, err
	}

	return subCategory, nil
}

func (u *useCase) ListSubCategories(ctx context.Context, categoryID uint) ([]SubCategory, error) {
	return u.repo.ListSubCategories(ctx, categoryID)
}

func (u *useCase) GetRecommendations(ctx context.Context, productID uint, limit int) (res []Product, err error) {
	defer logger.Track(ctx, "usecase.product", "GetRecommendations")(&err, map[string]interface{}{"product_id": productID, "limit": limit})
	if limit < 4 {
		limit = 4
	} else if limit > 8 {
		limit = 8
	}

	cacheKey := fmt.Sprintf("recommendations:product:%d", productID)

	// 1. Check Redis cache (Cache-Aside pattern)
	if u.rdb != nil {
		cachedJSON, err := u.rdb.Get(ctx, cacheKey).Result()
		if err == nil && cachedJSON != "" {
			var cachedProducts []Product
			if err := json.Unmarshal([]byte(cachedJSON), &cachedProducts); err == nil && len(cachedProducts) > 0 {
				if len(cachedProducts) > limit {
					return cachedProducts[:limit], nil
				}
				return cachedProducts, nil
			}
		}
	}

	// 2. Query Repository for pgvector cosine recommendation + top sellers fallback
	products, err := u.repo.GetRecommendations(ctx, productID, limit)
	if err != nil {
		logger.Warn(ctx, "usecase", fmt.Sprintf("failed to get recommendations for product %d: %v", productID, err), domain.ErrNotFound)
		return nil, domain.ErrNotFound
	}

	if len(products) > limit {
		products = products[:limit]
	}

	// 3. Populate Redis Cache (1-hour TTL = 3600 seconds)
	if u.rdb != nil && len(products) > 0 {
		if data, err := json.Marshal(products); err == nil {
			_ = u.rdb.Set(ctx, cacheKey, string(data), 3600*time.Second).Err()
		}
	}

	return products, nil
}
