package product

import (
	"math"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"tirenn-ai-commerce/internal/domain"
	"tirenn-ai-commerce/internal/response"
)

type Handler struct {
	useCase UseCase
}

func NewHandler(useCase UseCase) *Handler {
	return &Handler{useCase: useCase}
}

// RegisterRoutes sets up both public storefront routes and protected admin routes
func (h *Handler) RegisterRoutes(publicGroup *gin.RouterGroup, adminGroup *gin.RouterGroup) {
	// Public Catalog Routes
	publicGroup.GET("/products", h.ListProducts)
	publicGroup.GET("/products/:id", h.GetProduct)
	publicGroup.GET("/products/:id/recommendations", h.GetRecommendations)
	publicGroup.GET("/categories", h.ListCategories)
	publicGroup.GET("/sub-categories", h.ListSubCategories)

	// Protected Admin Routes
	if adminGroup != nil {
		adminGroup.GET("/products", h.AdminListProducts)
		adminGroup.POST("/products", h.CreateProduct)
		adminGroup.PUT("/products/:id", h.UpdateProduct)
		adminGroup.DELETE("/products/:id", h.DeleteProduct)
		adminGroup.POST("/products/:id/adjust-stock", h.AdjustStock)
		adminGroup.GET("/products/:id/stock-logs", h.GetStockLogs)
		adminGroup.POST("/categories", h.CreateCategory)
		adminGroup.POST("/sub-categories", h.CreateSubCategory)
	}
}

// ListProducts handles public storefront product listing & filtering
func (h *Handler) ListProducts(c *gin.Context) {
	var query ProductFilterQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		response.Error(c, "Invalid query parameters", domain.ErrBadRequest)
		return
	}

	query.IsAdmin = false
	products, total, err := h.useCase.ListProducts(c.Request.Context(), query)
	if err != nil {
		response.Error(c, "Failed to fetch products", err)
		return
	}

	limit := query.Limit
	if limit <= 0 {
		limit = 12
	}
	totalPages := int(math.Ceil(float64(total) / float64(limit)))

	meta := response.PaginationMeta{
		Page:      query.Page,
		Limit:     limit,
		TotalRows: total,
		TotalPage: totalPages,
	}

	response.SuccessWithMeta(c, http.StatusOK, "Products retrieved successfully", products, meta)
}

// AdminListProducts lists all products including inactive for admin
func (h *Handler) AdminListProducts(c *gin.Context) {
	var query ProductFilterQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		response.Error(c, "Invalid query parameters", domain.ErrBadRequest)
		return
	}

	query.IsAdmin = true
	products, total, err := h.useCase.ListProducts(c.Request.Context(), query)
	if err != nil {
		response.Error(c, "Failed to fetch products", err)
		return
	}

	limit := query.Limit
	if limit <= 0 {
		limit = 10
	}
	totalPages := int(math.Ceil(float64(total) / float64(limit)))

	meta := response.PaginationMeta{
		Page:      query.Page,
		Limit:     limit,
		TotalRows: total,
		TotalPage: totalPages,
	}

	response.SuccessWithMeta(c, http.StatusOK, "Admin products retrieved successfully", products, meta)
}

// GetProduct retrieves single product by ID or Slug
func (h *Handler) GetProduct(c *gin.Context) {
	param := c.Param("id")

	// If param is numeric, fetch by ID, otherwise by Slug
	if id, err := strconv.ParseUint(param, 10, 32); err == nil {
		product, err := h.useCase.GetProductByID(c.Request.Context(), uint(id))
		if err != nil {
			response.Error(c, "Product not found", domain.ErrNotFound)
			return
		}
		response.Success(c, http.StatusOK, "Product retrieved", product)
		return
	}

	product, err := h.useCase.GetProductBySlug(c.Request.Context(), param)
	if err != nil {
		response.Error(c, "Product not found", domain.ErrNotFound)
		return
	}

	response.Success(c, http.StatusOK, "Product retrieved", product)
}

// GetRecommendations retrieves AI-based product recommendations with fallback
func (h *Handler) GetRecommendations(c *gin.Context) {
	param := c.Param("id")
	id, err := strconv.ParseUint(param, 10, 32)
	if err != nil {
		response.Error(c, "Invalid product ID", domain.ErrBadRequest)
		return
	}

	limitStr := c.DefaultQuery("limit", "6")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 6
	}

	if limit < 4 {
		limit = 4
	} else if limit > 8 {
		limit = 8
	}

	products, err := h.useCase.GetRecommendations(c.Request.Context(), uint(id), limit)
	if err != nil {
		response.Error(c, "Product not found", domain.ErrNotFound)
		return
	}

	response.Success(c, http.StatusOK, "Recommendations retrieved successfully", products)
}

