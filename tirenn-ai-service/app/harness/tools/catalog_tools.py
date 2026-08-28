import logging
from typing import Dict, Any, Optional, List
from app.harness.tools.base import BaseTool
from app.core.config import settings
from app.repositories.product_repository import ProductRepository
from app.usecases.search_usecase import SearchUseCase

logger = logging.getLogger("ai-service.harness.tools.catalog")

def normalize_price(val: Optional[float]) -> Optional[float]:
    """Smart price normalizer: converts USD or shorthand thousands to Rupiah"""
    if val is None:
        return None
    try:
        val = float(val)
    except (ValueError, TypeError):
        return None
    if val <= 0:
        return None
    if val < 500:
        return val * 16000.0
    if 500 <= val < 1000:
        return val * 1000.0
    return val

class SearchProductsTool(BaseTool):
    """Tool for searching product catalog using semantic search and attribute filtering"""

    name = "search_products"
    description = "Search product catalog using semantic similarity, keyword matching, category, and price filters."

    def __init__(self, product_repo: ProductRepository, search_usecase: SearchUseCase):
        self.product_repo = product_repo
        self.search_usecase = search_usecase
        self._update_schema()

    def _update_schema(self):
        cats = self.product_repo.get_categories_map()
        cat_desc_list = [f"{k}: {v}" for k, v in cats.items()]
        cat_desc = ", ".join(cat_desc_list) if cat_desc_list else "1: Electronics, 2: Men Fashion, 3: Women Fashion, 4: Food & Drink, 5: Beauty"

        sub_cats = self.product_repo.get_sub_categories_map()
        sub_desc_list = [f"{k}: {v}" for k, v in sub_cats.items()]
        sub_desc = ", ".join(sub_desc_list) if sub_desc_list else "Sub-Category ID"

        self.parameters_schema = {
            "type": "object",
            "properties": {
                "query": {
                    "type": "string",
                    "description": "Search query or product description (e.g. 'wireless headphones', 'celana panjang pria', 'tas selempang wanita', 'kopi arabika'). Required."
                },
                "min_price": {
                    "type": "number",
                    "description": "Minimum price in IDR or USD. Empty if no lower bound."
                },
                "max_price": {
                    "type": "number",
                    "description": "Maximum price in IDR or USD. Empty if no upper bound."
                },
                "in_stock": {
                    "type": "boolean",
                    "description": "Filter only products with available stock (true/false)."
                },
                "category_id": {
                    "type": "integer",
                    "description": f"Main Category ID ({cat_desc})."
                },
                "sub_category_id": {
                    "type": "integer",
                    "description": f"Sub-Category ID ({sub_desc})."
                }
            },
            "required": ["query"]
        }

    async def execute(self, args: Dict[str, Any], context: Optional[Dict[str, Any]] = None) -> Dict[str, Any]:
        query = (args.get("query") or "").strip()
        cat_id = int(args.get("category_id") or 0)
        sub_cat_id = int(args.get("sub_category_id") or 0)
        min_price = normalize_price(args.get("min_price"))
        max_price = normalize_price(args.get("max_price"))
        in_stock = args.get("in_stock")

        cats = self.product_repo.get_categories_map()
        cat_name = cats.get(cat_id, "all")

        logger.info(
            f"🔎 [TOOL: search_products] query='{query}' | "
            f"category_id={cat_id} ({cat_name}) | "
            f"sub_category_id={sub_cat_id} | "
            f"min_price={min_price} | max_price={max_price} | in_stock={in_stock}"
        )

        results = self.search_usecase.execute(
            query=query,
            limit=settings.CHAT_SEARCH_LIMIT,
            category_id=cat_id,
            sub_category_id=sub_cat_id,
            score_threshold=settings.CHAT_SEARCH_SCORE_THRESHOLD,
            min_price=min_price,
            max_price=max_price,
            in_stock=in_stock
        )

        # Multi-stage adaptive fallback for broad queries
        if not results:
            results = self.search_usecase.execute(
                query=query,
                limit=settings.CHAT_SEARCH_LIMIT,
                category_id=cat_id,
                sub_category_id=sub_cat_id,
                score_threshold=settings.CHAT_SEARCH_FALLBACK_THRESHOLD,
                min_price=min_price,
                max_price=max_price,
                in_stock=in_stock
            )

        if not results and cat_id > 0:
            results = self.search_usecase.execute(
                query=cat_name if cat_name != "all" else query,
                limit=15,
                category_id=cat_id,
                sub_category_id=sub_cat_id,
                score_threshold=0.0,
                min_price=min_price,
                max_price=max_price,
                in_stock=in_stock
            )

        formatted = [
            {
                "id": r.id,
                "name": r.name,
                "sku": r.sku,
                "category_id": r.category_id,
                "sub_category_id": r.sub_category_id,
                "sub_category_name": r.sub_category_name,
                "price": r.price,
                "currency": r.currency or ("USD" if r.sku.startswith("EN-") else "IDR"),
                "image_url": r.image_url,
                "stock_quantity": r.stock_quantity,
                "in_stock": r.stock_quantity > 0,
                "score": round(r.score, 3)
            }
            for r in results
        ]

        llm_products = [
            {
                "id": p["id"],
                "name": p["name"],
                "sku": p["sku"],
                "price": p["price"],
                "currency": p["currency"],
                "stock_quantity": p["stock_quantity"],
                "in_stock": p["in_stock"]
            }
            for p in formatted
        ]

        return {
            "found_count": len(formatted),
            "products": llm_products,
            "_raw_query": query,
            "_full_products": formatted
        }


