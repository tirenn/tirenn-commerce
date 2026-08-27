# ==============================================================================
# Tirenn Commerce Infrastructure & QA Makefile
# ==============================================================================

.PHONY: help up down restart logs ps test-e2e qa-run qa-test

help:
	@echo "🏗️ Tirenn Infrastructure & QA Commands:"
	@echo ""
	@echo "  up        - Start core infra (PostgreSQL pgvector, Redis, Ollama, Loki, Promtail, Grafana)"
	@echo "  down      - Stop core infra"
	@echo "  restart   - Restart core infra"
	@echo "  logs      - Tail infra logs"
	@echo "  ps        - List running infra containers"
	@echo "  test-e2e  - Run Playwright E2E test suite"
	@echo "  qa-run    - Run interactive QA API test runner"
	@echo "  qa-test   - Run QA integration tests"

up:
	docker compose up -d

down:
	docker compose down

restart:
	docker compose restart

logs:
	docker compose logs -f

ps:
	docker compose ps

test-e2e:
	cd qa && npm run test:e2e

qa-run:
	go run ./qa/main.go

qa-test:
	cd qa && go test -v ./...
