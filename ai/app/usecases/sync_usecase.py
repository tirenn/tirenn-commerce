import logging
from typing import List, Tuple
import httpx
from app.core.config import settings
from app.domain.product import ProductIndexItem
from app.repositories.embedding_repository import EmbeddingRepository
from app.repositories.product_repository import ProductRepository

logger = logging.getLogger("ai-service.usecase.sync")

class SyncUseCase:
    """UseCase responsible for batch indexing products and syncing from backend REST API"""

    def __init__(self, embedding_repo: EmbeddingRepository, product_repo: ProductRepository):
        self.embedding_repo = embedding_repo
        self.product_repo = product_repo

    def _build_rich_text(self, p: ProductIndexItem) -> str:
        """Create structured bilingual-friendly semantic text representation of product"""
        parts = []
        if p.category_name:
            parts.append(f"Kategori: {p.category_name}")
        if p.sub_category_name:
            parts.append(f"Subkategori: {p.sub_category_name}")
        parts.append(f"Nama Produk: {p.name}")
        if p.sku:
            parts.append(f"SKU: {p.sku}")
        if p.badge:
            parts.append(f"Badge: {p.badge}")
        if p.description:
            parts.append(f"Deskripsi: {p.description}")
        if p.price > 0:
            parts.append(f"Harga: Rp {int(p.price):,}")
        return ". ".join(parts)

    def index_items(self, products: List[ProductIndexItem]) -> int:
        """Encode products into vectors and save to database"""
        if not products:
            return 0

        texts = [self._build_rich_text(p) for p in products]
        embeddings = self.embedding_repo.encode_batch(texts)

        pairs: List[Tuple[int, List[float]]] = [(p.id, emb) for p, emb in zip(products, embeddings)]
        return self.product_repo.save_embeddings(pairs)

    async def sync_from_backend(self) -> int:
        """Fetch all product pages from Go backend and index them in pgvector"""
        all_items: List[ProductIndexItem] = []
        page = 1
        limit = 50

        async with httpx.AsyncClient(timeout=15.0) as client:
            while True:
                resp = await client.get(f"{settings.BACKEND_API_URL}/products?page={page}&limit={limit}")
                if resp.status_code != 200:
                    break

                data = resp.json()
                products_raw = data.get("data", [])
                if not products_raw or not isinstance(products_raw, list):
                    break

                for p in products_raw:
                    cat = p.get("category") or {}
                    sub_cat = p.get("sub_category") or {}
                    all_items.append(
                        ProductIndexItem(
                            id=p.get("id"),
                            name=p.get("name", ""),
                            category_id=p.get("category_id", 0),
                            category_name=cat.get("name", ""),
                            sub_category_id=p.get("sub_category_id"),
                            sub_category_name=sub_cat.get("name", ""),
                            sku=p.get("sku", ""),
                            description=p.get("description", ""),
                            price=float(p.get("price", 0.0)),
                            currency=p.get("currency", "IDR"),
                            image_url=p.get("image_url", ""),
                            badge=p.get("badge", ""),
                            rating=float(p.get("rating", 5.0)),
                            stock_quantity=int(p.get("stock_quantity", 0)),
                        )
                    )

                meta = data.get("meta") or {}
                total_pages = meta.get("total_pages", 1)
                if page >= total_pages:
                    break
                page += 1

        count = self.index_items(all_items)
        logger.info(f"SyncUseCase: Synced and embedded {count} products in PostgreSQL pgvector.")
        return count
