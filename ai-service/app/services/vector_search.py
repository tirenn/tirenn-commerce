import logging
from typing import List, Optional
import psycopg2
from psycopg2.extras import RealDictCursor
from sentence_transformers import SentenceTransformer

from app.core.config import settings
from app.schemas.search import ProductIndexItem, ScoredProductResult

logger = logging.getLogger("ai-service.vector")

DEFAULT_SCORE_THRESHOLD = 0.60

class VectorSearchService:
    def __init__(self):
        logger.info(f"Loading multilingual embedding model: {settings.EMBEDDING_MODEL_NAME}...")
        self.embedding_model = SentenceTransformer(settings.EMBEDDING_MODEL_NAME)
        self.vector_size = self.embedding_model.get_embedding_dimension()
        logger.info(f"Loaded {settings.EMBEDDING_MODEL_NAME} with {self.vector_size} dimensions.")

        self._init_db()

    def _get_connection(self):
        """Establish connection to PostgreSQL"""
        return psycopg2.connect(
            host=settings.DB_HOST,
            port=settings.DB_PORT,
            user=settings.DB_USER,
            password=settings.DB_PASSWORD,
            dbname=settings.DB_NAME,
            connect_timeout=5
        )

    def _init_db(self):
        """Enable pgvector extension and create HNSW index if table exists"""
        try:
            with self._get_connection() as conn:
                with conn.cursor() as cur:
                    cur.execute("CREATE EXTENSION IF NOT EXISTS vector;")
                    cur.execute("""
                        SELECT EXISTS (
                            SELECT FROM information_schema.tables 
                            WHERE table_name = 'products'
                        );
                    """)
                    exists = cur.fetchone()[0]
                    if exists:
                        cur.execute("""
                            ALTER TABLE products 
                            ADD COLUMN IF NOT EXISTS embedding vector(384);
                        """)
                        cur.execute("""
                            CREATE INDEX IF NOT EXISTS products_embedding_hnsw_idx 
                            ON products USING hnsw (embedding vector_cosine_ops);
                        """)
                    conn.commit()
            logger.info("🐘 PostgreSQL pgvector extension & HNSW indexing verified.")
        except Exception as e:
            logger.warning(f"Could not initialize pgvector at startup: {e}")

    def _to_vector_str(self, vec: list) -> str:
        """Convert float list to valid PostgreSQL vector literal string: '[-0.01,0.02,...]'"""
        return "[" + ",".join(f"{x:.6f}" for x in vec) + "]"

    def _build_rich_text(self, p: ProductIndexItem) -> str:
        """Create structured semantic text representation of product in Bahasa Indonesia"""
        parts = []
        if p.category_name:
            parts.append(f"Kategori: {p.category_name}")
        parts.append(f"Nama Produk: {p.name}")
        if p.description:
            parts.append(f"Deskripsi: {p.description}")
        return ". ".join(parts)

    def index_products(self, products: List[ProductIndexItem]) -> int:
        """Batch update vector embeddings in PostgreSQL products table"""
        if not products:
            return 0

        texts = [self._build_rich_text(p) for p in products]
        embeddings = self.embedding_model.encode(
            texts,
            normalize_embeddings=True,
            show_progress_bar=False
        )

        updated_count = 0
        try:
            with self._get_connection() as conn:
                with conn.cursor() as cur:
                    for p, emb in zip(products, embeddings):
                        vec_str = self._to_vector_str(emb.tolist())
                        cur.execute("""
                            UPDATE products 
                            SET embedding = %s::vector 
                            WHERE id = %s;
                        """, (vec_str, p.id))
                        updated_count += 1
                    conn.commit()
            logger.info(f"Successfully embedded {updated_count} products in PostgreSQL pgvector.")
            return updated_count
        except Exception as e:
            logger.error(f"Error indexing products in pgvector: {e}", exc_info=True)
            return 0

    def search_semantic(
        self,
        query: str,
        limit: int = 12,
        category_id: int = 0,
        score_threshold: float = DEFAULT_SCORE_THRESHOLD
    ) -> List[ScoredProductResult]:
        """Perform semantic vector search using PostgreSQL Cosine Distance (<=>)"""
        clean_q = query.strip()
        query_vector = self.embedding_model.encode(
            clean_q,
            normalize_embeddings=True
        ).tolist()
        query_vec_str = self._to_vector_str(query_vector)

        sql = """
            SELECT 
                p.id,
                p.name,
                p.sku,
                p.category_id,
                p.price,
                COALESCE(p.image_url, '') AS image_url,
                p.stock_quantity,
                ROUND((1 - (p.embedding <=> %s::vector))::numeric, 4) AS score
            FROM products p
            WHERE p.is_active = true
              AND p.embedding IS NOT NULL
              AND (%s = 0 OR p.category_id = %s)
              AND (1 - (p.embedding <=> %s::vector)) >= %s
            ORDER BY p.embedding <=> %s::vector ASC
            LIMIT %s;
        """

        results: List[ScoredProductResult] = []
        try:
            with self._get_connection() as conn:
                with conn.cursor(cursor_factory=RealDictCursor) as cur:
                    cur.execute(sql, (
                        query_vec_str,
                        category_id,
                        category_id,
                        query_vec_str,
                        score_threshold,
                        query_vec_str,
                        limit
                    ))
                    rows = cur.fetchall()
                    for r in rows:
                        results.append(
                            ScoredProductResult(
                                id=r["id"],
                                score=float(r["score"]),
                                name=r["name"],
                                category_id=r["category_id"],
                                sku=r["sku"],
                                price=float(r["price"]),
                                image_url=r["image_url"],
                                stock_quantity=int(r["stock_quantity"] or 0),
                            )
                        )
            return results
        except Exception as e:
            logger.error(f"Error executing pgvector search: {e}", exc_info=True)
            return []

# Singleton instance
vector_service = VectorSearchService()
