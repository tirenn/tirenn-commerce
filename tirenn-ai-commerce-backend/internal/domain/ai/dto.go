package ai

import "tirenn-ai-commerce/internal/domain/ai/tools"

// ChatShopperRequest is the request payload for shopper chat
type ChatShopperRequest struct {
	Messages  []ChatMessage `json:"messages" binding:"required"`
	SessionID string        `json:"session_id"`
}

// ChatAdminRequest is the request payload for admin copilot chat
type ChatAdminRequest struct {
	Messages  []ChatMessage `json:"messages" binding:"required"`
	SessionID string        `json:"session_id"`
}

// ChatShopperResult is the final output of the ReAct agent harness
type ChatShopperResult struct {
	Reply             string                   `json:"reply"`
	ToolCalls         []ExecutedToolRecord     `json:"tool_calls"`
	SuggestedProducts []map[string]interface{} `json:"suggested_products"`
	CartAction        map[string]interface{}   `json:"cart_action,omitempty"`
	LatencyMs         float64                  `json:"latency_ms"`
}

// AskKnowledgeRequest represents a request to search/ask the RAG knowledge base
type AskKnowledgeRequest struct {
	Query   string `json:"query" binding:"required"`
	DocType string `json:"doc_type"`
	TopK    int    `json:"top_k"`
}

// RAGSearchResult aliases the canonical tool search result model
type RAGSearchResult = tools.RAGSearchResult

// ==============================================================================
// Strongly-Typed AI Tool Input DTOs
// ==============================================================================

// SearchProductsInput defines typed inputs for catalog search tool
type SearchProductsInput struct {
	Query    string  `json:"query"`
	Category string  `json:"category"`
	MinPrice float64 `json:"min_price"`
	MaxPrice float64 `json:"max_price"`
	InStock  bool    `json:"in_stock"`
	Limit    int     `json:"limit"`
}

// GetProductDetailInput defines typed inputs for product detail tool
type GetProductDetailInput struct {
	SKU       string `json:"sku"`
	ProductID int64  `json:"product_id"`
}

// CheckStockInput defines typed inputs for checking stock availability
type CheckStockInput struct {
	SKU       string `json:"sku"`
	ProductID int64  `json:"product_id"`
}

// AddToCartInput defines typed inputs for cart addition tool
type AddToCartInput struct {
	SKU       string `json:"sku"`
	ProductID int64  `json:"product_id"`
	Quantity  int    `json:"quantity"`
}

// ViewCartInput defines typed inputs for viewing cart contents
type ViewCartInput struct{}

// ExecutiveMetricsInput defines typed inputs for dashboard KPIs tool
type ExecutiveMetricsInput struct {
	Days int `json:"days"`
}

// RecentOrdersInput defines typed inputs for recent orders tool
type RecentOrdersInput struct {
	Limit int `json:"limit"`
}

// LowStockInput defines typed inputs for low stock alerts tool
type LowStockInput struct {
	Threshold int `json:"threshold"`
}

// AdjustStockInput defines typed inputs for stock adjustment tool
type AdjustStockInput struct {
	SKU            string `json:"sku"`
	ProductID      int64  `json:"product_id"`
	QuantityChange int    `json:"quantity_change"`
	Reason         string `json:"reason"`
	Confirmed      bool   `json:"confirmed"`
}

// SearchPoliciesInput defines typed inputs for store policies tool
type SearchPoliciesInput struct {
	Query string `json:"query"`
	TopK  int    `json:"top_k"`
}

// SearchAdminSOPInput defines typed inputs for internal SOP tool
type SearchAdminSOPInput struct {
	Query string `json:"query"`
	TopK  int    `json:"top_k"`
}
