package order

// CheckoutItemRequest defines product ID and quantity item in checkout
type CheckoutItemRequest struct {
	ProductID uint `json:"product_id" binding:"required"`
	Quantity  int  `json:"quantity" binding:"required,gt=0"`
}

// CheckoutRequest defines customer order placement payload
type CheckoutRequest struct {
	Items           []CheckoutItemRequest `json:"items" binding:"required,min=1"`
	ShippingName    string                `json:"shipping_name" binding:"required"`
	ShippingPhone   string                `json:"shipping_phone" binding:"required"`
	ShippingAddress string                `json:"shipping_address" binding:"required"`
	PaymentMethod   string                `json:"payment_method"`
	Currency        string                `json:"currency"`
	Notes           string                `json:"notes"`
}

// UpdateOrderStatusRequest defines merchant status transition payload
type UpdateOrderStatusRequest struct {
	Status string `json:"status" binding:"required,oneof=PENDING PAID PROCESSING SHIPPED COMPLETED CANCELLED"`
	Notes  string `json:"notes"`
}

// OrderFilterQuery defines pagination and search filters for orders
type OrderFilterQuery struct {
	Status string `form:"status"`
	Search string `form:"search"`
	Page   int    `form:"page,default=1"`
	Limit  int    `form:"limit,default=10"`
}
