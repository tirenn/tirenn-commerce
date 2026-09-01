package database

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/tirenn/commerce/backend/internal/config"
	"github.com/tirenn/commerce/backend/internal/domain/auth"
	"github.com/tirenn/commerce/backend/internal/domain/customer"
	"github.com/tirenn/commerce/backend/internal/domain/dashboard"
	"github.com/tirenn/commerce/backend/internal/domain/order"
	"github.com/tirenn/commerce/backend/internal/domain/product"
	"github.com/tirenn/commerce/backend/internal/router"
	"github.com/tirenn/commerce/backend/internal/security"
	"gorm.io/gorm"
)

type SeedCategory struct {
	ID            uint              `json:"id"`
	Name          string            `json:"name"`
	Slug          string            `json:"slug"`
	Icon          string            `json:"icon"`
	Description   string            `json:"description"`
	SubCategories []SeedSubCategory `json:"sub_categories"`
}

type SeedSubCategory struct {
	ID          uint   `json:"id"`
	CategoryID  uint   `json:"category_id"`
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Icon        string `json:"icon"`
	Description string `json:"description"`
}

type SeedProduct struct {
	CategoryID        uint    `json:"category_id"`
	SubCategoryID     uint    `json:"sub_category_id"`
	Name              string  `json:"name"`
	SKU               string  `json:"sku"`
	Price             float64 `json:"price"`
	Currency          string  `json:"currency"`
	StockQuantity     int     `json:"stock_quantity"`
	LowStockThreshold int     `json:"low_stock_threshold"`
	Badge             string  `json:"badge"`
	Rating            float64 `json:"rating"`
	ImageURL          string  `json:"image_url"`
	Description       string  `json:"description"`
}

type SeedDataFile struct {
	Categories []SeedCategory `json:"categories"`
	Products   []SeedProduct  `json:"products"`
}

func loadSeedData() (*SeedDataFile, error) {
	possiblePaths := []string{
		"data/products.json",
		"backend/data/products.json",
		"../backend/data/products.json",
		"../../backend/data/products.json",
	}

	var dataBytes []byte
	var err error

	for _, p := range possiblePaths {
		absPath, _ := filepath.Abs(p)
		if dataBytes, err = os.ReadFile(absPath); err == nil {
			log.Printf("📂 Found seed dataset at: %s", absPath)
			break
		}
	}

	if len(dataBytes) == 0 {
		return nil, fmt.Errorf("failed to locate data/products.json in search paths: %w", err)
	}

	var seedFile SeedDataFile
	if err := json.Unmarshal(dataBytes, &seedFile); err != nil {
		return nil, fmt.Errorf("failed to parse products.json: %w", err)
	}

	return &seedFile, nil
}

// Seed checks if products already exist, otherwise executes ForceSeed
func Seed(db *gorm.DB) error {
	cfg := config.LoadConfig()
	var productCount int64
	db.Model(&product.Product{}).Count(&productCount)
	if productCount < 50 {
		return ForceSeed(db, cfg)
	}
	return nil
}

