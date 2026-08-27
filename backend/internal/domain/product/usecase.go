package product

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
)

type UseCase interface {
	CreateProduct(ctx context.Context, req *CreateProductRequest) (*Product, error)
	GetProductByID(ctx context.Context, id uint) (*Product, error)
	GetProductBySlug(ctx context.Context, slug string) (*Product, error)
	UpdateProduct(ctx context.Context, id uint, req *UpdateProductRequest) (*Product, error)
	DeleteProduct(ctx context.Context, id uint) error
	ListProducts(ctx context.Context, filter ProductFilterQuery) ([]Product, int64, error)
	
	AdjustStock(ctx context.Context, productID uint, req *StockAdjustRequest, adminID uint) (*Product, error)
	GetStockLogs(ctx context.Context, productID uint) ([]StockAdjustmentLog, error)
	
	CreateCategory(ctx context.Context, req *CreateCategoryRequest) (*Category, error)
	ListCategories(ctx context.Context) ([]Category, error)
}

type useCase struct {
	repo Repository
}

func NewUseCase(repo Repository) UseCase {
	return &useCase{repo: repo}
}

func slugify(text string) string {
	reg := regexp.MustCompile("[^a-zA-Z0-9]+")
	slug := strings.ToLower(reg.ReplaceAllString(text, "-"))
	return strings.Trim(slug, "-")
}

func (u *useCase) CreateProduct(ctx context.Context, req *CreateProductRequest) (*Product, error) {
	// Verify category exists
	_, err := u.repo.FindCategoryByID(ctx, req.CategoryID)
	if err != nil {
		return nil, errors.New("selected category does not exist")
	}

	slug := slugify(req.Name)
	// Append timestamp if needed to ensure uniqueness
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
		return nil, err
	}

	return u.repo.FindByID(ctx, product.ID)
}

func (u *useCase) GetProductByID(ctx context.Context, id uint) (*Product, error) {
	return u.repo.FindByID(ctx, id)
}

func (u *useCase) GetProductBySlug(ctx context.Context, slug string) (*Product, error) {
	return u.repo.FindBySlug(ctx, slug)
}

func (u *useCase) UpdateProduct(ctx context.Context, id uint, req *UpdateProductRequest) (*Product, error) {
	product, err := u.repo.FindByID(ctx, id)
	if err != nil {
		return nil, errors.New("product not found")
	}

	if req.CategoryID != nil {
		_, err := u.repo.FindCategoryByID(ctx, *req.CategoryID)
		if err != nil {
			return nil, errors.New("selected category does not exist")
		}
		product.CategoryID = *req.CategoryID
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
		return nil, err
	}

	return u.repo.FindByID(ctx, id)
}

func (u *useCase) DeleteProduct(ctx context.Context, id uint) error {
	_, err := u.repo.FindByID(ctx, id)
	if err != nil {
		return errors.New("product not found")
	}
	return u.repo.Delete(ctx, id)
}

func (u *useCase) ListProducts(ctx context.Context, filter ProductFilterQuery) ([]Product, int64, error) {
	return u.repo.List(ctx, filter)
}

func (u *useCase) AdjustStock(ctx context.Context, productID uint, req *StockAdjustRequest, adminID uint) (*Product, error) {
	product, err := u.repo.FindByID(ctx, productID)
	if err != nil {
		return nil, errors.New("product not found")
	}

	prevStock := product.StockQuantity
	newStock := prevStock

	switch req.Type {
	case "ADD":
		newStock = prevStock + req.Amount
	case "SUBTRACT":
		if prevStock < req.Amount {
			return nil, errors.New("insufficient stock to subtract")
		}
		newStock = prevStock - req.Amount
	case "SET":
		newStock = req.Amount
	default:
		return nil, errors.New("invalid adjustment type")
	}

	log := &StockAdjustmentLog{
		ProductID:      productID,
		AdjustmentType: req.Type,
		Quantity:       req.Amount,
		PreviousStock:  prevStock,
		NewStock:       newStock,
		Reason:         req.Reason,
		AdjustedBy:     adminID,
	}

	if err := u.repo.AdjustStock(ctx, log); err != nil {
		return nil, err
	}

	return u.repo.FindByID(ctx, productID)
}

func (u *useCase) GetStockLogs(ctx context.Context, productID uint) ([]StockAdjustmentLog, error) {
	return u.repo.GetStockLogs(ctx, productID)
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
		return nil, err
	}
	return category, nil
}

func (u *useCase) ListCategories(ctx context.Context) ([]Category, error) {
	return u.repo.ListCategories(ctx)
}
