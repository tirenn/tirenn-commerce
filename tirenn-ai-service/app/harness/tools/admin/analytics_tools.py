import logging
from typing import Dict, Any, Optional
from app.harness.tools.base import BaseTool
from app.repositories.analytics_repository import AnalyticsRepository

logger = logging.getLogger("ai-service.harness.tools.admin.analytics")


class GetExecutiveDashboardMetricsTool(BaseTool):
    """Admin Tool for fetching real-time executive dashboard KPIs, total revenue, orders, customer volume, and sales metrics via direct SQL"""

    name = "get_executive_dashboard_metrics"
    description = "Retrieve real-time merchant executive dashboard metrics including total revenue, total orders, customer count, low stock warnings, and top-selling products directly from database."
    parameters_schema = {
        "type": "object",
        "properties": {},
        "required": []
    }

    def __init__(self, analytics_repo: AnalyticsRepository):
        self.analytics_repo = analytics_repo

    async def execute(self, args: Dict[str, Any], context: Optional[Dict[str, Any]] = None) -> Dict[str, Any]:
        logger.info("📊 [ADMIN TOOL: get_executive_dashboard_metrics] Fetching live KPIs via direct SQL...")

        summary = self.analytics_repo.get_dashboard_summary()
        top_products = self.analytics_repo.get_top_selling_products(limit=5)
        recent_orders = self.analytics_repo.get_recent_orders(limit=5)

        return {
            "status": "success",
            "dashboard_metrics": {
                "summary": summary,
                "top_selling_products": top_products,
                "recent_orders": recent_orders
            }
        }


class GetRecentOrdersOverviewTool(BaseTool):
    """Admin Tool for listing and inspecting recent customer orders and fulfillment statuses via direct SQL"""

    name = "get_recent_orders_overview"
    description = "List recent customer orders with order numbers, customer names, payment status (PAID, PENDING), shipping status, order totals, and order timestamps directly from database."
    parameters_schema = {
        "type": "object",
        "properties": {
            "limit": {
                "type": "integer",
                "description": "Number of recent orders to inspect (default: 10, max: 50)."
            },
            "status": {
                "type": "string",
                "description": "Optional filter by order status: 'PENDING', 'PAID', 'SHIPPED', 'COMPLETED', 'CANCELLED'."
            }
        }
    }

    def __init__(self, analytics_repo: AnalyticsRepository):
        self.analytics_repo = analytics_repo

    async def execute(self, args: Dict[str, Any], context: Optional[Dict[str, Any]] = None) -> Dict[str, Any]:
        limit = int(args.get("limit") or 10)
        status_filter = (args.get("status") or "").strip() or None

        logger.info(f"📋 [ADMIN TOOL: get_recent_orders_overview] limit={limit} | status={status_filter} (direct SQL)")

        orders = self.analytics_repo.get_recent_orders(limit=limit, status=status_filter)

        return {
            "status": "success",
            "total_returned": len(orders),
            "orders": orders
        }
