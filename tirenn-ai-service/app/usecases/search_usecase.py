import logging
from typing import List, Optional
from app.core.config import settings
from app.domain.product import ScoredProduct
from app.repositories.embedding_repository import EmbeddingRepository
from app.repositories.product_repository import ProductRepository

logger = logging.getLogger("ai-service.usecase.search")

class SearchUseCase:
    """UseCase orchestrating hybrid vector and keyword semantic catalog search"""

    def __init__(self, embedding_repo: EmbeddingRepository, product_repo: ProductRepository):
        self.embedding_repo = embedding_repo
        self.product_repo = product_repo

    def execute(
        self,
        query: str,
        limit: Optional[int] = None,
        category_id: int = 0,
        sub_category_id: int = 0,
        score_threshold: Optional[float] = None,
        min_price: Optional[float] = None,
        max_price: Optional[float] = None,
        in_stock: Optional[bool] = None,
        enable_hybrid: Optional[bool] = None
    ) -> List[ScoredProduct]:
        clean_q = query.strip()
        if not clean_q:
            return []

        limit_val = limit if limit is not None else settings.SEARCH_LIMIT
        threshold_val = score_threshold if score_threshold is not None else settings.DEFAULT_SEARCH_SCORE_THRESHOLD
        hybrid_val = enable_hybrid if enable_hybrid is not None else settings.ENABLE_HYBRID_SEARCH

        query_vector = self.embedding_repo.encode(clean_q)

        results = self.product_repo.search_hybrid(
            query_vector=query_vector,
            query_text=clean_q,
            category_id=category_id,
            sub_category_id=sub_category_id,
            score_threshold=threshold_val,
            min_price=min_price,
            max_price=max_price,
            in_stock=in_stock,
            limit=limit_val,
            enable_hybrid=hybrid_val
        )
        return results
