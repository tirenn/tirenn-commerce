import logging
from typing import List, Optional
from pydantic import BaseModel
from fastapi import APIRouter, HTTPException, Depends

from app.core.security import verify_internal_api_key
from app.domain.product import ScoredProduct, ProductIndexItem
from app.usecases.search_usecase import SearchUseCase
from app.usecases.sync_usecase import SyncUseCase

logger = logging.getLogger("ai-service.handler.catalog")

class SemanticSearchRequest(BaseModel):
    query: str
    limit: Optional[int] = 12
    category_id: Optional[int] = 0
    score_threshold: Optional[float] = None
    min_price: Optional[float] = None
    max_price: Optional[float] = None
    in_stock: Optional[bool] = None

class SemanticSearchResponse(BaseModel):
    success: bool
    query: str
    total_results: int
    data: List[ScoredProduct]

class IndexProductsRequest(BaseModel):
    products: List[ProductIndexItem]

class IndexProductsResponse(BaseModel):
    success: bool
    message: str
    indexed_count: int

def get_catalog_router(search_usecase: SearchUseCase, sync_usecase: SyncUseCase) -> APIRouter:
    """Factory creating catalog and search routes with injected usecases"""
    router = APIRouter(tags=["Catalog & Search"])

    @router.post("/search/semantic", response_model=SemanticSearchResponse)
    async def semantic_search(req: SemanticSearchRequest):
        try:
            results = search_usecase.execute(
                query=req.query,
                limit=req.limit,
                category_id=req.category_id or 0,
                score_threshold=req.score_threshold,
                min_price=req.min_price,
                max_price=req.max_price,
                in_stock=req.in_stock
            )
            return SemanticSearchResponse(
                success=True,
                query=req.query,
                total_results=len(results),
                data=results
            )
        except Exception as e:
            logger.error(f"SemanticSearch handler exception: {e}", exc_info=True)
            raise HTTPException(status_code=500, detail=str(e))

    @router.post("/index-products", response_model=IndexProductsResponse, dependencies=[Depends(verify_internal_api_key)])
    async def index_products(req: IndexProductsRequest):
        try:
            count = sync_usecase.index_items(req.products)
            return IndexProductsResponse(
                success=True,
                message=f"Indexed {count} products successfully",
                indexed_count=count
            )
        except Exception as e:
            logger.error(f"IndexProducts handler exception: {e}", exc_info=True)
            raise HTTPException(status_code=500, detail=str(e))

    @router.post("/sync-from-backend", dependencies=[Depends(verify_internal_api_key)])
    async def sync_from_backend():
        try:
            count = await sync_usecase.sync_from_backend()
            return {"success": True, "synced_products": count}
        except Exception as e:
            logger.error(f"SyncFromBackend handler exception: {e}", exc_info=True)
            raise HTTPException(status_code=500, detail=str(e))

    return router
