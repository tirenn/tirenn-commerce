package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"tirenn-ai-commerce/internal/client/ollama"
	"tirenn-ai-commerce/internal/config"
	"tirenn-ai-commerce/internal/database"
	"tirenn-ai-commerce/internal/domain/ai"
	"tirenn-ai-commerce/internal/domain/auth"
	"tirenn-ai-commerce/internal/domain/customer"
	"tirenn-ai-commerce/internal/domain/dashboard"
	"tirenn-ai-commerce/internal/domain/order"
	"tirenn-ai-commerce/internal/domain/product"
	"tirenn-ai-commerce/internal/router"
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
	authRepo := auth.NewRepository(db)
	productRepo := product.NewRepository(db)
	orderRepo := order.NewRepository(db)
	customerRepo := customer.NewRepository(db)
	dashboardRepo := dashboard.NewRepository(db)

	// External Service Client (Infrastructure Layer)
	ollamaClient := ollama.NewClient(cfg.OllamaURL, cfg.OllamaModel, cfg.OllamaEmbeddingModel)

	// AI Engine Subsystems (Domain Repositories)
	sessionRepo := ai.NewSessionRepository(rdb)
	knowledgeRepo := ai.NewKnowledgeRepository(db)
	ragCacheRepo := ai.NewRAGCacheRepository(rdb)

	// 5. Dependency Injection: UseCases
	authUseCase := auth.NewUseCase(authRepo, cfg)
	productUseCase := product.NewUseCase(productRepo, rdb, ollamaClient)
	orderUseCase := order.NewUseCase(orderRepo)
	customerUseCase := customer.NewUseCase(customerRepo)
	dashboardUseCase := dashboard.NewUseCase(dashboardRepo)
	shopperAIUseCase := ai.NewShopperUseCase(ollamaClient, sessionRepo, knowledgeRepo, ragCacheRepo, db, cfg)
	adminAIUseCase := ai.NewAdminUseCase(ollamaClient, sessionRepo, knowledgeRepo, ragCacheRepo, db, cfg)
	knowledgeUseCase := ai.NewKnowledgeUseCase(knowledgeRepo, ragCacheRepo, ollamaClient)

	// 6. Dependency Injection: Handlers
	handlers := &router.Handlers{
		Auth:      auth.NewHandler(authUseCase),
		Product:   product.NewHandler(productUseCase),
		Order:     order.NewHandler(orderUseCase),
		Customer:  customer.NewHandler(customerUseCase),
		Dashboard: dashboard.NewHandler(dashboardUseCase),
		AI:        ai.NewHandler(shopperAIUseCase, adminAIUseCase, knowledgeUseCase, sessionRepo, ollamaClient, db, cfg),
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
