package main

import (
	"context"
	"flag"
	"log"
	"time"

	"tirenn-ai-commerce/internal/client/ollama"
	"tirenn-ai-commerce/internal/config"
	"tirenn-ai-commerce/internal/database"
)

func main() {
	forceFlag := flag.Bool("force", true, "Force reset database and seed fresh data")
	flag.Parse()

	log.Println("🌱 Loading configuration for Database Seeder...")
	cfg := config.LoadConfig()

	log.Printf("🔌 Connecting to PostgreSQL at %s:%s (DB: %s)...", cfg.DBHost, cfg.DBPort, cfg.DBName)
	db, err := database.InitDB(cfg)
	if err != nil {
		log.Fatalf("❌ Database connection error: %v", err)
	}

	// Initialize Ollama client (with fallback to localhost for host CLI execution)
	ollamaURL := cfg.OllamaURL
	ollamaClient := ollama.NewClient(ollamaURL, cfg.OllamaModel, cfg.OllamaEmbeddingModel)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := ollamaClient.Ping(ctx); err != nil && ollamaURL != "http://localhost:11434" && ollamaURL != "http://127.0.0.1:11434" {
		log.Println("Ollama container URL unreachable from host CLI, attempting http://localhost:11434...")
		ollamaClient = ollama.NewClient("http://localhost:11434", cfg.OllamaModel, cfg.OllamaEmbeddingModel)
	}

	if *forceFlag {
		log.Println("🚀 Executing full Force Database Reset and 560-Product Seeding with Vector Embeddings...")
		if err := database.ForceSeed(db, ollamaClient); err != nil {
			log.Fatalf("❌ Database seeding failed: %v", err)
		}
	} else {
		log.Println("🌱 Executing Safe Database Seeding (if not already populated)...")
		if err := database.Seed(db, ollamaClient); err != nil {
			log.Fatalf("❌ Database seeding failed: %v", err)
		}
	}

	log.Println("✅ Database successfully seeded with 560 products, vector embeddings, users, categories, and initial orders!")
}
