import logging
from typing import List, Dict, Any, Optional
from app.repositories.product_repository import ProductRepository
from app.repositories.embedding_repository import EmbeddingRepository

logger = logging.getLogger("ai-service.usecase.recommendation")


class RecommendationUseCase:
    """UseCase coordinating AI product recommendations, similarity scoring, and co-occurrence fallback"""

    def __init__(
        self,
        product_repo: ProductRepository,
        embedding_repo: Optional[EmbeddingRepository] = None
    ):
        self.product_repo = product_repo
        self.embedding_repo = embedding_repo

    def get_recommendations(
        self,
        product_id: int,
        rec_type: str = "similar",
        limit: int = 6
    ) -> List[Dict[str, Any]]:
        """
        Retrieve recommendations for a given product.
        - Clamps limit between 4 and 8 (default 6).
        - Dispatches to 'similar' (vector + category boost + price corridor) or
          'frequently_bought_together' / 'cross_category' (co-occurrence + cross-category vector fallback).
        """
        # Clamp limit between 4 and 8
        try:
            val_limit = int(limit) if limit is not None else 6
        except (ValueError, TypeError):
            val_limit = 6

        clamped_limit = max(4, min(8, val_limit))
        clean_type = (rec_type or "similar").strip().lower()

        logger.info(
            f"Generating recommendations: product_id={product_id}, "
            f"type='{clean_type}', limit={clamped_limit}"
        )

        if clean_type in ("frequently_bought_together", "cross_category", "cart", "fbt"):
            results = self.product_repo.get_frequently_bought_together(
                product_id=product_id,
                limit=clamped_limit
            )
        else:  # Default to "similar" (Produk Serupa / You May Also Like)
            results = self.product_repo.get_similar_products(
                product_id=product_id,
                limit=clamped_limit
            )

        return results[:clamped_limit]
