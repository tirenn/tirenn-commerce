package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

const defaultBaseURL = "http://localhost:8080"

type QAClient struct {
	BaseURL    string
	HTTPClient *http.Client
	AdminToken string
	UserToken  string
}

func NewQAClient(baseURL string) *QAClient {
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	return &QAClient{
		BaseURL: baseURL,
		HTTPClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (c *QAClient) Do(method, path string, body interface{}, token string) (int, map[string]interface{}, error) {
	var bodyReader io.Reader
	if body != nil {
		jsonBytes, err := json.Marshal(body)
		if err != nil {
			return 0, nil, err
		}
		bodyReader = bytes.NewReader(jsonBytes)
	}

	url := c.BaseURL + path
	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		return 0, nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp.StatusCode, nil, err
	}

	var jsonResp map[string]interface{}
	if len(respBody) > 0 {
		_ = json.Unmarshal(respBody, &jsonResp)
	}

	return resp.StatusCode, jsonResp, nil
}

// -----------------------------------------------------------------------------
// 1. Healthcheck Test
// -----------------------------------------------------------------------------
func Test01_Healthcheck(t *testing.T) {
	client := NewQAClient("")
	code, resp, err := client.Do("GET", "/healthz", nil, "")
	if err != nil {
		t.Fatalf("Healthcheck request failed: %v", err)
	}

	if code != http.StatusOK {
		t.Errorf("Expected 200 OK, got %d", code)
	}

	if success, ok := resp["success"].(bool); !ok || !success {
		t.Errorf("Expected success=true, got %v", resp)
	}
	t.Logf("✅ Healthcheck passed: %v", resp["message"])
}

// -----------------------------------------------------------------------------
// 2. Auth & RBAC Test Suite
// -----------------------------------------------------------------------------
func Test02_AuthAndRBAC(t *testing.T) {
	client := NewQAClient("")

	// A. Login with Default Admin
	code, resp, err := client.Do("POST", "/api/v1/auth/login", map[string]string{
		"email":    "admin@gocommerce.com",
		"password": "Admin@123",
	}, "")
	if err != nil || code != http.StatusOK {
		t.Fatalf("Admin login failed: code=%d err=%v resp=%v", code, err, resp)
	}
	data := resp["data"].(map[string]interface{})
	adminToken := data["token"].(string)
	client.AdminToken = adminToken
	t.Log("✅ Admin login successful")

	// B. Register a new unique shopper
	uniqueEmail := fmt.Sprintf("qa_hero_%d@comic.com", time.Now().UnixNano())
	code, resp, err = client.Do("POST", "/api/v1/auth/register", map[string]string{
		"name":     "QA Automation Hero",
		"email":    uniqueEmail,
		"password": "SecretPassword@123",
		"phone":    "+1-555-QA-HERO",
		"address":  "123 Quality Street, Silicon Valley",
	}, "")
	if err != nil || code != http.StatusCreated {
		t.Fatalf("Customer registration failed: code=%d err=%v resp=%v", code, err, resp)
	}
	t.Log("✅ Customer registration successful")

	// C. Login with new customer
	code, resp, err = client.Do("POST", "/api/v1/auth/login", map[string]string{
		"email":    uniqueEmail,
		"password": "SecretPassword@123",
	}, "")
	if err != nil || code != http.StatusOK {
		t.Fatalf("Customer login failed: code=%d err=%v", code, err)
	}
	custData := resp["data"].(map[string]interface{})
	userToken := custData["token"].(string)
	client.UserToken = userToken
	t.Log("✅ Customer login token acquired")

	// D. Test Profile endpoint /api/v1/auth/me
	code, resp, err = client.Do("GET", "/api/v1/auth/me", nil, userToken)
	if code != http.StatusOK {
		t.Errorf("Profile check failed: code=%d", code)
	}
	profile := resp["data"].(map[string]interface{})
	if profile["email"] != uniqueEmail {
		t.Errorf("Expected email %s, got %s", uniqueEmail, profile["email"])
	}
	t.Log("✅ Profile retrieval verified")

	// E. RBAC Test: Customer attempting to access Admin Dashboard (Must be Forbidden 403)
	code, _, _ = client.Do("GET", "/api/v1/admin/dashboard", nil, userToken)
	if code != http.StatusForbidden {
		t.Errorf("Security flaw: Customer was not forbidden from admin dashboard! Got status %d", code)
	} else {
		t.Log("🛡️ RBAC Enforcement passed: Customer blocked from /admin/dashboard with 403")
	}

	// F. Admin accessing Admin Dashboard (Must be 200 OK)
	code, _, _ = client.Do("GET", "/api/v1/admin/dashboard", nil, adminToken)
	if code != http.StatusOK {
		t.Errorf("Admin failed to access dashboard! Got status %d", code)
	} else {
		t.Log("✅ Admin successfully accessed /admin/dashboard")
	}
}

// -----------------------------------------------------------------------------
// 3. Catalog Discovery, Search & Filtering Tests
// -----------------------------------------------------------------------------
func Test03_StorefrontCatalog(t *testing.T) {
	client := NewQAClient("")

	// A. List products
	code, resp, err := client.Do("GET", "/api/v1/products", nil, "")
	if err != nil || code != http.StatusOK {
		t.Fatalf("Product listing failed: code=%d err=%v", code, err)
	}
	items := resp["data"].([]interface{})
	if len(items) == 0 {
		t.Error("Expected seeded products in catalog, got 0")
	}
	t.Logf("✅ Catalog listing returned %d products", len(items))

	// B. Search keyword
	code, resp, _ = client.Do("GET", "/api/v1/products?search=Sentinel", nil, "")
	if code == http.StatusOK {
		searchResults := resp["data"].([]interface{})
		t.Logf("✅ Keyword search 'Sentinel' returned %d results", len(searchResults))
	}

	// C. Sort Price Ascending
	code, resp, _ = client.Do("GET", "/api/v1/products?sort=price_asc", nil, "")
	if code == http.StatusOK {
		sortedItems := resp["data"].([]interface{})
		if len(sortedItems) >= 2 {
			first := sortedItems[0].(map[string]interface{})["price"].(float64)
			second := sortedItems[1].(map[string]interface{})["price"].(float64)
			if first > second {
				t.Errorf("Price sort failed: %f > %f", first, second)
			} else {
				t.Logf("✅ Price ascending sorting verified ($%.2f <= $%.2f)", first, second)
			}
		}
	}

	// D. Fetch Categories
	code, resp, _ = client.Do("GET", "/api/v1/categories", nil, "")
	if code != http.StatusOK {
		t.Errorf("Failed to list categories, code=%d", code)
	} else {
		cats := resp["data"].([]interface{})
		t.Logf("✅ Retrieved %d categories", len(cats))
	}
}

// -----------------------------------------------------------------------------
// 4. Admin Product CRUD & Stock Adjustment Audit
// -----------------------------------------------------------------------------
func Test04_AdminProductAndStock(t *testing.T) {
	client := NewQAClient("")

	// Login admin
	_, loginResp, _ := client.Do("POST", "/api/v1/auth/login", map[string]string{
		"email":    "admin@gocommerce.com",
		"password": "Admin@123",
	}, "")
	adminToken := loginResp["data"].(map[string]interface{})["token"].(string)

	// A. Create new product
	sku := fmt.Sprintf("QA-PROD-%d", time.Now().UnixNano()%10000)
	createReq := map[string]interface{}{
		"category_id":         1,
		"name":                "QA Test Comic Artifact",
		"sku":                 sku,
		"description":         "High precision test commodity",
		"price":               29.99,
		"stock_quantity":      20,
		"low_stock_threshold": 5,
		"image_url":           "https://images.unsplash.com/photo-1612036782180-6f0b6cd846fe?w=400",
		"badge":               "TEST!",
	}

	code, resp, err := client.Do("POST", "/api/v1/admin/products", createReq, adminToken)
	if err != nil || code != http.StatusCreated {
		t.Fatalf("Product creation failed: code=%d err=%v resp=%v", code, err, resp)
	}
	createdProd := resp["data"].(map[string]interface{})
	prodID := uint(createdProd["id"].(float64))
	t.Logf("✅ Created Product ID %d (SKU: %s)", prodID, sku)

	// B. Adjust Stock (ADD 15)
	code, resp, _ = client.Do("POST", fmt.Sprintf("/api/v1/admin/products/%d/adjust-stock", prodID), map[string]interface{}{
		"type":   "ADD",
		"amount": 15,
		"reason": "QA Restock Shipment",
	}, adminToken)
	if code != http.StatusOK {
		t.Errorf("Stock adjustment failed: code=%d", code)
	}
	updatedProd := resp["data"].(map[string]interface{})
	if updatedProd["stock_quantity"].(float64) != 35 {
		t.Errorf("Expected stock 35, got %v", updatedProd["stock_quantity"])
	}
	t.Log("✅ Stock adjustment ADD passed (20 + 15 = 35)")

	// C. Inspect Stock Logs
	code, resp, _ = client.Do("GET", fmt.Sprintf("/api/v1/admin/products/%d/stock-logs", prodID), nil, adminToken)
	if code == http.StatusOK {
		logs := resp["data"].([]interface{})
		if len(logs) > 0 {
			t.Logf("✅ Verified stock adjustment audit trail (Found %d logs)", len(logs))
		}
	}
}

// -----------------------------------------------------------------------------
// 5. Checkout & Concurrency Overselling Prevention
// -----------------------------------------------------------------------------
func Test05_CheckoutAndConcurrency(t *testing.T) {
	client := NewQAClient("")

	// Login admin
	_, loginResp, _ := client.Do("POST", "/api/v1/auth/login", map[string]string{
		"email":    "admin@gocommerce.com",
		"password": "Admin@123",
	}, "")
	adminToken := loginResp["data"].(map[string]interface{})["token"].(string)

	// Login shopper
	_, shopResp, _ := client.Do("POST", "/api/v1/auth/login", map[string]string{
		"email":    "shopper@gocommerce.com",
		"password": "Shopper@123",
	}, "")
	shopperToken := shopResp["data"].(map[string]interface{})["token"].(string)

	// A. Create a limited product with exactly 1 unit in stock
	limitedSKU := fmt.Sprintf("LIMITED-%d", time.Now().UnixNano()%10000)
	_, createResp, _ := client.Do("POST", "/api/v1/admin/products", map[string]interface{}{
		"category_id":         1,
		"name":                "Ultra Rare Comic (Only 1 in World)",
		"sku":                 limitedSKU,
		"description":         "Only one copy exists",
		"price":               999.00,
		"stock_quantity":      1,
		"low_stock_threshold": 1,
	}, adminToken)
	productID := uint(createResp["data"].(map[string]interface{})["id"].(float64))
	t.Logf("✅ Created limited product ID %d with Stock = 1", productID)

	// B. Launch 10 simultaneous checkout requests concurrently!
	concurrency := 10
	var successCount int32
	var failCount int32
	var wg sync.WaitGroup

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(threadID int) {
			defer wg.Done()
			code, _, _ := client.Do("POST", "/api/v1/orders/checkout", map[string]interface{}{
				"items": []map[string]interface{}{
					{"product_id": productID, "quantity": 1},
				},
				"shipping_name":    fmt.Sprintf("Shopper #%d", threadID),
				"shipping_phone":   "+1-555-CONCURRENCY",
				"shipping_address": "123 Speed Lane",
				"payment_method":   "SIMULATED_CARD",
			}, shopperToken)

			if code == http.StatusCreated {
				atomic.AddInt32(&successCount, 1)
			} else {
				atomic.AddInt32(&failCount, 1)
			}
		}(i)
	}

	wg.Wait()

	t.Logf("⚡ Concurrency Test Results: Successes=%d, Rejections=%d (out of %d)", successCount, failCount, concurrency)

	if successCount != 1 {
		t.Errorf("CRITICAL RACE CONDITION: Exactly 1 checkout should succeed, but %d succeeded!", successCount)
	} else {
		t.Log("🔒 Concurrency Race Condition Test PASSED: No overselling occurred! Atomic row locks working flawlessly.")
	}

	// Verify remaining stock is now 0
	_, prodResp, _ := client.Do("GET", fmt.Sprintf("/api/v1/products/%d", productID), nil, "")
	remStock := prodResp["data"].(map[string]interface{})["stock_quantity"].(float64)
	if remStock != 0 {
		t.Errorf("Expected remaining stock to be 0, got %v", remStock)
	} else {
		t.Log("✅ Final stock verified = 0 (no negative inventory)")
	}
}