// ForceSeed resets the database tables and populates fresh smartphone/electronics catalog via endpoint calls
func ForceSeed(db *gorm.DB, cfg *config.Config) error {
	if cfg == nil {
		cfg = config.LoadConfig()
	}

	log.Println("🔄 Resetting database tables for fresh Smartphone & Electronics catalog seeding...")
	if err := db.Exec("TRUNCATE TABLE order_items, orders, stock_adjustment_logs, products, sub_categories, categories, users RESTART IDENTITY CASCADE;").Error; err != nil {
		return fmt.Errorf("truncate tables failed: %w", err)
	}

	// 1. Seed Users
	log.Println("👤 Seeding system users (Admin & Shoppers)...")
	adminPassword, _ := security.HashPassword("Admin@123")
	customerPassword, _ := security.HashPassword("Shopper@123")
	sarahPassword, _ := security.HashPassword("Sarah@123")

	admin := auth.User{
		Name:         "Tirenn Merchant Admin",
		Email:        "admin@gocommerce.com",
		PasswordHash: adminPassword,
		Role:         auth.RoleAdmin,
		Phone:        "+62 811-9876-5432",
		Address:      "Gedung Menara Sudirman Lt. 18, Jl. Jend. Sudirman Kav. 60, Jakarta Selatan",
		Status:       auth.StatusActive,
	}
	if err := db.Create(&admin).Error; err != nil {
		return fmt.Errorf("failed to seed admin: %w", err)
	}

	shopper := auth.User{
		Name:         "Budi Santoso",
		Email:        "shopper@gocommerce.com",
		PasswordHash: customerPassword,
		Role:         auth.RoleCustomer,
		Phone:        "+62 812-3456-7890",
		Address:      "Jl. Kebon Jeruk No. 45, RT 02 / RW 05, Jakarta Barat",
		Status:       auth.StatusActive,
	}
	if err := db.Create(&shopper).Error; err != nil {
		return fmt.Errorf("failed to seed shopper: %w", err)
	}

	sarah := auth.User{
		Name:         "Sarah Jenkins",
		Email:        "sarah@gocommerce.com",
		PasswordHash: sarahPassword,
		Role:         auth.RoleCustomer,
		Phone:        "+1 415-555-2671",
		Address:      "742 Evergreen Terrace, Springfield, OR 97477, USA",
		Status:       auth.StatusActive,
	}
	if err := db.Create(&sarah).Error; err != nil {
		return fmt.Errorf("failed to seed sarah: %w", err)
	}

	// 2. Load Seed Data from JSON
	seedData, err := loadSeedData()
	if err != nil {
		return fmt.Errorf("failed to load seed data: %w", err)
	}

	// 3. Seed Categories & Subcategories
	log.Printf("📦 Seeding %d Categories & Subcategories...", len(seedData.Categories))
	for _, c := range seedData.Categories {
		cat := product.Category{
			ID:          c.ID,
			Name:        c.Name,
			Slug:        c.Slug,
			Icon:        c.Icon,
			Description: c.Description,
		}
		if err := db.Create(&cat).Error; err != nil {
			return fmt.Errorf("failed to seed category %s: %w", c.Name, err)
		}

		for _, sc := range c.SubCategories {
			subCat := product.SubCategory{
				ID:          sc.ID,
				CategoryID:  c.ID,
				Name:        sc.Name,
				Slug:        sc.Slug,
				Icon:        sc.Icon,
				Description: sc.Description,
			}
			if err := db.Create(&subCat).Error; err != nil {
				return fmt.Errorf("failed to seed subcategory %s: %w", sc.Name, err)
			}
		}
	}

	// 4. Generate Admin JWT Token for Endpoint Invocation
	adminToken, err := security.GenerateJWT(admin.ID, admin.Email, string(admin.Role), admin.Name, cfg.JWTSecret, cfg.JWTExpireHours)
	if err != nil {
		return fmt.Errorf("failed to generate admin token: %w", err)
	}

	// 5. Initialize Router & Handlers for Clean Endpoint Invocation
	gin.SetMode(gin.ReleaseMode)
	authRepo := auth.NewRepository(db)
	authUC := auth.NewUseCase(authRepo, cfg)
	authHandler := auth.NewHandler(authUC)

	productRepo := product.NewRepository(db, cfg)
	productUC := product.NewUseCase(productRepo, nil)
	productHandler := product.NewHandler(productUC)

	orderRepo := order.NewRepository(db)
	orderUC := order.NewUseCase(orderRepo)
	orderHandler := order.NewHandler(orderUC)

	customerRepo := customer.NewRepository(db)
	customerUC := customer.NewUseCase(customerRepo)
	customerHandler := customer.NewHandler(customerUC)

	dashboardRepo := dashboard.NewRepository(db)
	dashboardUC := dashboard.NewUseCase(dashboardRepo)
	dashboardHandler := dashboard.NewHandler(dashboardUC)

	handlers := &router.Handlers{
		Auth:      authHandler,
		Product:   productHandler,
		Order:     orderHandler,
		Customer:  customerHandler,
		Dashboard: dashboardHandler,
	}

	appRouter := router.SetupRouter(cfg, handlers, nil)

	// 6. Seed Products by Hitting POST /api/v1/admin/products Endpoint
	log.Printf("🚀 Seeding %d Products by hitting POST /api/v1/admin/products endpoint...", len(seedData.Products))
	startTime := time.Now()
	successCount := 0

	for idx, p := range seedData.Products {
		subCatID := p.SubCategoryID
		isActive := true

		reqBody := product.CreateProductRequest{
			CategoryID:        p.CategoryID,
			SubCategoryID:     &subCatID,
			Name:              p.Name,
			SKU:               p.SKU,
			Description:       p.Description,
			Price:             p.Price,
			Currency:          p.Currency,
			StockQuantity:     p.StockQuantity,
			LowStockThreshold: p.LowStockThreshold,
			ImageURL:          p.ImageURL,
			IsActive:          &isActive,
			Badge:             p.Badge,
		}

		payloadBytes, err := json.Marshal(reqBody)
		if err != nil {
			return fmt.Errorf("failed to serialize product %d: %w", idx+1, err)
		}

		httpReq, err := http.NewRequest(http.MethodPost, "/api/v1/admin/products", bytes.NewReader(payloadBytes))
		if err != nil {
			return fmt.Errorf("failed to create http request for product %d: %w", idx+1, err)
		}

		httpReq.Header.Set("Content-Type", "application/json")
		httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", adminToken))

		recorder := httptest.NewRecorder()
		appRouter.ServeHTTP(recorder, httpReq)

		if recorder.Code != http.StatusCreated && recorder.Code != http.StatusOK {
			return fmt.Errorf("endpoint POST /api/v1/admin/products failed for SKU %s (status %d): %s", p.SKU, recorder.Code, recorder.Body.String())
		}

		// Update rating on the created product in DB if rating is specified
		if p.Rating > 0 {
			db.Model(&product.Product{}).Where("sku = ?", p.SKU).Update("rating", p.Rating)
		}

		successCount++
		if (idx+1)%10 == 0 || (idx+1) == len(seedData.Products) {
			log.Printf("  ✨ Seeded %d/%d products via Admin API endpoint...", idx+1, len(seedData.Products))
		}
	}

	duration := time.Since(startTime)
	log.Printf("✅ Successfully seeded %d products via endpoint in %.2fs!", successCount, duration.Seconds())

	// 7. Seed Sample Orders for Analytics & History
	log.Println("🛒 Seeding sample orders and stock audit logs...")
	var seededProducts []product.Product
	db.Limit(10).Find(&seededProducts)

	if len(seededProducts) >= 4 {
		order1 := order.Order{
			UserID:          shopper.ID,
			OrderNumber:     fmt.Sprintf("TRN-ORD-%d-001", time.Now().Unix()),
			TotalAmount:     seededProducts[0].Price + seededProducts[1].Price,
			Currency:        "IDR",
			Status:          order.StatusCompleted,
			ShippingName:    shopper.Name,
			ShippingPhone:   shopper.Phone,
			ShippingAddress: shopper.Address,
			PaymentMethod:   "SIMULATED_CARD",
			PaymentStatus:   order.PaymentSuccess,
			CreatedAt:       time.Now().Add(-48 * time.Hour),
		}
		db.Create(&order1)

		db.Create(&order.OrderItem{
			OrderID:      order1.ID,
			ProductID:    seededProducts[0].ID,
			ProductName:  seededProducts[0].Name,
			ProductSKU:   seededProducts[0].SKU,
			ProductImage: seededProducts[0].ImageURL,
			Quantity:     1,
			UnitPrice:    seededProducts[0].Price,
			Subtotal:     seededProducts[0].Price,
			Currency:     "IDR",
		})
		db.Create(&order.OrderItem{
			OrderID:      order1.ID,
			ProductID:    seededProducts[1].ID,
			ProductName:  seededProducts[1].Name,
			ProductSKU:   seededProducts[1].SKU,
			ProductImage: seededProducts[1].ImageURL,
			Quantity:     1,
			UnitPrice:    seededProducts[1].Price,
			Subtotal:     seededProducts[1].Price,
			Currency:     "IDR",
		})

		order2 := order.Order{
			UserID:          sarah.ID,
			OrderNumber:     fmt.Sprintf("TRN-ORD-%d-002", time.Now().Unix()),
			TotalAmount:     seededProducts[2].Price*2 + seededProducts[3].Price,
			Currency:        "IDR",
			Status:          order.StatusPaid,
			ShippingName:    sarah.Name,
			ShippingPhone:   sarah.Phone,
			ShippingAddress: sarah.Address,
			PaymentMethod:   "QRIS",
			PaymentStatus:   order.PaymentSuccess,
			CreatedAt:       time.Now().Add(-12 * time.Hour),
		}
		db.Create(&order2)

		db.Create(&order.OrderItem{
			OrderID:      order2.ID,
			ProductID:    seededProducts[2].ID,
			ProductName:  seededProducts[2].Name,
			ProductSKU:   seededProducts[2].SKU,
			ProductImage: seededProducts[2].ImageURL,
			Quantity:     2,
			UnitPrice:    seededProducts[2].Price,
			Subtotal:     seededProducts[2].Price * 2,
			Currency:     "IDR",
		})
		db.Create(&order.OrderItem{
			OrderID:      order2.ID,
			ProductID:    seededProducts[3].ID,
			ProductName:  seededProducts[3].Name,
			ProductSKU:   seededProducts[3].SKU,
			ProductImage: seededProducts[3].ImageURL,
			Quantity:     1,
			UnitPrice:    seededProducts[3].Price,
			Subtotal:     seededProducts[3].Price,
			Currency:     "IDR",
		})

		// Stock adjustment audit log
		db.Create(&product.StockAdjustmentLog{
			ProductID:      seededProducts[0].ID,
			AdjustmentType: "ADD",
			Quantity:       20,
			PreviousStock:  15,
			NewStock:       35,
			Reason:         "Penerimaan stok batch perdana dari distributor resmi",
			AdjustedBy:     admin.ID,
			CreatedAt:      time.Now().Add(-72 * time.Hour),
		})
	}

	log.Println("🎉 Database Seeding successfully completed!")
	return nil
}
