package router

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gocommerce-backend/internal/config"
	"gocommerce-backend/internal/domain/auth"
	"gocommerce-backend/internal/domain/customer"
	"gocommerce-backend/internal/domain/dashboard"
	"gocommerce-backend/internal/domain/order"
	"gocommerce-backend/internal/domain/product"
	"gocommerce-backend/internal/middleware"
	"gocommerce-backend/internal/utils"
)

// Handlers holds all domain HTTP handlers for route registration
type Handlers struct {
	Auth      *auth.Handler
	Product   *product.Handler
	Order     *order.Handler
	Customer  *customer.Handler
	Dashboard *dashboard.Handler
}

// SetupRouter initializes Gin engine, middleware, and all API endpoints
func SetupRouter(cfg *config.Config, handlers *Handlers) *gin.Engine {
	if cfg.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.Default()

	// Global Middlewares
	r.Use(middleware.SetupCORS())

	// Health Check Endpoint
	r.GET("/healthz", func(c *gin.Context) {
		utils.Success(c, http.StatusOK, "GoCommerce API is healthy 🚀", gin.H{
			"status":      "online",
			"database":    "mysql",
			"environment": cfg.Environment,
			"timestamp":   time.Now().UTC(),
		})
	})

	// API v1 Routing Tree
	v1 := r.Group("/api/v1")
	{
		// -------------------------------------------------------------
		// Public & Customer Auth Routes
		// -------------------------------------------------------------
		authGroup := v1.Group("/auth")
		{
			authGroup.POST("/register", handlers.Auth.Register)
			authGroup.POST("/login", handlers.Auth.Login)
			authGroup.GET("/me", middleware.JWTAuth(cfg), handlers.Auth.Me)
		}

		// -------------------------------------------------------------
		// Public Storefront Catalog Routes
		// -------------------------------------------------------------
		v1.GET("/products", handlers.Product.ListProducts)
		v1.GET("/products/:id", handlers.Product.GetProduct)
		v1.GET("/categories", handlers.Product.ListCategories)

		// -------------------------------------------------------------
		// Authenticated Customer Orders Routes
		// -------------------------------------------------------------
		orderGroup := v1.Group("/orders", middleware.JWTAuth(cfg))
		{
			orderGroup.POST("/checkout", handlers.Order.Checkout)
			orderGroup.GET("/my-orders", handlers.Order.ListMyOrders)
			orderGroup.GET("/:id", handlers.Order.GetOrder)
		}

		// -------------------------------------------------------------
		// Admin Back-Office Routes (Requires JWT & ADMIN Role)
		// -------------------------------------------------------------
		adminGroup := v1.Group("/admin", middleware.JWTAuth(cfg), middleware.RequireAdmin())
		{
			// Dashboard & Analytics
			adminGroup.GET("/dashboard", handlers.Dashboard.GetDashboard)

			// Products & Inventory Operations
			adminGroup.GET("/products", handlers.Product.AdminListProducts)
			adminGroup.POST("/products", handlers.Product.CreateProduct)
			adminGroup.PUT("/products/:id", handlers.Product.UpdateProduct)
			adminGroup.DELETE("/products/:id", handlers.Product.DeleteProduct)
			adminGroup.POST("/products/:id/adjust-stock", handlers.Product.AdjustStock)
			adminGroup.GET("/products/:id/stock-logs", handlers.Product.GetStockLogs)
			adminGroup.POST("/categories", handlers.Product.CreateCategory)

			// Order Fulfillment Operations
			adminGroup.GET("/orders", handlers.Order.AdminListOrders)
			adminGroup.PATCH("/orders/:id/status", handlers.Order.AdminUpdateOrderStatus)

			// Customer CRM Operations
			adminGroup.GET("/customers", handlers.Customer.ListCustomers)
			adminGroup.PATCH("/customers/:id/status", handlers.Customer.UpdateStatus)
		}
	}

	return r
}
