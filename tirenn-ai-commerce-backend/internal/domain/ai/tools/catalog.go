package tools

import (
	"context"
	"fmt"
	"log"
	"math"
	"strings"

	"github.com/pgvector/pgvector-go"
	"tirenn-ai-commerce/internal/client/ollama"
	"gorm.io/gorm"
)

// SearchProductsConfig defines hybrid search weights and similarity threshold parameters
type SearchProductsConfig struct {
	EnableHybridSearch          bool
	HybridVectorWeight          float64
	HybridTextWeight            float64
	ChatSearchScoreThreshold   float64
	ChatSearchFallbackThreshold float64
	ChatSearchLimit             int
}

// SearchProductsTool executes hybrid dense-vector + keyword ranking search on products
type SearchProductsTool struct {
	db           *gorm.DB
	ollamaClient *ollama.Client
	cfg          SearchProductsConfig
}

func NewSearchProductsTool(db *gorm.DB, ollamaClient *ollama.Client, cfgs ...SearchProductsConfig) *SearchProductsTool {
	cfg := SearchProductsConfig{
		EnableHybridSearch:          true,
		HybridVectorWeight:          0.70,
		HybridTextWeight:            0.30,
		ChatSearchScoreThreshold:   0.20,
		ChatSearchFallbackThreshold: 0.10,
		ChatSearchLimit:             6,
	}
	if len(cfgs) > 0 {
		if cfgs[0].HybridVectorWeight > 0 {
			cfg.HybridVectorWeight = cfgs[0].HybridVectorWeight
		}
		if cfgs[0].HybridTextWeight > 0 {
			cfg.HybridTextWeight = cfgs[0].HybridTextWeight
		}
		if cfgs[0].ChatSearchScoreThreshold > 0 {
			cfg.ChatSearchScoreThreshold = cfgs[0].ChatSearchScoreThreshold
		}
		if cfgs[0].ChatSearchFallbackThreshold > 0 {
			cfg.ChatSearchFallbackThreshold = cfgs[0].ChatSearchFallbackThreshold
		}
		if cfgs[0].ChatSearchLimit > 0 {
			cfg.ChatSearchLimit = cfgs[0].ChatSearchLimit
		}
		cfg.EnableHybridSearch = cfgs[0].EnableHybridSearch
	}

	return &SearchProductsTool{
		db:           db,
		ollamaClient: ollamaClient,
		cfg:          cfg,
	}
}

func (t *SearchProductsTool) Name() string {
	return "search_products"
}

func (t *SearchProductsTool) Description() string {
	return "Search products in store catalog using hybrid semantic vector search + keyword matching and structured filters (query, category, in_stock, min_price, max_price, limit)."
}

func (t *SearchProductsTool) ParametersSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"query": map[string]interface{}{
				"type":        "string",
				"description": "Product search keywords, description, or intent (e.g. 'wireless noise cancelling headphone', 'smartwatch').",
			},
			"category": map[string]interface{}{
				"type":        "string",
				"description": "Optional category name filter (e.g. 'Elektronik & Gadget', 'Fashion Pria', 'Smartwatch & Wearables').",
			},
			"in_stock": map[string]interface{}{
				"type":        "boolean",
				"description": "Filter products that are currently in stock (stock_quantity > 0).",
			},
			"min_price": map[string]interface{}{
				"type":        "number",
				"description": "Minimum product price filter.",
			},
			"max_price": map[string]interface{}{
				"type":        "number",
				"description": "Maximum product price filter.",
			},
			"limit": map[string]interface{}{
				"type":        "integer",
				"description": "Maximum number of products to return (default: 6, max: 12).",
			},
		},
		"required": []string{"query"},
	}
}

