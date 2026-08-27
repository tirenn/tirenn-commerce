import logging
from contextlib import asynccontextmanager
from fastapi import FastAPI
from fastapi.middleware.cors import CORSMiddleware
import httpx

from app.core.config import settings
from app.api.router import router as api_router

logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s [%(levelname)s] %(name)s: %(message)s"
)
logger = logging.getLogger("ai-service.main")

@asynccontextmanager
async def lifespan(app: FastAPI):
    logger.info("🧠 Tirenn AI Semantic Search Service starting up...")
    # Attempt auto-syncing products from Go backend on startup in background
    try:
        async with httpx.AsyncClient(timeout=3.0) as client:
            resp = await client.get(f"{settings.BACKEND_API_URL}/products?limit=200")
            if resp.status_code == 200:
                logger.info("Backend detected. Triggering initial vector indexing...")
                from app.api.router import sync_from_backend
                await sync_from_backend()
    except Exception as e:
        logger.info(f"Go backend not reachable at startup ({e}). Will sync when requested.")
    
    yield
    logger.info("🛑 Tirenn AI Semantic Search Service shutting down...")

app = FastAPI(
    title=settings.SERVICE_NAME,
    version="1.0.0",
    description="Microservice providing Vector Embeddings, Fast Semantic Search, and Product Intelligence for Tirenn Commerce.",
    lifespan=lifespan,
)

# CORS Configuration
app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

# Healthcheck
@app.get("/healthz", tags=["System"])
async def healthz():
    return {
        "status": "online",
        "service": settings.SERVICE_NAME,
        "environment": settings.ENVIRONMENT,
        "model": settings.EMBEDDING_MODEL_NAME,
    }

# Register API Router
app.include_router(api_router, prefix="/api/v1", tags=["Semantic Search"])

if __name__ == "__main__":
    import uvicorn
    uvicorn.run("app.main:app", host=settings.HOST, port=settings.PORT, reload=True)
