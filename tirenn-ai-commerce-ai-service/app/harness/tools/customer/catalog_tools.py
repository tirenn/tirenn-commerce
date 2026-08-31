import logging
from typing import Dict, Any, Optional, List, Tuple
from app.harness.tools.base import BaseTool
from app.core.config import settings
from app.repositories.product_repository import ProductRepository
from app.usecases.search_usecase import SearchUseCase

logger = logging.getLogger("ai-service.harness.tools.customer.catalog")


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

        if len(query) < 3 and cat_id == 0:
            return {
                "status": "insufficient_query",
                "found_count": 0,
                "products": [],
                "_raw_query": query,
                "_full_products": []
            }

        cats = self.product_repo.get_categories_map()
        if cat_id not in cats:
            cat_id = 0
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

        # Fallback relaxing category if LLM provided incorrect category
        if not results and (cat_id > 0 or sub_cat_id > 0):
            results = self.search_usecase.execute(
                query=query,
                limit=settings.CHAT_SEARCH_LIMIT,
                category_id=0,
                sub_category_id=0,
                score_threshold=settings.CHAT_SEARCH_FALLBACK_THRESHOLD,
                min_price=min_price,
                max_price=max_price,
                in_stock=in_stock
            )

        results = results[:settings.CHAT_SEARCH_LIMIT]

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


def _lookup_product_by_sku(
    sku: str,
    product_repo: ProductRepository
) -> Tuple[Optional[Any], str]:
    """Directly looks up a product strictly by SKU."""
    clean_sku = (sku or "").strip()
    if not clean_sku:
        return None, "need_clarification"

    prod = product_repo.get_product_by_sku(clean_sku)
    if prod:
        return prod, "found"

    return None, "not_found"


class GetProductDetailTool(BaseTool):
    """Tool for retrieving full specifications, materials, and features of a specific product by SKU"""

    name = "get_product_detail"
    description = "Get in-depth product details, specifications, materials, features, rating, and value proposition of a specific product by SKU."
    parameters_schema = {
        "type": "object",
        "properties": {
            "sku": {
                "type": "string",
                "description": "Product SKU (e.g. 'ID-AUD-001', 'ID-WCL-001', 'EN-AUD-003')."
            }
        },
        "required": ["sku"]
    }

    def __init__(self, product_repo: ProductRepository, search_usecase: Optional[SearchUseCase] = None):
        self.product_repo = product_repo

    async def execute(self, args: Dict[str, Any], context: Optional[Dict[str, Any]] = None) -> Dict[str, Any]:
        sku = (args.get("sku") or "").strip()
        logger.info(f"📖 [TOOL: get_product_detail] sku='{sku}'")

        if not sku:
            return {"status": "need_clarification"}

        prod, status = _lookup_product_by_sku(sku=sku, product_repo=self.product_repo)
        if status == "need_clarification":
            return {"status": "need_clarification"}
        if status == "not_found" or not prod:
            return {"status": "not_found", "sku": sku}

        curr = prod.currency or ("USD" if prod.sku.startswith("EN-") else "IDR")
        category_name = prod.category_name if hasattr(prod, "category_name") and prod.category_name else "Umum"
        sub_category_name = prod.sub_category_name if hasattr(prod, "sub_category_name") and prod.sub_category_name else ""

        return {
            "status": "found",
            "id": prod.id,
            "name": prod.name,
            "sku": prod.sku,
            "category": category_name,
            "sub_category": sub_category_name,
            "description": prod.description,
            "price": prod.price,
            "currency": curr,
            "rating": getattr(prod, "rating", 4.8),
            "badge": getattr(prod, "badge", ""),
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


class GetProductStockTool(BaseTool):
    """Tool for checking real-time inventory and pricing of specific product by SKU"""

    name = "get_product_stock"
    description = "Check real-time stock quantity, stock status (ready/low/out of stock), and price of a product by SKU."
    parameters_schema = {
        "type": "object",
        "properties": {
            "sku": {
                "type": "string",
                "description": "Product SKU to check (e.g. 'ID-AUD-001', 'ID-WCL-001', 'EN-AUD-003')."
            }
        },
        "required": ["sku"]
    }

    def __init__(self, product_repo: ProductRepository, search_usecase: Optional[SearchUseCase] = None):
        self.product_repo = product_repo

    async def execute(self, args: Dict[str, Any], context: Optional[Dict[str, Any]] = None) -> Dict[str, Any]:
        sku = (args.get("sku") or "").strip()
        logger.info(f"📦 [TOOL: get_product_stock] sku='{sku}'")

        if not sku:
            return {"status": "need_clarification"}

        prod, status = _lookup_product_by_sku(sku=sku, product_repo=self.product_repo)
        if status == "need_clarification":
            return {"status": "need_clarification"}
        if status == "not_found" or not prod:
            return {"status": "not_found", "sku": sku}

        curr = prod.currency or ("USD" if prod.sku.startswith("EN-") else "IDR")
        stock_status = "ready_stock"
        if prod.stock_quantity <= 0:
            stock_status = "out_of_stock"
        elif prod.stock_quantity <= 5:
            stock_status = "low_stock"

        return {
            "status": "found",
            "id": prod.id,
            "name": prod.name,
            "sku": prod.sku,
            "price": prod.price,
            "currency": curr,
            "stock_quantity": prod.stock_quantity,
            "stock_status": stock_status,
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
