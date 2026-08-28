import logging
from typing import List, Optional, Dict, Tuple
import psycopg2
from psycopg2.extras import RealDictCursor
from app.core.config import settings
from app.domain.product import Product, ScoredProduct

logger = logging.getLogger("ai-service.repository.product")

class ProductRepository:
    """Repository handling all direct PostgreSQL queries, pgvector HNSW search, and categories"""

    def __init__(self):
        self._init_db()

    def _get_connection(self):
        """Establish PostgreSQL database connection"""
        return psycopg2.connect(
            host=settings.DB_HOST,
            port=settings.DB_PORT,
            user=settings.DB_USER,
            password=settings.DB_PASSWORD,
            dbname=settings.DB_NAME,
            connect_timeout=5
        )

    def _init_db(self):
        """Ensure pgvector, pg_trgm extensions and indexes exist"""
        try:
            with self._get_connection() as conn:
                with conn.cursor() as cur:
                    cur.execute("CREATE EXTENSION IF NOT EXISTS vector;")
                    cur.execute("CREATE EXTENSION IF NOT EXISTS pg_trgm;")
                    cur.execute("""
                        SELECT EXISTS (
                            SELECT FROM information_schema.tables 
                            WHERE table_name = 'products'
                        );
                    """)
                    if cur.fetchone()[0]:
                        cur.execute("""
                            ALTER TABLE products 
                            ADD COLUMN IF NOT EXISTS embedding vector(384);
                        """)
                        cur.execute("""
                            CREATE INDEX IF NOT EXISTS products_embedding_hnsw_idx 
                            ON products USING hnsw (embedding vector_cosine_ops);
                        """)
                        cur.execute("""
                            CREATE INDEX IF NOT EXISTS products_trgm_search_idx 
                            ON products USING gin ((name || ' ' || COALESCE(description, '')) gin_trgm_ops);
                        """)
                    conn.commit()
            logger.info("🐘 PostgreSQL pgvector & pg_trgm extensions & indexes verified.")
        except Exception as e:
            logger.warning(f"Database initialization warning: {e}")

    def _to_vector_str(self, vec: List[float]) -> str:
        """Format float list into PostgreSQL vector literal '[0.12,0.34,...]'"""
        return "[" + ",".join(f"{x:.6f}" for x in vec) + "]"

    def save_embeddings(self, items: List[Tuple[int, List[float]]]) -> int:
        """Batch update vector embeddings for product IDs"""
        if not items:
            return 0

        updated_count = 0
        try:
            with self._get_connection() as conn:
                with conn.cursor() as cur:
                    for product_id, emb in items:
                        vec_str = self._to_vector_str(emb)
                        cur.execute("""
                            UPDATE products 
                            SET embedding = %s::vector 
                            WHERE id = %s;
                        """, (vec_str, product_id))
                        updated_count += 1
                    conn.commit()
            return updated_count
        except Exception as e:
            logger.error(f"Failed to save embeddings in database: {e}", exc_info=True)
            return 0

    def search_hybrid(
        self,
        query_vector: List[float],
        query_text: str,
        category_id: int = 0,
        sub_category_id: int = 0,
        score_threshold: float = 0.55,
        min_price: Optional[float] = None,
        max_price: Optional[float] = None,
        in_stock: Optional[bool] = None,
        limit: int = 12,
        enable_hybrid: bool = True
    ) -> List[ScoredProduct]:
        """Perform hybrid vector + trigram search on products table"""
        query_vec_str = self._to_vector_str(query_vector)
        clean_q = query_text.strip()
        vec_weight = settings.HYBRID_VECTOR_WEIGHT
        text_weight = settings.HYBRID_TEXT_WEIGHT

        if enable_hybrid and clean_q:
            score_expr = f"""
                ROUND((
                    ({vec_weight} * (1 - (p.embedding <=> %s::vector))) + 
                    ({text_weight} * similarity(p.name || ' ' || COALESCE(p.description, '') || ' ' || COALESCE(sc.name, ''), %s))
                )::numeric, 4)
            """
            order_expr = f"""
                (({vec_weight} * (1 - (p.embedding <=> %s::vector))) + 
                 ({text_weight} * similarity(p.name || ' ' || COALESCE(p.description, '') || ' ' || COALESCE(sc.name, ''), %s))) DESC
            """
            sql = f"""
                SELECT 
                    p.id,
                    p.name,
                    p.sku,
                    p.category_id,
                    p.sub_category_id,
                    COALESCE(sc.name, '') AS sub_category_name,
                    p.price,
                    COALESCE(p.currency, 'IDR') AS currency,
                    COALESCE(p.image_url, '') AS image_url,
                    p.stock_quantity,
                    {score_expr} AS score
                FROM products p
                LEFT JOIN sub_categories sc ON sc.id = p.sub_category_id
                WHERE p.is_active = true
                  AND p.embedding IS NOT NULL
                  AND (%s = 0 OR p.category_id = %s)
                  AND (%s = 0 OR p.sub_category_id = %s)
                  AND (%s::numeric IS NULL OR p.price >= %s::numeric)
                  AND (%s::numeric IS NULL OR p.price <= %s::numeric)
                  AND (%s::boolean IS NULL OR %s::boolean = false OR p.stock_quantity > 0)
                  AND ({score_expr}) >= %s
                ORDER BY {order_expr}
                LIMIT %s;
            """
            params = (
                query_vec_str, clean_q,
                category_id, category_id,
                sub_category_id, sub_category_id,
                min_price, min_price,
                max_price, max_price,
                in_stock, in_stock,
                query_vec_str, clean_q, score_threshold,
                query_vec_str, clean_q,
                limit
            )
        else:
            score_expr = "ROUND((1 - (p.embedding <=> %s::vector))::numeric, 4)"
            sql = f"""
                SELECT 
                    p.id,
                    p.name,
                    p.sku,
                    p.category_id,
                    p.sub_category_id,
                    COALESCE(sc.name, '') AS sub_category_name,
                    p.price,
                    COALESCE(p.currency, 'IDR') AS currency,
                    COALESCE(p.image_url, '') AS image_url,
                    p.stock_quantity,
                    {score_expr} AS score
                FROM products p
                LEFT JOIN sub_categories sc ON sc.id = p.sub_category_id
                WHERE p.is_active = true
                  AND p.embedding IS NOT NULL
                  AND (%s = 0 OR p.category_id = %s)
                  AND (%s = 0 OR p.sub_category_id = %s)
                  AND (%s::numeric IS NULL OR p.price >= %s::numeric)
                  AND (%s::numeric IS NULL OR p.price <= %s::numeric)
                  AND (%s::boolean IS NULL OR %s::boolean = false OR p.stock_quantity > 0)
                  AND ({score_expr}) >= %s
                ORDER BY p.embedding <=> %s::vector ASC
                LIMIT %s;
            """
            params = (
                query_vec_str,
                category_id, category_id,
                sub_category_id, sub_category_id,
                min_price, min_price,
                max_price, max_price,
                in_stock, in_stock,
                query_vec_str, score_threshold,
                query_vec_str,
                limit
            )

        results: List[ScoredProduct] = []
        try:
            with self._get_connection() as conn:
                with conn.cursor(cursor_factory=RealDictCursor) as cur:
                    cur.execute(sql, params)
                    rows = cur.fetchall()
                    for r in rows:
                        results.append(
                            ScoredProduct(
                                id=r["id"],
                                name=r["name"],
                                sku=r["sku"],
                                category_id=r["category_id"],
                                sub_category_id=r.get("sub_category_id"),
                                sub_category_name=r.get("sub_category_name", ""),
                                price=float(r["price"]),
                                currency=r.get("currency", "IDR"),
                                image_url=r["image_url"],
                                stock_quantity=int(r["stock_quantity"] or 0),
                                score=float(r["score"])
                            )
                        )
            return results
        except Exception as e:
            logger.error(f"Error executing hybrid search in PostgreSQL: {e}", exc_info=True)
            return []

    def get_product_by_id(self, product_id: int) -> Optional[Product]:
        """Direct SQL lookup for product details by ID"""
        sql = """
            SELECT p.id, p.name, p.sku, p.category_id, p.sub_category_id,
                   COALESCE(sc.name, '') AS sub_category_name,
                   p.price, COALESCE(p.currency, 'IDR') AS currency,
                   COALESCE(p.image_url, '') AS image_url,
                   p.stock_quantity, p.badge, p.description, p.is_active
            FROM products p
            LEFT JOIN sub_categories sc ON sc.id = p.sub_category_id
            WHERE p.id = %s AND p.is_active = true
            LIMIT 1;
        """
        try:
            with self._get_connection() as conn:
                with conn.cursor(cursor_factory=RealDictCursor) as cur:
                    cur.execute(sql, (product_id,))
                    row = cur.fetchone()
                    if row:
                        return Product(
                            id=row["id"],
                            name=row["name"],
                            sku=row["sku"],
                            category_id=row["category_id"],
                            sub_category_id=row.get("sub_category_id"),
                            sub_category_name=row.get("sub_category_name", ""),
                            price=float(row["price"]),
                            currency=row.get("currency", "IDR"),
                            image_url=row["image_url"],
                            stock_quantity=int(row["stock_quantity"] or 0),
                            badge=row["badge"] or "",
                            description=row["description"] or "",
                            is_active=row["is_active"]
                        )
            return None
        except Exception as e:
            logger.error(f"Error executing direct product lookup by ID ({product_id}): {e}", exc_info=True)
            return None

    def get_product_by_sku(self, sku: str) -> Optional[Product]:
        """Direct SQL lookup strictly by SKU"""
        clean_sku = sku.strip()
        sql = """
            SELECT p.id, p.name, p.sku, p.category_id, p.sub_category_id,
                   COALESCE(sc.name, '') AS sub_category_name,
                   p.price, COALESCE(p.currency, 'IDR') AS currency,
                   COALESCE(p.image_url, '') AS image_url,
                   p.stock_quantity, p.badge, p.description, p.is_active
            FROM products p
            LEFT JOIN sub_categories sc ON sc.id = p.sub_category_id
            WHERE UPPER(p.sku) = UPPER(%s) AND p.is_active = true
            LIMIT 1;
        """
        try:
            with self._get_connection() as conn:
                with conn.cursor(cursor_factory=RealDictCursor) as cur:
                    cur.execute(sql, (clean_sku,))
                    row = cur.fetchone()
                    if row:
                        return Product(
                            id=row["id"],
                            name=row["name"],
                            sku=row["sku"],
                            category_id=row["category_id"],
                            sub_category_id=row.get("sub_category_id"),
                            sub_category_name=row.get("sub_category_name", ""),
                            price=float(row["price"]),
                            currency=row.get("currency", "IDR"),
                            image_url=row["image_url"],
                            stock_quantity=int(row["stock_quantity"] or 0),
                            badge=row["badge"] or "",
                            description=row["description"] or "",
                            is_active=row["is_active"]
                        )
            return None
        except Exception as e:
            logger.error(f"Error executing direct product lookup by SKU ({sku}): {e}", exc_info=True)
            return None

    def get_categories_map(self) -> Dict[int, str]:
        """Fetch all categories dynamically from PostgreSQL database"""
        sql = "SELECT id, name FROM categories ORDER BY id ASC;"
        try:
            with self._get_connection() as conn:
                with conn.cursor(cursor_factory=RealDictCursor) as cur:
                    cur.execute(sql)
                    rows = cur.fetchall()
                    if rows:
                        return {r["id"]: r["name"] for r in rows}
            return {}
        except Exception as e:
            logger.error(f"Error fetching categories from database: {e}", exc_info=True)
            return {}

    def get_sub_categories_map(self) -> Dict[int, str]:
        """Fetch all sub-categories dynamically from PostgreSQL database"""
        sql = "SELECT id, name FROM sub_categories ORDER BY id ASC;"
        try:
            with self._get_connection() as conn:
                with conn.cursor(cursor_factory=RealDictCursor) as cur:
                    cur.execute(sql)
                    rows = cur.fetchall()
                    if rows:
                        return {r["id"]: r["name"] for r in rows}
            return {}
        except Exception as e:
            logger.error(f"Error fetching sub-categories from database: {e}", exc_info=True)
            return {}
