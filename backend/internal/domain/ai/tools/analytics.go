package tools

import (
	"context"
	"fmt"
	"log"

	"tirenn-ai-commerce/internal/domain/dashboard"
)

// GetExecutiveDashboardMetricsTool queries high-level merchant analytics via dashboard.Repository
type GetExecutiveDashboardMetricsTool struct {
	repo dashboard.Repository
}

func NewGetExecutiveDashboardMetricsTool(repo dashboard.Repository) *GetExecutiveDashboardMetricsTool {
	return &GetExecutiveDashboardMetricsTool{repo: repo}
}

func (t *GetExecutiveDashboardMetricsTool) Name() string {
	return "get_executive_dashboard_metrics"
}

func (t *GetExecutiveDashboardMetricsTool) Description() string {
	return "Retrieve real-time executive dashboard metrics: gross revenue, total order volume, active customer count, low-stock alerts, and top-selling products."
}

func (t *GetExecutiveDashboardMetricsTool) ParametersSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"period_days": map[string]interface{}{
				"type":        "integer",
				"description": "Historical days period for sales metrics (default: 30 days).",
			},
		},
	}
}

func (t *GetExecutiveDashboardMetricsTool) Execute(ctx context.Context, args map[string]interface{}, contextMap map[string]interface{}) (map[string]interface{}, error) {
	log.Printf("📊 [ADMIN TOOL: get_executive_dashboard_metrics] executing dashboard repository queries")

	summary, err := t.repo.GetSummary(ctx)
	if err != nil {
		return map[string]interface{}{"status": "error", "message": err.Error()}, nil
	}

	topProducts, _ := t.repo.GetTopSellingProducts(ctx, 5)

	return map[string]interface{}{
		"status": "success",
		"metrics": map[string]interface{}{
			"gross_revenue":        summary.TotalRevenue,
			"gross_revenue_format": fmt.Sprintf("Rp %.2f", summary.TotalRevenue),
			"total_orders":         summary.TotalOrders,
			"total_customers":      summary.TotalCustomers,
			"low_stock_alerts":     summary.LowStockCount,
			"top_selling_products": topProducts,
		},
	}, nil
}

// GetRecentOrdersOverviewTool lists recent customer orders via dashboard.Repository
type GetRecentOrdersOverviewTool struct {
	repo dashboard.Repository
}

func NewGetRecentOrdersOverviewTool(repo dashboard.Repository) *GetRecentOrdersOverviewTool {
	return &GetRecentOrdersOverviewTool{repo: repo}
}

func (t *GetRecentOrdersOverviewTool) Name() string {
	return "get_recent_orders_overview"
}

func (t *GetRecentOrdersOverviewTool) Description() string {
	return "Retrieve overview of latest store orders with customer names, totals, and fulfillment status."
}

func (t *GetRecentOrdersOverviewTool) ParametersSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"status": map[string]interface{}{
				"type":        "string",
				"description": "Filter by order status ('PENDING', 'PAID', 'PROCESSING', 'SHIPPED', 'DELIVERED', 'CANCELLED').",
			},
			"limit": map[string]interface{}{
				"type":        "integer",
				"description": "Maximum number of orders to return (default: 8).",
			},
		},
	}
}

func (t *GetRecentOrdersOverviewTool) Execute(ctx context.Context, args map[string]interface{}, contextMap map[string]interface{}) (map[string]interface{}, error) {
	limit := 8
	if l, ok := args["limit"].(float64); ok && l > 0 {
		limit = int(l)
	} else if l, ok := args["limit"].(int); ok && l > 0 {
		limit = l
	}

	orders, err := t.repo.GetRecentOrders(ctx, limit)
	if err != nil {
		return map[string]interface{}{"status": "error", "message": err.Error()}, nil
	}

	return map[string]interface{}{
		"status":      "success",
		"total_found": len(orders),
		"orders":      orders,
	}, nil
}
