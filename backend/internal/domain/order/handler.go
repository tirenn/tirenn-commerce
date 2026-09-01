package order

import (
	"math"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/tirenn/commerce/backend/internal/domain"
	"github.com/tirenn/commerce/backend/internal/domain/auth"
	"github.com/tirenn/commerce/backend/internal/response"
)

type Handler struct {
	useCase UseCase
}

func NewHandler(useCase UseCase) *Handler {
	return &Handler{useCase: useCase}
}

// RegisterRoutes sets up customer and admin order endpoints
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup, authMiddleware gin.HandlerFunc, adminGroup *gin.RouterGroup) {
	// Authenticated Customer Orders Routes
	orders := rg.Group("/orders", authMiddleware)
	{
		orders.POST("/checkout", h.Checkout)
		orders.GET("/my-orders", h.ListMyOrders)
		orders.GET("/:id", h.GetOrder)
	}

	// Protected Admin Order Routes
	if adminGroup != nil {
		adminGroup.GET("/orders", h.AdminListOrders)
		adminGroup.PUT("/orders/:id/status", h.AdminUpdateOrderStatus)
		adminGroup.PATCH("/orders/:id/status", h.AdminUpdateOrderStatus)
	}
}

// Checkout creates an order from cart items
func (h *Handler) Checkout(c *gin.Context) {
	userIDVal, exists := c.Get("userID")
	if !exists {
		response.Error(c, "Authentication required for checkout", domain.ErrUnauthorized)
		return
	}
	userID := userIDVal.(uint)

	var req CheckoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, "Invalid checkout payload", domain.ErrBadRequest)
		return
	}

	order, err := h.useCase.Checkout(c.Request.Context(), userID, &req)
	if err != nil {
		response.Error(c, err.Error(), err)
		return
	}

	response.Created(c, "Order placed successfully", order)
}

// GetOrder retrieves order details
func (h *Handler) GetOrder(c *gin.Context) {
	userIDVal, _ := c.Get("userID")
	userID, _ := userIDVal.(uint)
	roleVal, _ := c.Get("userRole")
	role, _ := roleVal.(string)
	isAdmin := role == auth.RoleAdmin

	idParam := c.Param("id")
	if id, err := strconv.ParseUint(idParam, 10, 32); err == nil {
		order, err := h.useCase.GetOrderByID(c.Request.Context(), uint(id), userID, isAdmin)
		if err != nil {
			response.Error(c, err.Error(), domain.ErrNotFound)
			return
		}
		response.Success(c, http.StatusOK, "Order retrieved", order)
		return
	}

	order, err := h.useCase.GetOrderByOrderNumber(c.Request.Context(), idParam, userID, isAdmin)
	if err != nil {
		response.Error(c, err.Error(), domain.ErrNotFound)
		return
	}
	response.Success(c, http.StatusOK, "Order retrieved", order)
}

// ListMyOrders lists orders belonging to the authenticated customer
func (h *Handler) ListMyOrders(c *gin.Context) {
	userIDVal, _ := c.Get("userID")
	userID, _ := userIDVal.(uint)

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	orders, total, err := h.useCase.ListCustomerOrders(c.Request.Context(), userID, page, limit)
	if err != nil {
		response.Error(c, "Failed to fetch orders", err)
		return
	}

	totalPages := int(math.Ceil(float64(total) / float64(limit)))
	meta := response.PaginationMeta{
		Page:      page,
		Limit:     limit,
		TotalRows: total,
		TotalPage: totalPages,
	}

	response.SuccessWithMeta(c, http.StatusOK, "Orders retrieved successfully", orders, meta)
}

// AdminListOrders lists all orders with status/search filter (Admin)
func (h *Handler) AdminListOrders(c *gin.Context) {
	var query OrderFilterQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		response.Error(c, "Invalid filter query parameters", domain.ErrBadRequest)
		return
	}

	orders, total, err := h.useCase.AdminListOrders(c.Request.Context(), query)
	if err != nil {
		response.Error(c, "Failed to fetch admin orders", err)
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

	response.SuccessWithMeta(c, http.StatusOK, "Orders retrieved successfully", orders, meta)
}

// AdminUpdateOrderStatus updates order status (Admin)
func (h *Handler) AdminUpdateOrderStatus(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		response.Error(c, "Invalid order ID", domain.ErrBadRequest)
		return
	}

	var req UpdateOrderStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, "Invalid status payload", domain.ErrBadRequest)
		return
	}

	order, err := h.useCase.AdminUpdateStatus(c.Request.Context(), uint(id), &req)
	if err != nil {
		response.Error(c, err.Error(), err)
		return
	}

	response.Success(c, http.StatusOK, "Order status updated successfully", order)
}
