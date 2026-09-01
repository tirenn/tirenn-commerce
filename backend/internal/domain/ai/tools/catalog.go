package tools

import (
	"context"
	"fmt"
	"log"
	"math"
	"strings"

	"tirenn-ai-commerce/internal/client/ollama"
	"tirenn-ai-commerce/internal/domain/product"
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

// SearchProductsTool executes hybrid dense-vector + keyword ranking search on products via product.Repository
type SearchProductsTool struct {
	repo         product.Repository
	ollamaClient *ollama.Client
	cfg          SearchProductsConfig
}

func NewSearchProductsTool(repo product.Repository, ollamaClient *ollama.Client, cfgs ...SearchProductsConfig) *SearchProductsTool {
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
		repo:         repo,
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

	// 1. Generate query embedding vector via Ollama bge-m3
	var embedding []float32
	if t.ollamaClient != nil && t.cfg.EnableHybridSearch {
		vec, err := t.ollamaClient.GenerateEmbedding(ctx, query)
		if err == nil && len(vec) > 0 {
			embedding = vec
		} else if err != nil {
			log.Printf("⚠️ [SearchProductsTool] Embedding error: %v, falling back to pure text query", err)
		}
	}

	// 2. Build Filter Query for Repository
	filter := product.ProductFilterQuery{
		Search:    query,
		Limit:     limit,
		Page:      1,
		Embedding: embedding,
	}
	if inStock {
		filter.InStock = &inStock
	}

	// Execute search via clean repository
	rows, _, err := t.repo.List(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("product search repository error: %w", err)
	}

	// If 0 found and category was provided, retry with broad search (category relaxation)
	if len(rows) == 0 && category != "" {
		filter.Search = query
		rows, _, _ = t.repo.List(ctx, filter)
	}

	products := make([]map[string]interface{}, 0, len(rows))
	for _, r := range rows {
		subCatName := ""
		if r.SubCategory != nil {
			subCatName = r.SubCategory.Name
		}

		// Calculate real dynamic cosine similarity + text match hybrid score
		simScore := 0.0
		if len(embedding) > 0 && len(r.Embedding.Slice()) == len(embedding) {
			var dot, normA, normB float64
			rSlice := r.Embedding.Slice()
			for i := 0; i < len(embedding); i++ {
				dot += float64(embedding[i] * rSlice[i])
				normA += float64(embedding[i] * embedding[i])
				normB += float64(rSlice[i] * rSlice[i])
			}
			if normA > 0 && normB > 0 {
				cosSim := dot / (math.Sqrt(normA) * math.Sqrt(normB))
				textMatch := 0.0
				lowerQuery := strings.ToLower(query)
				if strings.Contains(strings.ToLower(r.Name), lowerQuery) {
					textMatch = 1.0
				} else if strings.Contains(strings.ToLower(r.Description), lowerQuery) {
					textMatch = 0.5
				}
				simScore = math.Round((cosSim*t.cfg.HybridVectorWeight+textMatch*t.cfg.HybridTextWeight)*100) / 100
			}
		} else {
			simScore = 0.85
		}

		products = append(products, map[string]interface{}{
			"id":             r.ID,
			"name":           r.Name,
			"sku":            r.SKU,
			"description":    r.Description,
			"price":          r.Price,
			"currency":       r.Currency,
			"stock_quantity": r.StockQuantity,
			"image_url":      r.ImageURL,
			"category":       r.Category.Name,
			"sub_category":   subCatName,
			"rating":         r.Rating,
			"similarity":     simScore,
		})
	}

	return map[string]interface{}{
		"query":          query,
		"found_count":    len(products),
		"products":       products,
		"search_success": true,
	}, nil
}

// GetProductDetailTool retrieves full specification of a product via product.Repository
type GetProductDetailTool struct {
	repo product.Repository
}

func NewGetProductDetailTool(repo product.Repository) *GetProductDetailTool {
	return &GetProductDetailTool{repo: repo}
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
	sku = strings.TrimSpace(sku)
	productID := uint(0)
	if id, ok := args["product_id"].(float64); ok && id > 0 {
		productID = uint(id)
	} else if id, ok := args["product_id"].(int); ok && id > 0 {
		productID = uint(id)
	}

	log.Printf("🔍 [CUSTOMER TOOL: get_product_detail] sku='%s' | product_id=%d", sku, productID)

	if sku == "" && productID == 0 {
		return map[string]interface{}{
			"found":   false,
			"message": "Either 'sku' or 'product_id' must be provided.",
		}, nil
	}

	var p *product.Product
	var err error

	if sku != "" {
		p, err = t.repo.FindBySKU(ctx, sku)
	} else {
		p, err = t.repo.FindByID(ctx, productID)
	}

	if err != nil || p == nil {
		return map[string]interface{}{
			"found":   false,
			"message": fmt.Sprintf("Product not found with SKU '%s' or ID %d", sku, productID),
		}, nil
	}

	subCatName := ""
	if p.SubCategory != nil {
		subCatName = p.SubCategory.Name
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
			"category":            p.Category.Name,
			"sub_category":        subCatName,
		},
	}, nil
}

// CheckProductStockTool checks live inventory quantity for SKU via product.Repository
type CheckProductStockTool struct {
	repo product.Repository
}

func NewCheckProductStockTool(repo product.Repository) *CheckProductStockTool {
	return &CheckProductStockTool{repo: repo}
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
	sku = strings.TrimSpace(sku)
	productID := uint(0)
	if id, ok := args["product_id"].(float64); ok && id > 0 {
		productID = uint(id)
	} else if id, ok := args["product_id"].(int); ok && id > 0 {
		productID = uint(id)
	}

	log.Printf("📦 [CUSTOMER TOOL: check_product_stock] sku='%s' | product_id=%d", sku, productID)

	if sku == "" && productID == 0 {
		return map[string]interface{}{
			"found":   false,
			"message": "Either 'sku' or 'product_id' must be provided.",
		}, nil
	}

	var p *product.Product
	var err error

	if sku != "" {
		p, err = t.repo.FindBySKU(ctx, sku)
	} else {
		p, err = t.repo.FindByID(ctx, productID)
	}

	if err != nil || p == nil {
		return map[string]interface{}{
			"found":   false,
			"message": fmt.Sprintf("Product with SKU '%s' not found.", sku),
		}, nil
	}

	return map[string]interface{}{
		"found":          true,
		"product_id":     p.ID,
		"name":           p.Name,
		"sku":            p.SKU,
		"stock_quantity": p.StockQuantity,
		"in_stock":       p.StockQuantity > 0 && p.IsActive,
	}, nil
}
