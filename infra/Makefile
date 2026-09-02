# ==============================================================================
# Tirenn Commerce Infrastructure & QA Makefile
# ==============================================================================

INFRA_DIR := ../../infra

.PHONY: help up down restart logs ps test-e2e qa-run qa-test

help:
	@echo "🏗️ Tirenn Infrastructure & QA Commands:"
	@echo ""
	@echo "  up        - Start shared infra (PostgreSQL pgvector, Redis, Ollama, Loki, Promtail, Grafana)"
	@echo "  down      - Stop shared infra"
	@echo "  restart   - Restart shared infra"
	@echo "  logs      - Tail shared infra logs"
	@echo "  ps        - List running infra containers"
	@echo "  test-e2e  - Run Playwright E2E test suite"
	@echo "  qa-run    - Run interactive QA API test runner"
	@echo "  qa-test   - Run QA integration tests"

up:
	cd $(INFRA_DIR) && docker compose up -d

down:
	cd $(INFRA_DIR) && docker compose down

restart:
	cd $(INFRA_DIR) && docker compose restart

logs:
	cd $(INFRA_DIR) && docker compose logs -f

ps:
	cd $(INFRA_DIR) && docker compose ps

test-e2e:
	cd qa && npm run test:e2e

qa-run:
	go run ./qa/main.go

qa-test:
	cd qa && go test -v ./...