class CheckProductStockTool(BaseTool):
    """Tool for checking real-time inventory and pricing of specific product"""

    name = "check_product_stock"
    description = "Check real-time stock quantity and price of a product by name, keyword, or SKU."
    parameters_schema = {
        "type": "object",
        "properties": {
            "product_name_or_query": {
                "type": "string",
                "description": "Product name or SKU to check (e.g. 'AuraSound', 'AUD-001', 'Kopi Gayo')."
            },
            "product_id": {
                "type": "integer",
                "description": "Product ID if known."
            }
        }
    }

    def __init__(self, product_repo: ProductRepository, search_usecase: SearchUseCase):
        self.product_repo = product_repo
        self.search_usecase = search_usecase

    async def execute(self, args: Dict[str, Any], context: Optional[Dict[str, Any]] = None) -> Dict[str, Any]:
        p_id = args.get("product_id")
        query = (args.get("product_name_or_query") or "").strip()

        logger.info(f"📦 [TOOL: check_product_stock] product_name_or_query='{query}' | product_id={p_id}")

        prod = None
        if p_id and int(p_id) <= 2000:
            prod = self.product_repo.get_product_by_id(int(p_id))

        if not prod and query:
            prod = self.product_repo.get_product_by_sku_or_name(query)

        if not prod and query:
            search_res = self.search_usecase.execute(query=query, limit=1, score_threshold=0.10)
            if search_res:
                prod = self.product_repo.get_product_by_id(search_res[0].id)

        if not prod:
            return {"status": "not_found", "message": f"Product '{query or p_id}' not found in catalog."}

        curr = prod.currency or ("USD" if prod.sku.startswith("EN-") else "IDR")

        return {
            "status": "found",
            "id": prod.id,
            "name": prod.name,
            "sku": prod.sku,
            "price": prod.price,
            "currency": curr,
            "stock_quantity": prod.stock_quantity,
            "in_stock": prod.stock_quantity > 0,
            "_full_product": {
                "id": prod.id,
                "name": prod.name,
                "sku": prod.sku,
                "price": prod.price,
                "currency": curr,
                "image_url": prod.image_url,
                "stock_quantity": prod.stock_quantity,
                "in_stock": prod.stock_quantity > 0,
                "score": 1.0
            }
        }
