package tools

import (
	"context"
	"fmt"
	"strings"

	"tirenn-ai-commerce/internal/domain/product"
)

// AddToCartTool validates product and formats cart action for UI via product.Repository
type AddToCartTool struct {
	repo product.Repository
}

func NewAddToCartTool(repo product.Repository) *AddToCartTool {
	return &AddToCartTool{repo: repo}
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
				"description": "Product SKU to add to cart (e.g. 'ID-AUD-001', 'ID-WCL-001', 'EN-AUD-003').",
			},
			"qty": map[string]interface{}{
				"type":        "integer",
				"description": "Quantity to add to cart (default: 1).",
			},
			"quantity": map[string]interface{}{
				"type":        "integer",
				"description": "Alternative parameter for quantity (default: 1).",
			},
			"product_id": map[string]interface{}{
				"type":        "integer",
				"description": "Product numeric ID (optional if SKU is provided).",
			},
		},
		"required": []string{"sku"},
	}
}

func (t *AddToCartTool) Execute(ctx context.Context, args map[string]interface{}, contextMap map[string]interface{}) (map[string]interface{}, error) {
	sku, _ := args["sku"].(string)
	sku = strings.TrimSpace(sku)
	qty := 1
	if q, ok := args["qty"].(float64); ok && q > 0 {
		qty = int(q)
	} else if q, ok := args["qty"].(int); ok && q > 0 {
		qty = q
	} else if q, ok := args["quantity"].(float64); ok && q > 0 {
		qty = int(q)
	} else if q, ok := args["quantity"].(int); ok && q > 0 {
		qty = q
	}

	productID := uint(0)
	if pID, ok := args["product_id"].(float64); ok && pID > 0 {
		productID = uint(pID)
	} else if pID, ok := args["product_id"].(int); ok && pID > 0 {
		productID = uint(pID)
	}

	if sku == "" && productID == 0 {
		return map[string]interface{}{"action": "need_clarification", "status": "error", "message": "Product SKU is required."}, nil
	}

	var prod *product.Product
	var err error

	if sku != "" {
		prod, err = t.repo.FindBySKU(ctx, sku)
	} else {
		prod, err = t.repo.FindByID(ctx, productID)
	}

	if err != nil || prod == nil || !prod.IsActive {
		return map[string]interface{}{"action": "not_found", "status": "not_found", "sku": sku, "message": fmt.Sprintf("Product with SKU '%s' not found.", sku)}, nil
	}

	if prod.StockQuantity <= 0 {
		return map[string]interface{}{
			"action":         "out_of_stock",
			"status":         "out_of_stock",
			"id":             prod.ID,
			"sku":            prod.SKU,
			"name":           prod.Name,
			"stock_quantity": 0,
			"message":        fmt.Sprintf("Sorry, %s is currently out of stock.", prod.Name),
		}, nil
	}

	actualQty := qty
	if actualQty > prod.StockQuantity {
		actualQty = prod.StockQuantity
	}

	curr := prod.Currency
	if curr == "" {
		if strings.HasPrefix(prod.SKU, "EN-") {
			curr = "USD"
		} else {
			curr = "IDR"
		}
	}

	prodMap := map[string]interface{}{
		"id":             prod.ID,
		"name":           prod.Name,
		"sku":            prod.SKU,
		"price":          prod.Price,
		"currency":       curr,
		"image_url":      prod.ImageURL,
		"stock_quantity": prod.StockQuantity,
		"quantity":       actualQty,
	}

	catName := prod.Category.Name
	subCatName := ""
	if prod.SubCategory != nil {
		subCatName = prod.SubCategory.Name
	}

	fullProdMap := map[string]interface{}{
		"id":             prod.ID,
		"name":           prod.Name,
		"sku":            prod.SKU,
		"price":          prod.Price,
		"currency":       curr,
		"image_url":      prod.ImageURL,
		"stock_quantity": prod.StockQuantity,
		"category":       catName,
		"sub_category":   subCatName,
		"rating":         prod.Rating,
		"in_stock":       true,
	}

	cartAction := map[string]interface{}{
		"action":         "cart_added",
		"type":           "ADD_TO_CART",
		"id":             prod.ID,
		"product_id":     prod.ID,
		"product_name":   prod.Name,
		"name":           prod.Name,
		"sku":            prod.SKU,
		"price":          prod.Price,
		"currency":       curr,
		"quantity":       actualQty,
		"stock_quantity": prod.StockQuantity,
		"image_url":      prod.ImageURL,
		"product":        prodMap,
	}

	return map[string]interface{}{
		"action":         "cart_added",
		"status":         "success",
		"id":             prod.ID,
		"name":           prod.Name,
		"sku":            prod.SKU,
		"price":          prod.Price,
		"currency":       curr,
		"quantity":       actualQty,
		"stock_quantity": prod.StockQuantity,
		"product":        prodMap,
		"cart_action":    cartAction,
		"_full_product":  fullProdMap,
		"message":        fmt.Sprintf("Added %d unit(s) of %s (%s) to shopping cart.", actualQty, prod.Name, prod.SKU),
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
