package dashboard

import (
	"context"
	"time"

	"gocommerce-backend/internal/domain/auth"
	"gocommerce-backend/internal/domain/order"
	"gocommerce-backend/internal/domain/product"
	"gorm.io/gorm"
)

type Repository interface {
	GetSummary(ctx context.Context) (*DashboardSummary, error)
	GetRevenueTrends(ctx context.Context, days int) ([]RevenueChartPoint, error)
	GetTopSellingProducts(ctx context.Context, limit int) ([]TopSellingProduct, error)
	GetLowStockAlerts(ctx context.Context) ([]LowStockAlert, error)
	GetRecentOrders(ctx context.Context, limit int) ([]RecentOrderSummary, error)
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

func (r *repository) GetSummary(ctx context.Context) (*DashboardSummary, error) {
	var summary DashboardSummary

	// Total Revenue
	r.db.WithContext(ctx).Model(&order.Order{}).
		Where("status != ?", order.StatusCancelled).
		Select("COALESCE(SUM(total_amount), 0)").
		Scan(&summary.TotalRevenue)

	// Total Orders
	r.db.WithContext(ctx).Model(&order.Order{}).Count(&summary.TotalOrders)

	// Total Customers
	r.db.WithContext(ctx).Model(&auth.User{}).Where("role = ?", auth.RoleCustomer).Count(&summary.TotalCustomers)

	// Low Stock Products
	r.db.WithContext(ctx).Model(&product.Product{}).
		Where("stock_quantity <= low_stock_threshold AND is_active = ?", true).
		Count(&summary.LowStockCount)

	// Pending Orders
	r.db.WithContext(ctx).Model(&order.Order{}).Where("status = ?", order.StatusPending).Count(&summary.PendingOrdersCount)

	return &summary, nil
}

func (r *repository) GetRevenueTrends(ctx context.Context, days int) ([]RevenueChartPoint, error) {
	var points []RevenueChartPoint
	if days <= 0 {
		days = 7
	}

	startDate := time.Now().AddDate(0, 0, -days+1)

	type TrendRow struct {
		DateStr    string  `gorm:"column:date_str"`
		Revenue    float64 `gorm:"column:revenue"`
		OrderCount int64   `gorm:"column:order_count"`
	}
	var rows []TrendRow

	// Works in MySQL (and SQLite) using standard date formatting functions
	err := r.db.WithContext(ctx).Raw(`
		SELECT 
			DATE(created_at) as date_str, 
			COALESCE(SUM(total_amount), 0) as revenue, 
			COUNT(id) as order_count 
		FROM orders 
		WHERE created_at >= ? AND status != 'CANCELLED'
		GROUP BY DATE(created_at) 
		ORDER BY DATE(created_at) ASC
	`, startDate).Scan(&rows).Error

	if err != nil {
		return nil, err
	}

	dateMap := make(map[string]TrendRow)
	for _, row := range rows {
		dateMap[row.DateStr] = row
	}

	// Fill every day in the range even if revenue is 0
	for i := 0; i < days; i++ {
		d := startDate.AddDate(0, 0, i).Format("2006-01-02")
		if val, exists := dateMap[d]; exists {
			points = append(points, RevenueChartPoint{
				Date:       d,
				Revenue:    val.Revenue,
				OrderCount: val.OrderCount,
			})
		} else {
			points = append(points, RevenueChartPoint{
				Date:       d,
				Revenue:    0,
				OrderCount: 0,
			})
		}
	}

	return points, nil
}

func (r *repository) GetTopSellingProducts(ctx context.Context, limit int) ([]TopSellingProduct, error) {
	if limit <= 0 {
		limit = 5
	}
	var items []TopSellingProduct

	err := r.db.WithContext(ctx).Raw(`
		SELECT 
			order_items.product_id,
			order_items.product_name,
			order_items.product_sku,
			order_items.product_image,
			MAX(order_items.unit_price) as price,
			SUM(order_items.quantity) as total_sold,
			SUM(order_items.subtotal) as total_revenue
		FROM order_items
		INNER JOIN orders ON orders.id = order_items.order_id
		WHERE orders.status != 'CANCELLED'
		GROUP BY order_items.product_id, order_items.product_name, order_items.product_sku, order_items.product_image
		ORDER BY total_sold DESC
		LIMIT ?
	`, limit).Scan(&items).Error

	return items, err
}

func (r *repository) GetLowStockAlerts(ctx context.Context) ([]LowStockAlert, error) {
	var alerts []LowStockAlert
	err := r.db.WithContext(ctx).Model(&product.Product{}).
		Where("stock_quantity <= low_stock_threshold AND is_active = ?", true).
		Select("id as product_id, name as product_name, sku as product_sku, stock_quantity, low_stock_threshold").
		Order("stock_quantity ASC").
		Limit(10).
		Scan(&alerts).Error

	return alerts, err
}

func (r *repository) GetRecentOrders(ctx context.Context, limit int) ([]RecentOrderSummary, error) {
	if limit <= 0 {
		limit = 5
	}
	var summaries []RecentOrderSummary

	err := r.db.WithContext(ctx).Table("orders").
		Select("id, order_number, shipping_name as customer_name, total_amount, status, created_at").
		Order("created_at DESC").
		Limit(limit).
		Scan(&summaries).Error

	return summaries, err
}
