# ==============================================================================
# Tirenn Frontend Makefile
# ==============================================================================

.PHONY: help dev build preview lint docker-up docker-down docker-logs

help:
	@echo "🎨 Tirenn Frontend Management Commands:"
	@echo ""
	@echo "  💻 Local Development:"
	@echo "    make dev          - Run Vite dev server with hot module replacement (:3000)"
	@echo "    make build        - Compile TypeScript and build production bundle"
	@echo "    make preview      - Preview production build locally on :3000"
	@echo "    make lint         - Run ESLint code quality checks"
	@echo ""
	@echo "  🐳 Docker Containerization:"
	@echo "    make docker-up    - Build and start Nginx production container (:3000)"
	@echo "    make docker-down  - Stop Frontend container"
	@echo "    make docker-logs  - Tail Frontend container logs"

dev:
	npm run dev

build:
	npm run build

preview:
	npm run preview

lint:
	npm run lint

docker-up:
	docker compose up -d --build

docker-down:
	docker compose down

docker-logs:
	docker compose logs -f