// CreateProduct creates a new product (Admin)
func (h *Handler) CreateProduct(c *gin.Context) {
	var req CreateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, "Invalid product payload", domain.ErrBadRequest)
		return
	}

	product, err := h.useCase.CreateProduct(c.Request.Context(), &req)
	if err != nil {
		response.Error(c, err.Error(), err)
		return
	}

	response.Success(c, http.StatusCreated, "Product created successfully", product)
}

// UpdateProduct updates product details (Admin)
func (h *Handler) UpdateProduct(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		response.Error(c, "Invalid product ID", domain.ErrBadRequest)
		return
	}

	var req UpdateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, "Invalid update payload", domain.ErrBadRequest)
		return
	}

	product, err := h.useCase.UpdateProduct(c.Request.Context(), uint(id), &req)
	if err != nil {
		response.Error(c, err.Error(), err)
		return
	}

	response.Success(c, http.StatusOK, "Product updated successfully", product)
}

// DeleteProduct soft/hard deletes a product (Admin)
func (h *Handler) DeleteProduct(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		response.Error(c, "Invalid product ID", domain.ErrBadRequest)
		return
	}

	if err := h.useCase.DeleteProduct(c.Request.Context(), uint(id)); err != nil {
		response.Error(c, err.Error(), err)
		return
	}

	response.Success(c, http.StatusOK, "Product deleted successfully", nil)
}

// AdjustStock handles stock adjustment with reason log (Admin)
func (h *Handler) AdjustStock(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		response.Error(c, "Invalid product ID", domain.ErrBadRequest)
		return
	}

	var req StockAdjustRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, "Invalid stock adjustment payload", domain.ErrBadRequest)
		return
	}

	// Retrieve Admin ID from Context
	adminID := uint(1)
	if val, exists := c.Get("user_id"); exists {
		if uid, ok := val.(uint); ok {
			adminID = uid
		}
	}

	product, err := h.useCase.AdjustStock(c.Request.Context(), uint(id), &req, adminID)
	if err != nil {
		response.Error(c, err.Error(), err)
		return
	}

	response.Success(c, http.StatusOK, "Stock adjusted successfully", product)
}

// GetStockLogs retrieves audit logs for product stock
func (h *Handler) GetStockLogs(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		response.Error(c, "Invalid product ID", domain.ErrBadRequest)
		return
	}

	logs, err := h.useCase.GetStockLogs(c.Request.Context(), uint(id))
	if err != nil {
		response.Error(c, "Failed to fetch stock logs", err)
		return
	}

	response.Success(c, http.StatusOK, "Stock adjustment logs retrieved", logs)
}

// ListCategories lists all product categories
func (h *Handler) ListCategories(c *gin.Context) {
	categories, err := h.useCase.ListCategories(c.Request.Context())
	if err != nil {
		response.Error(c, "Failed to fetch categories", err)
		return
	}

	response.Success(c, http.StatusOK, "Categories retrieved successfully", categories)
}

// CreateCategory creates a new category (Admin)
func (h *Handler) CreateCategory(c *gin.Context) {
	var req CreateCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, "Invalid category payload", domain.ErrBadRequest)
		return
	}

	category, err := h.useCase.CreateCategory(c.Request.Context(), &req)
	if err != nil {
		response.Error(c, err.Error(), err)
		return
	}

	response.Success(c, http.StatusCreated, "Category created successfully", category)
}

// ListSubCategories lists subcategories (optionally filtered by category_id)
func (h *Handler) ListSubCategories(c *gin.Context) {
	catIDStr := c.Query("category_id")
	var catID uint
	if catIDStr != "" {
		if val, err := strconv.ParseUint(catIDStr, 10, 32); err == nil {
			catID = uint(val)
		}
	}

	subCategories, err := h.useCase.ListSubCategories(c.Request.Context(), catID)
	if err != nil {
		response.Error(c, "Failed to fetch sub-categories", err)
		return
	}

	response.Success(c, http.StatusOK, "Sub-categories retrieved successfully", subCategories)
}

// CreateSubCategory creates a new subcategory (Admin)
func (h *Handler) CreateSubCategory(c *gin.Context) {
	var req CreateSubCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, "Invalid sub-category payload", domain.ErrBadRequest)
		return
	}

	subCategory, err := h.useCase.CreateSubCategory(c.Request.Context(), &req)
	if err != nil {
		response.Error(c, err.Error(), err)
		return
	}

	response.Success(c, http.StatusCreated, "Sub-category created successfully", subCategory)
}
