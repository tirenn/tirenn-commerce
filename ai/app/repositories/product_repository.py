import logging
from typing import List, Optional, Dict, Tuple, Any
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

    def get_taxonomy_prompt_text(self) -> str:
        """Dynamically fetch all categories & subcategories from PostgreSQL and format for LLM context"""
        sql = """
            SELECT 
                c.id AS category_id,
                c.name AS category_name,
                COALESCE(
                    json_agg(
                        json_build_object('id', sc.id, 'name', sc.name)
                        ORDER BY sc.id
                    ) FILTER (WHERE sc.id IS NOT NULL),
                    '[]'
                ) AS sub_categories
            FROM categories c
            LEFT JOIN sub_categories sc ON sc.category_id = c.id
            GROUP BY c.id, c.name
            ORDER BY c.id ASC;
        """
        try:
            with self._get_connection() as conn:
                with conn.cursor(cursor_factory=RealDictCursor) as cur:
                    cur.execute(sql)
                    rows = cur.fetchall()
                    lines = []
                    for r in rows:
                        subcats = r.get("sub_categories", [])
                        if isinstance(subcats, str):
                            import json
                            subcats = json.loads(subcats)
                        sub_str = ", ".join(f"sub_category_id={s['id']}: '{s['name']}'" for s in subcats)
                        lines.append(f"- category_id={r['category_id']}: '{r['category_name']}'" + (f" (Subcategories: {sub_str})" if sub_str else ""))
                    return "\n".join(lines)
        except Exception as e:
            logger.error(f"Error fetching dynamic taxonomy from database: {e}", exc_info=True)
            return ""

    def get_low_stock_products(self, threshold: int = 10, limit: int = 20) -> List[Dict[str, Any]]:
        """Fetch products with stock quantity less than or equal to threshold"""
        sql = """
            SELECT p.id, p.name, p.sku, p.price, p.currency, p.stock_quantity, p.low_stock_threshold,
                   c.name as category_name, sc.name as sub_category_name
            FROM products p
            LEFT JOIN categories c ON p.category_id = c.id
            LEFT JOIN sub_categories sc ON p.sub_category_id = sc.id
            WHERE p.stock_quantity <= %s AND p.is_active = true
            ORDER BY p.stock_quantity ASC, p.id ASC
            LIMIT %s;
        """
        try:
            with self._get_connection() as conn:
                with conn.cursor(cursor_factory=RealDictCursor) as cur:
                    cur.execute(sql, (threshold, limit))
                    rows = cur.fetchall()
                    return [dict(r) for r in rows]
        except Exception as e:
            logger.error(f"Error fetching low stock products: {e}", exc_info=True)
            return []

    def adjust_stock(
        self,
        product_id: int,
        adjustment_type: str,
        amount: int,
        reason: str,
        adjusted_by: int = 1
    ) -> Dict[str, Any]:
        """
        Atomically adjust product inventory stock and record immutable audit log in PostgreSQL
        """
        adj_type = adjustment_type.upper().strip()
        try:
            with self._get_connection() as conn:
                with conn.cursor(cursor_factory=RealDictCursor) as cur:
                    # 1. Lock target product row
                    cur.execute("SELECT id, name, sku, price, stock_quantity, low_stock_threshold, is_active FROM products WHERE id = %s FOR UPDATE;", (product_id,))
                    prod = cur.fetchone()
                    if not prod:
                        return {"success": False, "error": f"Product #{product_id} not found."}

                    prev_stock = int(prod["stock_quantity"] or 0)

                    if adj_type == "ADD":
                        change_amount = amount
                        new_stock = prev_stock + amount
                    elif adj_type == "SUBTRACT":
                        change_amount = -amount
                        new_stock = max(0, prev_stock - amount)
                    else:  # SET
                        change_amount = amount - prev_stock
                        new_stock = max(0, amount)

                    # 2. Update product stock
                    cur.execute("UPDATE products SET stock_quantity = %s, updated_at = NOW() WHERE id = %s;", (new_stock, product_id))

                    # 3. Insert audit log
                    cur.execute("""
                        INSERT INTO stock_adjustment_logs (product_id, adjustment_type, quantity, previous_stock, new_stock, reason, adjusted_by, created_at)
                        VALUES (%s, %s, %s, %s, %s, %s, %s, NOW())
                        RETURNING id, created_at;
                    """, (product_id, adj_type, change_amount, prev_stock, new_stock, reason, adjusted_by))
                    log_row = cur.fetchone()

                    conn.commit()

                    return {
                        "success": True,
                        "product_id": product_id,
                        "product_name": prod["name"],
                        "sku": prod["sku"],
                        "previous_stock": prev_stock,
                        "new_stock": new_stock,
                        "change_amount": change_amount,
                        "audit_log_id": log_row["id"] if log_row else None
                    }
        except Exception as e:
            logger.error(f"Database error adjusting stock for product #{product_id}: {e}", exc_info=True)
            return {"success": False, "error": str(e)}

    def get_similar_products(self, product_id: int, limit: int = 6) -> List[Dict[str, Any]]:
        """
        Compute similar products using pgvector cosine similarity,
        category (+0.08) and subcategory (+0.15) affinity soft-boost,
        and dynamic price corridor (0.4 * target_price <= price <= 2.5 * target_price).
        Excludes the target product itself and ensures is_active = true.
        """
        try:
            with self._get_connection() as conn:
                with conn.cursor(cursor_factory=RealDictCursor) as cur:
                    # 1. Fetch target product details & embedding
                    cur.execute("""
                        SELECT id, name, category_id, sub_category_id, price, embedding::text AS embedding_str
                        FROM products
                        WHERE id = %s AND is_active = true
                        LIMIT 1;
                    """, (product_id,))
                    target = cur.fetchone()
                    if not target:
                        logger.warning(f"Target product #{product_id} not found or inactive for recommendations.")
                        return []

                    target_cat = target["category_id"]
                    target_subcat = target["sub_category_id"]
                    target_price = float(target["price"] or 0.0)
                    target_emb = target.get("embedding_str")

                    # Price corridor calculation: [0.4 * target_price, 2.5 * target_price]
                    min_price = target_price * 0.4 if target_price > 0 else 0.0
                    max_price = target_price * 2.5 if target_price > 0 else 999999999.0

                    if target_emb:
                        # Vector similarity + Category/Subcategory Soft-Boost + Dynamic Price Corridor
                        sql = """
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
                                COALESCE(p.badge, '') AS badge,
                                COALESCE(p.description, '') AS description,
                                ROUND((
                                    (1 - (p.embedding <=> %s::vector)) + 
                                    CASE 
                                        WHEN %s IS NOT NULL AND p.sub_category_id = %s THEN 0.15
                                        WHEN %s IS NOT NULL AND p.category_id = %s THEN 0.08
                                        ELSE 0.00
                                    END
                                )::numeric, 4) AS score
                            FROM products p
                            LEFT JOIN sub_categories sc ON sc.id = p.sub_category_id
                            WHERE p.is_active = true
                              AND p.id != %s
                              AND p.embedding IS NOT NULL
                              AND (p.price BETWEEN %s AND %s)
                            ORDER BY score DESC
                            LIMIT %s;
                        """
                        cur.execute(
                            sql,
                            (
                                target_emb,
                                target_subcat, target_subcat,
                                target_cat, target_cat,
                                product_id,
                                min_price, max_price,
                                limit
                            )
                        )
                        rows = cur.fetchall()

                        # Fallback if price corridor yielded fewer items than limit: widen corridor
                        if len(rows) < limit:
                            existing_ids = tuple([r["id"] for r in rows] + [product_id])
                            remaining = limit - len(rows)
                            cur.execute("""
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
                                    COALESCE(p.badge, '') AS badge,
                                    COALESCE(p.description, '') AS description,
                                    ROUND((
                                        (1 - (p.embedding <=> %s::vector)) + 
                                        CASE 
                                            WHEN %s IS NOT NULL AND p.sub_category_id = %s THEN 0.15
                                            WHEN %s IS NOT NULL AND p.category_id = %s THEN 0.08
                                            ELSE 0.00
                                        END
                                    )::numeric, 4) AS score
                                FROM products p
                                LEFT JOIN sub_categories sc ON sc.id = p.sub_category_id
                                WHERE p.is_active = true
                                  AND p.id NOT IN %s
                                  AND p.embedding IS NOT NULL
                                ORDER BY score DESC
                                LIMIT %s;
                            """, (target_emb, target_subcat, target_subcat, target_cat, target_cat, existing_ids, remaining))
                            widen_rows = cur.fetchall()
                            rows.extend(widen_rows)

                        results = []
                        for r in rows:
                            results.append({
                                "id": r["id"],
                                "name": r["name"],
                                "sku": r["sku"],
                                "category_id": r["category_id"],
                                "sub_category_id": r.get("sub_category_id"),
                                "sub_category_name": r.get("sub_category_name", ""),
                                "price": float(r["price"]),
                                "currency": r.get("currency", "IDR"),
                                "image_url": r.get("image_url", ""),
                                "stock_quantity": int(r.get("stock_quantity") or 0),
                                "badge": r.get("badge", ""),
                                "description": r.get("description", ""),
                                "score": float(r["score"]),
                                "reason": "similar_category_price"
                            })
                        return results
                    else:
                        # Cold start / no embedding on target product -> Category-based fallback
                        cur.execute("""
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
                                COALESCE(p.badge, '') AS badge,
                                COALESCE(p.description, '') AS description,
                                0.5000 AS score
                            FROM products p
                            LEFT JOIN sub_categories sc ON sc.id = p.sub_category_id
                            WHERE p.is_active = true
                              AND p.id != %s
                              AND (%s IS NULL OR p.category_id = %s)
                            ORDER BY p.stock_quantity DESC, p.id ASC
                            LIMIT %s;
                        """, (product_id, target_cat, target_cat, limit))
                        rows = cur.fetchall()
                        return [
                            {
                                "id": r["id"],
                                "name": r["name"],
                                "sku": r["sku"],
                                "category_id": r["category_id"],
                                "sub_category_id": r.get("sub_category_id"),
                                "sub_category_name": r.get("sub_category_name", ""),
                                "price": float(r["price"]),
                                "currency": r.get("currency", "IDR"),
                                "image_url": r.get("image_url", ""),
                                "stock_quantity": int(r.get("stock_quantity") or 0),
                                "badge": r.get("badge", ""),
                                "description": r.get("description", ""),
                                "score": float(r["score"]),
                                "reason": "category_fallback"
                            }
                            for r in rows
                        ]
        except Exception as e:
            logger.error(f"Error computing similar products for #{product_id}: {e}", exc_info=True)
            return []

    def get_frequently_bought_together(self, product_id: int, limit: int = 6) -> List[Dict[str, Any]]:
        """
        Compute frequently bought together products via co-occurrence aggregation on order_items.
        Falls back to cross-category vector similarity when co-occurrence count is low or empty.
        """
        results: List[Dict[str, Any]] = []
        seen_ids = {product_id}

        try:
            with self._get_connection() as conn:
                with conn.cursor(cursor_factory=RealDictCursor) as cur:
                    # 1. Co-occurrence query on order_items
                    cur.execute("""
                        SELECT 
                            oi2.product_id AS id,
                            COUNT(DISTINCT oi2.order_id) AS co_occurrence_count,
                            p.name,
                            p.sku,
                            p.category_id,
                            p.sub_category_id,
                            COALESCE(sc.name, '') AS sub_category_name,
                            p.price,
                            COALESCE(p.currency, 'IDR') AS currency,
                            COALESCE(p.image_url, '') AS image_url,
                            p.stock_quantity,
                            COALESCE(p.badge, '') AS badge,
                            COALESCE(p.description, '') AS description
                        FROM order_items oi1
                        INNER JOIN order_items oi2 ON oi1.order_id = oi2.order_id AND oi1.product_id != oi2.product_id
                        INNER JOIN orders o ON o.id = oi1.order_id
                        INNER JOIN products p ON p.id = oi2.product_id
                        LEFT JOIN sub_categories sc ON sc.id = p.sub_category_id
                        WHERE oi1.product_id = %s 
                          AND (o.status IS NULL OR o.status != 'CANCELLED')
                          AND p.is_active = true
                        GROUP BY oi2.product_id, p.id, p.name, p.sku, p.category_id, p.sub_category_id, sc.name, p.price, p.currency, p.image_url, p.stock_quantity, p.badge, p.description
                        ORDER BY co_occurrence_count DESC
                        LIMIT %s;
                    """, (product_id, limit))
                    co_rows = cur.fetchall()

                    for r in co_rows:
                        seen_ids.add(r["id"])
                        co_count = int(r["co_occurrence_count"])
                        score = round(min(1.0, 0.60 + 0.08 * co_count), 4)
                        results.append({
                            "id": r["id"],
                            "name": r["name"],
                            "sku": r["sku"],
                            "category_id": r["category_id"],
                            "sub_category_id": r.get("sub_category_id"),
                            "sub_category_name": r.get("sub_category_name", ""),
                            "price": float(r["price"]),
                            "currency": r.get("currency", "IDR"),
                            "image_url": r.get("image_url", ""),
                            "stock_quantity": int(r.get("stock_quantity") or 0),
                            "badge": r.get("badge", ""),
                            "description": r.get("description", ""),
                            "score": score,
                            "reason": "frequently_bought_together"
                        })

                    # 2. Cross-category vector fallback if co-occurrence count is low (< limit)
                    if len(results) < limit:
                        needed = limit - len(results)
                        cur.execute("""
                            SELECT id, category_id, embedding::text AS embedding_str
                            FROM products
                            WHERE id = %s AND is_active = true
                            LIMIT 1;
                        """, (product_id,))
                        target = cur.fetchone()

                        if target and target.get("embedding_str"):
                            target_emb = target["embedding_str"]
                            target_cat = target["category_id"]
                            excluded_ids = tuple(seen_ids)

                            # First try products in different categories
                            cur.execute("""
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
                                    COALESCE(p.badge, '') AS badge,
                                    COALESCE(p.description, '') AS description,
                                    ROUND((1 - (p.embedding <=> %s::vector))::numeric, 4) AS score
                                FROM products p
                                LEFT JOIN sub_categories sc ON sc.id = p.sub_category_id
                                WHERE p.is_active = true 
                                  AND p.embedding IS NOT NULL
                                  AND p.id NOT IN %s
                                  AND (%s IS NULL OR p.category_id != %s)
                                ORDER BY score DESC
                                LIMIT %s;
                            """, (target_emb, excluded_ids, target_cat, target_cat, needed))
                            cross_rows = cur.fetchall()

                            for r in cross_rows:
                                seen_ids.add(r["id"])
                                results.append({
                                    "id": r["id"],
                                    "name": r["name"],
                                    "sku": r["sku"],
                                    "category_id": r["category_id"],
                                    "sub_category_id": r.get("sub_category_id"),
                                    "sub_category_name": r.get("sub_category_name", ""),
                                    "price": float(r["price"]),
                                    "currency": r.get("currency", "IDR"),
                                    "image_url": r.get("image_url", ""),
                                    "stock_quantity": int(r.get("stock_quantity") or 0),
                                    "badge": r.get("badge", ""),
                                    "description": r.get("description", ""),
                                    "score": float(r["score"]),
                                    "reason": "cross_category_vector"
                                })

                        # If still needed, fill with any active products not seen yet
                        if len(results) < limit:
                            still_needed = limit - len(results)
                            excluded_ids = tuple(seen_ids)
                            cur.execute("""
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
                                    COALESCE(p.badge, '') AS badge,
                                    COALESCE(p.description, '') AS description,
                                    0.4500 AS score
                                FROM products p
                                LEFT JOIN sub_categories sc ON sc.id = p.sub_category_id
                                WHERE p.is_active = true
                                  AND p.id NOT IN %s
                                ORDER BY p.stock_quantity DESC, p.id ASC
                                LIMIT %s;
                            """, (excluded_ids, still_needed))
                            fill_rows = cur.fetchall()
                            for r in fill_rows:
                                seen_ids.add(r["id"])
                                results.append({
                                    "id": r["id"],
                                    "name": r["name"],
                                    "sku": r["sku"],
                                    "category_id": r["category_id"],
                                    "sub_category_id": r.get("sub_category_id"),
                                    "sub_category_name": r.get("sub_category_name", ""),
                                    "price": float(r["price"]),
                                    "currency": r.get("currency", "IDR"),
                                    "image_url": r.get("image_url", ""),
                                    "stock_quantity": int(r.get("stock_quantity") or 0),
                                    "badge": r.get("badge", ""),
                                    "description": r.get("description", ""),
                                    "score": float(r["score"]),
                                    "reason": "catalog_fallback"
                                })

            return results
        except Exception as e:
            logger.error(f"Error computing frequently bought together for #{product_id}: {e}", exc_info=True)
            return []

