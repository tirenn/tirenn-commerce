package product

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"gocommerce-backend/internal/logger"
)

type UseCase interface {
	CreateProduct(ctx context.Context, req *CreateProductRequest) (*Product, error)
	GetProductByID(ctx context.Context, id uint) (*Product, error)
	GetProductBySlug(ctx context.Context, slug string) (*Product, error)
	UpdateProduct(ctx context.Context, id uint, req *UpdateProductRequest) (*Product, error)
	DeleteProduct(ctx context.Context, id uint) error
	ListProducts(ctx context.Context, filter ProductFilterQuery) ([]Product, int64, error)
	SyncCatalogToAI(ctx context.Context) error

	AdjustStock(ctx context.Context, productID uint, req *StockAdjustRequest, adminID uint) (*Product, error)
	GetStockLogs(ctx context.Context, productID uint) ([]StockAdjustmentLog, error)

	CreateCategory(ctx context.Context, req *CreateCategoryRequest) (*Category, error)
	ListCategories(ctx context.Context) ([]Category, error)
}

type useCase struct {
	repo     Repository
	aiClient AIClient
}

func NewUseCase(repo Repository, aiClient AIClient) UseCase {
	return &useCase{
		repo:     repo,
		aiClient: aiClient,
	}
}

func slugify(text string) string {
	reg := regexp.MustCompile("[^a-zA-Z0-9]+")
	slug := strings.ToLower(reg.ReplaceAllString(text, "-"))
	return strings.Trim(slug, "-")
}

func (u *useCase) CreateProduct(ctx context.Context, req *CreateProductRequest) (*Product, error) {
	// Verify category exists
	cat, err := u.repo.FindCategoryByID(ctx, req.CategoryID)
	if err != nil {
		logger.Error(ctx, "usecase", "selected category does not exist", err)
		return nil, errors.New("selected category does not exist")
	}

	slug := slugify(req.Name)
	existing, _ := u.repo.FindBySlug(ctx, slug)
	if existing != nil {
		slug = fmt.Sprintf("%s-%d", slug, time.Now().Unix())
	}

	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	lowThreshold := 5
	if req.LowStockThreshold >= 0 {
		lowThreshold = req.LowStockThreshold
	}

	product := &Product{
		CategoryID:        req.CategoryID,
		Category:          *cat,
		Name:              req.Name,
		Slug:              slug,
		SKU:               strings.ToUpper(strings.TrimSpace(req.SKU)),
		Description:       req.Description,
		Price:             req.Price,
		StockQuantity:     req.StockQuantity,
		LowStockThreshold: lowThreshold,
		ImageURL:          req.ImageURL,
		IsActive:          isActive,
		Badge:             req.Badge,
		Rating:            5.0,
	}

	if err := u.repo.Create(ctx, product); err != nil {
		logger.Error(ctx, "usecase", "failed to create product in repository", err)
		return nil, err
	}

	created, err := u.repo.FindByID(ctx, product.ID)
	if err == nil && u.aiClient != nil {
		go func() {
			_ = u.aiClient.SyncProducts(context.Background(), []Product{*created})
		}()
	}

	return created, err
}

func (u *useCase) GetProductByID(ctx context.Context, id uint) (*Product, error) {
	product, err := u.repo.FindByID(ctx, id)
	if err != nil {
		logger.Warn(ctx, "usecase", fmt.Sprintf("product with ID %d not found", id), err)
		return nil, err
	}
	return product, nil
}

func (u *useCase) GetProductBySlug(ctx context.Context, slug string) (*Product, error) {
	product, err := u.repo.FindBySlug(ctx, slug)
	if err != nil {
		logger.Warn(ctx, "usecase", fmt.Sprintf("product with slug %s not found", slug), err)
		return nil, err
	}
	return product, nil
}

