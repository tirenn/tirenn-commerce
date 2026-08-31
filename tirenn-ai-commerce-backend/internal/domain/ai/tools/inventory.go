package tools

import (
	"context"
	"fmt"
	"log"
	"math"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// GetLowStockProductsTool retrieves products with inventory at or below threshold
type GetLowStockProductsTool struct {
	db *gorm.DB
}

func NewGetLowStockProductsTool(db *gorm.DB) *GetLowStockProductsTool {
	return &GetLowStockProductsTool{db: db}
}

func (t *GetLowStockProductsTool) Name() string {
	return "get_low_stock_products"
}

func (t *GetLowStockProductsTool) Description() string {
	return "Identify and list all inventory items running low on stock below threshold (default: 10 units)."
}

func (t *GetLowStockProductsTool) ParametersSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"threshold": map[string]interface{}{
				"type":        "integer",
				"description": "Stock threshold count (default: 10).",
			},
			"limit": map[string]interface{}{
				"type":        "integer",
				"description": "Maximum number of items to return (default: 10).",
			},
		},
	}
}

func (t *GetLowStockProductsTool) Execute(ctx context.Context, args map[string]interface{}, contextMap map[string]interface{}) (map[string]interface{}, error) {
	threshold := 10
	if th, ok := args["threshold"].(float64); ok && th > 0 {
		threshold = int(th)
	} else if th, ok := args["threshold"].(int); ok && th > 0 {
		threshold = th
	}

	limit := 10
	if l, ok := args["limit"].(float64); ok && l > 0 {
		limit = int(l)
	} else if l, ok := args["limit"].(int); ok && l > 0 {
		limit = l
	}

	var prods []InventoryLowStockProduct
	err := t.db.WithContext(ctx).Table("products p").
		Select("p.id, p.name, p.sku, p.stock_quantity, p.price, c.name as category_name").
		Joins("LEFT JOIN categories c ON c.id = p.category_id").
		Where("p.is_active = true AND p.stock_quantity <= ?", threshold).
		Order("p.stock_quantity ASC").
		Limit(limit).
		Scan(&prods).Error

	if err != nil {
		return map[string]interface{}{"status": "error", "message": err.Error()}, nil
	}

	return map[string]interface{}{
		"status":      "success",
		"threshold":   threshold,
		"total_found": len(prods),
		"products":    prods,
	}, nil
}

// AdjustProductStockTool performs warehouse stock adjustments with strict 2-step confirmation guardrail
type AdjustProductStockTool struct {
	db *gorm.DB
}

func NewAdjustProductStockTool(db *gorm.DB) *AdjustProductStockTool {
	return &AdjustProductStockTool{db: db}
}

func (t *AdjustProductStockTool) Name() string {
	return "adjust_product_stock"
}

func (t *AdjustProductStockTool) Description() string {
	return "Adjust inventory stock quantity (ADD, SUBTRACT, SET) with mandatory 2-step confirmation guardrail."
}

func (t *AdjustProductStockTool) ParametersSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"sku": map[string]interface{}{
				"type":        "string",
				"description": "Product SKU code (e.g. 'ID-AUD-001', 'EN-AUD-001').",
			},
			"product_id": map[string]interface{}{
				"type":        "integer",
				"description": "Product numeric ID.",
			},
			"adjustment_type": map[string]interface{}{
				"type":        "string",
				"enum":        []string{"ADD", "SUBTRACT", "SET"},
				"description": "Operation type: 'ADD' (increase stock), 'SUBTRACT' (decrease stock), or 'SET' (set exact stock).",
			},
			"amount": map[string]interface{}{
				"type":        "integer",
				"description": "Quantity amount to adjust by (positive integer).",
			},
			"reason": map[string]interface{}{
				"type":        "string",
				"description": "Mandatory audit reason for this inventory adjustment.",
			},
			"confirmed": map[string]interface{}{
				"type":        "boolean",
				"description": "True if the Admin has explicitly confirmed this stock modification. False for proposal preview.",
			},
		},
		"required": []string{"amount", "reason"},
	}
}

