# ==============================================================================
# Tirenn Commerce Backend Makefile
# Provides automation for Goose migrations, database seeding, development, builds, and Docker
# ==============================================================================

.PHONY: help run dev build test clean seed db-seed db-reset migrate-create migrate-up migrate-down migrate-status migrate-reset docker-up docker-down docker-logs

help:
	@echo "🛍️ Tirenn Commerce Backend Make Commands:"
	@echo ""
	@echo "  Database Migrations & Seeding:"
	@echo "    make migrate-create name=<migration_name>  - Create a new SQL migration file"
	@echo "    make migrate-up                            - Run all pending Up migrations"
	@echo "    make migrate-down                          - Roll back the latest Down migration"
	@echo "    make migrate-status                        - View migration status & version"
	@echo "    make migrate-reset                         - Roll back ALL migrations"
	@echo "    make seed                                  - Seed database with 560 products, users, categories"
	@echo "    make db-reset                              - Roll back migrations, re-apply, and seed fresh data"
	@echo ""
	@echo "  Application & Testing:"
	@echo "    make run                                   - Run the server locally on :8080"
	@echo "    make test                                  - Run unit tests"
	@echo "    make build                                 - Build server binary to bin/server"
	@echo ""
	@echo "  🐳 Docker Containerization:"
	@echo "    make docker-up                             - Build and start Backend container (:8080)"
	@echo "    make docker-down                           - Stop Backend container"
	@echo "    make docker-logs                           - Tail Backend container logs"

# ------------------------------------------------------------------------------
# Goose Database Migrations & Seeding
# ------------------------------------------------------------------------------

migrate-create:
	@if [ -z "$(name)" ]; then \
		echo "❌ Error: 'name' is required. Usage: make migrate-create name=your_migration_name"; \
		exit 1; \
	fi
	go run ./cmd/migrate create $(name) sql

migrate-up:
	go run ./cmd/migrate up

migrate-down:
	go run ./cmd/migrate down

migrate-status:
	go run ./cmd/migrate status

migrate-reset:
	go run ./cmd/migrate reset

seed:
	go run ./cmd/seed

db-seed:
	go run ./cmd/seed

db-reset:
	go run ./cmd/migrate reset
	go run ./cmd/migrate up
	go run ./cmd/seed

# ------------------------------------------------------------------------------
# Development & Build
# ------------------------------------------------------------------------------

run:
	go run ./cmd/server

dev:
	go run ./cmd/server

build:
	go build -ldflags="-s -w" -o bin/server ./cmd/server

test:
	go test -v ./...

clean:
	rm -rf bin/

# ------------------------------------------------------------------------------
# Docker Containerization
# ------------------------------------------------------------------------------

docker-up:
	docker compose up -d --build

docker-down:
	docker compose down

docker-logs:
	docker compose logs -f