func (t *SearchProductsTool) Execute(ctx context.Context, args map[string]interface{}, contextMap map[string]interface{}) (map[string]interface{}, error) {
	query, _ := args["query"].(string)
	category, _ := args["category"].(string)
	inStock, _ := args["in_stock"].(bool)
	limit := t.cfg.ChatSearchLimit
	if limit <= 0 {
		limit = 6
	}
	if l, ok := args["limit"].(float64); ok && l > 0 {
		limit = int(math.Min(l, 12))
	} else if l, ok := args["limit"].(int); ok && l > 0 {
		limit = int(math.Min(float64(l), 12))
	}

	log.Printf("🔍 [CUSTOMER TOOL: search_products] query='%s' | category='%s' | inStock=%v | limit=%d | hybrid=%v", query, category, inStock, limit, t.cfg.EnableHybridSearch)

	// 1. Generate query embedding vector
	embedding, err := t.ollamaClient.GenerateEmbedding(ctx, query)
	if err != nil {
		log.Printf("⚠️ [SearchProductsTool] Embedding error: %v, falling back to pure text query", err)
	}

	var rows []CatalogProductRow

	if len(embedding) == 0 || !t.cfg.EnableHybridSearch {
		// Keyword search fallback
		kwQuery := `
			SELECT 
				p.id, p.name, p.sku, p.description, p.price, p.currency, p.stock_quantity, p.image_url,
				c.name as category_name, sc.name as sub_cat_name, p.rating,
				0.85 as similarity
			FROM products p
			LEFT JOIN categories c ON p.category_id = c.id
			LEFT JOIN sub_categories sc ON p.sub_category_id = sc.id
			WHERE p.is_active = true AND (LOWER(p.name) LIKE ? OR LOWER(p.description) LIKE ? OR LOWER(p.sku) LIKE ?)
		`
		params := []interface{}{"%" + strings.ToLower(query) + "%", "%" + strings.ToLower(query) + "%", "%" + strings.ToLower(query) + "%"}

		if category != "" {
			kwQuery += " AND (LOWER(c.name) LIKE ? OR LOWER(sc.name) LIKE ?)"
			params = append(params, "%"+strings.ToLower(category)+"%", "%"+strings.ToLower(category)+"%")
		}
		if inStock {
			kwQuery += " AND p.stock_quantity > 0"
		}
		kwQuery += " ORDER BY p.stock_quantity DESC LIMIT ?"
		params = append(params, limit)

		if err := t.db.WithContext(ctx).Raw(kwQuery, params...).Scan(&rows).Error; err != nil {
			return nil, fmt.Errorf("keyword product search failed: %w", err)
		}
	} else {
		// PostgreSQL Hybrid Ranking: Vector Cosine Similarity * VectorWeight + Trigram/Text Match * TextWeight
		vec := pgvector.NewVector(embedding)
		vw := t.cfg.HybridVectorWeight
		tw := t.cfg.HybridTextWeight
		threshold := t.cfg.ChatSearchScoreThreshold

		baseSQL := fmt.Sprintf(`
			SELECT 
				p.id, p.name, p.sku, p.description, p.price, p.currency, p.stock_quantity, p.image_url,
				c.name as category_name, sc.name as sub_cat_name, p.rating,
				((1.0 - (p.embedding <=> ?)) * %.2f + (CASE 
					WHEN LOWER(p.name) LIKE ? THEN 1.0 
					WHEN LOWER(p.description) LIKE ? THEN 0.5 
					ELSE 0.0 
				END) * %.2f) as similarity
			FROM products p
			LEFT JOIN categories c ON p.category_id = c.id
			LEFT JOIN sub_categories sc ON p.sub_category_id = sc.id
			WHERE p.is_active = true AND p.embedding IS NOT NULL
		`, vw, tw)

		kwPattern := "%" + strings.ToLower(query) + "%"
		params := []interface{}{vec, kwPattern, kwPattern}

		if category != "" {
			baseSQL += " AND (LOWER(c.name) LIKE ? OR LOWER(sc.name) LIKE ?)"
			params = append(params, "%"+strings.ToLower(category)+"%", "%"+strings.ToLower(category)+"%")
		}
		if inStock {
			baseSQL += " AND p.stock_quantity > 0"
		}

		baseSQL += fmt.Sprintf(" AND ((1.0 - (p.embedding <=> ?)) * %.2f + (CASE WHEN LOWER(p.name) LIKE ? THEN 1.0 WHEN LOWER(p.description) LIKE ? THEN 0.5 ELSE 0.0 END) * %.2f) >= %.2f", vw, tw, threshold)
		params = append(params, vec, kwPattern, kwPattern)

		baseSQL += " ORDER BY similarity DESC LIMIT ?"
		params = append(params, limit)

		if err := t.db.WithContext(ctx).Raw(baseSQL, params...).Scan(&rows).Error; err != nil {
			return nil, fmt.Errorf("hybrid vector product search failed: %w", err)
		}

		// Fallback check if zero results found
		if len(rows) == 0 {
			fallbackSQL := `
				SELECT 
					p.id, p.name, p.sku, p.description, p.price, p.currency, p.stock_quantity, p.image_url,
					c.name as category_name, sc.name as sub_cat_name, p.rating,
					(1.0 - (p.embedding <=> ?)) as similarity
				FROM products p
				LEFT JOIN categories c ON p.category_id = c.id
				LEFT JOIN sub_categories sc ON p.sub_category_id = sc.id
				WHERE p.is_active = true AND p.embedding IS NOT NULL
				ORDER BY p.embedding <=> ? LIMIT ?
			`
			_ = t.db.WithContext(ctx).Raw(fallbackSQL, vec, vec, limit).Scan(&rows).Error
		}
	}

	products := make([]map[string]interface{}, 0, len(rows))
	for _, r := range rows {
		products = append(products, map[string]interface{}{
			"id":             r.ID,
			"name":           r.Name,
			"sku":            r.SKU,
			"description":    r.Description,
			"price":          r.Price,
			"currency":       r.Currency,
			"stock_quantity": r.StockQuantity,
			"image_url":      r.ImageURL,
			"category":       r.CategoryName,
			"sub_category":   r.SubCatName,
			"rating":         r.Rating,
			"similarity":     math.Round(r.Similarity*100) / 100,
		})
	}

	return map[string]interface{}{
		"query":          query,
		"found_count":    len(products),
		"products":       products,
		"search_success": true,
	}, nil
}

