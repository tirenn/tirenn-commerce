package database

import (
	"fmt"
	"log"
	"time"

	"gocommerce-backend/internal/config"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// InitDB initializes PostgreSQL database connection with pgvector extension
func InitDB(cfg *config.Config) (*gorm.DB, error) {
	dsn := cfg.Database.DSN()

	log.Printf("Connecting to PostgreSQL at %s:%s/%s...\n", cfg.DBHost, cfg.DBPort, cfg.DBName)
	dialector := postgres.Open(dsn)

	gormConfig := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
		NowFunc: func() time.Time {
			return time.Now().UTC()
		},
	}

	db, err := gorm.Open(dialector, gormConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to PostgreSQL (%s:%s): %w", cfg.DBHost, cfg.DBPort, err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve generic sql.DB: %w", err)
	}

	// Connection pool settings
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	// 1. Enable pgvector extension
	if err := db.Exec("CREATE EXTENSION IF NOT EXISTS vector;").Error; err != nil {
		log.Printf("Warning: Enabling pgvector extension produced: %v\n", err)
	} else {
		log.Println("🐘 PostgreSQL pgvector extension verified/enabled.")
	}

	// 2. Seed initial test data
	if err := Seed(db); err != nil {
		log.Printf("Warning: Seeding initial data produced: %v\n", err)
	}

	log.Println("🐘 PostgreSQL Database successfully connected and initialized.")
	return db, nil
}