func (u *useCase) UpdateProduct(ctx context.Context, id uint, req *UpdateProductRequest) (*Product, error) {
	product, err := u.repo.FindByID(ctx, id)
	if err != nil {
		logger.Error(ctx, "usecase", fmt.Sprintf("failed to find product %d for update", id), err)
		return nil, errors.New("product not found")
	}

	if req.CategoryID != nil {
		cat, err := u.repo.FindCategoryByID(ctx, *req.CategoryID)
		if err != nil {
			logger.Error(ctx, "usecase", fmt.Sprintf("category %d does not exist", *req.CategoryID), err)
			return nil, errors.New("selected category does not exist")
		}
		product.CategoryID = *req.CategoryID
		product.Category = *cat
	}

	if req.Name != nil && *req.Name != "" {
		product.Name = *req.Name
	}
	if req.SKU != nil && *req.SKU != "" {
		product.SKU = strings.ToUpper(strings.TrimSpace(*req.SKU))
	}
	if req.Description != nil {
		product.Description = *req.Description
	}
	if req.Price != nil && *req.Price > 0 {
		product.Price = *req.Price
	}
	if req.StockQuantity != nil && *req.StockQuantity >= 0 {
		product.StockQuantity = *req.StockQuantity
	}
	if req.LowStockThreshold != nil && *req.LowStockThreshold >= 0 {
		product.LowStockThreshold = *req.LowStockThreshold
	}
	if req.ImageURL != nil {
		product.ImageURL = *req.ImageURL
	}
	if req.IsActive != nil {
		product.IsActive = *req.IsActive
	}
	if req.Badge != nil {
		product.Badge = *req.Badge
	}

	if err := u.repo.Update(ctx, product); err != nil {
		logger.Error(ctx, "usecase", fmt.Sprintf("failed to update product %d in repository", id), err)
		return nil, err
	}

	updated, err := u.repo.FindByID(ctx, id)
	if err == nil && u.aiClient != nil {
		go func() {
			_ = u.aiClient.SyncProducts(context.Background(), []Product{*updated})
		}()
	}

	return updated, err
}

func (u *useCase) DeleteProduct(ctx context.Context, id uint) error {
	_, err := u.repo.FindByID(ctx, id)
	if err != nil {
		logger.Error(ctx, "usecase", fmt.Sprintf("failed to find product %d for deletion", id), err)
		return errors.New("product not found")
	}
	if err := u.repo.Delete(ctx, id); err != nil {
		logger.Error(ctx, "usecase", fmt.Sprintf("failed to delete product %d from repository", id), err)
		return err
	}
	return nil
}

func (u *useCase) ListProducts(ctx context.Context, filter ProductFilterQuery) ([]Product, int64, error) {
	// If semantic search is requested and search text is present, attempt vector search
	if filter.Semantic && strings.TrimSpace(filter.Search) != "" && u.aiClient != nil {
		limit := filter.Limit
		if limit < 1 {
			limit = 12
		}
		ids, err := u.aiClient.SearchSemantic(ctx, filter.Search, filter.CategoryID, limit)
		if err == nil && len(ids) > 0 {
			products, err := u.repo.FindByIDs(ctx, ids)
			if err == nil && len(products) > 0 {
				return products, int64(len(products)), nil
			}
		}
		logger.Warn(ctx, "usecase", "semantic search fallback to standard full-text query", err)
	}

	products, total, err := u.repo.List(ctx, filter)
	if err != nil {
		logger.Error(ctx, "usecase", "failed to list products from repository", err)
		return nil, 0, err
	}
	return products, total, nil
}

func (u *useCase) SyncCatalogToAI(ctx context.Context) error {
	if u.aiClient == nil {
		return nil
	}
	products, err := u.repo.ListAll(ctx)
	if err != nil {
		logger.Error(ctx, "usecase", "failed to list all products for AI sync", err)
		return err
	}
	if err := u.aiClient.SyncProducts(ctx, products); err != nil {
		logger.Error(ctx, "usecase", "failed to sync products to AI service", err)
		return err
	}
	return nil
}

func (u *useCase) AdjustStock(ctx context.Context, productID uint, req *StockAdjustRequest, adminID uint) (*Product, error) {
	product, err := u.repo.FindByID(ctx, productID)
	if err != nil {
		logger.Error(ctx, "usecase", fmt.Sprintf("failed to find product %d for stock adjustment", productID), err)
		return nil, errors.New("product not found")
	}

	prevStock := product.StockQuantity
	newStock := prevStock

	switch req.Type {
	case "ADD":
		newStock = prevStock + req.Amount
	case "SUBTRACT":
		if prevStock < req.Amount {
			errInsufficient := errors.New("insufficient stock to subtract")
			logger.Warn(ctx, "usecase", fmt.Sprintf("product %d has only %d stock, requested subtraction of %d", productID, prevStock, req.Amount), errInsufficient)
			return nil, errInsufficient
		}
		newStock = prevStock - req.Amount
	case "SET":
		newStock = req.Amount
	default:
		errInvalid := errors.New("invalid adjustment type")
		logger.Warn(ctx, "usecase", "invalid adjustment type provided", errInvalid)
		return nil, errInvalid
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
	logs, err := u.repo.GetStockLogs(ctx, productID)
	if err != nil {
		logger.Error(ctx, "usecase", fmt.Sprintf("failed to fetch stock logs for product %d", productID), err)
		return nil, err
	}
	return logs, nil
}

func (u *useCase) CreateCategory(ctx context.Context, req *CreateCategoryRequest) (*Category, error) {
	slug := slugify(req.Name)
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
	categories, err := u.repo.ListCategories(ctx)
	if err != nil {
		logger.Error(ctx, "usecase", "failed to list categories in repository", err)
		return nil, err
	}
	return categories, nil
}
