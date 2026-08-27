package product

import (
	"time"
)

type Category struct {
	ID          uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	Name        string    `gorm:"size:100;not null;uniqueIndex" json:"name"`
	Slug        string    `gorm:"size:120;not null;uniqueIndex" json:"slug"`
	Description string    `gorm:"type:text" json:"description"`
	Icon        string    `gorm:"size:50" json:"icon"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (Category) TableName() string {
	return "categories"
}

type Product struct {
	ID                uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	CategoryID        uint      `gorm:"not null;index" json:"category_id"`
	Category          Category  `gorm:"foreignKey:CategoryID;constraint:OnDelete:CASCADE" json:"category,omitempty"`
	Name              string    `gorm:"size:200;not null;index" json:"name"`
	Slug              string    `gorm:"size:220;not null;uniqueIndex" json:"slug"`
	SKU               string    `gorm:"size:100;not null;uniqueIndex" json:"sku"`
	Description       string    `gorm:"type:text" json:"description"`
	Price             float64   `gorm:"type:decimal(12,2);not null;default:0.00" json:"price"`
	StockQuantity     int       `gorm:"not null;default:0" json:"stock_quantity"`
	LowStockThreshold int       `gorm:"not null;default:5" json:"low_stock_threshold"`
	ImageURL          string    `gorm:"size:500" json:"image_url"`
	IsActive          bool      `gorm:"default:true;not null;index" json:"is_active"`
	Badge             string    `gorm:"size:50" json:"badge"` // e.g. "HOT!", "POW!", "SALE", "NEW"
	Rating            float64   `gorm:"type:decimal(3,2);default:5.00" json:"rating"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

func (Product) TableName() string {
	return "products"
}

type StockAdjustmentLog struct {
	ID             uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	ProductID      uint      `gorm:"not null;index" json:"product_id"`
	AdjustmentType string    `gorm:"size:20;not null" json:"adjustment_type"` // ADD, SUBTRACT, SET
	Quantity       int       `gorm:"not null" json:"quantity"`
	PreviousStock  int       `gorm:"not null" json:"previous_stock"`
	NewStock       int       `gorm:"not null" json:"new_stock"`
	Reason         string    `gorm:"size:255" json:"reason"` // Restock, Inventory Audit, Damaged
	AdjustedBy     uint      `gorm:"not null" json:"adjusted_by"`
	CreatedAt      time.Time `json:"created_at"`
}

func (StockAdjustmentLog) TableName() string {
	return "stock_adjustment_logs"
}

type CreateProductRequest struct {
	CategoryID        uint    `json:"category_id" binding:"required"`
	Name              string  `json:"name" binding:"required,min=2,max=200"`
	SKU               string  `json:"sku" binding:"required,min=2,max=100"`
	Description       string  `json:"description"`
	Price             float64 `json:"price" binding:"required,gt=0"`
	StockQuantity     int     `json:"stock_quantity" binding:"gte=0"`
	LowStockThreshold int     `json:"low_stock_threshold" binding:"gte=0"`
	ImageURL          string  `json:"image_url"`
	IsActive          *bool   `json:"is_active"`
	Badge             string  `json:"badge"`
}

type UpdateProductRequest struct {
	CategoryID        *uint    `json:"category_id"`
	Name              *string  `json:"name"`
	SKU               *string  `json:"sku"`
	Description       *string  `json:"description"`
	Price             *float64 `json:"price"`
	StockQuantity     *int     `json:"stock_quantity"`
	LowStockThreshold *int     `json:"low_stock_threshold"`
	ImageURL          *string  `json:"image_url"`
	IsActive          *bool    `json:"is_active"`
	Badge             *string  `json:"badge"`
}

type StockAdjustRequest struct {
	Type   string `json:"type" binding:"required,oneof=ADD SUBTRACT SET"` // ADD, SUBTRACT, SET
	Amount int    `json:"amount" binding:"required,gte=0"`
	Reason string `json:"reason" binding:"required"`
}

type CreateCategoryRequest struct {
	Name        string `json:"name" binding:"required,min=2,max=100"`
	Description string `json:"description"`
	Icon        string `json:"icon"`
}

type ProductFilterQuery struct {
	Search     string  `form:"search"`
	CategoryID uint    `form:"category_id"`
	MinPrice   float64 `form:"min_price"`
	MaxPrice   float64 `form:"max_price"`
	InStock    *bool   `form:"in_stock"`
	Sort       string  `form:"sort"` // "price_asc", "price_desc", "newest", "name_asc"
	Page       int     `form:"page,default=1"`
	Limit      int     `form:"limit,default=12"`
	IsAdmin    bool    `form:"-"`
}
