package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"gocommerce-backend/internal/config"
	"github.com/pressly/goose/v3"
)

func main() {
	cfg := config.LoadConfig()

	args := os.Args[1:]
	if len(args) == 0 {
		log.Println("Usage: go run ./cmd/migrate <command> [args...]")
		log.Println("Commands:")
		log.Println("  create <migration_name> [sql]  - Create a new migration file")
		log.Println("  up                             - Apply all pending migrations")
		log.Println("  down                           - Roll back the latest migration")
		log.Println("  status                         - Check migration status")
		log.Println("  reset                          - Roll back all migrations")
		log.Println("  version                        - Print current migration version")
		os.Exit(1)
	}

	migrationsDir := "./migrations"
	filteredArgs := make([]string, 0, len(args))
	for _, arg := range args {
		if strings.HasPrefix(arg, "-dir=") {
			migrationsDir = strings.TrimPrefix(arg, "-dir=")
		} else if arg != "-dir" {
			filteredArgs = append(filteredArgs, arg)
		}
	}

	if len(filteredArgs) == 0 {
		log.Fatal("Error: No migration command provided.")
	}

	command := filteredArgs[0]

	// Handle 'create' command without requiring a database connection
	if command == "create" {
		if len(filteredArgs) < 2 {
			log.Fatal("Error: Migration name is required. Usage: make migrate-create name=<migration_name>")
		}
		name := filteredArgs[1]
		migrationType := "sql"
		if len(filteredArgs) >= 3 {
			migrationType = filteredArgs[2]
		}
		if err := goose.Create(nil, migrationsDir, name, migrationType); err != nil {
			log.Fatalf("Failed to create migration: %v", err)
		}
		log.Printf("Created new migration in '%s'\n", migrationsDir)
		return
	}

	// Format MySQL DSN
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true",
		cfg.DBUser,
		cfg.DBPassword,
		cfg.DBHost,
		cfg.DBPort,
		cfg.DBName,
	)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatalf("Failed to open MySQL connection: %v", err)
	}
	defer db.Close()

	// Connection retry loop (handles MySQL container initialization delay)
	var pingErr error
	for attempts := 1; attempts <= 5; attempts++ {
		pingErr = db.Ping()
		if pingErr == nil {
			break
		}
		if attempts < 5 {
			log.Printf("Waiting for MySQL at %s:%s (attempt %d/5)...", cfg.DBHost, cfg.DBPort, attempts)
			time.Sleep(1500 * time.Millisecond)
		}
	}

	if pingErr != nil {
		log.Fatalf("Failed to ping MySQL at %s:%s (database: %s): %v\nPlease ensure MySQL is running and healthy (e.g. docker compose up -d mysql).",
			cfg.DBHost, cfg.DBPort, cfg.DBName, pingErr)
	}

	if err := goose.SetDialect("mysql"); err != nil {
		log.Fatalf("Failed to set goose dialect to mysql: %v", err)
	}

	log.Printf("Executing goose migration command '%s' on database '%s'...\n", command, cfg.DBName)

	var cmdArgs []string
	if len(filteredArgs) > 1 {
		cmdArgs = filteredArgs[1:]
	}

	if err := goose.Run(command, db, migrationsDir, cmdArgs...); err != nil {
		log.Fatalf("Goose migration '%s' failed: %v", command, err)
	}

	log.Println("Goose migration executed successfully! 🚀")
}
