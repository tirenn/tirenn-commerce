# ==============================================================================
# Tirenn Commerce Root Makefile
# ==============================================================================

.PHONY: help migrate-create migrate-up migrate-down migrate-status migrate-reset backend-run frontend-run docker-up docker-down qa-test qa-run test-e2e

help:
	@echo "🛍️ Tirenn Commerce Project Commands:"
	@echo ""
	@echo "  Database Migrations (Goose / MySQL):"
	@echo "    make migrate-create name=<name>  - Create a new migration in backend/migrations/"
	@echo "    make migrate-up                  - Run pending migrations"
	@echo "    make migrate-down                - Roll back the latest migration"
	@echo "    make migrate-status              - Check migration status"
	@echo "    make migrate-reset               - Reset all migrations"
	@echo ""
	@echo "  Automated QA Testing:"
	@echo "    make test-e2e                    - Run Playwright E2E browser tests (Storefront & Admin)"
	@echo "    make qa-run                      - Run interactive QA API validation runner"
	@echo "    make qa-test                     - Run QA integration test suite"
	@echo ""
	@echo "  Docker Compose:"
	@echo "    make docker-up                   - Start MySQL, Backend, and Frontend"
	@echo "    make docker-down                 - Stop all services"
	@echo ""
	@echo "  Local Development:"
	@echo "    make backend-run                 - Start Go backend"
	@echo "    make frontend-run                - Start React frontend"

migrate-create:
	@$(MAKE) -C backend migrate-create name=$(name)

migrate-up:
	@$(MAKE) -C backend migrate-up

migrate-down:
	@$(MAKE) -C backend migrate-down

migrate-status:
	@$(MAKE) -C backend migrate-status

migrate-reset:
	@$(MAKE) -C backend migrate-reset

backend-run:
	@$(MAKE) -C backend run

frontend-run:
	cd frontend && npm run dev

test-e2e:
	cd qa && npm run test:e2e

qa-run:
	go run ./qa/main.go

qa-test:
	cd qa && go test -v ./...

docker-up:
	docker compose up -d --build

docker-down:
	docker compose down
