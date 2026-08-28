import logging
import asyncio
from contextlib import asynccontextmanager
from fastapi import FastAPI
from fastapi.middleware.cors import CORSMiddleware
import httpx

from app.core.config import settings
from app.core.security import (
    SecurityHeadersMiddleware,
    RequestBodySizeLimitMiddleware,
    RateLimitMiddleware,
)
from app.repositories.embedding_repository import EmbeddingRepository
from app.repositories.product_repository import ProductRepository
from app.repositories.llm_repository import LLMRepository

from app.usecases.search_usecase import SearchUseCase
from app.usecases.sync_usecase import SyncUseCase
from app.usecases.shopper_usecase import ShopperUseCase

from app.handlers.chat_handler import get_chat_router
from app.handlers.catalog_handler import get_catalog_router

logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s [%(levelname)s] %(name)s: %(message)s"
)
logger = logging.getLogger("ai-service.main")

# ==============================================================================
# Dependency Injection Container (Clean Architecture)
# ==============================================================================

# 1. Repositories
embedding_repo = EmbeddingRepository()
product_repo = ProductRepository()
llm_repo = LLMRepository()

# 2. UseCases
search_usecase = SearchUseCase(embedding_repo=embedding_repo, product_repo=product_repo)
sync_usecase = SyncUseCase(embedding_repo=embedding_repo, product_repo=product_repo)
shopper_usecase = ShopperUseCase(
    llm_repo=llm_repo,
    product_repo=product_repo,
    search_usecase=search_usecase
)

# 3. Handlers
chat_router = get_chat_router(shopper_usecase=shopper_usecase)
catalog_router = get_catalog_router(search_usecase=search_usecase, sync_usecase=sync_usecase)

async def _bg_sync():
    """Initial vector indexing sync on application boot"""
    try:
        await asyncio.sleep(1.5)
        async with httpx.AsyncClient(timeout=3.0) as client:
            resp = await client.get(f"{settings.BACKEND_API_URL}/products?limit=200")
            if resp.status_code == 200:
                logger.info("Backend detected. Triggering initial vector indexing in background...")
                await sync_usecase.sync_from_backend()
    except Exception as e:
        logger.info(f"Go backend not reachable at startup ({e}). Will sync on demand.")

@asynccontextmanager
async def lifespan(app: FastAPI):
    logger.info("🧠 Tirenn AI Service starting up with Clean Architecture (Handler-UseCase-Repository)...")
    asyncio.create_task(llm_repo.ensure_model_available())
    asyncio.create_task(_bg_sync())
    yield
    logger.info("🛑 Tirenn AI Service shutting down...")

app = FastAPI(
    title=settings.SERVICE_NAME,
    version="1.0.0",
    description="Microservice providing Vector Embeddings, Fast Semantic Search, and Product Intelligence for Tirenn Commerce.",
    lifespan=lifespan,
)

# Parse CORS allowed origins from .env
allowed_origins = [origin.strip() for origin in settings.CORS_ORIGINS.split(",") if origin.strip()]
if not allowed_origins:
    allowed_origins = ["*"]

# Middleware Stack
app.add_middleware(RateLimitMiddleware)
app.add_middleware(RequestBodySizeLimitMiddleware)
app.add_middleware(SecurityHeadersMiddleware)
app.add_middleware(
    CORSMiddleware,
    allow_origins=allowed_origins,
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

# System Healthcheck
@app.get("/healthz", tags=["System"])
async def healthz():
    return {
        "status": "online",
        "service": settings.SERVICE_NAME,
        "environment": settings.ENVIRONMENT,
        "model": settings.EMBEDDING_MODEL_NAME,
        "architecture": "Clean Architecture (Handler-UseCase-Repository)"
    }

# Register Handlers
app.include_router(chat_router, prefix="/api/v1")
app.include_router(catalog_router, prefix="/api/v1")

if __name__ == "__main__":
    import uvicorn
    uvicorn.run("app.main:app", host=settings.HOST, port=settings.PORT, reload=True)
