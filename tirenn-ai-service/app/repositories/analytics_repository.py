import logging
from typing import List, Optional, Dict, Any
import psycopg2
from psycopg2.extras import RealDictCursor
from app.core.config import settings

logger = logging.getLogger("ai-service.repository.analytics")


class AnalyticsRepository:
    """Enterprise Analytics & Orders Repository executing high-performance direct SQL queries on PostgreSQL"""

    def __init__(self):
        self.db_params = {
            "host": settings.DB_HOST,
            "port": settings.DB_PORT,
            "user": settings.DB_USER,
            "password": settings.DB_PASSWORD,
            "dbname": settings.DB_NAME,
            "connect_timeout": 5
        }

    def _get_connection(self):
        return psycopg2.connect(**self.db_params)

    def get_dashboard_summary(self) -> Dict[str, Any]:
        """Fetch real-time executive dashboard KPIs and financial summary directly from PostgreSQL"""
        summary: Dict[str, Any] = {
            "total_revenue": 0.0,
            "total_orders": 0,
            "total_customers": 0,
            "low_stock_count": 0,
            "pending_orders_count": 0,
        }

        try:
            with self._get_connection() as conn:
                with conn.cursor(cursor_factory=RealDictCursor) as cur:
                    # 1. Total Revenue (excluding cancelled orders)
                    cur.execute("SELECT COALESCE(SUM(total_amount), 0) as total_revenue FROM orders WHERE status != 'CANCELLED'")
                    row = cur.fetchone()
                    if row:
                        summary["total_revenue"] = float(row["total_revenue"])

                    # 2. Total Orders Count
                    cur.execute("SELECT COUNT(*) as total_orders FROM orders")
                    row = cur.fetchone()
                    if row:
                        summary["total_orders"] = int(row["total_orders"])

                    # 3. Total Registered Customers
                    cur.execute("SELECT COUNT(*) as total_customers FROM users WHERE role = 'CUSTOMER'")
                    row = cur.fetchone()
                    if row:
                        summary["total_customers"] = int(row["total_customers"])

                    # 4. Low Stock Products Count
                    cur.execute("SELECT COUNT(*) as low_stock_count FROM products WHERE stock_quantity <= low_stock_threshold AND is_active = true")
                    row = cur.fetchone()
                    if row:
                        summary["low_stock_count"] = int(row["low_stock_count"])

                    # 5. Pending Orders Count
                    cur.execute("SELECT COUNT(*) as pending_orders FROM orders WHERE status = 'PENDING'")
                    row = cur.fetchone()
                    if row:
                        summary["pending_orders_count"] = int(row["pending_orders"])

        except Exception as e:
            logger.error(f"Error fetching dashboard summary from database: {e}", exc_info=True)

        return summary

    def get_top_selling_products(self, limit: int = 5) -> List[Dict[str, Any]]:
        """Fetch top-selling products by units sold directly from order_items in PostgreSQL"""
        query = """
            SELECT 
                oi.product_id,
                oi.product_name,
                oi.product_sku,
                oi.product_image,
                MAX(oi.unit_price) as price,
                SUM(oi.quantity) as total_sold,
                SUM(oi.subtotal) as total_revenue
            FROM order_items oi
            INNER JOIN orders o ON o.id = oi.order_id
            WHERE o.status != 'CANCELLED'
            GROUP BY oi.product_id, oi.product_name, oi.product_sku, oi.product_image
            ORDER BY total_sold DESC
            LIMIT %s;
        """
        try:
            with self._get_connection() as conn:
                with conn.cursor(cursor_factory=RealDictCursor) as cur:
                    cur.execute(query, (limit,))
                    rows = cur.fetchall()
                    return [
                        {
                            "product_id": r["product_id"],
                            "product_name": r["product_name"],
                            "product_sku": r["product_sku"],
                            "product_image": r["product_image"],
                            "price": float(r["price"] or 0),
                            "total_sold": int(r["total_sold"] or 0),
                            "total_revenue": float(r["total_revenue"] or 0),
                        }
                        for r in rows
                    ]
        except Exception as e:
            logger.error(f"Error fetching top selling products from database: {e}", exc_info=True)
            return []

    def get_recent_orders(self, limit: int = 10, status: Optional[str] = None) -> List[Dict[str, Any]]:
        """Fetch recent order fulfillment list directly from PostgreSQL"""
        params: List[Any] = []
        where_clause = ""
        if status:
            where_clause = "WHERE status = %s"
            params.append(status.upper().strip())

        query = f"""
            SELECT 
                id,
                order_number,
                user_id,
                shipping_name as customer_name,
                shipping_phone,
                shipping_address,
                total_amount,
                status,
                payment_method,
                payment_status,
                notes,
                created_at,
                updated_at
            FROM orders
            {where_clause}
            ORDER BY created_at DESC
            LIMIT %s;
        """
        params.append(limit)

        try:
            with self._get_connection() as conn:
                with conn.cursor(cursor_factory=RealDictCursor) as cur:
                    cur.execute(query, tuple(params))
                    rows = cur.fetchall()
                    return [
                        {
                            "id": r["id"],
                            "order_number": r["order_number"],
                            "customer_name": r["customer_name"],
                            "shipping_phone": r["shipping_phone"],
                            "shipping_address": r["shipping_address"],
                            "total_amount": float(r["total_amount"] or 0),
                            "status": r["status"],
                            "payment_method": r["payment_method"],
                            "payment_status": r["payment_status"],
                            "created_at": r["created_at"].isoformat() if r["created_at"] else "",
                        }
                        for r in rows
                    ]
        except Exception as e:
            logger.error(f"Error fetching recent orders from database: {e}", exc_info=True)
            return []
