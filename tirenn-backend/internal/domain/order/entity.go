package order

import (
	"time"

	"gocommerce-backend/internal/domain/auth"
	"gocommerce-backend/internal/domain/product"
)

const (
	StatusPending    = "PENDING"
	StatusPaid       = "PAID"
	StatusProcessing = "PROCESSING"
	StatusShipped    = "SHIPPED"
	StatusCompleted  = "COMPLETED"
	StatusCancelled  = "CANCELLED"

	PaymentPending = "UNPAID"
	PaymentSuccess = "PAID"
	PaymentFailed  = "FAILED"
)

type Order struct {
	ID              uint            `gorm:"primaryKey;autoIncrement" json:"id"`
	OrderNumber     string          `gorm:"size:50;uniqueIndex;not null" json:"order_number"`
	UserID          uint            `gorm:"not null;index" json:"user_id"`
	User            auth.User       `gorm:"foreignKey:UserID" json:"user,omitempty"`
	TotalAmount     float64         `gorm:"type:decimal(12,2);not null;default:0.00" json:"total_amount"`
	Status          string          `gorm:"size:30;default:'PENDING';not null;index" json:"status"`
	ShippingName    string          `gorm:"size:100;not null" json:"shipping_name"`
	ShippingPhone   string          `gorm:"size:30;not null" json:"shipping_phone"`
	ShippingAddress string          `gorm:"type:text;not null" json:"shipping_address"`
	PaymentMethod   string          `gorm:"size:50;default:'SIMULATED_CARD';not null" json:"payment_method"`
	PaymentStatus   string          `gorm:"size:30;default:'PAID';not null" json:"payment_status"`
	Notes           string          `gorm:"type:text" json:"notes"`
	Items           []OrderItem     `gorm:"foreignKey:OrderID;constraint:OnDelete:CASCADE" json:"items"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
}

func (Order) TableName() string {
	return "orders"
}

type OrderItem struct {
	ID           uint            `gorm:"primaryKey;autoIncrement" json:"id"`
	OrderID      uint            `gorm:"not null;index" json:"order_id"`
	ProductID    uint            `gorm:"not null;index" json:"product_id"`
	Product      product.Product `gorm:"foreignKey:ProductID" json:"product,omitempty"`
	ProductName  string          `gorm:"size:200;not null" json:"product_name"`
	ProductSKU   string          `gorm:"size:100;not null" json:"product_sku"`
	ProductImage string          `gorm:"size:500" json:"product_image"`
	Quantity     int             `gorm:"not null" json:"quantity"`
	UnitPrice    float64         `gorm:"type:decimal(12,2);not null" json:"unit_price"`
	Subtotal     float64         `gorm:"type:decimal(12,2);not null" json:"subtotal"`
}

func (OrderItem) TableName() string {
	return "order_items"
}

type CheckoutItemRequest struct {
	ProductID uint `json:"product_id" binding:"required"`
	Quantity  int  `json:"quantity" binding:"required,gt=0"`
}

type CheckoutRequest struct {
	Items           []CheckoutItemRequest `json:"items" binding:"required,min=1"`
	ShippingName    string                `json:"shipping_name" binding:"required"`
	ShippingPhone   string                `json:"shipping_phone" binding:"required"`
	ShippingAddress string                `json:"shipping_address" binding:"required"`
	PaymentMethod   string                `json:"payment_method"`
	Notes           string                `json:"notes"`
}

type UpdateOrderStatusRequest struct {
	Status string `json:"status" binding:"required,oneof=PENDING PAID PROCESSING SHIPPED COMPLETED CANCELLED"`
	Notes  string `json:"notes"`
}

type OrderFilterQuery struct {
	Status string `form:"status"`
	Search string `form:"search"`
	Page   int    `form:"page,default=1"`
	Limit  int    `form:"limit,default=10"`
}
