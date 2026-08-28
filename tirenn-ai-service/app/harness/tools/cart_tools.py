import logging
from typing import Dict, Any, Optional
from app.harness.tools.base import BaseTool
from app.repositories.product_repository import ProductRepository
from app.usecases.search_usecase import SearchUseCase

logger = logging.getLogger("ai-service.harness.tools.cart")

class AddToCartTool(BaseTool):
    """Tool for adding items to shopping cart for both guests and authenticated users"""

    name = "add_to_cart"
    description = "Add a selected product to customer's shopping cart (works for both guests and logged-in users)."
    parameters_schema = {
        "type": "object",
        "properties": {
            "product_name_or_query": {
                "type": "string",
                "description": "Product name or SKU to add to cart (e.g. 'AuraSound', 'AUD-001')."
            },
            "product_id": {
                "type": "integer",
                "description": "Product ID if known."
            },
            "quantity": {
                "type": "integer",
                "description": "Quantity count (default: 1)."
            }
        }
    }

    def __init__(self, product_repo: ProductRepository, search_usecase: SearchUseCase):
        self.product_repo = product_repo
        self.search_usecase = search_usecase

    async def execute(self, args: Dict[str, Any], context: Optional[Dict[str, Any]] = None) -> Dict[str, Any]:
        p_id = args.get("product_id")
        query = (args.get("product_name_or_query") or "").strip()
        qty = int(args.get("quantity") or 1)
        if qty <= 0:
            qty = 1

        is_authenticated = context.get("is_authenticated", False) if context else False

        logger.info(
            f"🛒 [TOOL: add_to_cart] product_name_or_query='{query}' | "
            f"product_id={p_id} | quantity={qty} | is_authenticated={is_authenticated}"
        )

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
            return {
                "action": "not_found",
                "message": f"Cannot add to cart: Product '{query or p_id}' not found."
            }

        if prod.stock_quantity <= 0:
            return {
                "action": "out_of_stock",
                "message": f"Sorry, '{prod.name}' is currently out of stock."
            }

        actual_qty = min(qty, prod.stock_quantity)
        curr = prod.currency or ("USD" if prod.sku.startswith("EN-") else "IDR")

        return {
            "action": "cart_added",
            "message": f"'{prod.name}' ({actual_qty} pcs) has been successfully added to your shopping cart.",
            "product": {
                "id": prod.id,
                "name": prod.name,
                "sku": prod.sku,
                "price": prod.price,
                "currency": curr,
                "image_url": prod.image_url,
                "stock_quantity": prod.stock_quantity,
                "quantity": actual_qty
            },
            "_full_product": {
                "id": prod.id,
                "name": prod.name,
                "sku": prod.sku,
                "price": prod.price,
                "currency": curr,
                "image_url": prod.image_url,
                "stock_quantity": prod.stock_quantity,
                "in_stock": True,
                "score": 1.0
            }
        }
