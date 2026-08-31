package tools

import (
	"context"
	"fmt"
	"log"

	"gorm.io/gorm"
)

// GetExecutiveDashboardMetricsTool queries high-level merchant analytics directly from database
type GetExecutiveDashboardMetricsTool struct {
	db *gorm.DB
}

func NewGetExecutiveDashboardMetricsTool(db *gorm.DB) *GetExecutiveDashboardMetricsTool {
	return &GetExecutiveDashboardMetricsTool{db: db}
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
	log.Printf("📊 [ADMIN TOOL: get_executive_dashboard_metrics] executing direct SQL aggregation")

	var totalRevenue float64
	var totalOrders int64
	var totalCustomers int64
	var lowStockCount int64

	t.db.WithContext(ctx).Table("orders").Where("status != 'CANCELLED'").Select("COALESCE(SUM(total_amount), 0)").Row().Scan(&totalRevenue)
	t.db.WithContext(ctx).Table("orders").Count(&totalOrders)
	t.db.WithContext(ctx).Table("users").Where("role = 'CUSTOMER'").Count(&totalCustomers)
	t.db.WithContext(ctx).Table("products").Where("is_active = true AND stock_quantity <= low_stock_threshold").Count(&lowStockCount)

	var topProducts []AnalyticsTopProduct
	t.db.WithContext(ctx).Raw(`
		SELECT 
			p.id, p.name, p.sku, 
			COALESCE(SUM(oi.quantity), 0) as total_sold,
			COALESCE(SUM(oi.subtotal), 0) as total_sales
		FROM products p
		INNER JOIN order_items oi ON oi.product_id = p.id
		INNER JOIN orders o ON oi.order_id = o.id
		WHERE o.status != 'CANCELLED'
		GROUP BY p.id, p.name, p.sku
		ORDER BY total_sold DESC
		LIMIT 5
	`).Scan(&topProducts)

	return map[string]interface{}{
		"status": "success",
		"metrics": map[string]interface{}{
			"gross_revenue":        totalRevenue,
			"gross_revenue_format": fmt.Sprintf("Rp %.2f", totalRevenue),
			"total_orders":         totalOrders,
			"total_customers":      totalCustomers,
			"low_stock_alerts":     lowStockCount,
			"top_selling_products": topProducts,
		},
	}, nil
}

// GetRecentOrdersOverviewTool lists recent customer orders
type GetRecentOrdersOverviewTool struct {
	db *gorm.DB
}

func NewGetRecentOrdersOverviewTool(db *gorm.DB) *GetRecentOrdersOverviewTool {
	return &GetRecentOrdersOverviewTool{db: db}
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

	status, _ := args["status"].(string)

	var orders []AnalyticsOrderRow
	query := t.db.WithContext(ctx).Table("orders o").
		Select("o.id, o.order_number, u.name as customer_name, u.email as customer_email, o.total_amount, o.status, (SELECT COUNT(*) FROM order_items oi WHERE oi.order_id = o.id) as item_count, o.created_at").
		Joins("LEFT JOIN users u ON u.id = o.user_id").
		Order("o.id DESC").
		Limit(limit)

	if status != "" {
		query = query.Where("o.status = ?", status)
	}

	if err := query.Scan(&orders).Error; err != nil {
		return map[string]interface{}{"status": "error", "message": err.Error()}, nil
	}

	return map[string]interface{}{
		"status":      "success",
		"total_found": len(orders),
		"orders":      orders,
	}, nil
}
