package dashboard

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"tirenn-ai-commerce/internal/response"
)

type Handler struct {
	useCase UseCase
}

func NewHandler(useCase UseCase) *Handler {
	return &Handler{useCase: useCase}
}

// RegisterRoutes registers dashboard analytics routes for admin
func (h *Handler) RegisterRoutes(adminGroup *gin.RouterGroup) {
	if adminGroup != nil {
		adminGroup.GET("/dashboard", h.GetDashboard)
	}
}

// GetDashboard returns the full admin analytics data (KPIs, Charts, Top Products, Alerts)
func (h *Handler) GetDashboard(c *gin.Context) {
	days, _ := strconv.Atoi(c.DefaultQuery("days", "7"))
	if days <= 0 || days > 90 {
		days = 7
	}

	data, err := h.useCase.GetDashboardData(c.Request.Context(), days)
	if err != nil {
		response.Error(c, "Failed to load dashboard data", err)
		return
	}

	response.Success(c, http.StatusOK, "Dashboard data retrieved successfully", data)
}
