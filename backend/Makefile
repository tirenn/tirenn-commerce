# ==============================================================================
# Tirenn Commerce Backend Makefile
# Provides automation for Goose migrations, development, and builds
# ==============================================================================

.PHONY: help run dev build test clean migrate-create migrate-up migrate-down migrate-status migrate-reset

help:
	@echo "🛍️ Tirenn Commerce Backend Make Commands:"
	@echo ""
	@echo "  Database Migrations (Goose):"
	@echo "    make migrate-create name=<migration_name>  - Create a new SQL migration file"
	@echo "    make migrate-up                            - Run all pending Up migrations"
	@echo "    make migrate-down                          - Roll back the latest Down migration"
	@echo "    make migrate-status                        - View migration status & version"
	@echo "    make migrate-reset                         - Roll back ALL migrations"
	@echo ""
	@echo "  Application & Testing:"
	@echo "    make run                                   - Run the server locally"
	@echo "    make test                                  - Run unit tests"
	@echo "    make build                                 - Build server binary to bin/server"
	@echo ""

# ------------------------------------------------------------------------------
# Goose Database Migrations
# ------------------------------------------------------------------------------

## Create a new migration file: make migrate-create name=add_new_table
migrate-create:
	@if [ -z "$(name)" ]; then \
		echo "❌ Error: 'name' is required. Usage: make migrate-create name=your_migration_name"; \
		exit 1; \
	fi
	go run ./cmd/migrate create $(name) sql

## Apply all pending Up migrations
migrate-up:
	go run ./cmd/migrate up

## Roll back the single most recent migration
migrate-down:
	go run ./cmd/migrate down

## Check the current migration status
migrate-status:
	go run ./cmd/migrate status

## Roll back all migrations
migrate-reset:
	go run ./cmd/migrate reset

# ------------------------------------------------------------------------------
# Development & Build
# ------------------------------------------------------------------------------

run:
	go run ./cmd/server

dev:
	go run ./cmd/server

test:
	go test -v ./...

build:
	go build -v -o bin/server ./cmd/server

clean:
	rm -rf bin/
