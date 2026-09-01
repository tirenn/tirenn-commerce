package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/tirenn/commerce/backend/internal/client/ai"
	"github.com/tirenn/commerce/backend/internal/config"
	"github.com/tirenn/commerce/backend/internal/database"
	"github.com/tirenn/commerce/backend/internal/domain/auth"
	"github.com/tirenn/commerce/backend/internal/domain/customer"
	"github.com/tirenn/commerce/backend/internal/domain/dashboard"
	"github.com/tirenn/commerce/backend/internal/domain/order"
	"github.com/tirenn/commerce/backend/internal/domain/product"
	"github.com/tirenn/commerce/backend/internal/router"
)

func main() {
	// 1. Load Configurations
	cfg := config.LoadConfig()

	// 2. Initialize PostgreSQL Database
	db, err := database.InitDB(cfg)
	if err != nil {
		log.Fatalf("Fatal: Database initialization failed: %v", err)
	}

	// 3. Initialize Redis Cache & Rate Limiter
	rdb, err := database.InitRedis(cfg)
	if err != nil {
		log.Printf("Warning: Redis connection error: %v (rate limiter will operate in pass-through mode)\n", err)
	}

	// 4. Dependency Injection: Repositories & External Adapters
	aiClient := ai.NewClient(cfg.AIServiceURL, cfg.InternalAPIKey)
	authRepo := auth.NewRepository(db)
	productRepo := product.NewRepository(db, cfg)
	orderRepo := order.NewRepository(db)
	customerRepo := customer.NewRepository(db)
	dashboardRepo := dashboard.NewRepository(db)

	// 5. Dependency Injection: UseCases
	authUseCase := auth.NewUseCase(authRepo, cfg)
	productUseCase := product.NewUseCase(productRepo, aiClient)
	orderUseCase := order.NewUseCase(orderRepo)
	customerUseCase := customer.NewUseCase(customerRepo)
	dashboardUseCase := dashboard.NewUseCase(dashboardRepo)

	// 6. Dependency Injection: Handlers
	handlers := &router.Handlers{
		Auth:      auth.NewHandler(authUseCase),
		Product:   product.NewHandler(productUseCase),
		Order:     order.NewHandler(orderUseCase),
		Customer:  customer.NewHandler(customerUseCase),
		Dashboard: dashboard.NewHandler(dashboardUseCase),
	}

	// 7. Setup Router from dedicated router package
	engine := router.SetupRouter(cfg, handlers, rdb)

	// 8. Server Configuration with Graceful Shutdown
	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      engine,
		ReadTimeout:  120 * time.Second,
		WriteTimeout: 180 * time.Second,
		IdleTimeout:  300 * time.Second,
	}

	go func() {
		log.Printf("💥 Tirenn Commerce Backend Server running on port %s 🚀\n", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server Listen error: %s\n", err)
		}
	}()

	// Graceful shutdown listener
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down Tirenn Commerce server gracefully...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}

	if rdb != nil {
		_ = rdb.Close()
	}

	log.Println("Tirenn Commerce server exited gracefully.")
}
