# ==============================================================================
# Tirenn AI Service Makefile
# ==============================================================================

.PHONY: help run dev docker-up docker-down docker-logs test sync

help:
	@echo "🤖 Tirenn AI Service Management Commands:"
	@echo ""
	@echo "  💻 Local Development:"
	@echo "    make dev          - Run FastAPI uvicorn server with hot reload (:8000)"
	@echo "    make run          - Run FastAPI uvicorn server in production mode"
	@echo "    make sync         - Trigger initial pgvector catalog embedding sync"
	@echo "    make test         - Run pytest test suite"
	@echo ""
	@echo "  🐳 Docker Containerization:"
	@echo "    make docker-up    - Build and start AI Service container (:8000)"
	@echo "    make docker-down  - Stop AI Service container"
	@echo "    make docker-logs  - Tail AI Service container logs"

dev:
	uvicorn app.main:app --host 0.0.0.0 --port 8000 --reload

run:
	uvicorn app.main:app --host 0.0.0.0 --port 8000 --workers 4

sync:
	python -c "from app.repositories.product_repository import ProductRepository; from app.repositories.embedding_repository import EmbeddingRepository; from app.usecases.sync_usecase import SyncUseCase; s = SyncUseCase(EmbeddingRepository(), ProductRepository()); import asyncio; asyncio.run(s.execute())"

test:
	pytest -v

docker-up:
	docker compose up -d --build

docker-down:
	docker compose down

docker-logs:
	docker compose logs -f
