# ==============================================================================
# Tirenn Commerce Root Makefile
# ==============================================================================

.PHONY: help migrate-create migrate-up migrate-down migrate-status migrate-reset \
        backend-run frontend-run ai-run \
        infra-up infra-down backend-up backend-down ai-up ai-down frontend-up frontend-down \
        docker-up docker-down qa-test qa-run test-e2e

help:
	@echo "🛍️ Tirenn Commerce Modular Project Commands:"
	@echo ""
	@echo "  🏗️ Modular Docker Stacks (Isolated Restarts):"
	@echo "    make infra-up       - Start core infra (Postgres, Redis, Ollama, Loki, Promtail, Grafana)"
	@echo "    make infra-down     - Stop core infra"
	@echo "    make backend-up     - Build & start ONLY Go Backend (:8080)"
	@echo "    make backend-down   - Stop Go Backend"
	@echo "    make ai-up          - Build & start ONLY Python AI Service (:8000)"
	@echo "    make ai-down        - Stop Python AI Service"
	@echo "    make frontend-up    - Build & start ONLY React Frontend (:3000)"
	@echo "    make frontend-down  - Stop React Frontend"
	@echo "    make docker-up      - Start all modular stacks"
	@echo "    make docker-down    - Stop all containers across all stacks"
	@echo ""
	@echo "  🧪 Automated QA Testing:"
	@echo "    make test-e2e       - Run Playwright E2E browser tests (Storefront & Admin)"
	@echo "    make qa-run         - Run interactive QA API validation runner"
	@echo "    make qa-test        - Run QA integration test suite"
	@echo ""
	@echo "  💻 Bare-Metal Local Development:"
	@echo "    make backend-run    - Start Go backend directly on :8080"
	@echo "    make frontend-run   - Start React frontend directly on :3000"
	@echo "    make ai-run         - Start Python AI service directly on :8000"

# --- Database Migrations ---
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

# --- Local Bare-Metal Dev ---
backend-run:
	@$(MAKE) -C backend run

frontend-run:
	cd frontend && npm run dev

ai-run:
	cd ai-service && uvicorn app.main:app --host 0.0.0.0 --port 8000 --reload

# --- QA & Testing ---
test-e2e:
	cd qa && npm run test:e2e

qa-run:
	go run ./qa/main.go

qa-test:
	cd qa && go test -v ./...

# --- Modular Docker Stacks ---
infra-up:
	docker compose up -d

infra-down:
	docker compose down

backend-up:
	cd backend && docker compose up -d --build

backend-down:
	cd backend && docker compose down

ai-up:
	cd ai-service && docker compose up -d --build

ai-down:
	cd ai-service && docker compose down

frontend-up:
	cd frontend && docker compose up -d --build

frontend-down:
	cd frontend && docker compose down

docker-up: infra-up backend-up ai-up frontend-up

docker-down:
	-cd frontend && docker compose down
	-cd backend && docker compose down
	-cd ai-service && docker compose down
	-docker compose down
