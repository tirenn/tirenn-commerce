import os
import logging
from typing import List, Dict, Any, Optional
from fastembed import TextEmbedding
from qdrant_client import QdrantClient
from qdrant_client.http import models as qmodels

from app.core.config import settings
from app.schemas.search import ProductIndexItem, ScoredProductResult

logger = logging.getLogger("ai-service.vector")

COLLECTION_NAME = "tirenn_products"

# Relevance threshold: products with cosine similarity below this cutoff are ignored
DEFAULT_SCORE_THRESHOLD = 0.65

class VectorSearchService:
    def __init__(self):
        logger.info(f"Initializing FastEmbed model: {settings.EMBEDDING_MODEL_NAME}...")
        self.embedding_model = TextEmbedding(model_name=settings.EMBEDDING_MODEL_NAME)
        
        # Determine vector size by embedding a dummy string
        sample_vec = list(self.embedding_model.embed(["sample test"]))[0]
        self.vector_size = len(sample_vec)
        logger.info(f"Vector dimensions initialized: {self.vector_size}")

        # Initialize Qdrant Client (In-memory or local persistent)
        if settings.QDRANT_STORAGE_PATH:
            os.makedirs(settings.QDRANT_STORAGE_PATH, exist_ok=True)
            self.client = QdrantClient(path=settings.QDRANT_STORAGE_PATH)
        else:
            self.client = QdrantClient(":memory:")

        self._ensure_collection()

    def _ensure_collection(self):
        """Create Qdrant collection if not exists"""
        collections = self.client.get_collections().collections
        exists = any(c.name == COLLECTION_NAME for c in collections)
        if not exists:
            logger.info(f"Creating Qdrant collection '{COLLECTION_NAME}'...")
            self.client.create_collection(
                collection_name=COLLECTION_NAME,
                vectors_config=qmodels.VectorParams(
                    size=self.vector_size,
                    distance=qmodels.Distance.COSINE
                )
            )
            # Create payload index on category_id for fast filtering
            self.client.create_payload_index(
                collection_name=COLLECTION_NAME,
                field_name="category_id",
                field_schema=qmodels.PayloadSchemaType.INTEGER,
            )

    def _build_rich_text(self, p: ProductIndexItem) -> str:
        """Create rich semantic text representation of product"""
        cat_text = f" Kategori: {p.category_name}." if p.category_name else ""
        desc_text = f" Deskripsi: {p.description}." if p.description else ""
        return f"{p.name}. SKU: {p.sku}.{cat_text}{desc_text}"

    def index_products(self, products: List[ProductIndexItem]) -> int:
        """Batch index products into Qdrant vector store"""
        if not products:
            return 0

        texts = [self._build_rich_text(p) for p in products]
        embeddings = list(self.embedding_model.embed(texts))

        points = []
        for p, emb in zip(products, embeddings):
            points.append(
                qmodels.PointStruct(
                    id=p.id,
                    vector=emb.tolist(),
                    payload={
                        "product_id": p.id,
                        "name": p.name,
                        "sku": p.sku,
                        "category_id": p.category_id,
                        "category_name": p.category_name,
                        "price": p.price,
                        "image_url": p.image_url or "",
                        "stock_quantity": p.stock_quantity or 0,
                        "description": p.description,
                    }
                )
            )

        self.client.upsert(
            collection_name=COLLECTION_NAME,
            points=points,
            wait=True
        )
        logger.info(f"Successfully indexed {len(points)} products into Qdrant vector collection.")
        return len(points)

    def search_semantic(
        self,
        query: str,
        limit: int = 12,
        category_id: int = 0,
        score_threshold: float = DEFAULT_SCORE_THRESHOLD
    ) -> List[ScoredProductResult]:
        """Perform semantic nearest-neighbor search using cosine similarity with relevance threshold"""
        query_vector = list(self.embedding_model.embed([query]))[0].tolist()

        query_filter = None
        if category_id > 0:
            query_filter = qmodels.Filter(
                must=[
                    qmodels.FieldCondition(
                        key="category_id",
                        match=qmodels.MatchValue(value=category_id)
                    )
                ]
            )

        if hasattr(self.client, "query_points"):
            response = self.client.query_points(
                collection_name=COLLECTION_NAME,
                query=query_vector,
                query_filter=query_filter,
                limit=limit,
                score_threshold=score_threshold,
                with_payload=True,
            )
            search_results = response.points
        else:
            search_results = self.client.search(
                collection_name=COLLECTION_NAME,
                query_vector=query_vector,
                query_filter=query_filter,
                limit=limit,
                score_threshold=score_threshold,
                with_payload=True,
            )

        results: List[ScoredProductResult] = []
        for r in search_results:
            payload = r.payload or {}
            results.append(
                ScoredProductResult(
                    id=payload.get("product_id", int(r.id)),
                    score=float(r.score),
                    name=payload.get("name", ""),
                    category_id=payload.get("category_id", 0),
                    sku=payload.get("sku", ""),
                    price=float(payload.get("price", 0.0)),
                    image_url=payload.get("image_url", ""),
                    stock_quantity=int(payload.get("stock_quantity", 0)),
                )
            )

        return results

# Singleton instance
vector_service = VectorSearchService()
