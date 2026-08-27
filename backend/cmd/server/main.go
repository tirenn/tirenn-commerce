package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"gocommerce-backend/internal/config"
	"gocommerce-backend/internal/database"
	"gocommerce-backend/internal/domain/auth"
	"gocommerce-backend/internal/domain/customer"
	"gocommerce-backend/internal/domain/dashboard"
	"gocommerce-backend/internal/domain/order"
	"gocommerce-backend/internal/domain/product"
	"gocommerce-backend/internal/router"
)

func main() {
	// 1. Load Configurations
	cfg := config.LoadConfig()

	// 2. Initialize MySQL Database
	db, err := database.InitDB(cfg)
	if err != nil {
		log.Fatalf("Fatal: Database initialization failed: %v", err)
	}

	// 3. Dependency Injection: Repositories & Clients
	authRepo := auth.NewRepository(db)
	productRepo := product.NewRepository(db)
	orderRepo := order.NewRepository(db)
	customerRepo := customer.NewRepository(db)
	dashboardRepo := dashboard.NewRepository(db)
	aiClient := product.NewAIClient(cfg.AIServiceURL)

	// 4. Dependency Injection: UseCases
	authUseCase := auth.NewUseCase(authRepo, cfg)
	productUseCase := product.NewUseCase(productRepo, aiClient)
	orderUseCase := order.NewUseCase(orderRepo)
	customerUseCase := customer.NewUseCase(customerRepo)
	dashboardUseCase := dashboard.NewUseCase(dashboardRepo)

	// 5. Dependency Injection: Handlers
	handlers := &router.Handlers{
		Auth:      auth.NewHandler(authUseCase),
		Product:   product.NewHandler(productUseCase),
		Order:     order.NewHandler(orderUseCase),
		Customer:  customer.NewHandler(customerUseCase),
		Dashboard: dashboard.NewHandler(dashboardUseCase),
	}

	// 6. Setup Router from dedicated router package
	engine := router.SetupRouter(cfg, handlers)

	// 7. Server Configuration with Graceful Shutdown
	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      engine,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
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

	log.Println("Tirenn Commerce server exited cleanly.")
}
