package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/tirenn/commerce/backend/internal/domain/product"
	"github.com/tirenn/commerce/backend/internal/logger"
)

type Client interface {
	SearchSemantic(ctx context.Context, query string, categoryID uint, limit int) ([]uint, error)
	SyncProducts(ctx context.Context, products []product.Product) error
	GetRecommendations(ctx context.Context, productID uint, limit int) ([]uint, error)
}

type client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

func NewClient(baseURL string, apiKey string) Client {
	return &client{
		baseURL: baseURL,
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

type semanticSearchReq struct {
	Query      string `json:"query"`
	Limit      int    `json:"limit"`
	CategoryID uint   `json:"category_id"`
}

type scoredItem struct {
	ID    uint    `json:"id"`
	Score float64 `json:"score"`
}

type semanticSearchResp struct {
	Success bool         `json:"success"`
	Data    []scoredItem `json:"data"`
}

func (c *client) SearchSemantic(ctx context.Context, query string, categoryID uint, limit int) ([]uint, error) {
	if c.baseURL == "" {
		return nil, fmt.Errorf("AI service URL not configured")
	}

	bodyData, err := json.Marshal(semanticSearchReq{
		Query:      query,
		Limit:      limit,
		CategoryID: categoryID,
	})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/v1/search/semantic", bytes.NewBuffer(bodyData))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("X-API-Key", c.apiKey)
	}
	if reqID := logger.GetRequestID(ctx); reqID != "" {
		req.Header.Set("X-Request-ID", reqID)
	}
	if traceID := logger.GetTraceID(ctx); traceID != "" {
		req.Header.Set("X-Trace-ID", traceID)
	}
	if spanID := logger.GetSpanID(ctx); spanID != "" {
		req.Header.Set("X-Span-ID", spanID)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("AI service returned status code %d", resp.StatusCode)
	}

	var parsed semanticSearchResp
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, err
	}

	ids := make([]uint, 0, len(parsed.Data))
	for _, item := range parsed.Data {
		ids = append(ids, item.ID)
	}
	return ids, nil
}

type productIndexPayload struct {
	ID           uint    `json:"id"`
	Name         string  `json:"name"`
	CategoryID   uint    `json:"category_id"`
	CategoryName string  `json:"category_name"`
	SKU          string  `json:"sku"`
	Description  string  `json:"description"`
	Price        float64 `json:"price"`
	ImageURL     string  `json:"image_url"`
	Badge        string  `json:"badge"`
	Rating       float64 `json:"rating"`
	Stock        int     `json:"stock_quantity"`
}

func (c *client) SyncProducts(ctx context.Context, products []product.Product) error {
	if c.baseURL == "" || len(products) == 0 {
		return nil
	}

	items := make([]productIndexPayload, 0, len(products))
	for _, p := range products {
		catName := ""
		if p.Category.Name != "" {
			catName = p.Category.Name
		}
		items = append(items, productIndexPayload{
			ID:           p.ID,
			Name:         p.Name,
			CategoryID:   p.CategoryID,
			CategoryName: catName,
			SKU:          p.SKU,
			Description:  p.Description,
			Price:        p.Price,
			ImageURL:     p.ImageURL,
			Badge:        p.Badge,
			Rating:       p.Rating,
			Stock:        p.StockQuantity,
		})
	}

	bodyData, err := json.Marshal(map[string]any{
		"products": items,
	})
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/v1/index-products", bytes.NewBuffer(bodyData))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("X-API-Key", c.apiKey)
	}
	if reqID := logger.GetRequestID(ctx); reqID != "" {
		req.Header.Set("X-Request-ID", reqID)
	}
	if traceID := logger.GetTraceID(ctx); traceID != "" {
		req.Header.Set("X-Trace-ID", traceID)
	}
	if spanID := logger.GetSpanID(ctx); spanID != "" {
		req.Header.Set("X-Span-ID", spanID)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

type recommendationItem struct {
	ID uint `json:"id"`
}

type recommendationResp struct {
	Success bool                 `json:"success"`
	Data    []recommendationItem `json:"data"`
}

func (c *client) GetRecommendations(ctx context.Context, productID uint, limit int) ([]uint, error) {
	if c.baseURL == "" {
		return nil, fmt.Errorf("AI service URL not configured")
	}

	url := fmt.Sprintf("%s/api/v1/recommendations/%d?limit=%d", c.baseURL, productID, limit)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	if c.apiKey != "" {
		req.Header.Set("X-API-Key", c.apiKey)
	}
	if reqID := logger.GetRequestID(ctx); reqID != "" {
		req.Header.Set("X-Request-ID", reqID)
	}
	if traceID := logger.GetTraceID(ctx); traceID != "" {
		req.Header.Set("X-Trace-ID", traceID)
	}
	if spanID := logger.GetSpanID(ctx); spanID != "" {
		req.Header.Set("X-Span-ID", spanID)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("AI service recommendations returned status code %d", resp.StatusCode)
	}

	var parsed recommendationResp
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, err
	}

	ids := make([]uint, 0, len(parsed.Data))
	for _, item := range parsed.Data {
		ids = append(ids, item.ID)
	}
	return ids, nil
}
