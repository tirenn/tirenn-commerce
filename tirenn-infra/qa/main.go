package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type Runner struct {
	BaseURL     string
	HTTPClient  *http.Client
	AdminToken  string
	ShopperToken string
	PassedTests int
	FailedTests int
}

func NewRunner(baseURL string) *Runner {
	return &Runner{
		BaseURL: baseURL,
		HTTPClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (r *Runner) Request(method, path string, body interface{}, token string) (int, map[string]interface{}, error) {
	var bodyReader io.Reader
	if body != nil {
		jsonBytes, err := json.Marshal(body)
		if err != nil {
			return 0, nil, err
		}
		bodyReader = bytes.NewReader(jsonBytes)
	}

	req, err := http.NewRequest(method, r.BaseURL+path, bodyReader)
	if err != nil {
		return 0, nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := r.HTTPClient.Do(req)
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

func (r *Runner) Assert(testName string, condition bool, details string) {
	if condition {
		r.PassedTests++
		fmt.Printf("  ✅ [PASS] %s\n", testName)
	} else {
		r.FailedTests++
		fmt.Printf("  ❌ [FAIL] %s - %s\n", testName, details)
	}
}

func main() {
	baseURL := "http://localhost:8080"
	if len(os.Args) > 1 {
		baseURL = os.Args[1]
	}

	fmt.Println(strings.Repeat("=", 75))
	fmt.Println("🚀 GOCOMMERCE END-TO-END QA AUTOMATION TEST SUITE")
	fmt.Printf("🎯 Target API: %s\n", baseURL)
	fmt.Printf("⏰ Time: %s\n", time.Now().Format("2006-01-02 15:04:05"))
	fmt.Println(strings.Repeat("=", 75))

	runner := NewRunner(baseURL)

	// Wait up to 30s for server readiness
	fmt.Println("\n⏳ Waiting for GoCommerce API to be online...")
	ready := false
	for i := 0; i < 30; i++ {
		code, _, err := runner.Request("GET", "/healthz", nil, "")
		if err == nil && code == http.StatusOK {
			ready = true
			fmt.Println("⚡ GoCommerce API is ONLINE and HEALTHY!")
			break
		}
		time.Sleep(1 * time.Second)
	}

	if !ready {
		fmt.Println("❌ Error: GoCommerce server is not reachable at", baseURL)
		os.Exit(1)
	}

	// 1. Healthcheck Test
	fmt.Println("\n--- [SUITE 1: System Health & Status] ---")
	code, resp, _ := runner.Request("GET", "/healthz", nil, "")
	runner.Assert("Health Endpoint Returns 200", code == http.StatusOK, fmt.Sprintf("Got status %d", code))
	runner.Assert("Health API Response Success Flag", resp["success"] == true, "success != true")

	// 2. Auth & RBAC Test
	fmt.Println("\n--- [SUITE 2: Authentication & RBAC Security] ---")
	// Admin Login
	code, resp, _ = runner.Request("POST", "/api/v1/auth/login", map[string]string{
		"email":    "admin@gocommerce.com",
		"password": "Admin@123",
	}, "")
	runner.Assert("Admin Login (admin@gocommerce.com)", code == http.StatusOK, "Admin login failed")
	if data, ok := resp["data"].(map[string]interface{}); ok {
		runner.AdminToken = data["token"].(string)
	}

	// Customer Registration & Login
	uniqueEmail := fmt.Sprintf("qa_tester_%d@comic.com", time.Now().UnixNano()%100000)
	code, _, _ = runner.Request("POST", "/api/v1/auth/register", map[string]string{
		"name":     "QA Automation Tester",
		"email":    uniqueEmail,
		"password": "Password@123",
		"phone":    "+1-555-TEST",
		"address":  "123 QA Boulevard, Gotham",
	}, "")
	runner.Assert("Customer Registration", code == http.StatusCreated, "Registration failed")

	code, resp, _ = runner.Request("POST", "/api/v1/auth/login", map[string]string{
		"email":    uniqueEmail,
		"password": "Password@123",
	}, "")
	runner.Assert("Customer Login", code == http.StatusOK, "Login failed")
	if data, ok := resp["data"].(map[string]interface{}); ok {
		runner.ShopperToken = data["token"].(string)
	}

	// Profile check
	code, resp, _ = runner.Request("GET", "/api/v1/auth/me", nil, runner.ShopperToken)
	runner.Assert("Authenticated Profile (/auth/me)", code == http.StatusOK, "Profile fetch failed")

	// RBAC Privilege Escalation Check
	code, _, _ = runner.Request("GET", "/api/v1/admin/dashboard", nil, runner.ShopperToken)
	runner.Assert("RBAC: Customer blocked from /admin/dashboard (403)", code == http.StatusForbidden, fmt.Sprintf("Got %d instead of 403", code))

	code, _, _ = runner.Request("GET", "/api/v1/admin/dashboard", nil, runner.AdminToken)
	runner.Assert("RBAC: Admin granted access to /admin/dashboard (200)", code == http.StatusOK, fmt.Sprintf("Got %d", code))

	// 3. Storefront Catalog & Search
	fmt.Println("\n--- [SUITE 3: Catalog Discovery, Search & Filters] ---")
	code, resp, _ = runner.Request("GET", "/api/v1/products?limit=10", nil, "")
	runner.Assert("Fetch Product Catalog", code == http.StatusOK, "Products listing failed")
	var sampleProductID uint = 1
	if items, ok := resp["data"].([]interface{}); ok && len(items) > 0 {
		first := items[0].(map[string]interface{})
		sampleProductID = uint(first["id"].(float64))
	}

	code, resp, _ = runner.Request("GET", "/api/v1/categories", nil, "")
	runner.Assert("Fetch Categories List", code == http.StatusOK, "Category listing failed")

	code, resp, _ = runner.Request("GET", fmt.Sprintf("/api/v1/products/%d", sampleProductID), nil, "")
	runner.Assert("Fetch Single Product Details", code == http.StatusOK, "PDP fetch failed")

	// 4. Admin Catalog & Stock Management
	fmt.Println("\n--- [SUITE 4: Admin Product CRUD & Stock Adjuster] ---")
	sku := fmt.Sprintf("QA-AUTO-%d", time.Now().UnixNano()%100000)
	code, resp, _ = runner.Request("POST", "/api/v1/admin/products", map[string]interface{}{
		"category_id":         1,
		"name":                "QA Automatic Comic Issue",
		"sku":                 sku,
		"description":         "Created via automated QA test pipeline",
		"price":               19.99,
		"stock_quantity":      10,
		"low_stock_threshold": 3,
		"badge":               "QA!",
	}, runner.AdminToken)
	runner.Assert("Admin Create Product", code == http.StatusCreated, "Product creation failed")

	var qaProdID uint
	if data, ok := resp["data"].(map[string]interface{}); ok {
		qaProdID = uint(data["id"].(float64))
	}

	// Adjust stock ADD 20
	code, resp, _ = runner.Request("POST", fmt.Sprintf("/api/v1/admin/products/%d/adjust-stock", qaProdID), map[string]interface{}{
		"type":   "ADD",
		"amount": 20,
		"reason": "QA Automated Restock",
	}, runner.AdminToken)
	runner.Assert("Admin Adjust Stock ADD 20", code == http.StatusOK && resp["data"].(map[string]interface{})["stock_quantity"].(float64) == 30, "Stock was not updated to 30")

	// Check Stock Logs
	code, resp, _ = runner.Request("GET", fmt.Sprintf("/api/v1/admin/products/%d/stock-logs", qaProdID), nil, runner.AdminToken)
	runner.Assert("Admin View Stock Audit Logs", code == http.StatusOK, "Stock log fetch failed")

	// 5. Checkout & Concurrency Overselling Test
	fmt.Println("\n--- [SUITE 5: Atomic Checkout & Concurrency Overselling Test] ---")
	// Create product with stock = 1
	limitSKU := fmt.Sprintf("RARE-%d", time.Now().UnixNano()%100000)
	_, createResp, _ := runner.Request("POST", "/api/v1/admin/products", map[string]interface{}{
		"category_id":         1,
		"name":                "Ultra Rare One-Of-A-Kind Comic",
		"sku":                 limitSKU,
		"price":               500.00,
		"stock_quantity":      1,
		"low_stock_threshold": 1,
	}, runner.AdminToken)
	rareID := uint(createResp["data"].(map[string]interface{})["id"].(float64))

	concurrency := 8
	var successCount int32
	var failCount int32
	var wg sync.WaitGroup

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			c, _, _ := runner.Request("POST", "/api/v1/orders/checkout", map[string]interface{}{
				"items": []map[string]interface{}{
					{"product_id": rareID, "quantity": 1},
				},
				"shipping_name":    fmt.Sprintf("Concurrent User %d", id),
				"shipping_phone":   "+1-555-RACE",
				"shipping_address": "404 Race Track",
			}, runner.ShopperToken)

			if c == http.StatusCreated {
				atomic.AddInt32(&successCount, 1)
			} else {
				atomic.AddInt32(&failCount, 1)
			}
		}(i)
	}
	wg.Wait()

	runner.Assert("Atomic Concurrency: Exactly 1 checkout succeeded", successCount == 1, fmt.Sprintf("%d succeeded out of %d", successCount, concurrency))
	runner.Assert("Atomic Concurrency: Remaining 7 checkouts rejected", failCount == int32(concurrency-1), fmt.Sprintf("%d failed", failCount))

	// Verify remaining stock is 0
	_, prodCheck, _ := runner.Request("GET", fmt.Sprintf("/api/v1/products/%d", rareID), nil, "")
	runner.Assert("Atomic Concurrency: Final Stock == 0 (No Negative Oversell)", prodCheck["data"].(map[string]interface{})["stock_quantity"].(float64) == 0, "Stock was not 0")

	// 6. Order Lifecycle & Restock
	fmt.Println("\n--- [SUITE 6: Order Lifecycle & Restock on Cancel] ---")
	// Place an order to cancel
	_, orderPlaced, _ := runner.Request("POST", "/api/v1/orders/checkout", map[string]interface{}{
		"items": []map[string]interface{}{
			{"product_id": qaProdID, "quantity": 2},
		},
		"shipping_name":    "Cancel Test Hero",
		"shipping_phone":   "+1-555-CANCEL",
		"shipping_address": "789 Gotham Alley",
	}, runner.ShopperToken)
	cancelOrderID := uint(orderPlaced["data"].(map[string]interface{})["id"].(float64))

	// Admin updates status: SHIPPED
	code, _, _ = runner.Request("PATCH", fmt.Sprintf("/api/v1/admin/orders/%d/status", cancelOrderID), map[string]string{
		"status": "SHIPPED",
	}, runner.AdminToken)
	runner.Assert("Admin Transition Status to SHIPPED", code == http.StatusOK, "Status update to SHIPPED failed")

	// Admin cancels order (restocks)
	code, _, _ = runner.Request("PATCH", fmt.Sprintf("/api/v1/admin/orders/%d/status", cancelOrderID), map[string]string{
		"status": "CANCELLED",
	}, runner.AdminToken)
	runner.Assert("Admin Cancel Order (Trigger Restock)", code == http.StatusOK, "Status update to CANCELLED failed")

	// 7. Customer CRM & Dashboard
	fmt.Println("\n--- [SUITE 7: Customer CRM & Analytics Dashboard] ---")
	code, resp, _ = runner.Request("GET", "/api/v1/admin/customers", nil, runner.AdminToken)
	runner.Assert("Admin List Customers with Spend Metrics", code == http.StatusOK, "CRM list failed")

	code, resp, _ = runner.Request("GET", "/api/v1/admin/dashboard", nil, runner.AdminToken)
	runner.Assert("Admin Fetch Full Dashboard Analytics", code == http.StatusOK && resp["data"] != nil, "Dashboard fetch failed")

	// Final Summary Report
	fmt.Println("\n" + strings.Repeat("=", 75))
	fmt.Println("📊 QA AUTOMATION SUMMARY REPORT")
	fmt.Printf("  Total Assertions Checked: %d\n", runner.PassedTests+runner.FailedTests)
	fmt.Printf("  ✅ Passed: %d\n", runner.PassedTests)
	fmt.Printf("  ❌ Failed: %d\n", runner.FailedTests)
	if runner.FailedTests == 0 {
		fmt.Println("🎉 ALL QA FUNCTIONAL & CONCURRENCY TESTS PASSED WITH 100% SUCCESS RATE!")
	} else {
		fmt.Printf("⚠️ %d test(s) failed. Check details above.\n", runner.FailedTests)
	}
	fmt.Println(strings.Repeat("=", 75))
}
