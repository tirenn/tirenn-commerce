package product

import (
	"math"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gocommerce-backend/internal/utils"
)

type Handler struct {
	useCase UseCase
}

func NewHandler(useCase UseCase) *Handler {
	return &Handler{useCase: useCase}
}

// ListProducts handles public storefront product listing & filtering
func (h *Handler) ListProducts(c *gin.Context) {
	var query ProductFilterQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		utils.BadRequest(c, "Invalid query parameters", err.Error())
		return
	}

	query.IsAdmin = false
	products, total, err := h.useCase.ListProducts(c.Request.Context(), query)
	if err != nil {
		utils.InternalServerError(c, "Failed to fetch products", err.Error())
		return
	}

	limit := query.Limit
	if limit <= 0 {
		limit = 12
	}
	totalPages := int(math.Ceil(float64(total) / float64(limit)))

	meta := utils.PaginationMeta{
		Page:      query.Page,
		Limit:     limit,
		TotalRows: total,
		TotalPage: totalPages,
	}

	utils.SuccessWithMeta(c, http.StatusOK, "Products retrieved successfully", products, meta)
}

// AdminListProducts lists all products including inactive for admin
func (h *Handler) AdminListProducts(c *gin.Context) {
	var query ProductFilterQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		utils.BadRequest(c, "Invalid query parameters", err.Error())
		return
	}

	query.IsAdmin = true
	products, total, err := h.useCase.ListProducts(c.Request.Context(), query)
	if err != nil {
		utils.InternalServerError(c, "Failed to fetch products", err.Error())
		return
	}

	limit := query.Limit
	if limit <= 0 {
		limit = 10
	}
	totalPages := int(math.Ceil(float64(total) / float64(limit)))

	meta := utils.PaginationMeta{
		Page:      query.Page,
		Limit:     limit,
		TotalRows: total,
		TotalPage: totalPages,
	}

	utils.SuccessWithMeta(c, http.StatusOK, "Admin products retrieved successfully", products, meta)
}

// GetProduct retrieves single product by ID or Slug
func (h *Handler) GetProduct(c *gin.Context) {
	param := c.Param("id")
	
	// If param is numeric, fetch by ID, otherwise by Slug
	if id, err := strconv.ParseUint(param, 10, 32); err == nil {
		product, err := h.useCase.GetProductByID(c.Request.Context(), uint(id))
		if err != nil {
			utils.NotFound(c, "Product not found")
			return
		}
		utils.Success(c, http.StatusOK, "Product retrieved", product)
		return
	}

	product, err := h.useCase.GetProductBySlug(c.Request.Context(), param)
	if err != nil {
		utils.NotFound(c, "Product not found")
		return
	}

	utils.Success(c, http.StatusOK, "Product retrieved", product)
}

// CreateProduct creates a new product (Admin)
func (h *Handler) CreateProduct(c *gin.Context) {
	var req CreateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "Invalid product payload", err.Error())
		return
	}

	product, err := h.useCase.CreateProduct(c.Request.Context(), &req)
	if err != nil {
		utils.BadRequest(c, err.Error(), nil)
		return
	}

	utils.Success(c, http.StatusCreated, "Product created successfully", product)
}

// UpdateProduct updates product details (Admin)
func (h *Handler) UpdateProduct(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		utils.BadRequest(c, "Invalid product ID", nil)
		return
	}

	var req UpdateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "Invalid update payload", err.Error())
		return
	}

	product, err := h.useCase.UpdateProduct(c.Request.Context(), uint(id), &req)
	if err != nil {
		utils.BadRequest(c, err.Error(), nil)
		return
	}

	utils.Success(c, http.StatusOK, "Product updated successfully", product)
}

// DeleteProduct deletes a product (Admin)
func (h *Handler) DeleteProduct(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		utils.BadRequest(c, "Invalid product ID", nil)
		return
	}

	if err := h.useCase.DeleteProduct(c.Request.Context(), uint(id)); err != nil {
		utils.BadRequest(c, err.Error(), nil)
		return
	}

	utils.Success(c, http.StatusOK, "Product deleted successfully", nil)
}

// AdjustStock adjusts product stock count (Admin)
func (h *Handler) AdjustStock(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		utils.BadRequest(c, "Invalid product ID", nil)
		return
	}

	adminIDVal, _ := c.Get("userID")
	adminID, _ := adminIDVal.(uint)

	var req StockAdjustRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "Invalid stock adjustment payload", err.Error())
		return
	}

	product, err := h.useCase.AdjustStock(c.Request.Context(), uint(id), &req, adminID)
	if err != nil {
		utils.BadRequest(c, err.Error(), nil)
		return
	}

	utils.Success(c, http.StatusOK, "Stock adjusted successfully", product)
}

// GetStockLogs retrieves audit logs for product stock
func (h *Handler) GetStockLogs(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		utils.BadRequest(c, "Invalid product ID", nil)
		return
	}

	logs, err := h.useCase.GetStockLogs(c.Request.Context(), uint(id))
	if err != nil {
		utils.InternalServerError(c, "Failed to fetch stock logs", err.Error())
		return
	}

	utils.Success(c, http.StatusOK, "Stock adjustment logs retrieved", logs)
}

// ListCategories lists all product categories
func (h *Handler) ListCategories(c *gin.Context) {
	categories, err := h.useCase.ListCategories(c.Request.Context())
	if err != nil {
		utils.InternalServerError(c, "Failed to fetch categories", err.Error())
		return
	}

	utils.Success(c, http.StatusOK, "Categories retrieved successfully", categories)
}

// CreateCategory creates a new category (Admin)
func (h *Handler) CreateCategory(c *gin.Context) {
	var req CreateCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "Invalid category payload", err.Error())
		return
	}

	category, err := h.useCase.CreateCategory(c.Request.Context(), &req)
	if err != nil {
		utils.BadRequest(c, err.Error(), nil)
		return
	}

	utils.Success(c, http.StatusCreated, "Category created successfully", category)
}
