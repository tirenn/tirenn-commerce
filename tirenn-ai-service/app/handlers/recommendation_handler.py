import logging
from typing import List, Optional, Any
from pydantic import BaseModel, Field
from fastapi import APIRouter, HTTPException, Query, Path

from app.usecases.recommendation_usecase import RecommendationUseCase

logger = logging.getLogger("ai-service.handler.recommendation")


class RecommendationItem(BaseModel):
    id: int
    score: float
    reason: str
    name: Optional[str] = ""
    sku: Optional[str] = ""
    category_id: Optional[int] = None
    sub_category_id: Optional[int] = None
    sub_category_name: Optional[str] = ""
    price: Optional[float] = 0.0
    currency: Optional[str] = "IDR"
    image_url: Optional[str] = ""
    stock_quantity: Optional[int] = 0
    badge: Optional[str] = ""
    description: Optional[str] = ""


class RecommendationResponse(BaseModel):
    success: bool = True
    product_id: int
    recommendations: List[RecommendationItem]
    data: Optional[List[RecommendationItem]] = None
    total: int


def get_recommendation_router(recommendation_usecase: RecommendationUseCase) -> APIRouter:
    """Factory creating product recommendation routes with injected usecase"""
    router = APIRouter(tags=["Product Recommendations"])

    @router.get(
        "/products/{id}/recommendations",
        response_model=RecommendationResponse,
        summary="Get Real-Time AI Product Recommendations",
        description="Retrieve similar items or frequently-bought-together add-on products for a given product ID."
    )
    async def get_recommendations(
        id: int = Path(..., description="Target Product ID", ge=1),
        limit: int = Query(6, description="Number of recommendations (clamped between 4 and 8)"),
        type: Optional[str] = Query("similar", description="Recommendation strategy: 'similar' or 'cross_category'/'frequently_bought_together'"),
        rec_type: Optional[str] = Query(None, description="Alias for recommendation type")
    ):
        try:
            effective_type = rec_type or type or "similar"
            raw_results = recommendation_usecase.get_recommendations(
                product_id=id,
                rec_type=effective_type,
                limit=limit
            )

            items = [RecommendationItem(**item) for item in raw_results]

            return RecommendationResponse(
                success=True,
                product_id=id,
                recommendations=items,
                data=items,
                total=len(items)
            )
        except Exception as e:
            logger.error(f"Error handling recommendation request for product #{id}: {e}", exc_info=True)
            raise HTTPException(status_code=500, detail=str(e))

    return router
