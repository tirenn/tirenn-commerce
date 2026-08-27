package order

import (
	"math"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gocommerce-backend/internal/domain/auth"
	"gocommerce-backend/internal/utils"
)

type Handler struct {
	useCase UseCase
}

func NewHandler(useCase UseCase) *Handler {
	return &Handler{useCase: useCase}
}

// Checkout creates an order from cart items
func (h *Handler) Checkout(c *gin.Context) {
	userIDVal, exists := c.Get("userID")
	if !exists {
		utils.Unauthorized(c, "Authentication required for checkout")
		return
	}
	userID := userIDVal.(uint)

	var req CheckoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "Invalid checkout payload", err.Error())
		return
	}

	order, err := h.useCase.Checkout(c.Request.Context(), userID, &req)
	if err != nil {
		utils.BadRequest(c, err.Error(), nil)
		return
	}

	utils.Success(c, http.StatusCreated, "Order placed successfully", order)
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
			utils.NotFound(c, err.Error())
			return
		}
		utils.Success(c, http.StatusOK, "Order retrieved", order)
		return
	}

	order, err := h.useCase.GetOrderByOrderNumber(c.Request.Context(), idParam, userID, isAdmin)
	if err != nil {
		utils.NotFound(c, err.Error())
		return
	}
	utils.Success(c, http.StatusOK, "Order retrieved", order)
}

// ListMyOrders lists orders belonging to the authenticated customer
func (h *Handler) ListMyOrders(c *gin.Context) {
	userIDVal, _ := c.Get("userID")
	userID, _ := userIDVal.(uint)

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	orders, total, err := h.useCase.ListCustomerOrders(c.Request.Context(), userID, page, limit)
	if err != nil {
		utils.InternalServerError(c, "Failed to fetch orders", err.Error())
		return
	}

	totalPages := int(math.Ceil(float64(total) / float64(limit)))
	meta := utils.PaginationMeta{
		Page:      page,
		Limit:     limit,
		TotalRows: total,
		TotalPage: totalPages,
	}

	utils.SuccessWithMeta(c, http.StatusOK, "Orders retrieved successfully", orders, meta)
}

// AdminListOrders lists all orders with status/search filter (Admin)
func (h *Handler) AdminListOrders(c *gin.Context) {
	var query OrderFilterQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		utils.BadRequest(c, "Invalid query parameters", err.Error())
		return
	}

	orders, total, err := h.useCase.ListAllOrders(c.Request.Context(), query)
	if err != nil {
		utils.InternalServerError(c, "Failed to fetch orders", err.Error())
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

	utils.SuccessWithMeta(c, http.StatusOK, "All orders retrieved successfully", orders, meta)
}

// AdminUpdateOrderStatus updates order status (Admin)
func (h *Handler) AdminUpdateOrderStatus(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		utils.BadRequest(c, "Invalid order ID", nil)
		return
	}

	var req UpdateOrderStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "Invalid status update payload", err.Error())
		return
	}

	if err := h.useCase.UpdateOrderStatus(c.Request.Context(), uint(id), &req); err != nil {
		utils.BadRequest(c, err.Error(), nil)
		return
	}

	utils.Success(c, http.StatusOK, "Order status updated successfully", nil)
}