// GetProductDetailTool retrieves full specification of a product
type GetProductDetailTool struct {
	db *gorm.DB
}

func NewGetProductDetailTool(db *gorm.DB) *GetProductDetailTool {
	return &GetProductDetailTool{db: db}
}

func (t *GetProductDetailTool) Name() string {
	return "get_product_detail"
}

func (t *GetProductDetailTool) Description() string {
	return "Retrieve complete product details, specifications, live stock, pricing, and category by SKU or product ID."
}

func (t *GetProductDetailTool) ParametersSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"sku": map[string]interface{}{
				"type":        "string",
				"description": "Product SKU code (e.g. 'EN-AUD-001', 'ID-CMP-001').",
			},
			"product_id": map[string]interface{}{
				"type":        "integer",
				"description": "Product database primary key ID.",
			},
		},
	}
}

func (t *GetProductDetailTool) Execute(ctx context.Context, args map[string]interface{}, contextMap map[string]interface{}) (map[string]interface{}, error) {
	sku, _ := args["sku"].(string)
	productID := int64(0)
	if id, ok := args["product_id"].(float64); ok {
		productID = int64(id)
	} else if id, ok := args["product_id"].(int); ok {
		productID = int64(id)
	}

	log.Printf("🔍 [CUSTOMER TOOL: get_product_detail] sku='%s' | product_id=%d", sku, productID)

	if sku == "" && productID == 0 {
		return map[string]interface{}{
			"found":   false,
			"message": "Either 'sku' or 'product_id' must be provided.",
		}, nil
	}

	var p CatalogProductDetail

	query := `
		SELECT 
			p.id, p.name, p.slug, p.sku, p.description, p.price, p.currency, 
			p.stock_quantity, p.low_stock_threshold, p.image_url, p.is_active, p.badge, p.rating,
			c.name as category_name, sc.name as sub_cat_name
		FROM products p
		LEFT JOIN categories c ON p.category_id = c.id
		LEFT JOIN sub_categories sc ON p.sub_category_id = sc.id
		WHERE p.is_active = true AND `

	var err error
	if sku != "" {
		err = t.db.WithContext(ctx).Raw(query+"UPPER(p.sku) = ?", strings.ToUpper(strings.TrimSpace(sku))).Scan(&p).Error
	} else {
		err = t.db.WithContext(ctx).Raw(query+"p.id = ?", productID).Scan(&p).Error
	}

	if err != nil || p.ID == 0 {
		return map[string]interface{}{
			"found":   false,
			"message": fmt.Sprintf("Product not found with SKU '%s' or ID %d", sku, productID),
		}, nil
	}

	return map[string]interface{}{
		"found": true,
		"product": map[string]interface{}{
			"id":                  p.ID,
			"name":                p.Name,
			"slug":                p.Slug,
			"sku":                 p.SKU,
			"description":         p.Description,
			"price":               p.Price,
			"currency":            p.Currency,
			"stock_quantity":      p.StockQuantity,
			"in_stock":            p.StockQuantity > 0,
			"low_stock_threshold": p.LowStockThreshold,
			"image_url":           p.ImageURL,
			"badge":               p.Badge,
			"rating":              p.Rating,
			"category":            p.CategoryName,
			"sub_category":        p.SubCatName,
		},
	}, nil
}

