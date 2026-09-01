import logging
from typing import Dict, Any, Optional
from app.harness.tools.base import BaseTool
from app.repositories.product_repository import ProductRepository

logger = logging.getLogger("ai-service.harness.tools.admin.inventory")


class GetLowStockProductsTool(BaseTool):
    """Admin Tool for checking inventory items running low on stock via direct SQL"""

    name = "get_low_stock_products"
    description = "Retrieve list of products with low stock levels (below specified threshold, default <= 10 units) directly from database to alert store manager for restocking."
    parameters_schema = {
        "type": "object",
        "properties": {
            "threshold": {
                "type": "integer",
                "description": "Stock quantity ceiling threshold to consider as low stock (default: 10)."
            },
            "limit": {
                "type": "integer",
                "description": "Maximum number of low-stock items to retrieve (default: 20)."
            }
        }
    }

    def __init__(self, product_repo: ProductRepository):
        self.product_repo = product_repo

    async def execute(self, args: Dict[str, Any], context: Optional[Dict[str, Any]] = None) -> Dict[str, Any]:
        threshold = int(args.get("threshold") or 10)
        limit = int(args.get("limit") or 20)

        logger.info(f"⚠️ [ADMIN TOOL: get_low_stock_products] threshold={threshold} | limit={limit} (direct SQL)")

        items = self.product_repo.get_low_stock_products(threshold=threshold, limit=limit)

        return {
            "status": "success",
            "threshold": threshold,
            "low_stock_count": len(items),
            "products": items
        }


class AdjustProductStockTool(BaseTool):
    """Admin Tool for adjusting product inventory stock (ADD, SUBTRACT, SET) directly via atomic SQL transaction with 2-step confirmation guardrail"""

    name = "adjust_product_stock"
    description = "Adjust product inventory stock in database with guardrail confirmation. Specify product SKU (or ID), adjustment type ('ADD', 'SUBTRACT', 'SET'), quantity amount, audit reason, and 'confirmed' boolean flag."
    parameters_schema = {
        "type": "object",
        "properties": {
            "sku": {
                "type": "string",
                "description": "Product SKU to adjust (e.g. 'ID-AUD-001', 'EN-AUD-002')."
            },
            "product_id": {
                "type": "integer",
                "description": "Product ID (optional if SKU is provided)."
            },
            "type": {
                "type": "string",
                "enum": ["ADD", "SUBTRACT", "SET"],
                "description": "Adjustment type: 'ADD' to add units, 'SUBTRACT' to remove units, 'SET' to override exact stock quantity. Required."
            },
            "amount": {
                "type": "integer",
                "description": "Quantity amount to add, subtract, or set (must be >= 0). Required."
            },
            "reason": {
                "type": "string",
                "description": "Official audit reason for the inventory adjustment (e.g. 'Restock PT Jaya', 'Damaged goods', 'Physical audit adjustment'). Required."
            },
            "confirmed": {
                "type": "boolean",
                "description": "CRITICAL GUARDRAIL: Set to true ONLY when the admin has explicitly confirmed/approved the proposed stock change. If false (or unconfirmed), the tool returns a change preview and blocks execution until confirmed."
            }
        },
        "required": ["type", "amount", "reason"]
    }

    def __init__(self, product_repo: ProductRepository):
        self.product_repo = product_repo

    async def execute(self, args: Dict[str, Any], context: Optional[Dict[str, Any]] = None) -> Dict[str, Any]:
        sku = (args.get("sku") or "").strip()
        product_id = args.get("product_id")
        adj_type = (args.get("type") or "ADD").upper().strip()
        amount = abs(int(args.get("amount") or 0))
        reason = (args.get("reason") or "AI Admin Copilot Adjustment").strip()
        confirmed = bool(args.get("confirmed", False))

        context = context or {}
        admin_id = context.get("admin_id", 1)

        logger.info(f"⚡ [ADMIN TOOL: adjust_product_stock] sku='{sku}' | id={product_id} | type={adj_type} | amount={amount} | confirmed={confirmed} | reason='{reason}' (direct SQL)")

        if adj_type not in ["ADD", "SUBTRACT", "SET"]:
            return {"status": "error", "message": "Invalid adjustment type. Must be 'ADD', 'SUBTRACT', or 'SET'."}

        if amount == 0 and adj_type != "SET":
            return {"status": "error", "message": "Amount must be greater than 0 for ADD or SUBTRACT operations."}

        target_id = product_id
        target_prod = None

        if sku:
            target_prod = self.product_repo.get_product_by_sku(sku)
            if target_prod:
                target_id = target_prod.id
        elif product_id:
            target_prod = self.product_repo.get_product_by_id(product_id)

        if not target_id or not target_prod:
            return {"status": "not_found", "message": f"Product with SKU '{sku}' or ID {product_id} was not found."}

        current_stock = target_prod.stock_quantity
        if adj_type == "ADD":
            projected_stock = current_stock + amount
        elif adj_type == "SUBTRACT":
            projected_stock = max(0, current_stock - amount)
        else:  # SET
            projected_stock = amount

        # =====================================================================
        # GUARDRAIL INTERCEPTION: Require Explicit Admin Confirmation
        # =====================================================================
        if not confirmed:
            logger.info(f"🛑 [GUARDRAIL ACTIVE] Blocking unconfirmed stock mutation for SKU '{target_prod.sku}'")
            return {
                "status": "requires_confirmation",
                "guardrail": "CONFIRMATION_REQUIRED",
                "product_id": target_id,
                "product_name": target_prod.name,
                "sku": target_prod.sku,
                "adjustment_type": adj_type,
                "amount": amount,
                "current_stock": current_stock,
                "projected_stock": projected_stock,
                "reason": reason,
                "instruction": "Present these proposed stock adjustment details clearly to the Admin (Product Name, SKU, Operation Type, Current Stock, Projected New Stock, and Audit Reason) and ask for their explicit confirmation before applying changes."
            }

        # =====================================================================
        # CONFIRMED: Execute Direct SQL Stock Adjustment & Audit Logging
        # =====================================================================
        result = self.product_repo.adjust_stock(
            product_id=target_id,
            adjustment_type=adj_type,
            amount=amount,
            reason=reason,
            adjusted_by=admin_id
        )

        if result.get("success"):
            return {
                "status": "success",
                "product_name": target_prod.name,
                "sku": target_prod.sku,
                "previous_stock": result["previous_stock"],
                "current_stock": result["new_stock"],
                "adjustment_type": adj_type,
                "amount": amount,
                "reason": reason,
                "audit_log_id": result.get("audit_log_id")
            }
        else:
            return {
                "status": "error",
                "message": result.get("error", "Failed to adjust stock in database.")
            }
