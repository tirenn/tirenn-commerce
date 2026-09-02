import logging
import asyncio
from contextlib import asynccontextmanager
from fastapi import FastAPI
from fastapi.middleware.cors import CORSMiddleware
import httpx

from app.core.config import settings
from app.core.logging_config import setup_logging, DistributedTracingMiddleware, get_current_request_id, get_current_trace_id, get_current_span_id
from app.core.security import (
    SecurityHeadersMiddleware,
    RequestBodySizeLimitMiddleware,
    RateLimitMiddleware,
)
from app.repositories.embedding_repository import EmbeddingRepository
from app.repositories.product_repository import ProductRepository
from app.repositories.knowledge_repository import KnowledgeRepository
from app.repositories.llm_repository import LLMRepository
from app.repositories.analytics_repository import AnalyticsRepository

from app.usecases.search_usecase import SearchUseCase
from app.usecases.sync_usecase import SyncUseCase
from app.usecases.knowledge_usecase import KnowledgeUseCase
from app.usecases.shopper_usecase import ShopperUseCase
from app.usecases.admin_usecase import AdminUseCase
from app.usecases.recommendation_usecase import RecommendationUseCase

from app.handlers.chat_handler import get_chat_router
from app.handlers.catalog_handler import get_catalog_router
from app.handlers.knowledge_handler import get_knowledge_router
from app.handlers.recommendation_handler import get_recommendation_router

# Initialize unified structured logging with Request ID
setup_logging()
logger = logging.getLogger("ai-service.main")

# ==============================================================================
# Dependency Injection Container (Clean Architecture)
# ==============================================================================

# 1. Repositories & Core Services
embedding_repo = EmbeddingRepository()
product_repo = ProductRepository()
knowledge_repo = KnowledgeRepository()
llm_repo = LLMRepository()
analytics_repo = AnalyticsRepository()
from app.core.llm_cache import LLMSemanticCache
llm_cache = LLMSemanticCache()

# 2. UseCases
search_usecase = SearchUseCase(embedding_repo=embedding_repo, product_repo=product_repo)
sync_usecase = SyncUseCase(embedding_repo=embedding_repo, product_repo=product_repo)
knowledge_usecase = KnowledgeUseCase(knowledge_repo=knowledge_repo, embedding_repo=embedding_repo)
shopper_usecase = ShopperUseCase(
    llm_repo=llm_repo,
    product_repo=product_repo,
    search_usecase=search_usecase,
    knowledge_usecase=knowledge_usecase,
    embedding_repo=embedding_repo,
    llm_cache=llm_cache
)
admin_usecase = AdminUseCase(
    llm_repo=llm_repo,
    product_repo=product_repo,
    knowledge_usecase=knowledge_usecase,
    analytics_repo=analytics_repo
)
recommendation_usecase = RecommendationUseCase(
    product_repo=product_repo,
    embedding_repo=embedding_repo
)

# 3. Handlers
chat_router = get_chat_router(shopper_usecase=shopper_usecase, admin_usecase=admin_usecase)
catalog_router = get_catalog_router(search_usecase=search_usecase, sync_usecase=sync_usecase)
knowledge_router = get_knowledge_router(knowledge_usecase=knowledge_usecase)
recommendation_router = get_recommendation_router(recommendation_usecase=recommendation_usecase)

async def _bg_sync():
    """Initial vector indexing sync on application boot"""
    try:
        await asyncio.sleep(1.5)
        headers = {"X-Request-ID": get_current_request_id()}
        async with httpx.AsyncClient(timeout=3.0) as client:
            resp = await client.get(f"{settings.BACKEND_API_URL}/products?limit=200", headers=headers)
            if resp.status_code == 200:
                logger.info("Backend detected. Triggering initial vector indexing in background...")
                await sync_usecase.sync_from_backend()
    except Exception as e:
        logger.info(f"Go backend not reachable at startup ({e}). Will sync on demand.")

@asynccontextmanager
async def lifespan(app: FastAPI):
    logger.info("🧠 Tirenn AI Service starting up with Clean Architecture & Vector RAG...")
    asyncio.create_task(llm_repo.ensure_model_available())
    asyncio.create_task(_bg_sync())
    yield
    logger.info("🛑 Tirenn AI Service shutting down...")

app = FastAPI(
    title=settings.SERVICE_NAME,
    version="1.0.0",
    description="Microservice providing Vector Embeddings, Fast Semantic Search, RAG Knowledge Base, and Shopper Agent for Tirenn Commerce.",
    lifespan=lifespan,
)

# Parse CORS allowed origins from .env
allowed_origins = [origin.strip() for origin in settings.CORS_ORIGINS.split(",") if origin.strip()]
if not allowed_origins:
    allowed_origins = ["*"]

# Middleware Stack (Executed in reverse order of registration)
app.add_middleware(DistributedTracingMiddleware)
app.add_middleware(RateLimitMiddleware)
app.add_middleware(RequestBodySizeLimitMiddleware)
app.add_middleware(SecurityHeadersMiddleware)
app.add_middleware(
    CORSMiddleware,
    allow_origins=allowed_origins,
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
    expose_headers=["X-Trace-ID", "X-Span-ID", "X-Request-ID", "X-Response-Time-Ms", "X-Correlation-ID"],
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
app.include_router(knowledge_router, prefix="/api/v1")
app.include_router(recommendation_router, prefix="/api/v1")

# ==============================================================================
# Global Exception Handlers (All Errors Logged with Request ID & Stack Trace)
# ==============================================================================

from fastapi.exceptions import RequestValidationError
from fastapi.responses import JSONResponse
from fastapi import HTTPException, Request

@app.exception_handler(RequestValidationError)
async def validation_exception_handler(request: Request, exc: RequestValidationError):
    req_id = get_current_request_id()
    logger.warning(
        f"⚠️ Validation Error on {request.method} {request.url.path} | "
        f"errors={exc.errors()}"
    )
    return JSONResponse(
        status_code=422,
        content={
            "success": False,
            "error": "Validation Error",
            "details": exc.errors(),
            "request_id": req_id
        },
        headers={"X-Request-ID": req_id}
    )

@app.exception_handler(HTTPException)
async def http_exception_handler(request: Request, exc: HTTPException):
    req_id = get_current_request_id()
    if exc.status_code >= 500:
        logger.error(f"🔥 HTTP {exc.status_code} on {request.method} {request.url.path}: {exc.detail}")
    else:
        logger.warning(f"⚠️ HTTP {exc.status_code} on {request.method} {request.url.path}: {exc.detail}")
    return JSONResponse(
        status_code=exc.status_code,
        content={
            "success": False,
            "error": exc.detail,
            "request_id": req_id
        },
        headers={"X-Request-ID": req_id}
    )

@app.exception_handler(Exception)
async def unhandled_exception_handler(request: Request, exc: Exception):
    req_id = get_current_request_id()
    logger.error(
        f"💥 [500 INTERNAL SERVER ERROR] Unhandled exception on {request.method} {request.url.path}: {exc}",
        exc_info=True
    )
    return JSONResponse(
        status_code=500,
        content={
            "success": False,
            "error": "Internal Server Error",
            "message": str(exc),
            "request_id": req_id
        },
        headers={"X-Request-ID": req_id}
    )

if __name__ == "__main__":
    import uvicorn
    uvicorn.run("app.main:app", host=settings.HOST, port=settings.PORT, reload=True)
