package customer

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

// RegisterRoutes registers customer management routes for admin
func (h *Handler) RegisterRoutes(adminGroup *gin.RouterGroup) {
	if adminGroup != nil {
		adminGroup.GET("/customers", h.ListCustomers)
		adminGroup.PUT("/customers/:id/status", h.UpdateStatus)
	}
}

// ListCustomers returns a list of registered customers with metrics (Admin)
func (h *Handler) ListCustomers(c *gin.Context) {
	var query CustomerFilterQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		response.Error(c, "Invalid query parameters", domain.ErrBadRequest)
		return
	}

	customers, total, err := h.useCase.ListCustomers(c.Request.Context(), query)
	if err != nil {
		response.Error(c, "Failed to fetch customers", err)
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

	response.SuccessWithMeta(c, http.StatusOK, "Customers retrieved successfully", customers, meta)
}

// UpdateStatus toggles a customer's active or suspended status (Admin)
func (h *Handler) UpdateStatus(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		response.Error(c, "Invalid customer ID", domain.ErrBadRequest)
		return
	}

	var req UpdateCustomerStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, "Invalid status payload", domain.ErrBadRequest)
		return
	}

	if err := h.useCase.UpdateStatus(c.Request.Context(), uint(id), &req); err != nil {
		response.Error(c, err.Error(), err)
		return
	}

	response.Success(c, http.StatusOK, "Customer status updated successfully", nil)
}