// CheckProductStockTool checks live inventory quantity for SKU
type CheckProductStockTool struct {
	db *gorm.DB
}

func NewCheckProductStockTool(db *gorm.DB) *CheckProductStockTool {
	return &CheckProductStockTool{db: db}
}

func (t *CheckProductStockTool) Name() string {
	return "check_product_stock"
}

func (t *CheckProductStockTool) Description() string {
	return "Check the current real-time stock availability of a product by SKU or product ID."
}

func (t *CheckProductStockTool) ParametersSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"sku": map[string]interface{}{
				"type":        "string",
				"description": "Product SKU code.",
			},
			"product_id": map[string]interface{}{
				"type":        "integer",
				"description": "Product database primary key ID.",
			},
		},
	}
}

func (t *CheckProductStockTool) Execute(ctx context.Context, args map[string]interface{}, contextMap map[string]interface{}) (map[string]interface{}, error) {
	sku, _ := args["sku"].(string)
	productID := int64(0)
	if id, ok := args["product_id"].(float64); ok {
		productID = int64(id)
	}

	var row struct {
		ID            int64  `json:"id"`
		Name          string `json:"name"`
		SKU           string `json:"sku"`
		StockQuantity int    `json:"stock_quantity"`
		IsActive      bool   `json:"is_active"`
	}

	var err error
	if sku != "" {
		err = t.db.WithContext(ctx).Raw("SELECT id, name, sku, stock_quantity, is_active FROM products WHERE UPPER(sku) = ?", strings.ToUpper(strings.TrimSpace(sku))).Scan(&row).Error
	} else {
		err = t.db.WithContext(ctx).Raw("SELECT id, name, sku, stock_quantity, is_active FROM products WHERE id = ?", productID).Scan(&row).Error
	}

	if err != nil || row.ID == 0 {
		return map[string]interface{}{
			"found":   false,
			"message": "Product not found.",
		}, nil
	}

	return map[string]interface{}{
		"found":          true,
		"product_id":     row.ID,
		"name":           row.Name,
		"sku":            row.SKU,
		"stock_quantity": row.StockQuantity,
		"in_stock":       row.StockQuantity > 0 && row.IsActive,
	}, nil
}
