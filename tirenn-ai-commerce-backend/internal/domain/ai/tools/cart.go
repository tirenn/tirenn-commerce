package tools

import (
	"context"
	"fmt"
	"strings"

	"gorm.io/gorm"
)

// AddToCartTool validates product and formats cart action for UI
type AddToCartTool struct {
	db *gorm.DB
}

func NewAddToCartTool(db *gorm.DB) *AddToCartTool {
	return &AddToCartTool{db: db}
}

func (t *AddToCartTool) Name() string {
	return "add_to_cart"
}

func (t *AddToCartTool) Description() string {
	return "Add a specific product to the customer's shopping cart with requested quantity."
}

func (t *AddToCartTool) ParametersSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"sku": map[string]interface{}{
				"type":        "string",
				"description": "Product SKU code.",
			},
			"product_id": map[string]interface{}{
				"type":        "integer",
				"description": "Product numeric ID.",
			},
			"quantity": map[string]interface{}{
				"type":        "integer",
				"description": "Quantity to add to cart (default: 1).",
			},
		},
	}
}

func (t *AddToCartTool) Execute(ctx context.Context, args map[string]interface{}, contextMap map[string]interface{}) (map[string]interface{}, error) {
	sku, _ := args["sku"].(string)
	sku = strings.TrimSpace(sku)
	qty := 1
	if q, ok := args["quantity"].(float64); ok && q > 0 {
		qty = int(q)
	} else if q, ok := args["quantity"].(int); ok && q > 0 {
		qty = q
	}

	var prod CartProductInfo
	query := t.db.WithContext(ctx).Table("products").Select("id, name, sku, price, currency, stock_quantity, image_url").Where("is_active = true")
	if sku != "" {
		query = query.Where("sku = ?", sku)
	} else if pID, ok := args["product_id"].(float64); ok && pID > 0 {
		query = query.Where("id = ?", int64(pID))
	} else if pID, ok := args["product_id"].(int); ok && pID > 0 {
		query = query.Where("id = ?", int64(pID))
	} else {
		return map[string]interface{}{"status": "error", "message": "SKU or product_id required."}, nil
	}

	if err := query.First(&prod).Error; err != nil {
		return map[string]interface{}{"status": "not_found", "message": "Product not found."}, nil
	}

	if prod.StockQuantity < qty {
		return map[string]interface{}{
			"status":         "out_of_stock",
			"message":        fmt.Sprintf("Insufficient stock for %s. Available: %d", prod.Name, prod.StockQuantity),
			"available_qty": prod.StockQuantity,
		}, nil
	}

	prodMap := map[string]interface{}{
		"id":             prod.ID,
		"name":           prod.Name,
		"sku":            prod.SKU,
		"price":          prod.Price,
		"currency":       prod.Currency,
		"image_url":      prod.ImageURL,
		"stock_quantity": prod.StockQuantity,
	}

	cartAction := map[string]interface{}{
		"type":         "ADD_TO_CART",
		"product_id":   prod.ID,
		"product_name": prod.Name,
		"sku":          prod.SKU,
		"price":        prod.Price,
		"currency":     prod.Currency,
		"quantity":     qty,
		"image_url":    prod.ImageURL,
		"product":      prodMap,
	}

	return map[string]interface{}{
		"status":        "success",
		"message":       fmt.Sprintf("Added %d unit(s) of %s to shopping cart.", qty, prod.Name),
		"cart_action":   cartAction,
		"_full_product": prodMap,
	}, nil
}

// ViewCartTool inspects current shopping cart
type ViewCartTool struct{}

func NewViewCartTool() *ViewCartTool {
	return &ViewCartTool{}
}

func (t *ViewCartTool) Name() string {
	return "view_cart"
}

func (t *ViewCartTool) Description() string {
	return "View current items in the customer's shopping cart."
}

func (t *ViewCartTool) ParametersSchema() map[string]interface{} {
	return map[string]interface{}{
		"type":       "object",
		"properties": map[string]interface{}{},
	}
}

func (t *ViewCartTool) Execute(ctx context.Context, args map[string]interface{}, contextMap map[string]interface{}) (map[string]interface{}, error) {
	return map[string]interface{}{
		"status":  "success",
		"message": "Directing customer to view shopping cart.",
		"cart_action": map[string]interface{}{
			"type": "VIEW_CART",
		},
	}, nil
}
