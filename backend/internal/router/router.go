package router

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
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
func SetupRouter(cfg *config.Config, handlers *Handlers, rdb *redis.Client) *gin.Engine {
	if cfg.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.Default()

	// Global Middlewares
	r.Use(middleware.RequestID())
	r.Use(middleware.SetupCORS())
	r.Use(middleware.StructuredLogger())
	r.Use(middleware.RateLimiter(rdb, cfg))

	// Health Check Endpoint
	r.GET("/healthz", func(c *gin.Context) {
		utils.Success(c, http.StatusOK, "Tirenn Commerce API is healthy 🚀", gin.H{
			"status":      "online",
			"database":    "postgres",
			"redis":       rdb != nil,
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
		// Protected Merchant / Admin Routes
		// -------------------------------------------------------------
		admin := v1.Group("/admin", middleware.JWTAuth(cfg))
		{
			// Dashboard & Financial KPIs
			admin.GET("/dashboard", handlers.Dashboard.GetDashboard)

			// Catalog & Inventory Management
			admin.POST("/products", handlers.Product.CreateProduct)
			admin.PUT("/products/:id", handlers.Product.UpdateProduct)
			admin.DELETE("/products/:id", handlers.Product.DeleteProduct)
			admin.POST("/products/:id/adjust-stock", handlers.Product.AdjustStock)
			admin.GET("/products/:id/stock-logs", handlers.Product.GetStockLogs)
			admin.POST("/categories", handlers.Product.CreateCategory)

			// Customer CRM Management
			admin.GET("/customers", handlers.Customer.ListCustomers)
			admin.PATCH("/customers/:id/status", handlers.Customer.UpdateStatus)

			// Order Fulfillment Management
			admin.GET("/orders", handlers.Order.AdminListOrders)
			admin.PATCH("/orders/:id/status", handlers.Order.AdminUpdateOrderStatus)
		}
	}

	return r
}
