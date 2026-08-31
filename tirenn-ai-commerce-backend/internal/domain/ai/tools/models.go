package tools

import (
	"time"
)

// CatalogProductRow represents scanned catalog product result from SQL
type CatalogProductRow struct {
	ID            int64   `json:"id"`
	Name          string  `json:"name"`
	SKU           string  `json:"sku"`
	Description   string  `json:"description"`
	Price         float64 `json:"price"`
	Currency      string  `json:"currency"`
	StockQuantity int     `json:"stock_quantity"`
	ImageURL      string  `json:"image_url"`
	CategoryName  string  `json:"category_name"`
	SubCatName    string  `json:"sub_cat_name"`
	Rating        float64 `json:"rating"`
	Similarity    float64 `json:"similarity"`
}

// CatalogProductDetail represents detailed product view from SQL
type CatalogProductDetail struct {
	ID                int64   `json:"id"`
	Name              string  `json:"name"`
	Slug              string  `json:"slug"`
	SKU               string  `json:"sku"`
	Description       string  `json:"description"`
	Price             float64 `json:"price"`
	Currency          string  `json:"currency"`
	StockQuantity     int     `json:"stock_quantity"`
	LowStockThreshold int     `json:"low_stock_threshold"`
	ImageURL          string  `json:"image_url"`
	IsActive          bool    `json:"is_active"`
	Badge             string  `json:"badge"`
	Rating            float64 `json:"rating"`
	CategoryName      string  `json:"category_name"`
	SubCatName        string  `json:"sub_cat_name"`
}

// CartProductInfo represents minimal product info needed for cart operations
type CartProductInfo struct {
	ID            int64   `json:"id"`
	Name          string  `json:"name"`
	SKU           string  `json:"sku"`
	Price         float64 `json:"price"`
	Currency      string  `json:"currency"`
	StockQuantity int     `json:"stock_quantity"`
	ImageURL      string  `json:"image_url"`
}

// AnalyticsTopProduct represents top performing product in analytics
type AnalyticsTopProduct struct {
	ID         int64   `json:"id"`
	Name       string  `json:"name"`
	SKU        string  `json:"sku"`
	TotalSold  int64   `json:"total_sold"`
	TotalSales float64 `json:"total_sales"`
}

// AnalyticsOrderRow represents order summary row in analytics
type AnalyticsOrderRow struct {
	ID            int64     `json:"id"`
	OrderNumber   string    `json:"order_number"`
	CustomerName  string    `json:"customer_name"`
	CustomerEmail string    `json:"customer_email"`
	TotalAmount   float64   `json:"total_amount"`
	Status        string    `json:"status"`
	ItemCount     int       `json:"item_count"`
	CreatedAt     time.Time `json:"created_at"`
}

// InventoryLowStockProduct represents low stock alert row
type InventoryLowStockProduct struct {
	ID            int64   `json:"id"`
	Name          string  `json:"name"`
	SKU           string  `json:"sku"`
	StockQuantity int     `json:"stock_quantity"`
	Price         float64 `json:"price"`
	CategoryName  string  `json:"category_name"`
}

// InventoryProductRecord represents database record for inventory transactions
type InventoryProductRecord struct {
	ID            int64  `json:"id"`
	Name          string `json:"name"`
	SKU           string `json:"sku"`
	StockQuantity int    `json:"stock_quantity"`
}

// InventoryStockAdjustmentLog represents stock adjustment audit log entry
type InventoryStockAdjustmentLog struct {
	ID             int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	ProductID      int64     `gorm:"not null;index" json:"product_id"`
	AdjustmentType string    `gorm:"size:20;not null" json:"adjustment_type"`
	Quantity       int       `gorm:"not null" json:"quantity"`
	PreviousStock  int       `gorm:"not null" json:"previous_stock"`
	NewStock       int       `gorm:"not null" json:"new_stock"`
	Reason         string    `gorm:"size:255" json:"reason"`
	AdjustedBy     int64     `gorm:"not null" json:"adjusted_by"`
	CreatedAt      time.Time `gorm:"autoCreateTime" json:"created_at"`
}

func (InventoryStockAdjustmentLog) TableName() string {
	return "stock_adjustment_logs"
}

// RAGSearchResult represents a retrieved chunk with similarity score
type RAGSearchResult struct {
	ChunkID       int64   `json:"chunk_id"`
	DocumentID    int64   `json:"document_id"`
	DocumentTitle string  `json:"document_title"`
	DocType       string  `json:"doc_type"`
	PageNumber    int     `json:"page_number"`
	Content       string  `json:"content"`
	Similarity    float64 `json:"similarity"`
}
