package main

import (
	"flag"
	"log"

	"github.com/tirenn/commerce/backend/internal/config"
	"github.com/tirenn/commerce/backend/internal/database"
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

	if *forceFlag {
		log.Println("🚀 Executing full Force Database Reset and Smartphone Catalog Seeding...")
		if err := database.ForceSeed(db, cfg); err != nil {
			log.Fatalf("❌ Database seeding failed: %v", err)
		}
	} else {
		log.Println("🌱 Executing Safe Database Seeding (if not already populated)...")
		if err := database.Seed(db); err != nil {
			log.Fatalf("❌ Database seeding failed: %v", err)
		}
	}

	log.Println("✅ Database successfully seeded with smartphone & electronics catalog, users, and categories!")
}
