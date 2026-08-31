package product

// CreateProductRequest defines the input payload for creating a new catalog product
type CreateProductRequest struct {
	CategoryID        uint    `json:"category_id" binding:"required"`
	SubCategoryID     *uint   `json:"sub_category_id"`
	Name              string  `json:"name" binding:"required,min=2,max=200"`
	SKU               string  `json:"sku" binding:"required,min=2,max=100"`
	Description       string  `json:"description"`
	Price             float64 `json:"price" binding:"required,gt=0"`
	Currency          string  `json:"currency"`
	StockQuantity     int     `json:"stock_quantity" binding:"gte=0"`
	LowStockThreshold int     `json:"low_stock_threshold" binding:"gte=0"`
	ImageURL          string  `json:"image_url"`
	IsActive          *bool   `json:"is_active"`
	Badge             string  `json:"badge"`
}

// UpdateProductRequest defines the payload for updating an existing product
type UpdateProductRequest struct {
	CategoryID        *uint    `json:"category_id"`
	SubCategoryID     *uint    `json:"sub_category_id"`
	Name              *string  `json:"name"`
	SKU               *string  `json:"sku"`
	Description       *string  `json:"description"`
	Price             *float64 `json:"price"`
	Currency          *string  `json:"currency"`
	StockQuantity     *int     `json:"stock_quantity"`
	LowStockThreshold *int     `json:"low_stock_threshold"`
	ImageURL          *string  `json:"image_url"`
	IsActive          *bool    `json:"is_active"`
	Badge             *string  `json:"badge"`
}

// StockAdjustRequest defines input for inventory stock adjustment
type StockAdjustRequest struct {
	Type   string `json:"type" binding:"required,oneof=ADD SUBTRACT SET"` // ADD, SUBTRACT, SET
	Amount int    `json:"amount" binding:"required,gte=0"`
	Reason string `json:"reason" binding:"required"`
}

// CreateCategoryRequest defines input for category creation
type CreateCategoryRequest struct {
	Name        string `json:"name" binding:"required,min=2,max=100"`
	Description string `json:"description"`
	Icon        string `json:"icon"`
}

// CreateSubCategoryRequest defines input for sub-category creation
type CreateSubCategoryRequest struct {
	CategoryID  uint   `json:"category_id" binding:"required"`
	Name        string `json:"name" binding:"required,min=2,max=100"`
	Description string `json:"description"`
	Icon        string `json:"icon"`
}

// ProductFilterQuery defines query parameters for listing and filtering products
type ProductFilterQuery struct {
	Search        string  `form:"search"`
	Semantic      bool    `form:"semantic"`
	CategoryID    uint    `form:"category_id"`
	SubCategoryID uint    `form:"sub_category_id"`
	MinPrice      float64 `form:"min_price"`
	MaxPrice      float64 `form:"max_price"`
	Currency      string  `form:"currency"`
	InStock       *bool   `form:"in_stock"`
	Sort          string  `form:"sort"` // "price_asc", "price_desc", "newest", "name_asc"
	Page          int     `form:"page,default=1"`
	Limit         int     `form:"limit,default=12"`
	IsAdmin       bool    `form:"-"`
}
