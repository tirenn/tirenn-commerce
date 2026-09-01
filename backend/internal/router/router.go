package router

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/tirenn/commerce/backend/internal/config"
	"github.com/tirenn/commerce/backend/internal/domain/auth"
	"github.com/tirenn/commerce/backend/internal/domain/customer"
	"github.com/tirenn/commerce/backend/internal/domain/dashboard"
	"github.com/tirenn/commerce/backend/internal/domain/order"
	"github.com/tirenn/commerce/backend/internal/domain/product"
	"github.com/tirenn/commerce/backend/internal/middleware"
	"github.com/tirenn/commerce/backend/internal/response"
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
		response.Success(c, http.StatusOK, "Tirenn Commerce API is healthy 🚀", gin.H{
			"status":      "online",
			"database":    "postgres",
			"redis":       rdb != nil,
			"environment": cfg.Environment,
			"timestamp":   time.Now().UTC(),
		})
	})

	// API v1 Routing Tree
	v1 := r.Group("/api/v1")
	jwtAuth := middleware.JWTAuth(cfg)
	adminGroup := v1.Group("/admin", jwtAuth, middleware.RequireAdmin())

	// Modular Domain Route Registrations
	handlers.Auth.RegisterRoutes(v1, jwtAuth)
	handlers.Product.RegisterRoutes(v1, adminGroup)
	handlers.Order.RegisterRoutes(v1, jwtAuth, adminGroup)
	handlers.Customer.RegisterRoutes(adminGroup)
	handlers.Dashboard.RegisterRoutes(adminGroup)

	return r
}
