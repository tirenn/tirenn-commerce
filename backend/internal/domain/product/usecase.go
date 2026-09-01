package product

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/tirenn/commerce/backend/internal/domain"
	"github.com/tirenn/commerce/backend/internal/logger"
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

type AIClient interface {
	SearchSemantic(ctx context.Context, query string, categoryID uint, limit int) ([]uint, error)
	SyncProducts(ctx context.Context, products []Product) error
	GetRecommendations(ctx context.Context, productID uint, limit int) ([]uint, error)
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

	currency := "IDR"
	if req.Currency != "" {
		currency = req.Currency
	}

	product := &Product{
		CategoryID:        req.CategoryID,
		Category:          *cat,
		SubCategoryID:     req.SubCategoryID,
		Name:              req.Name,
		Slug:              slug,
		SKU:               strings.ToUpper(strings.TrimSpace(req.SKU)),
		Description:       req.Description,
		Price:             req.Price,
		Currency:          currency,
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
	p, err := u.repo.FindByID(ctx, id)
	if err != nil {
		logger.Warn(ctx, "usecase", fmt.Sprintf("product not found with id %d", id), err)
		return nil, errors.New("product not found")
	}
	return p, nil
}

func (u *useCase) GetProductBySlug(ctx context.Context, slug string) (*Product, error) {
	p, err := u.repo.FindBySlug(ctx, slug)
	if err != nil {
		logger.Warn(ctx, "usecase", fmt.Sprintf("product not found with slug %s", slug), err)
		return nil, errors.New("product not found")
	}
	return p, nil
}

func (u *useCase) UpdateProduct(ctx context.Context, id uint, req *UpdateProductRequest) (*Product, error) {
	p, err := u.repo.FindByID(ctx, id)
	if err != nil {
		logger.Warn(ctx, "usecase", fmt.Sprintf("failed to find product for update with id %d", id), err)
		return nil, errors.New("product not found")
	}

	if req.CategoryID != nil {
		cat, err := u.repo.FindCategoryByID(ctx, *req.CategoryID)
		if err != nil {
			logger.Error(ctx, "usecase", "category does not exist for update", err)
			return nil, errors.New("category does not exist")
		}
		p.CategoryID = *req.CategoryID
		p.Category = *cat
	}

	if req.SubCategoryID != nil {
		p.SubCategoryID = req.SubCategoryID
	}

	if req.Name != nil {
		p.Name = *req.Name
	}
	if req.SKU != nil {
		p.SKU = strings.ToUpper(strings.TrimSpace(*req.SKU))
	}
	if req.Description != nil {
		p.Description = *req.Description
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

	if err := u.repo.Update(ctx, p); err != nil {
		logger.Error(ctx, "usecase", fmt.Sprintf("failed to update product %d in repository", id), err)
		return nil, err
	}

	updated, err := u.repo.FindByID(ctx, p.ID)
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
		logger.Warn(ctx, "usecase", fmt.Sprintf("product not found for deletion with id %d", id), err)
		return errors.New("product not found")
	}

	if err := u.repo.Delete(ctx, id); err != nil {
		logger.Error(ctx, "usecase", fmt.Sprintf("failed to delete product %d in repository", id), err)
		return err
	}
	return nil
}

func (u *useCase) ListProducts(ctx context.Context, filter ProductFilterQuery) ([]Product, int64, error) {
	return u.repo.List(ctx, filter)
}

func (u *useCase) AdjustStock(ctx context.Context, productID uint, req *StockAdjustRequest, adminID uint) (*Product, error) {
	p, err := u.repo.FindByID(ctx, productID)
	if err != nil {
		logger.Warn(ctx, "usecase", fmt.Sprintf("product not found for stock adjustment with id %d", productID), err)
		return nil, errors.New("product not found")
	}

	prevStock := p.StockQuantity
	var newStock int

	switch req.Type {
	case "ADD":
		newStock = prevStock + req.Amount
	case "SUBTRACT":
		if prevStock < req.Amount {
			errStock := errors.New("insufficient stock to subtract")
			logger.Warn(ctx, "usecase", fmt.Sprintf("insufficient stock for product %d", productID), errStock)
			return nil, errStock
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

func (u *useCase) CreateSubCategory(ctx context.Context, req *CreateSubCategoryRequest) (*SubCategory, error) {
	slug := slugify(req.Name)
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
	subCategories, err := u.repo.ListSubCategories(ctx, categoryID)
	if err != nil {
		logger.Error(ctx, "usecase", "failed to list sub categories in repository", err)
		return nil, err
	}
	return subCategories, nil
}

func (u *useCase) GetRecommendations(ctx context.Context, productID uint, limit int) ([]Product, error) {
	if limit < 4 {
		limit = 4
	}
	if limit > 8 {
		limit = 8
	}

	p, err := u.repo.FindByID(ctx, productID)
	if err != nil || p == nil {
		return nil, domain.ErrNotFound
	}

	// 1. Try AI service recommendations if available
	if u.aiClient != nil {
		ids, err := u.aiClient.GetRecommendations(ctx, productID, limit)
		if err == nil && len(ids) > 0 {
			prods, err := u.repo.FindByIDs(ctx, ids)
			if err == nil && len(prods) > 0 {
				return prods, nil
			}
		}
	}

	// 2. Fallback to vector similarity from repository
	recs, err := u.repo.GetRecommendations(ctx, productID, limit)
	if err == nil && len(recs) >= limit {
		return recs, nil
	}

	// 3. Fallback to category top sellers
	catSellers, err := u.repo.GetCategoryTopSellers(ctx, p.CategoryID, productID, limit)
	if err == nil && len(catSellers) > 0 {
		return catSellers, nil
	}

	// 4. Fallback to overall top sellers
	return u.repo.GetOverallTopSellers(ctx, productID, limit)
}
