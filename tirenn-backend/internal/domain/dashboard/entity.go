package dashboard

import "time"

type DashboardSummary struct {
	TotalRevenue       float64 `json:"total_revenue"`
	TotalOrders        int64   `json:"total_orders"`
	TotalCustomers     int64   `json:"total_customers"`
	LowStockCount      int64   `json:"low_stock_count"`
	PendingOrdersCount int64   `json:"pending_orders_count"`
}

type RevenueChartPoint struct {
	Date       string  `json:"date"`
	Revenue    float64 `json:"revenue"`
	OrderCount int64   `json:"order_count"`
}

type TopSellingProduct struct {
	ProductID    uint    `json:"product_id"`
	ProductName  string  `json:"product_name"`
	ProductSKU   string  `json:"product_sku"`
	ProductImage string  `json:"product_image"`
	Price        float64 `json:"price"`
	TotalSold    int64   `json:"total_sold"`
	TotalRevenue float64 `json:"total_revenue"`
}

type LowStockAlert struct {
	ProductID         uint   `json:"product_id"`
	ProductName       string `json:"product_name"`
	ProductSKU        string `json:"product_sku"`
	StockQuantity     int    `json:"stock_quantity"`
	LowStockThreshold int    `json:"low_stock_threshold"`
}

type RecentOrderSummary struct {
	ID           uint      `json:"id"`
	OrderNumber  string    `json:"order_number"`
	CustomerName string    `json:"customer_name"`
	TotalAmount  float64   `json:"total_amount"`
	Status       string    `json:"status"`
	CreatedAt    time.Time `json:"created_at"`
}

type DashboardResponse struct {
	Summary            DashboardSummary     `json:"summary"`
	RevenueTrends      []RevenueChartPoint  `json:"revenue_trends"`
	TopSellingProducts []TopSellingProduct  `json:"top_selling_products"`
	LowStockAlerts     []LowStockAlert      `json:"low_stock_alerts"`
	RecentOrders       []RecentOrderSummary `json:"recent_orders"`
}
