package database

import (
	"fmt"
	"log"
	"time"

	"gocommerce-backend/internal/config"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// InitDB initializes strict MySQL database connection
func InitDB(cfg *config.Config) (*gorm.DB, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		cfg.DBUser,
		cfg.DBPassword,
		cfg.DBHost,
		cfg.DBPort,
		cfg.DBName,
	)

	log.Printf("Connecting to MySQL at %s:%s/%s...\n", cfg.DBHost, cfg.DBPort, cfg.DBName)
	dialector := mysql.Open(dsn)

	gormConfig := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
		NowFunc: func() time.Time {
			return time.Now().UTC()
		},
	}

	db, err := gorm.Open(dialector, gormConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MySQL (%s:%s): %w", cfg.DBHost, cfg.DBPort, err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve generic sql.DB: %w", err)
	}

	// Connection pool settings
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	// Seed initial test data
	if err := Seed(db); err != nil {
		log.Printf("Warning: Seeding initial data produced: %v\n", err)
	}

	log.Println("MySQL Database successfully connected and initialized.")
	return db, nil
}
