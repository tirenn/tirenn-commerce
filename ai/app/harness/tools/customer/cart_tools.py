import logging
from typing import Dict, Any, Optional, List
from app.harness.tools.base import BaseTool
from app.repositories.product_repository import ProductRepository
from app.usecases.search_usecase import SearchUseCase

logger = logging.getLogger("ai-service.harness.tools.customer.cart")


class AddToCartTool(BaseTool):
    """Tool for adding items to shopping cart for both guests and authenticated users by SKU"""

    name = "add_to_cart"
    description = "Add a selected product to customer's shopping cart. Pass product SKU and quantity."
    parameters_schema = {
        "type": "object",
        "properties": {
            "sku": {
                "type": "string",
                "description": "Product SKU to add to cart (e.g. 'ID-AUD-001', 'ID-WCL-001', 'EN-AUD-003')."
            },
            "qty": {
                "type": "integer",
                "description": "Quantity to add to cart (default: 1)."
            }
        },
        "required": ["sku"]
    }

    def __init__(self, product_repo: ProductRepository, search_usecase: Optional[SearchUseCase] = None):
        self.product_repo = product_repo

    async def execute(self, args: Dict[str, Any], context: Optional[Dict[str, Any]] = None) -> Dict[str, Any]:
        sku = (args.get("sku") or "").strip()
        qty = int(args.get("qty") or 1)
        if qty <= 0:
            qty = 1

        context = context or {}
        is_authenticated = context.get("is_authenticated", False)

        logger.info(
            f"🛒 [TOOL: add_to_cart] sku='{sku}' | qty={qty} | is_authenticated={is_authenticated}"
        )

        if not sku:
            return {"action": "need_clarification"}

        # Direct SQL lookup strictly by SKU
        prod = self.product_repo.get_product_by_sku(sku)
        if not prod:
            return {"action": "not_found", "sku": sku}

        # Server-side stock re-validation
        if prod.stock_quantity <= 0:
            return {
                "action": "out_of_stock",
                "id": prod.id,
                "sku": prod.sku,
                "name": prod.name,
                "stock_quantity": 0
            }

        actual_qty = min(qty, prod.stock_quantity)
        curr = prod.currency or ("USD" if prod.sku.startswith("EN-") else "IDR")

        return {
            "action": "cart_added",
            "id": prod.id,
            "name": prod.name,
            "sku": prod.sku,
            "price": prod.price,
            "currency": curr,
            "quantity": actual_qty,
            "stock_quantity": prod.stock_quantity,
            "product": {
                "id": prod.id,
                "name": prod.name,
                "sku": prod.sku,
                "price": prod.price,
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


class ViewCartTool(BaseTool):
    """Tool for viewing items currently in customer's shopping cart"""

    name = "view_cart"
    description = "View the current items, quantities, and status of customer's shopping cart."
    parameters_schema = {
        "type": "object",
        "properties": {}
    }

    def __init__(self, product_repo: ProductRepository):
        self.product_repo = product_repo

    async def execute(self, args: Dict[str, Any], context: Optional[Dict[str, Any]] = None) -> Dict[str, Any]:
        context = context or {}
        cart_items = context.get("cart_items", [])

        if not cart_items:
            return {
                "status": "empty",
                "item_count": 0,
                "items": []
            }

        return {
            "status": "active",
            "item_count": len(cart_items),
            "items": cart_items
        }