// -----------------------------------------------------------------------------
// 6. Order Lifecycle & Status Transitions
// -----------------------------------------------------------------------------
func Test06_OrderLifecycleAndRestock(t *testing.T) {
	client := NewQAClient("")

	_, loginResp, _ := client.Do("POST", "/api/v1/auth/login", map[string]string{
		"email":    "admin@gocommerce.com",
		"password": "Admin@123",
	}, "")
	adminToken := loginResp["data"].(map[string]interface{})["token"].(string)

	_, shopResp, _ := client.Do("POST", "/api/v1/auth/login", map[string]string{
		"email":    "shopper@gocommerce.com",
		"password": "Shopper@123",
	}, "")
	shopperToken := shopResp["data"].(map[string]interface{})["token"].(string)

	// Fetch a product
	_, prodList, _ := client.Do("GET", "/api/v1/products?limit=1", nil, "")
	prod := prodList["data"].([]interface{})[0].(map[string]interface{})
	pID := uint(prod["id"].(float64))
	prevStock := prod["stock_quantity"].(float64)

	// Place order
	code, orderResp, _ := client.Do("POST", "/api/v1/orders/checkout", map[string]interface{}{
		"items": []map[string]interface{}{
			{"product_id": pID, "quantity": 1},
		},
		"shipping_name":    "Test Cancel Restock",
		"shipping_phone":   "+1-555-000-0000",
		"shipping_address": "456 Test Blvd",
	}, shopperToken)

	if code != http.StatusCreated {
		t.Fatalf("Failed to create order for lifecycle test, code=%d", code)
	}

	orderID := uint(orderResp["data"].(map[string]interface{})["id"].(float64))

	// Status transition: Processing -> Shipped
	code, _, _ = client.Do("PATCH", fmt.Sprintf("/api/v1/admin/orders/%d/status", orderID), map[string]string{
		"status": "SHIPPED",
		"notes":  "Dispatched via Gotham Express",
	}, adminToken)
	if code != http.StatusOK {
		t.Errorf("Failed to update status to SHIPPED, code=%d", code)
	} else {
		t.Log("✅ Order status transitioned to SHIPPED")
	}

	// Status transition: Cancel & Restock
	code, _, _ = client.Do("PATCH", fmt.Sprintf("/api/v1/admin/orders/%d/status", orderID), map[string]string{
		"status": "CANCELLED",
		"notes":  "Customer requested cancellation",
	}, adminToken)
	if code != http.StatusOK {
		t.Errorf("Failed to cancel order, code=%d", code)
	} else {
		t.Log("✅ Order successfully CANCELLED")
	}

	// Verify product stock was automatically restored back
	_, prodAfterCancel, _ := client.Do("GET", fmt.Sprintf("/api/v1/products/%d", pID), nil, "")
	restoredStock := prodAfterCancel["data"].(map[string]interface{})["stock_quantity"].(float64)
	if restoredStock != prevStock {
		t.Errorf("Stock was not restored! Expected %f, got %f", prevStock, restoredStock)
	} else {
		t.Log("🔄 Restock Verification PASSED: Product stock automatically restored on order cancellation!")
	}
}
