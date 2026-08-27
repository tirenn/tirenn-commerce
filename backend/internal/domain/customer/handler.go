package customer

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

// ListCustomers returns a list of registered customers with metrics (Admin)
func (h *Handler) ListCustomers(c *gin.Context) {
	var query CustomerFilterQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		utils.BadRequest(c, "Invalid query parameters", err.Error())
		return
	}

	customers, total, err := h.useCase.ListCustomers(c.Request.Context(), query)
	if err != nil {
		utils.InternalServerError(c, "Failed to fetch customers", err.Error())
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

	utils.SuccessWithMeta(c, http.StatusOK, "Customers retrieved successfully", customers, meta)
}

// UpdateStatus toggles a customer's active or suspended status (Admin)
func (h *Handler) UpdateStatus(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		utils.BadRequest(c, "Invalid customer ID", nil)
		return
	}

	var req UpdateCustomerStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "Invalid status payload", err.Error())
		return
	}

	if err := h.useCase.UpdateStatus(c.Request.Context(), uint(id), &req); err != nil {
		utils.BadRequest(c, err.Error(), nil)
		return
	}

	utils.Success(c, http.StatusOK, "Customer status updated successfully", nil)
}