func (t *AdjustProductStockTool) Execute(ctx context.Context, args map[string]interface{}, contextMap map[string]interface{}) (map[string]interface{}, error) {
	sku, _ := args["sku"].(string)
	sku = strings.TrimSpace(sku)

	adjType, _ := args["adjustment_type"].(string)
	if adjType == "" {
		if tVal, ok := args["type"].(string); ok {
			adjType = tVal
		} else {
			adjType = "ADD"
		}
	}
	adjType = strings.ToUpper(strings.TrimSpace(adjType))

	rawAmount := 0
	if a, ok := args["amount"].(float64); ok {
		rawAmount = int(math.Abs(a))
	} else if a, ok := args["amount"].(int); ok {
		rawAmount = int(math.Abs(float64(a)))
	}

	reason, _ := args["reason"].(string)
	if reason == "" {
		reason = fmt.Sprintf("Stock adjustment (%s)", sku)
	}

	confirmed, _ := args["confirmed"].(bool)

	var prod InventoryProductRecord
	q := t.db.WithContext(ctx).Table("products").Select("id, name, sku, stock_quantity").Where("is_active = true")
	if sku != "" {
		q = q.Where("sku = ?", sku)
	} else if pID, ok := args["product_id"].(float64); ok && pID > 0 {
		q = q.Where("id = ?", int64(pID))
	} else if pID, ok := args["product_id"].(int); ok && pID > 0 {
		q = q.Where("id = ?", int64(pID))
	} else {
		return map[string]interface{}{"status": "error", "message": "SKU or product_id required."}, nil
	}

	if err := q.First(&prod).Error; err != nil {
		return map[string]interface{}{"status": "not_found", "message": "Product not found in catalog."}, nil
	}

	projectedStock := prod.StockQuantity
	qtyChange := rawAmount
	switch adjType {
	case "ADD":
		projectedStock = prod.StockQuantity + rawAmount
	case "SUBTRACT":
		projectedStock = prod.StockQuantity - rawAmount
		qtyChange = -rawAmount
		if projectedStock < 0 {
			return map[string]interface{}{
				"status":        "error",
				"message":       fmt.Sprintf("Cannot subtract %d units. Current stock is only %d units.", rawAmount, prod.StockQuantity),
				"current_stock": prod.StockQuantity,
			}, nil
		}
	case "SET":
		projectedStock = rawAmount
		qtyChange = rawAmount - prod.StockQuantity
	}

	// STEP 1: Guardrail Active
	if !confirmed {
		log.Printf("🛑 [GUARDRAIL ACTIVE] Blocking unconfirmed stock mutation for SKU '%s'", prod.SKU)
		return map[string]interface{}{
			"status":              "requires_confirmation",
			"product_id":          prod.ID,
			"product_name":        prod.Name,
			"sku":                 prod.SKU,
			"adjustment_type":     adjType,
			"amount":              rawAmount,
			"current_stock":       prod.StockQuantity,
			"projected_new_stock": projectedStock,
			"reason":              reason,
			"message":             fmt.Sprintf("Stock adjustment proposal: %s (%s) from %d -> %d units. Awaiting confirmation.", prod.Name, prod.SKU, prod.StockQuantity, projectedStock),
			"instruction_for_llm": "Present these adjustment details to the admin and ask for confirmation. When the admin confirms, call adjust_product_stock again with confirmed=true.",
		}, nil
	}

	// STEP 2: Execute Atomic PostgreSQL Transaction
	adminID := int64(1)
	if aID, ok := contextMap["admin_id"].(int64); ok && aID > 0 {
		adminID = aID
	} else if aID, ok := contextMap["admin_id"].(float64); ok && aID > 0 {
		adminID = int64(aID)
	}

	var auditLogID int64
	var finalNewStock int

	err := t.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var lockedProd InventoryProductRecord
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Table("products").
			Where("id = ?", prod.ID).First(&lockedProd).Error; err != nil {
			return err
		}

		prevStock := lockedProd.StockQuantity
		newStock := prevStock
		switch adjType {
		case "ADD":
			newStock = prevStock + rawAmount
		case "SUBTRACT":
			newStock = prevStock - rawAmount
			if newStock < 0 {
				return fmt.Errorf("insufficient stock: current %d, attempted reduction %d", prevStock, rawAmount)
			}
		case "SET":
			newStock = rawAmount
		}

		// Update product stock
		if err := tx.Table("products").Where("id = ?", prod.ID).Update("stock_quantity", newStock).Error; err != nil {
			return err
		}

		// Insert stock adjustment audit log
		auditLog := InventoryStockAdjustmentLog{
			ProductID:      prod.ID,
			AdjustmentType: adjType,
			Quantity:       qtyChange,
			PreviousStock:  prevStock,
			NewStock:       newStock,
			Reason:         reason,
			AdjustedBy:     adminID,
		}

		if err := tx.Table("stock_adjustment_logs").Create(&auditLog).Error; err != nil {
			return err
		}

		auditLogID = auditLog.ID
		finalNewStock = newStock
		return nil
	})

	if err != nil {
		log.Printf("❌ [AdjustProductStockTool] Transaction failed: %v", err)
		return map[string]interface{}{"status": "error", "message": err.Error()}, nil
	}

	log.Printf("✅ [AdjustProductStockTool] Stock committed: %s (%s) %d -> %d (Audit Log #%d)", prod.Name, prod.SKU, prod.StockQuantity, finalNewStock, auditLogID)

	return map[string]interface{}{
		"status":          "success",
		"product_id":      prod.ID,
		"product_name":    prod.Name,
		"sku":             prod.SKU,
		"adjustment_type": adjType,
		"amount":          rawAmount,
		"previous_stock":  prod.StockQuantity,
		"new_stock":       finalNewStock,
		"audit_log_id":    auditLogID,
		"reason":          reason,
		"message":         fmt.Sprintf("Stock successfully updated to %d units (Audit Log #%d).", finalNewStock, auditLogID),
	}, nil
}
