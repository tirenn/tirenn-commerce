package product

import (
	"time"

	"github.com/pgvector/pgvector-go"
)

type Category struct {
	ID            uint          `gorm:"primaryKey;autoIncrement" json:"id"`
	Name          string        `gorm:"size:100;not null;uniqueIndex" json:"name"`
	Slug          string        `gorm:"size:120;not null;uniqueIndex" json:"slug"`
	Description   string        `gorm:"type:text" json:"description"`
	Icon          string        `gorm:"size:50" json:"icon"`
	SubCategories []SubCategory `gorm:"foreignKey:CategoryID;constraint:OnDelete:CASCADE" json:"sub_categories,omitempty"`
	CreatedAt     time.Time     `json:"created_at"`
	UpdatedAt     time.Time     `json:"updated_at"`
}

func (Category) TableName() string {
	return "categories"
}

type SubCategory struct {
	ID          uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	CategoryID  uint      `gorm:"not null;index" json:"category_id"`
	Category    *Category `gorm:"foreignKey:CategoryID;constraint:OnDelete:CASCADE" json:"category,omitempty"`
	Name        string    `gorm:"size:100;not null" json:"name"`
	Slug        string    `gorm:"size:120;not null;uniqueIndex" json:"slug"`
	Description string    `gorm:"type:text" json:"description"`
	Icon        string    `gorm:"size:50" json:"icon"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (SubCategory) TableName() string {
	return "sub_categories"
}

type Product struct {
	ID                uint             `gorm:"primaryKey;autoIncrement" json:"id"`
	CategoryID        uint             `gorm:"not null;index" json:"category_id"`
	Category          Category         `gorm:"foreignKey:CategoryID;constraint:OnDelete:CASCADE" json:"category,omitempty"`
	SubCategoryID     *uint            `gorm:"index" json:"sub_category_id,omitempty"`
	SubCategory       *SubCategory     `gorm:"foreignKey:SubCategoryID;constraint:OnDelete:SET NULL" json:"sub_category,omitempty"`
	Name              string           `gorm:"size:200;not null;index" json:"name"`
	Slug              string           `gorm:"size:220;not null;uniqueIndex" json:"slug"`
	SKU               string           `gorm:"size:100;not null;uniqueIndex" json:"sku"`
	Description       string           `gorm:"type:text" json:"description"`
	Price             float64          `gorm:"type:decimal(12,2);not null;default:0.00" json:"price"`
	Currency          string           `gorm:"size:10;not null;default:'IDR'" json:"currency"`
	StockQuantity     int              `gorm:"not null;default:0" json:"stock_quantity"`
	LowStockThreshold int              `gorm:"not null;default:5" json:"low_stock_threshold"`
	ImageURL          string           `gorm:"size:500" json:"image_url"`
	IsActive          bool             `gorm:"default:true;not null;index" json:"is_active"`
	Badge             string           `gorm:"size:50" json:"badge"`
	Rating            float64          `gorm:"type:decimal(3,2);default:5.00" json:"rating"`
	Embedding         *pgvector.Vector `gorm:"type:vector(384)" json:"-"`
	CreatedAt         time.Time        `json:"created_at"`
	UpdatedAt         time.Time        `json:"updated_at"`
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
