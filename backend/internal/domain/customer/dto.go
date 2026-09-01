package customer

import "time"

// CustomerListItem represents enriched customer aggregate data
type CustomerListItem struct {
	ID           uint      `json:"id"`
	Name         string    `json:"name"`
	Email        string    `json:"email"`
	Phone        string    `json:"phone"`
	Address      string    `json:"address"`
	Status       string    `json:"status"`
	TotalOrders  int64     `json:"total_orders"`
	TotalSpent   float64   `json:"total_spent"`
	RegisteredAt time.Time `json:"registered_at"`
}

// UpdateCustomerStatusRequest defines input for status updates (Admin)
type UpdateCustomerStatusRequest struct {
	Status string `json:"status" binding:"required,oneof=ACTIVE SUSPENDED"`
}

// CustomerFilterQuery defines pagination and search filters for customer list
type CustomerFilterQuery struct {
	Search string `form:"search"`
	Status string `form:"status"`
	Page   int    `form:"page,default=1"`
	Limit  int    `form:"limit,default=10"`
}
